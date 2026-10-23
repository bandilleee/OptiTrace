\set ON_ERROR_STOP on
BEGIN;
INSERT INTO jobs(id,tenant_id,seed,count,created_at) VALUES ('acceptance-lease','demo',7,2,now()-interval '1 day');
SET LOCAL ROLE optitrace_test_login;
DO $$ DECLARE selected text; n int; BEGIN
 SELECT id INTO selected FROM claim_job('old-token');
 IF selected IS DISTINCT FROM 'acceptance-lease' THEN RAISE EXCEPTION 'wrong initial lease %',selected; END IF;
 SELECT count(*) INTO n FROM claim_job('parallel-token');
 IF n<>0 THEN RAISE EXCEPTION 'unexpired lease claimed twice'; END IF;
END $$;
RESET ROLE;
UPDATE jobs SET lease_until=now()-interval '1 second' WHERE id='acceptance-lease';
SET LOCAL ROLE optitrace_test_login;
DO $$ DECLARE selected text; BEGIN
 SELECT id INTO selected FROM claim_job('replacement-token');
 IF selected IS DISTINCT FROM 'acceptance-lease' THEN RAISE EXCEPTION 'expired lease not recovered'; END IF;
 PERFORM finish_job('acceptance-lease','old-token','{"marker":"stale"}',NULL);
END $$;
RESET ROLE;
DO $$ BEGIN IF (SELECT status FROM jobs WHERE id='acceptance-lease')<>'running' THEN RAISE EXCEPTION 'stale worker overwrote replacement'; END IF; END $$;
SET LOCAL ROLE optitrace_test_login;
SELECT finish_job('acceptance-lease','replacement-token','{"marker":"valid"}',NULL);
SELECT finish_job('acceptance-lease','replacement-token','{"marker":"duplicate"}',NULL);
RESET ROLE;
DO $$ BEGIN IF (SELECT result->>'marker' FROM jobs WHERE id='acceptance-lease')<>'valid' THEN RAISE EXCEPTION 'duplicate completion overwrote report'; END IF; END $$;
INSERT INTO jobs(id,tenant_id,seed,count,status,attempts,lease_until) VALUES ('acceptance-exhausted','demo',7,2,'running',3,now()-interval '1 second');
SET LOCAL ROLE optitrace_test_login;
SELECT expire_jobs();
RESET ROLE;
DO $$ BEGIN IF (SELECT status FROM jobs WHERE id='acceptance-exhausted')<>'failed' THEN RAISE EXCEPTION 'exhausted lease not failed'; END IF; END $$;
SET LOCAL ROLE optitrace_test_login;
SELECT set_config('app.tenant_id','other',true);
DO $$ DECLARE table_name text; n int; BEGIN
 FOREACH table_name IN ARRAY ARRAY['resources','audit','idempotency','memberships','invitations','credentials','jobs'] LOOP
  EXECUTE format('SELECT count(*) FROM %I WHERE tenant_id=''demo''',table_name) INTO n;
  IF n<>0 THEN RAISE EXCEPTION 'cross-tenant leak table % count %',table_name,n; END IF;
 END LOOP;
 BEGIN
  INSERT INTO resources(id,tenant_id,kind,data) VALUES('rls-forgery','demo','test','{}');
  RAISE EXCEPTION 'RLS accepted forged tenant write';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  UPDATE audit SET action='tampered';
  RAISE EXCEPTION 'audit UPDATE permitted';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $$;
ROLLBACK;
SELECT 'PASS: lease exclusion/recovery/token fencing/duplicate completion/exhaustion; seven-table RLS; forged write and audit mutation rejection' AS acceptance;
