-- Run through scripts/hosting/Initialize-PortfolioDatabase.ps1 with owner credentials.
-- app_password is supplied privately through stdin as a psql variable, never an argument.
-- Fresh installation only: do not overwrite a pre-existing role or its memberships.
BEGIN;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='optitrace_login') THEN
    RAISE EXCEPTION 'optitrace_login already exists; verify ownership before an explicit credential rotation';
  END IF;
  CREATE ROLE optitrace_login LOGIN INHERIT;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='optitrace_login'
    AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication)) THEN
    RAISE EXCEPTION 'application role must not have elevated attributes';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_auth_members WHERE member=(SELECT oid FROM pg_roles WHERE rolname='optitrace_login')) THEN
    RAISE EXCEPTION 'new application role unexpectedly has memberships';
  END IF;
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO optitrace_login',current_database());
END $$;
ALTER ROLE optitrace_login PASSWORD :'app_password';
COMMIT;
