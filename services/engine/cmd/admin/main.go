// Command admin provisions a tenant and its first invitation using an operator
// database connection. It is deliberately not exposed by the web API.
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"github.com/optitrace/optitrace/services/engine/internal/storage"
	"os"
	"strings"
)

func main() {
	tenant := flag.String("tenant", "", "Tenant identifier")
	name := flag.String("name", "", "Tenant display name (creates tenant)")
	email := flag.String("email", "", "Invite a Google account email")
	roles := flag.String("roles", "tenant_admin,integration_developer,auditor", "Comma-separated roles")
	flag.Parse()
	if *tenant == "" || *email == "" || !strings.Contains(*email, "@") {
		panic("-tenant and valid -email required; optionally -name to create tenant")
	}
	r := strings.Split(*roles, ",")
	for _, v := range r {
		valid := false
		for _, x := range []string{"tenant_admin", "integration_developer", "auditor", "policy_author", "policy_reviewer"} {
			valid = valid || v == x
		}
		if !valid {
			panic("invalid role")
		}
	}
	ctx := context.Background()
	db, e := storage.Open(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		panic(e)
	}
	defer db.Close()
	tx, e := storage.Tenant(ctx, db, *tenant)
	if e != nil {
		panic(e)
	}
	defer tx.Rollback(ctx)
	if *name != "" {
		if _, e = tx.Exec(ctx, `INSERT INTO tenants(id,name) VALUES($1,$2) ON CONFLICT(id) DO NOTHING`, *tenant, *name); e != nil {
			panic(e)
		}
	}
	id := identity.Random()
	if _, e = tx.Exec(ctx, `INSERT INTO invitations(id,tenant_id,email,roles) VALUES($1,$2,$3,$4)`, id, *tenant, strings.ToLower(*email), r); e != nil {
		panic(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO audit(tenant_id,actor_id,action,resource_id) VALUES($1,'platform_operator','tenant.invited',$2)`, *tenant, id); e != nil {
		panic(e)
	}
	if e = tx.Commit(ctx); e != nil {
		panic(e)
	}
	fmt.Printf("Invitation created: %s\n", id)
}
