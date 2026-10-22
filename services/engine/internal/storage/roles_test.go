package storage

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestApplicationRoleHardening(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL required for disposable PostgreSQL role acceptance")
	}
	ctx := context.Background()
	owner, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err = Migrate(ctx, owner); err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	loginName, groupName := "hardening_login_"+suffix, "hardening_group_"+suffix
	loginIdent, groupIdent := pgx.Identifier{loginName}.Sanitize(), pgx.Identifier{groupName}.Sanitize()
	if _, err = owner.Exec(ctx, "CREATE ROLE "+groupIdent+" NOLOGIN CREATEROLE CREATEDB"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, e := owner.Exec(ctx, "DROP ROLE "+groupIdent); e != nil {
			t.Error(e)
		}
	}()
	if _, err = owner.Exec(ctx, "CREATE ROLE "+loginIdent+" LOGIN PASSWORD 'local-test-only' CREATEROLE CREATEDB"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, e := owner.Exec(ctx, "DROP ROLE "+loginIdent); e != nil {
			t.Error(e)
		}
	}()
	if _, err = owner.Exec(ctx, "GRANT "+groupIdent+" TO "+loginIdent); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = url.UserPassword(loginName, "local-test-only")
	app, err := Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if err = CheckRuntimeRole(ctx, app); err == nil {
		t.Fatal("elevated application login accepted")
	}
	if err = HardenLogin(ctx, owner, loginName); err != nil {
		t.Fatal(err)
	}
	if err = CheckRuntimeRole(ctx, app); err != nil {
		t.Fatalf("hardened login rejected: %v", err)
	}
	var unexpected int
	if err = owner.QueryRow(ctx, `SELECT count(*) FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.roleid WHERE m.member=$1::regrole AND r.rolname<>'optitrace_app'`, loginName).Scan(&unexpected); err != nil || unexpected != 0 {
		t.Fatalf("unexpected inherited groups: %d %v", unexpected, err)
	}
	var current string
	if err = owner.QueryRow(ctx, "SELECT current_user").Scan(&current); err != nil {
		t.Fatal(err)
	}
	if err = HardenLogin(ctx, owner, current); err == nil {
		t.Fatal("migration and app login allowed to be identical")
	}
}
