package server

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"github.com/optitrace/optitrace/services/engine/internal/identity"
	"log/slog"
	"os"
	"time"
)

// RunJob claims one durable lease. A ten-minute job deadline is shorter than its
// lease; token-guarded completion prevents late workers overwriting a retry.
func RunJob(ctx context.Context, db *pgxpool.Pool, engine *evaluation.Engine) (bool, error) {
	token := identity.Random()
	var id string
	var seed int64
	var count int
	e := db.QueryRow(ctx, `SELECT id,seed,count FROM claim_job($1)`, token).Scan(&id, &seed, &count)
	if e == pgx.ErrNoRows {
		_, e = db.Exec(ctx, `SELECT expire_jobs()`)
		return false, e
	}
	if e != nil {
		return false, e
	}
	jobctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	report, runerr := engine.Benchmark(jobctx, seed, count)
	complete, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()
	if runerr != nil {
		_, e = db.Exec(complete, `SELECT finish_job($1,$2,NULL,$3)`, id, token, runerr.Error())
		slog.Warn("benchmark failed", "id", id, "error", runerr)
		return true, e
	}
	if os.Getenv("REPORT_BUCKET") != "" {
		report.ArtifactObject = "benchmarks/" + id + "/" + token + ".json"
	}
	raw, e := json.Marshal(report)
	if e != nil {
		return true, e
	}
	if e = uploadReport(complete, report.ArtifactObject, raw); e != nil {
		slog.Warn("benchmark failed", "id", id, "error", e)
		_, finishErr := db.Exec(complete, `SELECT finish_job($1,$2,NULL,$3)`, id, token, e.Error())
		return true, finishErr
	}
	_, e = db.Exec(complete, `SELECT finish_job($1,$2,$3,NULL)`, id, token, raw)
	return true, e
}
