package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HardenLogin removes managed-service default privilege groups before granting
// the deliberately narrow application role. The operator/migration login is separate.
func HardenLogin(ctx context.Context, db *pgxpool.Pool, user string) error {
	var current string
	if e := db.QueryRow(ctx, `SELECT current_user`).Scan(&current); e != nil {
		return e
	}
	if user == current {
		return fmt.Errorf("application and migration login must differ")
	}
	rows, e := db.Query(ctx, `SELECT parent.rolname FROM pg_auth_members m JOIN pg_roles parent ON parent.oid=m.roleid JOIN pg_roles child ON child.oid=m.member WHERE child.rolname=$1 AND parent.rolname<>'optitrace_app'`, user)
	if e != nil {
		return e
	}
	groups := []string{}
	for rows.Next() {
		var group string
		if e = rows.Scan(&group); e != nil {
			rows.Close()
			return e
		}
		groups = append(groups, group)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return e
	}
	for _, g := range groups {
		if _, e = db.Exec(ctx, "REVOKE "+pgx.Identifier{g}.Sanitize()+" FROM "+pgx.Identifier{user}.Sanitize()); e != nil {
			return e
		}
	}
	if _, e = db.Exec(ctx, "ALTER ROLE "+pgx.Identifier{user}.Sanitize()+" NOCREATEROLE NOCREATEDB"); e != nil {
		return e
	}
	if _, e = db.Exec(ctx, "GRANT optitrace_app TO "+pgx.Identifier{user}.Sanitize()); e != nil {
		return e
	}
	return checkRole(ctx, db, user)
}
func checkRole(ctx context.Context, db *pgxpool.Pool, user string) error {
	var elevated bool
	e := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE pg_has_role($1,oid,'member') AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolname='cloudsqlsuperuser')) OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','p') AND pg_has_role($1,c.relowner,'member'))`, user).Scan(&elevated)
	if e != nil {
		return e
	}
	if elevated {
		return fmt.Errorf("application role %s has elevated privileges or owns application tables", user)
	}
	return nil
}
func CheckRuntimeRole(ctx context.Context, db *pgxpool.Pool) error {
	var user string
	if e := db.QueryRow(ctx, `SELECT current_user`).Scan(&user); e != nil {
		return e
	}
	return checkRole(ctx, db, user)
}
