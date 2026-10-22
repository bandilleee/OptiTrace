package server

import (
	"github.com/jackc/pgx/v5"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"net/http"
	"strings"
	"time"
)

var validRoles = []string{"tenant_admin", "policy_author", "policy_reviewer", "integration_developer", "auditor"}

func rolesValid(roles []string) bool {
	if len(roles) == 0 || len(roles) > 5 {
		return false
	}
	seen := map[string]bool{}
	for _, role := range roles {
		if !has(validRoles, role) || seen[role] {
			return false
		}
		seen[role] = true
	}
	return true
}
func (s *Server) admin(w http.ResponseWriter, r *http.Request, tx pgx.Tx, p principal, tid, kind string) error {
	id := r.PathValue("id")
	if r.Method == "GET" {
		items := []map[string]any{}
		query := ""
		switch kind {
		case "memberships":
			query = `SELECT m.id,u.email,m.roles FROM memberships m JOIN users u ON u.id=m.user_id WHERE m.tenant_id=$1`
		case "invitations":
			query = `SELECT id,email,roles FROM invitations WHERE tenant_id=$1 AND accepted_by IS NULL`
		case "credentials":
			query = `SELECT id,name,scopes FROM credentials WHERE tenant_id=$1 AND NOT revoked`
		case "audit":
			query = `SELECT id::text,actor_id,action,resource_id,created_at FROM audit WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 200`
		}
		rows, e := tx.Query(r.Context(), query, tid)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var key, a, b string
			var roles []string
			if kind == "audit" {
				var at time.Time
				if e = rows.Scan(&key, &a, &b, &id, &at); e != nil {
					return e
				}
				items = append(items, map[string]any{"id": key, "actor_id": a, "action": b, "resource_id": id, "created_at": at})
			} else {
				if e = rows.Scan(&key, &a, &roles); e != nil {
					return e
				}
				if kind == "credentials" {
					items = append(items, map[string]any{"id": key, "name": a, "scopes": roles})
				} else {
					items = append(items, map[string]any{"id": key, "email": a, "roles": roles})
				}
			}
		}
		if e = rows.Err(); e != nil {
			return e
		}
		write(w, 200, map[string]any{"items": items})
		return nil
	}
	var out any
	action := ""
	switch {
	case kind == "memberships" && r.Method == "PATCH" && id != "":
		if _, e := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, tid+":memberships"); e != nil {
			return e
		}
		var b struct {
			Roles []string `json:"roles"`
		}
		if !decode(w, r, &b) {
			return nil
		}
		if !rolesValid(b.Roles) {
			fail(w, 400, "invalid_roles", "Valid unique tenant roles required")
			return nil
		}
		var current []string
		var userID string
		if e := tx.QueryRow(r.Context(), `SELECT roles,user_id FROM memberships WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, id, tid).Scan(&current, &userID); e == pgx.ErrNoRows {
			fail(w, 404, "not_found", "Membership not found")
			return nil
		} else if e != nil {
			return e
		}
		if has(current, "tenant_admin") && !has(b.Roles, "tenant_admin") {
			var count int
			if e := tx.QueryRow(r.Context(), `SELECT count(*) FROM memberships WHERE tenant_id=$1 AND 'tenant_admin'=ANY(roles)`, tid).Scan(&count); e != nil {
				return e
			}
			if count <= 1 {
				fail(w, 409, "last_administrator", "Cannot remove the last tenant administrator")
				return nil
			}
		}
		if _, e := tx.Exec(r.Context(), `UPDATE memberships SET roles=$3 WHERE id=$1 AND tenant_id=$2`, id, tid, b.Roles); e != nil {
			return e
		}
		out = map[string]any{"id": id, "roles": b.Roles}
		action = "membership.updated"
	case kind == "invitations" && r.Method == "POST" && id == "":
		var b struct {
			Email string   `json:"email"`
			Roles []string `json:"roles"`
		}
		if !decode(w, r, &b) {
			return nil
		}
		b.Email = strings.ToLower(strings.TrimSpace(b.Email))
		if !invitationEmailValid(b.Email) || !rolesValid(b.Roles) {
			fail(w, 400, "invalid_invitation", "Valid email and roles required")
			return nil
		}
		id = identity.Random()
		_, e := tx.Exec(r.Context(), `INSERT INTO invitations(id,tenant_id,email,roles) VALUES($1,$2,$3,$4)`, id, tid, b.Email, b.Roles)
		if e != nil {
			return e
		}
		out = map[string]any{"id": id, "email": b.Email, "roles": b.Roles}
		action = "invitation.created"
	case kind == "credentials" && r.Method == "POST" && id == "":
		var b struct {
			Name   string   `json:"name"`
			Scopes []string `json:"scopes"`
		}
		if !decode(w, r, &b) {
			return nil
		}
		b.Name = strings.TrimSpace(b.Name)
		if !resourceNameValid(b.Name) || len(b.Scopes) == 0 || len(b.Scopes) > 2 {
			fail(w, 400, "invalid_credential", "Name and scopes required")
			return nil
		}
		seenScopes := map[string]bool{}
		for _, scope := range b.Scopes {
			if seenScopes[scope] || (scope != "evaluations:read" && scope != "evaluations:write") {
				fail(w, 400, "invalid_scope", "Valid unique scopes required")
				return nil
			}
			seenScopes[scope] = true
		}
		id = identity.Random()
		secret := "ot_" + identity.Random()
		if _, e := tx.Exec(r.Context(), `INSERT INTO credentials(id,tenant_id,name,hash,scopes) VALUES($1,$2,$3,$4,$5)`, id, tid, b.Name, identity.Hash(secret), b.Scopes); e != nil {
			return e
		}
		out = map[string]any{"credential": map[string]any{"id": id, "name": b.Name, "scopes": b.Scopes}, "secret": secret}
		action = "credential.created"
	case kind == "credentials" && r.Method == "DELETE" && id != "":
		tag, e := tx.Exec(r.Context(), `UPDATE credentials SET revoked=true WHERE id=$1 AND tenant_id=$2 AND NOT revoked`, id, tid)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			fail(w, 404, "not_found", "Credential not found")
			return nil
		}
		out = map[string]string{"id": id, "status": "revoked"}
		action = "credential.revoked"
	default:
		fail(w, 405, "method_not_allowed", "Method not allowed")
		return nil
	}
	if e := audit(r.Context(), tx, tid, p.id, action, id); e != nil {
		return e
	}
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	return commitWrite(w, r, tx, status, out)
}
