package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

type supabaseConfig struct{ authURL, publishableKey, secretKey, callback string }
type supabaseIdentity struct {
	Provider string `json:"provider"`
	ID       string `json:"provider_id"`
	Data     struct {
		Subject  string `json:"sub"`
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	} `json:"identity_data"`
}
type supabaseUser struct {
	ID          string             `json:"id"`
	Email       string             `json:"email"`
	Audience    string             `json:"aud"`
	ConfirmedAt *time.Time         `json:"email_confirmed_at"`
	BannedUntil *time.Time         `json:"banned_until"`
	DeletedAt   *time.Time         `json:"deleted_at"`
	Anonymous   bool               `json:"is_anonymous"`
	Identities  []supabaseIdentity `json:"identities"`
}

var supabaseUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func newSupabaseProvider(origin string, production bool) (*Provider, error) {
	base, e := url.Parse(strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"))
	if e != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.Path != "" {
		return nil, errors.New("SUPABASE_URL must be a project origin without a path, credentials, query, or fragment")
	}
	local := isLoopback(base.Hostname())
	if base.Scheme != "https" && !(base.Scheme == "http" && local && !production) || production && local {
		return nil, errors.New("Supabase requires HTTPS; loopback HTTP is allowed only for local development and protocol tests")
	}
	app, e := url.Parse(origin)
	if e != nil || app.Host == "" || app.User != nil || app.RawQuery != "" || app.Fragment != "" || app.Path != "" || app.Scheme != "https" && !(app.Scheme == "http" && isLoopback(app.Hostname()) && !production) {
		return nil, errors.New("APP_ORIGIN must be an HTTPS origin (local HTTP allowed outside production)")
	}
	callbackBase := strings.TrimRight(os.Getenv("API_PUBLIC_URL"), "/")
	if callbackBase == "" {
		callbackBase = strings.TrimRight(origin, "/") + "/api"
	}
	callback, e := url.Parse(callbackBase + "/v1/auth/callback")
	if e != nil || callback.User != nil || callback.RawQuery != "" || callback.Fragment != "" || callback.Scheme != app.Scheme || !strings.EqualFold(callback.Host, app.Host) {
		return nil, errors.New("API_PUBLIC_URL must share APP_ORIGIN for same-origin authentication cookies")
	}
	publicKey, secretKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY"), os.Getenv("SUPABASE_SECRET_KEY")
	if publicKey == "" || secretKey == "" || publicKey == secretKey {
		return nil, errors.New("distinct SUPABASE_PUBLISHABLE_KEY and server-only SUPABASE_SECRET_KEY are required")
	}
	if !strings.HasPrefix(secretKey, "sb_secret_") && !(strings.HasPrefix(secretKey, "eyJ") && strings.Count(secretKey, ".") == 2) {
		return nil, errors.New("SUPABASE_SECRET_KEY must be a server secret key or legacy service_role JWT")
	}
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &Provider{Mode: "supabase", Client: client, OAuth: oauth2.Config{RedirectURL: callback.String()}, supabase: &supabaseConfig{base.String() + "/auth/v1", publicKey, secretKey, callback.String()}}, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (p *Provider) supabaseAuthorizationURL(state, verifier string) string {
	callback, _ := url.Parse(p.supabase.callback)
	query := callback.Query()
	query.Set("state", state)
	callback.RawQuery = query.Encode()
	query = url.Values{"provider": {"google"}, "redirect_to": {callback.String()}, "scopes": {"openid email profile"}, "code_challenge": {Hash(verifier)}, "code_challenge_method": {"s256"}}
	return p.supabase.authURL + "/authorize?" + query.Encode()
}

// Identity is established through authenticated server-to-server /user, never
// from token-response user metadata or unsigned JWT payloads. A Google access
// token from this PKCE flow must also authenticate at Google's fixed UserInfo
// endpoint and match the linked Google subject; an email/password account with
// a historic Google identity alone is insufficient.
func (p *Provider) supabaseExchange(ctx context.Context, code, verifier string) (Account, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if code == "" || len(code) > 4096 || len(verifier) < 43 || len(verifier) > 128 {
		return Account{}, errors.New("invalid Supabase PKCE transaction")
	}
	var token struct {
		AccessToken   string `json:"access_token"`
		ProviderToken string `json:"provider_token"`
		TokenType     string `json:"token_type"`
	}
	if e := p.supabaseRequest(ctx, "POST", "/token?grant_type=pkce", map[string]string{"auth_code": code, "code_verifier": verifier}, "", false, &token); e != nil {
		return Account{}, e
	}
	if token.AccessToken == "" || token.ProviderToken == "" || !strings.EqualFold(token.TokenType, "bearer") {
		return Account{}, errors.New("Supabase Google OAuth credentials missing")
	}
	var user supabaseUser
	if e := p.supabaseRequest(ctx, "GET", "/user", nil, token.AccessToken, false, &user); e != nil {
		return Account{}, e
	}
	if !validSupabaseUser(user) {
		return Account{}, errors.New("Supabase account is not an active confirmed Google account")
	}
	var google struct {
		Subject  string `json:"sub"`
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}
	req, e := http.NewRequestWithContext(ctx, "GET", googleUserInfoURL, nil)
	if e != nil {
		return Account{}, errors.New("Google identity verification unavailable")
	}
	req.Header.Set("Authorization", "Bearer "+token.ProviderToken)
	if e = p.decodeProviderResponse(req, &google); e != nil {
		return Account{}, e
	}
	if google.Subject == "" || !google.Verified || !strings.EqualFold(google.Email, user.Email) {
		return Account{}, errors.New("Google identity does not match the Supabase account")
	}
	matched := false
	for _, id := range user.Identities {
		if id.Provider == "google" && id.Data.Subject == google.Subject && id.Data.Verified && strings.EqualFold(id.Data.Email, google.Email) {
			matched = true
		}
	}
	if !matched {
		return Account{}, errors.New("Google subject does not match the Supabase identity")
	}
	// Recheck ban/deletion through the admin endpoint at login as well as every
	// five minutes thereafter. No Supabase or Google token needs to be persisted.
	if !p.supabaseEnabled(ctx, user.ID) {
		return Account{}, errors.New("Supabase account is unavailable or disabled")
	}
	return Account{ID: user.ID, Email: strings.ToLower(user.Email), Token: ""}, nil
}

func validSupabaseUser(user supabaseUser) bool {
	if !supabaseUUID.MatchString(user.ID) || user.Audience != "authenticated" || user.Anonymous || user.ConfirmedAt == nil || user.ConfirmedAt.IsZero() || user.ConfirmedAt.After(time.Now().Add(time.Minute)) || user.DeletedAt != nil && !user.DeletedAt.IsZero() || user.BannedUntil != nil && user.BannedUntil.After(time.Now()) {
		return false
	}
	address, e := mail.ParseAddress(user.Email)
	if e != nil || address.Address != user.Email {
		return false
	}
	for _, id := range user.Identities {
		if id.Provider == "google" && id.Data.Subject != "" && id.Data.Verified && strings.EqualFold(id.Data.Email, user.Email) {
			return true
		}
	}
	return false
}

func (p *Provider) supabaseEnabled(ctx context.Context, id string) bool {
	if !supabaseUUID.MatchString(id) {
		return false
	}
	var user supabaseUser
	if e := p.supabaseRequest(ctx, "GET", "/admin/users/"+url.PathEscape(id), nil, "", true, &user); e != nil {
		return false
	}
	return user.ID == id && validSupabaseUser(user)
}

func (p *Provider) supabaseRequest(ctx context.Context, method, path string, body any, token string, admin bool, out any) error {
	var reader io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		if e != nil {
			return errors.New("invalid authentication request")
		}
		reader = bytes.NewReader(raw)
	}
	req, e := http.NewRequestWithContext(ctx, method, p.supabase.authURL+path, reader)
	if e != nil {
		return errors.New("invalid authentication endpoint")
	}
	key := p.supabase.publishableKey
	if admin {
		key = p.supabase.secretKey
	}
	req.Header.Set("apikey", key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if admin && !strings.HasPrefix(key, "sb_secret_") {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return p.decodeProviderResponse(req, out)
}

func (p *Provider) decodeProviderResponse(req *http.Request, out any) error {
	response, e := p.Client.Do(req)
	if e != nil {
		return errors.New("identity provider request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("identity provider rejected request (HTTP %d)", response.StatusCode)
	}
	reader := io.LimitReader(response.Body, 1<<20)
	decoder := json.NewDecoder(reader)
	if decoder.Decode(out) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("invalid identity provider response")
	}
	return nil
}
