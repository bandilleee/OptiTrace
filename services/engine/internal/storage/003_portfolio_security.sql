-- Append-only per-call events avoid contending on the evaluation resource JSON
-- row. Intentions commit before each supplier call; completions commit afterward.
CREATE UNIQUE INDEX resources_tenant_identity ON public.resources(tenant_id,id);
CREATE TABLE public.evaluation_events (
 tenant_id text NOT NULL,
 evaluation_id text NOT NULL,
 capability text NOT NULL CHECK(capability IN ('number_verification','sim_swap','device_swap')),
 phase text NOT NULL CHECK(phase IN ('starting','completed')),
 check_data jsonb NOT NULL CHECK(jsonb_typeof(check_data)='object' AND coalesce(jsonb_typeof(check_data->'capability')='string' AND check_data->>'capability'=capability,false)),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(tenant_id,evaluation_id,capability,phase),
 FOREIGN KEY(tenant_id,evaluation_id) REFERENCES public.resources(tenant_id,id)
);
ALTER TABLE public.evaluation_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.evaluation_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_evaluation_events ON public.evaluation_events
 USING(tenant_id=current_tenant()) WITH CHECK(tenant_id=current_tenant());
REVOKE ALL ON public.evaluation_events FROM PUBLIC, optitrace_app;
GRANT SELECT, INSERT ON public.evaluation_events TO optitrace_app;

-- Supabase's public Data API is not OptiTrace's application API. Its default
-- roles must never see sessions, invitations, evidence or the tenant catalogue.
-- Touch only our own objects, not auth.*, storage.* or unrelated public objects.
DO $$
DECLARE target_role text; target_table text; target_function text;
BEGIN
 FOREACH target_role IN ARRAY ARRAY['PUBLIC','anon','authenticated','service_role'] LOOP
  IF target_role='PUBLIC' OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=target_role) THEN
   FOREACH target_table IN ARRAY ARRAY['schema_migrations','tenants','users','memberships','invitations','sessions','login_transactions','resources','credentials','audit','jobs','idempotency','evaluation_events'] LOOP
    EXECUTE format('REVOKE ALL ON TABLE public.%I FROM %s',target_table,CASE WHEN target_role='PUBLIC' THEN 'PUBLIC' ELSE quote_ident(target_role) END);
   END LOOP;
   EXECUTE format('REVOKE ALL ON SEQUENCE public.audit_id_seq FROM %s',CASE WHEN target_role='PUBLIC' THEN 'PUBLIC' ELSE quote_ident(target_role) END);
   FOREACH target_function IN ARRAY ARRAY['auth_memberships(text)','auth_credential(text)','auth_accept(text,text)','claim_job(text)','finish_job(text,text,jsonb,text)','expire_jobs()'] LOOP
    EXECUTE format('REVOKE ALL ON FUNCTION public.%s FROM %s',target_function,CASE WHEN target_role='PUBLIC' THEN 'PUBLIC' ELSE quote_ident(target_role) END);
   END LOOP;
  END IF;
 END LOOP;
END $$;
