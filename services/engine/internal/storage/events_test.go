package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestAppendOnlyEvaluationEvents(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("requires disposable TEST_DATABASE_URL")
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
	suffix := fmt.Sprint(time.Now().UnixNano())
	tenant, other, id, login := "events-"+suffix, "other-events-"+suffix, "attempt-"+suffix, "events_login_"+suffix
	if _, err = owner.Exec(ctx, `INSERT INTO tenants(id,name) VALUES($1,'Synthetic test'),($2,'Separate synthetic test')`, tenant, other); err != nil {
		t.Fatal(err)
	}
	defer func() {
		owner.Exec(ctx, `DELETE FROM evaluation_events WHERE tenant_id=$1`, tenant)
		owner.Exec(ctx, `DELETE FROM resources WHERE id=$1`, id)
		owner.Exec(ctx, `DELETE FROM tenants WHERE id=ANY($1::text[])`, []string{tenant, other})
	}()
	if _, err = owner.Exec(ctx, `INSERT INTO resources(id,tenant_id,kind,data) VALUES($1,$2,'evaluation_attempt','{}')`, id, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = owner.Exec(ctx, "CREATE ROLE "+pgx.Identifier{login}.Sanitize()+" LOGIN PASSWORD 'synthetic-events-only'"); err != nil {
		t.Fatal(err)
	}
	defer owner.Exec(ctx, "DROP ROLE "+pgx.Identifier{login}.Sanitize())
	if _, err = owner.Exec(ctx, "GRANT optitrace_app TO "+pgx.Identifier{login}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(databaseURL)
	parsed.User = url.UserPassword(login, "synthetic-events-only")
	app, err := Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if err = CheckRuntimeRole(ctx, app); err != nil {
		t.Fatal(err)
	}

	// The old resource JSON update takes this row lock. New event inserts must
	// still complete while it is held; increasing a journal timeout cannot pass.
	locked, err := owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Rollback(ctx)
	if _, err = locked.Exec(ctx, `SELECT id FROM resources WHERE id=$1 FOR NO KEY UPDATE`, id); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	errors := make(chan error, 6)
	var wg sync.WaitGroup
	for _, capability := range []string{"number_verification", "sim_swap", "device_swap"} {
		for _, phase := range []string{"starting", "completed"} {
			wg.Add(1)
			go func(capability, phase string) {
				defer wg.Done()
				check, _ := json.Marshal(map[string]string{"capability": capability, "status": "unknown"})
				errors <- AppendEvaluationEvent(bounded, app, tenant, id, capability, phase, check)
			}(capability, phase)
		}
	}
	wg.Wait()
	close(errors)
	for e := range errors {
		if e != nil {
			t.Fatal(e)
		}
	}
	locked.Rollback(ctx)
	for _, invalid := range []string{`{}`, `{"capability":null}`, `{"capability":"device_swap"}`} {
		// Use a valid phase/new capability so phase/unique constraints cannot mask
		// the missing/null/mismatched capability CHECK tested below.
		probeID := id + "-shape"
		if _, err = owner.Exec(ctx, `INSERT INTO resources(id,tenant_id,kind,data) VALUES($1,$2,'evaluation_attempt','{}') ON CONFLICT DO NOTHING`, probeID, tenant); err != nil {
			t.Fatal(err)
		}
		if err = AppendEvaluationEvent(ctx, app, tenant, probeID, "sim_swap", "starting", json.RawMessage(invalid)); err == nil {
			t.Fatal("invalid capability accepted")
		}
		owner.Exec(ctx, `DELETE FROM resources WHERE id=$1`, probeID)
	}
	check := json.RawMessage(`{"capability":"sim_swap"}`)
	if err = AppendEvaluationEvent(ctx, app, tenant, id, "sim_swap", "completed", check); err == nil {
		t.Fatal("duplicate completion accepted")
	}
	if err = AppendEvaluationEvent(ctx, app, other, id, "sim_swap", "completed", check); err == nil {
		t.Fatal("cross-tenant foreign key accepted")
	}
	tx, err := Tenant(ctx, app, other)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM evaluation_events WHERE evaluation_id=$1`, id).Scan(&count)
	tx.Rollback(ctx)
	if err != nil || count != 0 {
		t.Fatalf("cross-tenant visibility: %d %v", count, err)
	}
	for _, statement := range []string{`UPDATE evaluation_events SET phase='completed' WHERE evaluation_id=$1`, `DELETE FROM evaluation_events WHERE evaluation_id=$1`} {
		tx, err = Tenant(ctx, app, tenant)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(ctx, statement, id)
		tx.Rollback(ctx)
		if err == nil {
			t.Fatal("application mutated journal")
		}
	}
	tx, err = Tenant(ctx, app, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	items := []map[string]any{{"id": id, "status": "execution_unconfirmed", "journal": []map[string]string{{"event": "legacy"}}}}
	if err = EnrichEvaluationJournals(ctx, tx, items); err != nil {
		t.Fatal(err)
	}
	if got := len(items[0]["journal"].([]json.RawMessage)); got != 7 {
		t.Fatalf("journal count %d, want legacy+6", got)
	}
}

func TestDataAPIRolesDenied(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("requires disposable TEST_DATABASE_URL")
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
	tx, err := owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	// Exercise the migration's actual denial block against Supabase-like default
	// grants. Everything, including test roles, rolls back in this transaction.
	for _, role := range []string{"anon", "authenticated", "service_role"} {
		_, err = tx.Exec(ctx, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='`+role+`') THEN CREATE ROLE `+role+`; END IF; END $$; GRANT ALL ON ALL TABLES IN SCHEMA public TO `+role+`; GRANT ALL ON ALL FUNCTIONS IN SCHEMA public TO `+role)
		if err != nil {
			t.Fatal(err)
		}
	}
	start := strings.Index(portfolioSecurity, "DO $$")
	if start < 0 {
		t.Fatal("denial block missing")
	}
	if _, err = tx.Exec(ctx, portfolioSecurity[start:]); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"anon", "authenticated", "service_role"} {
		for _, table := range []string{"users", "tenants", "sessions", "invitations", "resources", "evaluation_events", "jobs", "audit"} {
			var allowed bool
			err = tx.QueryRow(ctx, `SELECT has_table_privilege($1,$2,'SELECT,INSERT,UPDATE,DELETE')`, role, table).Scan(&allowed)
			if err != nil || allowed {
				t.Fatalf("Data API %s access to %s: %v %v", role, table, allowed, err)
			}
		}
		var allowed bool
		err = tx.QueryRow(ctx, `SELECT has_function_privilege($1,'public.auth_memberships(text)','EXECUTE')`, role).Scan(&allowed)
		if err != nil || allowed {
			t.Fatalf("Data API security definer execution: %s %v %v", role, allowed, err)
		}
	}
}
