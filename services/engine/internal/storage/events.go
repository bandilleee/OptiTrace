package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppendEvaluationEvent commits one independent event row. A commit error is
// uncertain; callers must preserve the reservation and must not replay calls.
func AppendEvaluationEvent(ctx context.Context, db *pgxpool.Pool, tenantID, id, capability, phase string, check json.RawMessage) error {
	tx, err := Tenant(ctx, db, tenantID)
	if err != nil {
		return fmt.Errorf("journal begin: %w", err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	_, err = tx.Exec(ctx, `INSERT INTO evaluation_events(tenant_id,evaluation_id,capability,phase,check_data) VALUES($1,$2,$3,$4,$5)`, tenantID, id, capability, phase, check)
	if err != nil {
		return fmt.Errorf("journal insert: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("journal commit acknowledgement: %w", err)
	}
	return nil
}

// EnrichEvaluationJournals retains legacy resource-embedded events and adds new
// rows in one query, including for a page of unconfirmed attempts. The caller's
// transaction-local tenant and RLS determine visibility; client IDs grant none.
func EnrichEvaluationJournals(ctx context.Context, tx pgx.Tx, items []map[string]any) error {
	ids := []string{}
	for _, item := range items {
		if item["status"] != "execution_unconfirmed" {
			continue
		}
		id, ok := item["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("invalid persisted evaluation identity")
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	events := map[string][]json.RawMessage{}
	rows, err := tx.Query(ctx, `SELECT evaluation_id,jsonb_build_object('event',phase,'check',check_data,'recorded_at',recorded_at) FROM evaluation_events WHERE evaluation_id=ANY($1::text[]) ORDER BY recorded_at,capability,phase`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var entry []byte
		if err = rows.Scan(&id, &entry); err != nil {
			return err
		}
		events[id] = append(events[id], json.RawMessage(entry))
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for _, item := range items {
		if item["status"] != "execution_unconfirmed" {
			continue
		}
		entries := []json.RawMessage{}
		if item["journal"] != nil {
			raw, err := json.Marshal(item["journal"])
			if err != nil {
				return err
			}
			if err = json.Unmarshal(raw, &entries); err != nil {
				return err
			}
		}
		item["journal"] = append(entries, events[item["id"].(string)]...)
	}
	return nil
}
