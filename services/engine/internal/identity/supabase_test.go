package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type supabaseTestTransport struct {
	base      http.RoundTripper
	googleURL string
}

func (t supabaseTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.String() == googleUserInfoURL {
		copy := r.Clone(r.Context())
		u, _ := url.Parse(t.googleURL + "/google/userinfo")
		copy.URL = u
		return t.base.RoundTrip(copy)
	}
	return t.base.RoundTrip(r)
}

type supabaseDouble struct {
	mu                                    sync.Mutex
	server                                *httptest.Server
	user                                  supabaseUser
	googleSubject, googleEmail            string
	googleVerified                        bool
	providerToken                         bool
	userStatus, adminStatus, googleStatus int
	codes                                 map[string]string
	lastVerifier                          string
	adminRequests                         int
	redirect                              string
}

func newSupabaseDouble(t *testing.T) *supabaseDouble {
	t.Helper()
	confirmed := time.Now().Add(-time.Hour).UTC()
	d := &supabaseDouble{user: supabaseUser{ID: "11111111-2222-4333-8444-555555555555", Email: "invited@optitrace.test", Audience: "authenticated", ConfirmedAt: &confirmed}, googleSubject: "google-subject-1", googleEmail: "invited@optitrace.test", googleVerified: true, providerToken: true, userStatus: 200, adminStatus: 200, googleStatus: 200, codes: map[string]string{}}
	googleID := supabaseIdentity{Provider: "google"}
	googleID.Data.Subject = d.googleSubject
	googleID.Data.Email = d.googleEmail
	googleID.Data.Verified = true
	d.user.Identities = []supabaseIdentity{googleID}
	d.server = httptest.NewServer(http.HandlerFunc(d.serve))
	t.Cleanup(d.server.Close)
	return d
}
func (d *supabaseDouble) serve(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if d.redirect != "" {
		http.Redirect(w, r, d.redirect, 302)
		return
	}
	switch r.URL.Path {
	case "/auth/v1/authorize":
		q := r.URL.Query()
		if q.Get("provider") != "google" || q.Get("prompt") != "select_account" || q.Get("code_challenge_method") != "s256" || len(q.Get("code_challenge")) != 43 || q.Has("state") {
			http.Error(w, "invalid provider or upstream-state override", 400)
			return
		}
		callback, e := url.Parse(q.Get("redirect_to"))
		if e != nil || callback.Query().Get("state") == "" {
			http.Error(w, "missing callback state", 400)
			return
		}
		code := Random()
		d.codes[code] = q.Get("code_challenge")
		out := callback.Query()
		out.Set("code", code)
		callback.RawQuery = out.Encode()
		http.Redirect(w, r, callback.String(), 302)
	case "/auth/v1/token":
		if r.Method != "POST" || r.Header.Get("apikey") != "sb_publishable_test" || r.Header.Get("Authorization") != "" || r.URL.Query().Get("grant_type") != "pkce" {
			http.Error(w, "incorrect token credentials", 401)
			return
		}
		var body struct {
			Code     string `json:"auth_code"`
			Verifier string `json:"code_verifier"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		d.lastVerifier = body.Verifier
		challenge, ok := d.codes[body.Code]
		if !ok || challenge != Hash(body.Verifier) {
			http.Error(w, "invalid_grant", 400)
			return
		}
		delete(d.codes, body.Code)
		provider := ""
		if d.providerToken {
			provider = "google-provider-token"
		}
		json.NewEncoder(w).Encode(map[string]any{"access_token": "opaque-supabase-access-token", "provider_token": provider, "refresh_token": "must-never-persist-refresh-token", "token_type": "bearer", "user": map[string]string{"id": "untrusted-token-response-user", "email": "attacker@example.com"}})
	case "/auth/v1/user":
		if r.Header.Get("apikey") != "sb_publishable_test" || r.Header.Get("Authorization") != "Bearer opaque-supabase-access-token" {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.WriteHeader(d.userStatus)
		if d.userStatus == 200 {
			json.NewEncoder(w).Encode(d.user)
		}
	case "/auth/v1/admin/users/11111111-2222-4333-8444-555555555555":
		d.adminRequests++
		if r.Header.Get("apikey") != "sb_secret_test" || r.Header.Get("Authorization") != "" {
			http.Error(w, "admin secret must be on apikey only", 401)
			return
		}
		w.WriteHeader(d.adminStatus)
		if d.adminStatus == 200 {
			json.NewEncoder(w).Encode(d.user)
		}
	case "/google/userinfo":
		if r.Header.Get("Authorization") != "Bearer google-provider-token" || r.Header.Get("apikey") != "" {
			http.Error(w, "wrong Google bearer or leaked Supabase key", 401)
			return
		}
		w.WriteHeader(d.googleStatus)
		if d.googleStatus == 200 {
			json.NewEncoder(w).Encode(map[string]any{"sub": d.googleSubject, "email": d.googleEmail, "email_verified": d.googleVerified})
		}
	default:
		http.NotFound(w, r)
	}
}
func (d *supabaseDouble) provider(t *testing.T) *Provider {
	t.Helper()
	t.Setenv("AUTH_MODE", "supabase")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_ORIGIN", "http://localhost:3000")
	t.Setenv("API_PUBLIC_URL", "http://localhost:3000/api")
	t.Setenv("SUPABASE_URL", d.server.URL)
	t.Setenv("SUPABASE_PUBLISHABLE_KEY", "sb_publishable_test")
	t.Setenv("SUPABASE_SECRET_KEY", "sb_secret_test")
	p, e := New(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	p.Client.Transport = supabaseTestTransport{http.DefaultTransport, d.server.URL}
	return p
}
func (d *supabaseDouble) code(t *testing.T, p *Provider, verifier string) string {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, e := client.Get(p.AuthorizationURL("browser-state", "local-nonce", verifier))
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	if res.StatusCode != 302 {
		t.Fatalf("authorize: %d", res.StatusCode)
	}
	callback, e := url.Parse(res.Header.Get("Location"))
	if e != nil {
		t.Fatal(e)
	}
	if callback.Query().Get("state") != "browser-state" {
		t.Fatal("local transaction state not preserved")
	}
	return callback.Query().Get("code")
}

func TestSupabaseGooglePKCEProtocol(t *testing.T) {
	d := newSupabaseDouble(t)
	p := d.provider(t)
	verifier := Random()
	code := d.code(t, p, verifier)
	account, e := p.Exchange(context.Background(), code, verifier, "local-nonce")
	if e != nil {
		t.Fatal(e)
	}
	if account.ID != d.user.ID || account.Email != d.user.Email || account.Token != "" {
		t.Fatalf("account must come from authenticated /user with no retained provider credentials: %+v", account)
	}
	if d.lastVerifier != verifier || d.adminRequests != 1 {
		t.Fatal("missing PKCE or initial admin recheck")
	}
	if _, e = p.Exchange(context.Background(), code, verifier, ""); e == nil {
		t.Fatal("replayed code accepted")
	}
	if _, e = p.Exchange(context.Background(), d.code(t, p, verifier), Random(), ""); e == nil {
		t.Fatal("wrong PKCE verifier accepted")
	}
	if !p.Enabled(context.Background(), "already-expired-unused-access-token", account.ID) {
		t.Fatal("admin status lookup must outlive access token")
	}
	if p.Enabled(context.Background(), "", "../other-user") {
		t.Fatal("non-UUID admin lookup accepted")
	}
}

func TestSupabaseRejectsUntrustedOrNonGoogleIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*supabaseDouble)
	}{
		{"unconfirmed", func(d *supabaseDouble) { d.user.ConfirmedAt = nil }},
		{"anonymous", func(d *supabaseDouble) { d.user.Anonymous = true }},
		{"wrong audience", func(d *supabaseDouble) { d.user.Audience = "service_role" }},
		{"invalid UUID", func(d *supabaseDouble) { d.user.ID = "untrusted-user" }},
		{"email-only account", func(d *supabaseDouble) { d.user.Identities[0].Provider = "email" }},
		{"missing provider access token", func(d *supabaseDouble) { d.providerToken = false }},
		{"unverified Google identity", func(d *supabaseDouble) { d.user.Identities[0].Data.Verified = false }},
		{"missing explicit identity subject", func(d *supabaseDouble) {
			d.user.Identities[0].ID = d.googleSubject
			d.user.Identities[0].Data.Subject = ""
		}},
		{"different Google subject", func(d *supabaseDouble) { d.googleSubject = "different-google-user" }},
		{"different Google email", func(d *supabaseDouble) { d.googleEmail = "attacker@example.com" }},
		{"unverified Google UserInfo", func(d *supabaseDouble) { d.googleVerified = false }},
		{"invalid bearer", func(d *supabaseDouble) { d.userStatus = 401 }},
		{"Google rejected provider token", func(d *supabaseDouble) { d.googleStatus = 401 }},
		{"deleted", func(d *supabaseDouble) { now := time.Now(); d.user.DeletedAt = &now }},
		{"banned", func(d *supabaseDouble) { future := time.Now().Add(time.Hour); d.user.BannedUntil = &future }},
		{"admin missing user", func(d *supabaseDouble) { d.adminStatus = 404 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := newSupabaseDouble(t)
			p := d.provider(t)
			verifier := Random()
			code := d.code(t, p, verifier)
			test.change(d)
			if _, e := p.Exchange(context.Background(), code, verifier, ""); e == nil {
				t.Fatal("invalid identity accepted")
			}
		})
	}
}

func TestSupabaseAdminStatusFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*supabaseDouble)
	}{{"banned", func(d *supabaseDouble) { future := time.Now().Add(time.Hour); d.user.BannedUntil = &future }}, {"deleted", func(d *supabaseDouble) { now := time.Now(); d.user.DeletedAt = &now }}, {"unconfirmed", func(d *supabaseDouble) { d.user.ConfirmedAt = nil }}, {"missing", func(d *supabaseDouble) { d.adminStatus = 404 }}, {"provider unavailable", func(d *supabaseDouble) { d.adminStatus = 503 }}, {"wrong user", func(d *supabaseDouble) { d.user.ID = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee" }}, {"unlinked Google identity", func(d *supabaseDouble) { d.user.Identities = nil }}} {
		t.Run(test.name, func(t *testing.T) {
			d := newSupabaseDouble(t)
			p := d.provider(t)
			test.change(d)
			if p.Enabled(context.Background(), "ignored", "11111111-2222-4333-8444-555555555555") {
				t.Fatal("disabled/unavailable account permitted")
			}
		})
	}
}

func TestSupabaseCredentialsAreNotForwardedOnRedirect(t *testing.T) {
	d := newSupabaseDouble(t)
	p := d.provider(t)
	requests := 0
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Error("identity credentials forwarded to redirect destination")
	}))
	defer foreign.Close()
	d.redirect = foreign.URL
	if p.Enabled(context.Background(), "", d.user.ID) {
		t.Fatal("redirect accepted")
	}
	if requests != 0 {
		t.Fatal("provider redirect followed")
	}
}

func TestSupabaseHostedConfiguration(t *testing.T) {
	d := newSupabaseDouble(t)
	_ = d.provider(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ORIGIN", "https://optitrace.example")
	t.Setenv("API_PUBLIC_URL", "https://optitrace.example/api")
	if _, e := New(context.Background()); e == nil {
		t.Fatal("hosted loopback Supabase endpoint accepted")
	}
	t.Setenv("SUPABASE_URL", "https://example.supabase.co")
	p, e := New(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	authorization := p.AuthorizationURL("state", "nonce", Random())
	if strings.Contains(authorization, "sb_secret_") || strings.Contains(authorization, "sb_publishable_") {
		t.Fatal("keys leaked in authorization URL")
	}
	t.Setenv("API_PUBLIC_URL", "https://other.example/api")
	if _, e := New(context.Background()); e == nil {
		t.Fatal("cross-origin callback accepted")
	}
	t.Setenv("API_PUBLIC_URL", "https://optitrace.example/api")
	t.Setenv("SUPABASE_SECRET_KEY", "")
	if _, e := New(context.Background()); e == nil {
		t.Fatal("missing admin secret accepted")
	}
}
