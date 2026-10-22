package storage

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	p, e := pgxpool.New(ctx, url)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	return p, nil
}
func Migrate(ctx context.Context, p *pgxpool.Pool) error {
	tx, e := p.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(784331)"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY,checksum text NOT NULL,applied_at timestamptz NOT NULL DEFAULT now())`); e != nil {
		return e
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(schema)))
	var applied string
	e = tx.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version=1`).Scan(&applied)
	if e != nil && e != pgx.ErrNoRows {
		return e
	}
	if e == pgx.ErrNoRows {
		if _, e = tx.Exec(ctx, schema); e != nil {
			return fmt.Errorf("schema: %w", e)
		}
		if _, e = tx.Exec(ctx, `INSERT INTO schema_migrations(version,checksum) VALUES(1,$1)`, checksum); e != nil {
			return e
		}
	} else if applied != checksum {
		return fmt.Errorf("migration 1 checksum changed; add a new version instead")
	}
	return tx.Commit(ctx)
}
func Tenant(ctx context.Context, p *pgxpool.Pool, id string) (pgx.Tx, error) {
	tx, e := p.Begin(ctx)
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "SELECT set_config('app.tenant_id',$1,true)", id)
	if e != nil {
		tx.Rollback(ctx)
		return nil, e
	}
	return tx, nil
}
func Seed(ctx context.Context, p *pgxpool.Pool) error {
	tx, e := p.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `INSERT INTO tenants VALUES('demo','OptiTrace demonstration'),('other','Separate workspace') ON CONFLICT DO NOTHING;
 INSERT INTO users VALUES('seed-admin','admin@optitrace.test'),('seed-author','author@optitrace.test'),('seed-reviewer','reviewer@optitrace.test'),('seed-auditor','auditor@optitrace.test'),('seed-outsider','outsider@optitrace.test') ON CONFLICT DO NOTHING;
 INSERT INTO memberships VALUES('m-admin','demo','seed-admin',ARRAY['tenant_admin','integration_developer','auditor']),('m-author','demo','seed-author',ARRAY['policy_author','integration_developer']),('m-reviewer','demo','seed-reviewer',ARRAY['policy_reviewer','auditor']),('m-auditor','demo','seed-auditor',ARRAY['auditor']),('m-outsider','other','seed-outsider',ARRAY['tenant_admin','integration_developer']) ON CONFLICT DO NOTHING;
 INSERT INTO invitations(id,tenant_id,email,roles) SELECT 'invite-'||memberships.id,tenant_id,email,roles FROM memberships JOIN users ON users.id=memberships.user_id ON CONFLICT DO NOTHING;`)
	if e != nil {
		return e
	}
	for _, profile := range []string{"complete", "conditional"} {
		_, e = tx.Exec(ctx, `SELECT set_config('app.tenant_id','demo',true)`)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO resources(id,tenant_id,kind,data) VALUES($1,'demo','policy',jsonb_build_object('id',$1::text,'tenant_id','demo','name',$2::text,'profile',$2::text,'version',1,'status','active','author_id','seed-author','reviewer_id','seed-reviewer','created_at',now())) ON CONFLICT DO NOTHING`, "policy-"+profile, profile)
		if e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
