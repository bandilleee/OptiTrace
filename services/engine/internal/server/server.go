package server

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
	"github.com/optitrace/optitrace/services/engine/internal/storage/dbgen"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type login struct {
	nonce, verifier string
	expires         time.Time
}
type Server struct {
	DB       *pgxpool.Pool
	Engine   *evaluation.Engine
	Identity *identity.Provider
	Origin   string
	secure   bool
}
type principal struct {
	id, csrf, tenantID string
	key                bool
	scopes             []string
}
type tenant struct {
	ID, Name string
	Roles    []string
}

func New(db *pgxpool.Pool, e *evaluation.Engine, p *identity.Provider) *Server {
	return &Server{DB: db, Engine: e, Identity: p, Origin: os.Getenv("APP_ORIGIN"), secure: strings.HasPrefix(os.Getenv("APP_ORIGIN"), "https://")}
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		fail(w, 400, "invalid_request", "Invalid JSON request")
		return false
	}
	if d.Decode(&struct{}{}) != io.EOF {
		fail(w, 400, "invalid_request", "Exactly one JSON object required")
		return false
	}
	return true
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	if os.Getenv("PUBLIC_DEMO_ENABLED") == "true" {
		m.Handle("POST /v1/demo/evaluations", newPublicDemo(s.Engine))
	}
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, c := context.WithTimeout(r.Context(), time.Second)
		defer c()
		if s.DB.Ping(ctx) != nil {
			fail(w, 503, "database_unavailable", "Database unavailable")
			return
		}
		write(w, 200, map[string]string{"status": "ok"})
	})
	m.HandleFunc("GET /v1/auth/login", s.login)
	m.HandleFunc("GET /v1/auth/callback", s.callback)
	m.HandleFunc("GET /v1/auth/me", s.me)
	m.HandleFunc("POST /v1/auth/logout", s.logout)
	m.HandleFunc("/v1/tenants/{tenant}/{resource}", s.resource)
	m.HandleFunc("/v1/tenants/{tenant}/{resource}/{id}", s.resource)
	m.HandleFunc("/v1/tenants/{tenant}/{resource}/{id}/{action}", s.resource)
	return otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()
		if requestID == "00000000000000000000000000000000" {
			requestID = identity.Random()
		}
		w.Header().Set("X-Request-Id", requestID)
		defer func() {
			slog.Info("http request", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "elapsed_ms", time.Since(started).Milliseconds())
		}()
		defer func() {
			if x := recover(); x != nil {
				slog.Error("request panic", "error", x)
				fail(w, 500, "internal_error", "Request failed")
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		m.ServeHTTP(w, r)
	}), "optitrace.api")
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	state := identity.Random()
	l := login{identity.Random(), identity.Random(), time.Now().Add(5 * time.Minute)}
	// Direct OIDC validates this nonce against the ID token. Supabase manages
	// its upstream OIDC nonce/state; our local transaction uses state + PKCE,
	// retaining the nonce column only for compatibility with the direct flow.
	s.DB.Exec(r.Context(), `DELETE FROM login_transactions WHERE expires_at<now()`)
	tag, err := s.DB.Exec(r.Context(), `INSERT INTO login_transactions(hash,nonce,verifier,expires_at) SELECT $1,$2,$3,$4 WHERE (SELECT count(*) FROM login_transactions)<10000`, identity.Hash(state), l.nonce, l.verifier, l.expires)
	if err != nil {
		fail(w, 503, "login_unavailable", "Could not start login")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 429, "login_limit", "Too many pending login transactions")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "optitrace_login", Value: state, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 300})
	http.Redirect(w, r, s.Identity.AuthorizationURL(state, l.nonce, l.verifier), 302)
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	cookie, e := r.Cookie("optitrace_login")
	state := r.URL.Query().Get("state")
	if e != nil || state == "" || cookie.Value != state {
		fail(w, 401, "invalid_login", "Invalid login transaction")
		return
	}
	var l login
	e = s.DB.QueryRow(r.Context(), `DELETE FROM login_transactions WHERE hash=$1 AND expires_at>now() RETURNING nonce,verifier,expires_at`, identity.Hash(state)).Scan(&l.nonce, &l.verifier, &l.expires)
	http.SetCookie(w, &http.Cookie{Name: "optitrace_login", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode})
	if e != nil || time.Now().After(l.expires) {
		fail(w, 401, "invalid_login", "Expired or replayed login")
		return
	}
	a, e := s.Identity.Exchange(r.Context(), r.URL.Query().Get("code"), l.verifier, l.nonce)
	if e != nil {
		slog.Warn("identity login rejected")
		fail(w, 401, "invalid_identity", "Identity verification failed")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		fail(w, 503, "database_unavailable", "Database unavailable")
		return
	}
	defer tx.Rollback(r.Context())
	var existing bool
	e = tx.QueryRow(r.Context(), `SELECT auth_accept($1,$2)`, a.ID, a.Email).Scan(&existing)
	if e != nil || !existing {
		fail(w, 403, "invitation_required", "An invitation is required")
		return
	}
	secret, csrf := identity.Random(), identity.Random()
	if e == nil {
		e = dbgen.New(tx).SaveSession(r.Context(), dbgen.SaveSessionParams{Hash: identity.Hash(secret), UserID: pgtype.Text{String: a.ID, Valid: true}, Csrf: csrf, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(8 * time.Hour), Valid: true}, ProviderToken: a.Token})
	}
	if e != nil || tx.Commit(r.Context()) != nil {
		fail(w, 500, "session_failed", "Could not establish session")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "optitrace_session", Value: secret, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 28800})
	http.Redirect(w, r, s.Origin, 302)
}
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) (principal, bool) {
	p := principal{}
	if token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); token != "" && token != r.Header.Get("Authorization") {
		e := s.DB.QueryRow(r.Context(), `SELECT id,tenant_id,scopes FROM auth_credential($1)`, identity.Hash(token)).Scan(&p.id, &p.tenantID, &p.scopes)
		if e != nil {
			fail(w, 401, "invalid_credential", "Invalid credential")
			return p, false
		}
		p.key = true
		return p, true
	}
	c, e := r.Cookie("optitrace_session")
	if e != nil {
		fail(w, 401, "unauthenticated", "Sign in required")
		return p, false
	}
	session, e := dbgen.New(s.DB).GetSession(r.Context(), identity.Hash(c.Value))
	if e != nil {
		fail(w, 401, "unauthenticated", "Session expired")
		return p, false
	}
	p.id, p.csrf = session.UserID, session.Csrf
	token, checked := session.ProviderToken, session.ProviderChecked.Time
	if time.Since(checked) >= 5*time.Minute {
		if !s.Identity.Enabled(r.Context(), token, p.id) {
			s.DB.Exec(r.Context(), `DELETE FROM sessions WHERE hash=$1`, identity.Hash(c.Value))
			fail(w, 401, "account_disabled", "Identity unavailable or disabled; sign in again")
			return p, false
		}
		s.DB.Exec(r.Context(), `UPDATE sessions SET provider_checked=now() WHERE hash=$1`, identity.Hash(c.Value))
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if r.Header.Get("Origin") != s.Origin || r.Header.Get("X-CSRF-Token") != p.csrf {
			fail(w, 403, "csrf_failed", "Valid origin and CSRF token required")
			return p, false
		}
	}
	return p, true
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	if p.key {
		fail(w, 403, "session_required", "Browser session required")
		return
	}
	var email string
	if s.DB.QueryRow(r.Context(), `SELECT email FROM users WHERE id=$1`, p.id).Scan(&email) != nil {
		fail(w, 401, "unknown_user", "Unknown user")
		return
	}
	rows, e := s.DB.Query(r.Context(), `SELECT id,name,roles FROM auth_memberships($1) ORDER BY id`, p.id)
	if e != nil {
		fail(w, 500, "database_error", "Could not read memberships")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var t tenant
		rows.Scan(&t.ID, &t.Name, &t.Roles)
		items = append(items, map[string]any{"id": t.ID, "name": t.Name, "roles": t.Roles})
	}
	write(w, 200, map[string]any{"user": map[string]string{"id": p.id, "email": email}, "tenants": items, "csrf_token": p.csrf})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	if p.key {
		fail(w, 403, "session_required", "Browser session required")
		return
	}
	c, _ := r.Cookie("optitrace_session")
	s.DB.Exec(r.Context(), `DELETE FROM sessions WHERE hash=$1`, identity.Hash(c.Value))
	http.SetCookie(w, &http.Cookie{Name: "optitrace_session", Value: "", Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(204)
}
func has(values []string, wants ...string) bool {
	for _, v := range values {
		for _, want := range wants {
			if v == want {
				return true
			}
		}
	}
	return false
}
func (s *Server) resource(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	tid, kind := r.PathValue("tenant"), r.PathValue("resource")
	roles := []string{}
	if p.key {
		if p.tenantID != tid {
			fail(w, 403, "forbidden", "Tenant access denied")
			return
		}
		if kind != "evaluations" || !has(p.scopes, "evaluations:read") && r.Method == "GET" || !has(p.scopes, "evaluations:write") && r.Method != "GET" {
			fail(w, 403, "insufficient_scope", "Credential scope denied")
			return
		}
		roles = []string{"integration_developer"}
	} else if s.DB.QueryRow(r.Context(), `SELECT roles FROM auth_memberships($1) WHERE id=$2`, p.id, tid).Scan(&roles) != nil {
		fail(w, 403, "forbidden", "Tenant access denied")
		return
	}
	allowed := false
	if r.Method == "GET" {
		switch kind {
		case "policies":
			allowed = has(roles, "policy_author", "policy_reviewer", "integration_developer", "auditor")
		case "evaluations", "benchmarks":
			allowed = has(roles, "integration_developer", "auditor")
		case "audit":
			allowed = has(roles, "auditor")
		case "memberships", "invitations", "credentials":
			allowed = has(roles, "tenant_admin")
		}
	} else {
		switch kind {
		case "policies":
			if r.PathValue("action") == "approve" || r.PathValue("action") == "activate" {
				allowed = has(roles, "policy_reviewer")
			} else {
				allowed = has(roles, "policy_author")
			}
		case "evaluations", "benchmarks":
			allowed = has(roles, "integration_developer")
		case "memberships", "invitations", "credentials":
			allowed = has(roles, "tenant_admin")
		}
	}
	if (kind == "memberships" || kind == "invitations" || kind == "credentials") && !has(roles, "tenant_admin") {
		allowed = false
	}
	if !allowed {
		fail(w, 403, "forbidden", "Role does not permit this action")
		return
	}
	// Once an authenticated evaluation starts, finish and persist its billable
	// trace even when the caller disconnects. The service owns a bounded deadline.
	if kind == "evaluations" && r.Method == "POST" {
		executionCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
		defer cancel()
		r = r.WithContext(executionCtx)
	}
	tx, e := storage.Tenant(r.Context(), s.DB, tid)
	if e != nil {
		fail(w, 503, "database_unavailable", "Database unavailable")
		return
	}
	defer tx.Rollback(r.Context())
	switch kind {
	case "policies":
		e = s.policies(w, r, tx, p, tid)
	case "evaluations":
		e = s.evaluations(w, r, tx, p, tid)
	case "benchmarks":
		e = s.benchmarks(w, r, tx, p, tid)
	case "memberships", "invitations", "credentials", "audit":
		e = s.admin(w, r, tx, p, tid, kind)
	default:
		fail(w, 404, "not_found", "Resource not found")
	}
	if e != nil {
		slog.Error("resource failure", "resource", kind, "error", e.Error())
		fail(w, 500, "database_error", "Could not complete request")
	}
}
func audit(ctx context.Context, tx pgx.Tx, tid, actor, action, id string) error {
	return dbgen.New(tx).AuditEvent(ctx, dbgen.AuditEventParams{TenantID: tid, ActorID: actor, Action: action, ResourceID: id})
}
func persist(ctx context.Context, tx pgx.Tx, id, tid, kind string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return dbgen.New(tx).InsertResource(ctx, dbgen.InsertResourceParams{ID: id, TenantID: tid, Kind: kind, Data: b})
}
func load(ctx context.Context, tx pgx.Tx, kind, id string) (map[string]any, error) {
	b, e := dbgen.New(tx).GetResource(ctx, dbgen.GetResourceParams{ID: id, Kind: kind})
	v := map[string]any{}
	if e == nil {
		e = json.Unmarshal(b, &v)
	}
	return v, e
}
func list(ctx context.Context, tx pgx.Tx, kind string) ([]json.RawMessage, error) {
	rows, e := dbgen.New(tx).ListResources(ctx, kind)
	if e != nil {
		return nil, e
	}
	items := []json.RawMessage{}
	for _, b := range rows {
		items = append(items, json.RawMessage(b))
	}
	return items, nil
}
func commitWrite(w http.ResponseWriter, r *http.Request, tx pgx.Tx, status int, v any) error {
	if e := tx.Commit(r.Context()); e != nil {
		return e
	}
	write(w, status, v)
	return nil
}
