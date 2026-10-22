-- Local synthetic development only. Hosted credentials are generated into Secret Manager.
CREATE ROLE optitrace_login LOGIN PASSWORD 'local-app-only' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
