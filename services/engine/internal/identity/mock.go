package identity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type mockCode struct {
	email, nonce, challenge, redirect string
	expires                           time.Time
}
type Mock struct {
	issuer   string
	key      *rsa.PrivateKey
	mu       sync.Mutex
	codes    map[string]mockCode
	tokens   map[string]string
	disabled map[string]bool
}

func NewMock(issuer string) (*Mock, error) {
	k, e := rsa.GenerateKey(rand.Reader, 2048)
	return &Mock{issuer: issuer, key: k, codes: map[string]mockCode{}, tokens: map[string]string{}, disabled: map[string]bool{}}, e
}
func (m *Mock) SetDisabled(id string, value bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disabled[id] = value
}
func (m *Mock) jwt(email, nonce string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"local","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{"iss": m.issuer, "aud": "optitrace-local", "sub": mockID(email), "email": email, "email_verified": true, "nonce": nonce, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()})
	s := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	h := sha256.Sum256([]byte(s))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, m.key, crypto.SHA256, h[:])
	return s + "." + base64.RawURLEncoding.EncodeToString(sig)
}
func mockID(email string) string { return "seed-" + strings.Split(email, "@")[0] }
func (m *Mock) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		mockJSON(w, map[string]any{"issuer": m.issuer, "authorization_endpoint": m.issuer + "/authorize", "token_endpoint": m.issuer + "/token", "jwks_uri": m.issuer + "/jwks", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}, "code_challenge_methods_supported": []string{"S256"}})
	})
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, r *http.Request) {
		mockJSON(w, map[string]any{"keys": []any{map[string]string{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "local", "n": base64.RawURLEncoding.EncodeToString(m.key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(m.key.E)).Bytes())}}})
	})
	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		redirect, e := url.Parse(q.Get("redirect_uri"))
		if e != nil || redirect.Hostname() != "localhost" && redirect.Hostname() != "127.0.0.1" || q.Get("client_id") != "optitrace-local" || q.Get("code_challenge_method") != "S256" {
			http.Error(w, "invalid local authorization request", 400)
			return
		}
		email := q.Get("email")
		if email == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, "<h1>OptiTrace synthetic identity provider</h1><p>Choose a seeded test identity</p>")
			for _, name := range []string{"admin", "author", "reviewer", "auditor", "outsider", "uninvited"} {
				v := r.URL.Query()
				v.Set("email", name+"@optitrace.test")
				fmt.Fprintf(w, `<p><a href="/authorize?%s">%s@optitrace.test</a></p>`, template.HTMLEscapeString(v.Encode()), name)
			}
			return
		}
		valid := false
		for _, n := range []string{"admin", "author", "reviewer", "auditor", "outsider", "uninvited"} {
			valid = valid || email == n+"@optitrace.test"
		}
		if !valid {
			http.Error(w, "unknown synthetic identity", 400)
			return
		}
		code := Random()
		m.mu.Lock()
		m.codes[code] = mockCode{email, q.Get("nonce"), q.Get("code_challenge"), q.Get("redirect_uri"), time.Now().Add(2 * time.Minute)}
		m.mu.Unlock()
		out := redirect.Query()
		out.Set("code", code)
		out.Set("state", q.Get("state"))
		redirect.RawQuery = out.Encode()
		http.Redirect(w, r, redirect.String(), 302)
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		m.mu.Lock()
		c, ok := m.codes[r.Form.Get("code")]
		delete(m.codes, r.Form.Get("code"))
		m.mu.Unlock()
		id, secret, _ := r.BasicAuth()
		if id == "" {
			id = r.Form.Get("client_id")
			secret = r.Form.Get("client_secret")
		}
		if !ok || time.Now().After(c.expires) || Hash(r.Form.Get("code_verifier")) != c.challenge || r.Form.Get("redirect_uri") != c.redirect || id != "optitrace-local" || secret != "local-secret" {
			http.Error(w, "invalid_grant", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		mockJSON(w, map[string]any{"access_token": Random(), "token_type": "Bearer", "expires_in": 3600, "id_token": m.jwt(c.email, c.nonce)})
	})
	mux.HandleFunc("POST /identity/v1/accounts:signInWithIdp", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PostBody string `json:"postBody"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body) != nil {
			http.Error(w, "invalid body", 400)
			return
		}
		v, _ := url.ParseQuery(body.PostBody)
		parts := strings.Split(v.Get("id_token"), ".")
		if len(parts) != 3 {
			http.Error(w, "invalid id token", 400)
			return
		}
		raw, e := base64.RawURLEncoding.DecodeString(parts[1])
		var claims struct{ Email string }
		if e != nil || json.Unmarshal(raw, &claims) != nil {
			http.Error(w, "invalid claims", 400)
			return
		}
		h := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
		sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
		if rsa.VerifyPKCS1v15(&m.key.PublicKey, crypto.SHA256, h[:], sig) != nil {
			http.Error(w, "bad signature", 400)
			return
		}
		token := Random()
		m.mu.Lock()
		m.tokens[token] = mockID(claims.Email)
		m.mu.Unlock()
		mockJSON(w, map[string]string{"localId": mockID(claims.Email), "email": claims.Email, "idToken": token})
	})
	mux.HandleFunc("POST /identity/v1/accounts:lookup", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Token string `json:"idToken"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		m.mu.Lock()
		id := m.tokens[b.Token]
		disabled := m.disabled[id]
		m.mu.Unlock()
		if id == "" {
			http.Error(w, "invalid token", 401)
			return
		}
		mockJSON(w, map[string]any{"users": []any{map[string]any{"localId": id, "disabled": disabled}}})
	})
	return mux
}
func mockJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
