-- name: AuditEvent :exec
INSERT INTO audit(tenant_id,actor_id,action,resource_id) VALUES($1,$2,$3,$4);

-- name: ListResources :many
SELECT data FROM resources WHERE kind=$1 ORDER BY created_at DESC LIMIT 100;

-- name: GetResource :one
SELECT data FROM resources WHERE id=$1 AND kind=$2;

-- name: InsertResource :exec
INSERT INTO resources(id,tenant_id,kind,data) VALUES($1,$2,$3,$4);

-- name: GetSession :one
SELECT coalesce(user_id,'') AS user_id, csrf, provider_token, provider_checked FROM sessions WHERE hash=$1 AND expires_at>now();

-- name: SaveSession :exec
INSERT INTO sessions(hash,user_id,csrf,expires_at,provider_token) VALUES($1,$2,$3,$4,$5);

-- name: GetIdempotency :one
SELECT request_hash,response FROM idempotency WHERE tenant_id=$1 AND key=$2 FOR UPDATE;

-- name: ReserveIdempotency :execrows
INSERT INTO idempotency(tenant_id,key,request_hash) VALUES($1,$2,$3) ON CONFLICT DO NOTHING;

-- name: CompleteIdempotency :exec
UPDATE idempotency SET response=$3 WHERE tenant_id=$1 AND key=$2;


-- name: ListEvaluations :many
SELECT data FROM resources WHERE kind IN ('evaluation','evaluation_attempt') ORDER BY created_at DESC LIMIT 100;

-- name: FinalizeEvaluation :execrows
UPDATE resources SET kind='evaluation',data=$2 WHERE id=$1 AND kind='evaluation_attempt';
