package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
)

const invitedSupabaseID = "11111111-2222-4333-8444-555555555555"
const outsiderSupabaseID = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"

type supabaseServerCode struct{ challenge, user string }
type supabaseServerDouble struct {
	mu                        sync.Mutex
	codes                     map[string]supabaseServerCode
	adminCalls, tokenCalls    int
	ban, deleted, unavailable bool
}

func (d *supabaseServerDouble) user(id string) map[string]any {
	email := "invited@optitrace.test"
	if id == outsiderSupabaseID {
		email = "not-invited@optitrace.test"
	}
	user := map[string]any{"id": id, "email": email, "aud": "authenticated", "email_confirmed_at": time.Now().Add(-time.Hour).UTC(), "is_anonymous": false, "identities": []any{map[string]any{"provider": "google", "identity_data": map[string]any{"sub": "google-" + id, "email": email, "email_verified": true}}}}
	if d.ban {
		user["banned_until"] = time.Now().Add(time.Hour).UTC()
	}
	return user
}
func (d *supabaseServerDouble) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/auth/v1/authorize":
		q := r.URL.Query()
		if q.Get("provider") != "google" || q.Get("code_challenge_method") != "s256" || q.Get("code_challenge") == "" || q.Get("state") != "" {
			http.Error(w, "invalid authorize", 400)
			return
		}
		callback, _ := url.Parse(q.Get("redirect_to"))
		id := invitedSupabaseID
		if q.Get("test_identity") == "outsider" {
			id = outsiderSupabaseID
		}
		code := identity.Random()
		d.codes[code] = supabaseServerCode{q.Get("code_challenge"), id}
		query := callback.Query()
		query.Set("code", code)
		callback.RawQuery = query.Encode()
		http.Redirect(w, r, callback.String(), 302)
	case r.URL.Path == "/auth/v1/token":
		d.tokenCalls++
		if r.Header.Get("apikey") != "sb_publishable_test" || r.URL.Query().Get("grant_type") != "pkce" {
			http.Error(w, "wrong token credentials", 401)
			return
		}
		var b struct {
			Code     string `json:"auth_code"`
			Verifier string `json:"code_verifier"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		code, ok := d.codes[b.Code]
		if !ok || code.challenge != identity.Hash(b.Verifier) {
			http.Error(w, "wrong PKCE", 400)
			return
		}
		delete(d.codes, b.Code)
		json.NewEncoder(w).Encode(map[string]string{"access_token": "sb-access-" + code.user, "provider_token": "google-access-" + code.user, "token_type": "bearer", "refresh_token": "not-retained"})
	case r.URL.Path == "/auth/v1/user":
		if r.Header.Get("apikey") != "sb_publishable_test" {
			http.Error(w, "wrong project key", 401)
			return
		}
		id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer sb-access-")
		if id != invitedSupabaseID && id != outsiderSupabaseID {
			http.Error(w, "unknown bearer", 401)
			return
		}
		json.NewEncoder(w).Encode(d.user(id))
	case strings.HasPrefix(r.URL.Path, "/auth/v1/admin/users/"):
		d.adminCalls++
		if r.Header.Get("apikey") != "sb_secret_test" || r.Header.Get("Authorization") != "" {
			http.Error(w, "wrong admin key handling", 401)
			return
		}
		if d.deleted {
			http.Error(w, "deleted", 404)
			return
		}
		if d.unavailable {
			http.Error(w, "unavailable", 503)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/auth/v1/admin/users/")
		json.NewEncoder(w).Encode(d.user(id))
	case r.URL.Path == "/google/userinfo":
		id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer google-access-")
		if id != invitedSupabaseID && id != outsiderSupabaseID || r.Header.Get("apikey") != "" {
			http.Error(w, "wrong Google credentials or leaked key", 401)
			return
		}
		email := d.user(id)["email"]
		json.NewEncoder(w).Encode(map[string]any{"sub": "google-" + id, "email": email, "email_verified": true})
	default:
		http.NotFound(w, r)
	}
}

type supabaseServerTransport struct{ googleBase string }

func (t supabaseServerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.String() == "https://openidconnect.googleapis.com/v1/userinfo" {
		copy := r.Clone(r.Context())
		u, _ := url.Parse(t.googleBase + "/google/userinfo")
		copy.URL = u
		return http.DefaultTransport.RoundTrip(copy)
	}
	return http.DefaultTransport.RoundTrip(r)
}

// Uses a separate disposable PostgreSQL database, never the running application
// database. The protocol double implements REST/PKCE, not an auth bypass.
func TestSupabaseBrowserSessionProtocol(t *testing.T) {
	databaseURL := os.Getenv("TEST_SUPABASE_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_SUPABASE_DATABASE_URL required; use an isolated disposable database")
	}
	ctx := context.Background()
	owner, e := storage.Open(ctx, databaseURL)
	if e != nil {
		t.Fatal(e)
	}
	defer owner.Close()
	if e = storage.Migrate(ctx, owner); e != nil {
		t.Fatal(e)
	}
	if _, e = owner.Exec(ctx, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='supabase_auth_test_login') THEN CREATE ROLE supabase_auth_test_login LOGIN PASSWORD 'synthetic-only'; END IF; END $$; GRANT optitrace_app TO supabase_auth_test_login; INSERT INTO tenants(id,name) VALUES('supabase-auth-test','Supabase auth protocol test') ON CONFLICT DO NOTHING; INSERT INTO invitations(id,tenant_id,email,roles) VALUES('supabase-first-invite','supabase-auth-test','invited@optitrace.test',ARRAY['integration_developer','auditor']) ON CONFLICT DO NOTHING;`); e != nil {
		t.Fatal(e)
	}
	parsed, e := url.Parse(databaseURL)
	if e != nil {
		t.Fatal(e)
	}
	parsed.User = url.UserPassword("supabase_auth_test_login", "synthetic-only")
	app, e := storage.Open(ctx, parsed.String())
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	double := &supabaseServerDouble{codes: map[string]supabaseServerCode{}}
	providerAPI := httptest.NewServer(double)
	defer providerAPI.Close()
	s := New(app, nil, nil)
	api := httptest.NewServer(s.Handler())
	defer api.Close()
	s.Origin = api.URL
	t.Setenv("AUTH_MODE", "supabase")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_ORIGIN", api.URL)
	t.Setenv("API_PUBLIC_URL", api.URL)
	t.Setenv("SUPABASE_URL", providerAPI.URL)
	t.Setenv("SUPABASE_PUBLISHABLE_KEY", "sb_publishable_test")
	t.Setenv("SUPABASE_SECRET_KEY", "sb_secret_test")
	s.Identity, e = identity.New(ctx)
	if e != nil {
		t.Fatal(e)
	}
	s.Identity.Client.Transport = supabaseServerTransport{providerAPI.URL}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	get := func(c *http.Client, address string, want int) *http.Response {
		t.Helper()
		res, e := c.Get(address)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			raw, _ := io.ReadAll(res.Body)
			res.Body.Close()
			t.Fatalf("GET %s: %d want%d: %s", strings.Split(address, "?")[0], res.StatusCode, want, raw)
		}
		return res
	}
	start := func(choice string) (string, *http.Cookie) {
		t.Helper()
		res := get(client, api.URL+"/v1/auth/login", 302)
		authorize := res.Header.Get("Location")
		var loginCookie *http.Cookie
		for _, cookie := range res.Cookies() {
			if cookie.Name == "optitrace_login" {
				loginCookie = cookie
			}
		}
		res.Body.Close()
		if loginCookie == nil || !loginCookie.HttpOnly || loginCookie.SameSite != http.SameSiteLaxMode {
			t.Fatal("missing protected login cookie")
		}
		if strings.Contains(authorize, "sb_secret_") {
			t.Fatal("admin secret exposed")
		}
		res = get(client, authorize+"&test_identity="+choice, 302)
		callback := res.Header.Get("Location")
		res.Body.Close()
		return callback, loginCookie
	}
	callback, loginCookie := start("invited")
	bad, _ := url.Parse(callback)
	q := bad.Query()
	q.Set("state", "attacker-state")
	bad.RawQuery = q.Encode()
	get(client, bad.String(), 401).Body.Close()
	noCookies := &http.Client{CheckRedirect: client.CheckRedirect}
	get(noCookies, callback, 401).Body.Close()
	res := get(client, callback, 302)
	var sessionCookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "optitrace_session" {
			sessionCookie = c
		}
	}
	res.Body.Close()
	if sessionCookie == nil || !sessionCookie.HttpOnly || sessionCookie.MaxAge != 28800 || sessionCookie.Value == "" {
		t.Fatal("missing opaque eight-hour session")
	}
	res = get(client, api.URL+"/v1/auth/me", 200)
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	var me struct {
		User    struct{ ID, Email string }
		CSRF    string `json:"csrf_token"`
		Tenants []struct{ ID string }
	}
	if e = json.Unmarshal(raw, &me); e != nil {
		t.Fatal(e)
	}
	if me.User.ID != invitedSupabaseID || me.User.Email != "invited@optitrace.test" || len(me.Tenants) != 1 || me.Tenants[0].ID != "supabase-auth-test" || me.CSRF == "" {
		t.Fatal("stable Supabase UUID membership binding failed")
	}
	if strings.Contains(string(raw), "sb_secret_") || strings.Contains(string(raw), "sb-access-") || strings.Contains(string(raw), "google-access-") {
		t.Fatal("provider credentials exposed to browser")
	}
	var storedHash, providerToken string
	if e = owner.QueryRow(ctx, `SELECT hash,provider_token FROM sessions WHERE hash=$1`, identity.Hash(sessionCookie.Value)).Scan(&storedHash, &providerToken); e != nil {
		t.Fatal(e)
	}
	if storedHash == sessionCookie.Value || providerToken != "" {
		t.Fatal("session secret or provider tokens persisted")
	}
	// Atomically consumed local state prevents replay before any provider call.
	double.mu.Lock()
	calls := double.tokenCalls
	double.mu.Unlock()
	req, _ := http.NewRequest("GET", callback, nil)
	req.AddCookie(loginCookie)
	res, e = noCookies.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 401 {
		t.Fatal("callback state replay accepted")
	}
	double.mu.Lock()
	if double.tokenCalls != calls {
		double.mu.Unlock()
		t.Fatal("replayed callback reached provider token endpoint")
	}
	double.mu.Unlock()
	// An authenticated Supabase/Google account still needs an OptiTrace invitation.
	outsiderClientJar, _ := cookiejar.New(nil)
	savedClient := client
	client = &http.Client{Jar: outsiderClientJar, CheckRedirect: savedClient.CheckRedirect}
	outsider, _ := start("outsider")
	get(client, outsider, 403).Body.Close()
	client = savedClient
	// Credential expiry does not shorten the app session: account status uses a
	// server admin lookup and the database intentionally has no access token.
	if _, e = owner.Exec(ctx, `UPDATE sessions SET provider_checked=now()-interval '6 minutes' WHERE hash=$1`, storedHash); e != nil {
		t.Fatal(e)
	}
	get(client, api.URL+"/v1/auth/me", 200).Body.Close()
	mutation := func(origin, csrf string, want int) {
		t.Helper()
		req, _ := http.NewRequest("POST", api.URL+"/v1/auth/logout", bytes.NewBufferString(`{}`))
		req.Header.Set("Origin", origin)
		req.Header.Set("X-CSRF-Token", csrf)
		res, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("logout status%d want%d", res.StatusCode, want)
		}
	}
	mutation("https://attacker.example", me.CSRF, 403)
	mutation(api.URL, "incorrect-csrf", 403)
	double.mu.Lock()
	double.ban = true
	double.mu.Unlock()
	if _, e = owner.Exec(ctx, `UPDATE sessions SET provider_checked=now()-interval '6 minutes' WHERE hash=$1`, storedHash); e != nil {
		t.Fatal(e)
	}
	get(client, api.URL+"/v1/auth/me", 401).Body.Close()
	var retained bool
	if e = owner.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sessions WHERE hash=$1)`, storedHash).Scan(&retained); e != nil || retained {
		t.Fatal("banned user session was not revoked")
	}
	// A deleted provider user also invalidates a newly established app session.
	double.mu.Lock()
	double.ban = false
	double.mu.Unlock()
	callback, _ = start("invited")
	get(client, callback, 302).Body.Close()
	double.mu.Lock()
	double.deleted = true
	double.mu.Unlock()
	if _, e = owner.Exec(ctx, `UPDATE sessions SET provider_checked=now()-interval '6 minutes' WHERE user_id=$1`, invitedSupabaseID); e != nil {
		t.Fatal(e)
	}
	get(client, api.URL+"/v1/auth/me", 401).Body.Close()

	// Transactions are durable across API instances, not tied to a Go process.
	double.mu.Lock()
	double.deleted = false
	double.mu.Unlock()
	callback, loginCookie = start("invited")
	otherInstance := New(app, nil, s.Identity)
	otherInstance.Origin = api.URL
	otherAPI := httptest.NewServer(otherInstance.Handler())
	defer otherAPI.Close()
	crossInstance, _ := url.Parse(callback)
	otherURL, _ := url.Parse(otherAPI.URL)
	crossInstance.Host = otherURL.Host
	req, _ = http.NewRequest("GET", crossInstance.String(), nil)
	req.AddCookie(loginCookie)
	res, e = noCookies.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 302 {
		t.Fatalf("cross-instance callback failed: %d", res.StatusCode)
	}
	// An expired local state is refused before the one-time provider code is used.
	callback, loginCookie = start("invited")
	if _, e = owner.Exec(ctx, `UPDATE login_transactions SET expires_at=now()-interval '1 minute' WHERE hash=$1`, identity.Hash(loginCookie.Value)); e != nil {
		t.Fatal(e)
	}
	double.mu.Lock()
	calls = double.tokenCalls
	double.mu.Unlock()
	get(client, callback, 401).Body.Close()
	double.mu.Lock()
	if double.tokenCalls != calls {
		double.mu.Unlock()
		t.Fatal("expired callback reached token endpoint")
	}
	double.mu.Unlock()
}
