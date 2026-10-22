package identity

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOIDCExchangeSecurity(t *testing.T) {
	var mock *Mock
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mock.Handler().ServeHTTP(w, r) }))
	defer h.Close()
	var e error
	mock, e = NewMock(h.URL)
	if e != nil {
		t.Fatal(e)
	}
	t.Setenv("AUTH_MODE", "mock")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_ORIGIN", "http://localhost:3000")
	t.Setenv("API_PUBLIC_URL", "http://localhost:3000/api")
	t.Setenv("MOCK_ISSUER", h.URL)
	t.Setenv("MOCK_INTERNAL_URL", "")
	p, e := New(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	code := func(verifier, nonce string) string {
		t.Helper()
		v := url.Values{"client_id": {"optitrace-local"}, "redirect_uri": {p.OAuth.RedirectURL}, "code_challenge_method": {"S256"}, "code_challenge": {Hash(verifier)}, "nonce": {nonce}, "email": {"admin@optitrace.test"}}
		c := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		r, e := c.Get(h.URL + "/authorize?" + v.Encode())
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		u, e := url.Parse(r.Header.Get("Location"))
		if e != nil {
			t.Fatal(e)
		}
		return u.Query().Get("code")
	}
	c := code("proof", "nonce")
	a, e := p.Exchange(context.Background(), c, "proof", "nonce")
	if e != nil || a.ID != "seed-admin" {
		t.Fatalf("exchange: %+v %v", a, e)
	}
	if !p.Enabled(context.Background(), a.Token, a.ID) {
		t.Fatal("account lookup failed")
	}
	mock.SetDisabled(a.ID, true)
	if p.Enabled(context.Background(), a.Token, a.ID) {
		t.Fatal("disabled provider account enabled")
	}
	mock.SetDisabled(a.ID, false)
	for name, change := range map[string]func(map[string]any){"issuer": func(c map[string]any) { c["iss"] = "https://wrong.example" }, "audience": func(c map[string]any) { c["aud"] = "wrong-client" }, "expiration": func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }} {
		t.Run(name, func(t *testing.T) {
			parts := strings.Split(mock.jwt("admin@optitrace.test", "nonce"), ".")
			raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
			var claims map[string]any
			json.Unmarshal(raw, &claims)
			change(claims)
			raw, _ = json.Marshal(claims)
			unsigned := parts[0] + "." + base64.RawURLEncoding.EncodeToString(raw)
			hash := sha256.Sum256([]byte(unsigned))
			signature, e := rsa.SignPKCS1v15(rand.Reader, mock.key, crypto.SHA256, hash[:])
			if e != nil {
				t.Fatal(e)
			}
			if _, e = p.Verifier.Verify(context.Background(), unsigned+"."+base64.RawURLEncoding.EncodeToString(signature)); e == nil {
				t.Fatal("invalid signed identity accepted")
			}
		})
	}
	if _, e = p.Exchange(context.Background(), c, "proof", "nonce"); e == nil {
		t.Fatal("authorization code replay accepted")
	}
	if _, e = p.Exchange(context.Background(), code("proof", "nonce"), "wrong", "nonce"); e == nil {
		t.Fatal("PKCE mismatch accepted")
	}
	if _, e = p.Exchange(context.Background(), code("proof", "nonce"), "proof", "wrong"); e == nil {
		t.Fatal("nonce mismatch accepted")
	}
	if _, e = p.Verifier.Verify(context.Background(), "forged.token.signature"); e == nil {
		t.Fatal("forged identity accepted")
	}
}
func TestHostedRejectsMock(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_MODE", "mock")
	t.Setenv("APP_ORIGIN", "https://optitrace.example")
	if _, e := New(context.Background()); e == nil {
		t.Fatal("production accepted mock provider")
	}
}
