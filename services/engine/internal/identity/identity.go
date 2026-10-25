package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func Random() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

type Account struct {
	ID    string
	Email string
	Token string
}
type Provider struct {
	Mode                string
	OAuth               oauth2.Config
	Verifier            *oidc.IDTokenVerifier
	Client              *http.Client
	IdentityURL, APIKey string
	supabase            *supabaseConfig
}
type rewrite struct {
	base               http.RoundTripper
	external, internal string
}

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	c := req.Clone(req.Context())
	u := *req.URL
	c.URL = &u
	if strings.HasPrefix(c.URL.String(), r.external) {
		v, e := url.Parse(strings.Replace(c.URL.String(), r.external, r.internal, 1))
		if e != nil {
			return nil, e
		}
		c.URL = v
	}
	return r.base.RoundTrip(c)
}
func New(ctx context.Context) (*Provider, error) {
	mode := os.Getenv("AUTH_MODE")
	prod := os.Getenv("APP_ENV") == "production"
	origin := os.Getenv("APP_ORIGIN")
	if mode != "google" && mode != "mock" && mode != "supabase" {
		return nil, errors.New("AUTH_MODE must be supabase, google, or mock")
	}
	if prod && (mode == "mock" || !strings.HasPrefix(origin, "https://")) {
		return nil, errors.New("hosted configuration requires a configured auth provider and HTTPS")
	}
	if mode == "supabase" {
		return newSupabaseProvider(origin, prod)
	}
	issuer := "https://accounts.google.com"
	identityURL := "https://identitytoolkit.googleapis.com/v1"
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	secret := os.Getenv("GOOGLE_CLIENT_SECRET")
	client := &http.Client{Timeout: 10 * time.Second}
	if mode == "mock" {
		u, e := url.Parse(origin)
		if e != nil || !(u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
			return nil, errors.New("mock requires localhost application origin")
		}
		issuer = os.Getenv("MOCK_ISSUER")
		if issuer == "" {
			issuer = "http://localhost:8090"
		}
		mockURL, parseErr := url.Parse(issuer)
		if parseErr != nil || mockURL.Hostname() != "localhost" && mockURL.Hostname() != "127.0.0.1" {
			return nil, errors.New("mock issuer must be localhost")
		}
		identityURL = issuer + "/identity/v1"
		clientID = "optitrace-local"
		secret = "local-secret"
		if internal := os.Getenv("MOCK_INTERNAL_URL"); internal != "" {
			client.Transport = rewrite{http.DefaultTransport, issuer, internal}
		}
	}
	if mode == "google" && (clientID == "" || secret == "" || os.Getenv("IDENTITY_PLATFORM_API_KEY") == "" || os.Getenv("GOOGLE_CLOUD_PROJECT") == "") {
		return nil, errors.New("Google client ID, secret, Identity Platform API key, and project ID required")
	}
	ctx = oidc.ClientContext(ctx, client)
	p, e := oidc.NewProvider(ctx, issuer)
	if e != nil {
		return nil, e
	}
	callback := strings.TrimRight(os.Getenv("API_PUBLIC_URL"), "/") + "/v1/auth/callback"
	if os.Getenv("API_PUBLIC_URL") == "" {
		callback = strings.TrimRight(origin, "/") + "/api/v1/auth/callback"
	}
	return &Provider{Mode: mode, OAuth: oauth2.Config{ClientID: clientID, ClientSecret: secret, Endpoint: p.Endpoint(), RedirectURL: callback, Scopes: []string{oidc.ScopeOpenID, "email", "profile"}}, Verifier: p.Verifier(&oidc.Config{ClientID: clientID}), Client: client, IdentityURL: identityURL, APIKey: os.Getenv("IDENTITY_PLATFORM_API_KEY")}, nil
}

// AuthorizationURL keeps Supabase's own upstream OAuth state separate from the
// browser-bound OptiTrace transaction carried in the callback's query string.
func (p *Provider) AuthorizationURL(state, nonce, verifier string) string {
	if p.supabase != nil {
		return p.supabaseAuthorizationURL(state, verifier)
	}
	return p.OAuth.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce), oauth2.S256ChallengeOption(verifier))
}
func (p *Provider) Exchange(ctx context.Context, code, verifier, nonce string) (Account, error) {
	if p.supabase != nil {
		return p.supabaseExchange(ctx, code, verifier)
	}
	ctx = oidc.ClientContext(ctx, p.Client)
	t, e := p.OAuth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if e != nil {
		return Account{}, e
	}
	raw, ok := t.Extra("id_token").(string)
	if !ok {
		return Account{}, errors.New("missing id token")
	}
	id, e := p.Verifier.Verify(ctx, raw)
	if e != nil {
		return Account{}, e
	}
	var claims struct {
		Nonce    string `json:"nonce"`
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}
	if e = id.Claims(&claims); e != nil {
		return Account{}, e
	}
	if claims.Nonce != nonce || !claims.Verified {
		return Account{}, errors.New("nonce or verified email invalid")
	}
	body, _ := json.Marshal(map[string]any{"postBody": url.Values{"id_token": {raw}, "providerId": {"google.com"}}.Encode(), "requestUri": p.OAuth.RedirectURL, "returnSecureToken": true})
	req, _ := http.NewRequestWithContext(ctx, "POST", p.IdentityURL+"/accounts:signInWithIdp?key="+url.QueryEscape(p.APIKey), strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	res, e := p.Client.Do(req)
	if e != nil {
		return Account{}, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return Account{}, fmt.Errorf("identity exchange rejected (%d)", res.StatusCode)
	}
	var account struct {
		LocalID string `json:"localId"`
		Email   string `json:"email"`
		Token   string `json:"idToken"`
	}
	e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&account)
	if e != nil || account.LocalID == "" || account.Email != claims.Email || account.Token == "" {
		return Account{}, errors.New("invalid Identity Platform response")
	}
	return Account{account.LocalID, account.Email, account.Token}, nil
}
func (p *Provider) Enabled(ctx context.Context, token, id string) bool {
	if p.supabase != nil {
		return p.supabaseEnabled(ctx, id)
	}
	client := p.Client
	endpoint := p.IdentityURL + "/accounts:lookup?key=" + url.QueryEscape(p.APIKey)
	payload := any(map[string]string{"idToken": token})
	if os.Getenv("AUTH_MODE") == "google" {
		var e error
		client, e = google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if e != nil {
			return false
		}
		client.Timeout = 10 * time.Second
		endpoint = p.IdentityURL + "/projects/" + url.PathEscape(os.Getenv("GOOGLE_CLOUD_PROJECT")) + "/accounts:lookup"
		payload = map[string]any{"localId": []string{id}}
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	res, e := client.Do(req)
	if e != nil {
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return false
	}
	var out struct {
		Users []struct {
			ID       string `json:"localId"`
			Disabled bool   `json:"disabled"`
		}
	}
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out) != nil {
		return false
	}
	return len(out.Users) == 1 && out.Users[0].ID == id && !out.Users[0].Disabled
}
