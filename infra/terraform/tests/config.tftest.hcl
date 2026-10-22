mock_provider "google" {}
mock_provider "random" {}

variables {
  project_id                = "optitrace-test-project"
  app_origin                = "https://optitrace.example.test"
  engine_image              = "africa-south1-docker.pkg.dev/optitrace-test-project/images/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  web_image                 = "africa-south1-docker.pkg.dev/optitrace-test-project/images/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  google_client_id          = "test-client.apps.googleusercontent.com"
  google_client_secret      = "test-only-not-a-real-secret"
  identity_platform_api_key = "test-only-not-a-real-key"
  github_repository         = "example/optitrace"
}

run "staging_plan" {
  command = plan
  assert {
    condition     = google_sql_database_instance.postgres.settings[0].availability_type == "ZONAL"
    error_message = "Staging capacity unexpectedly changed."
  }
  assert {
    condition     = length(google_sql_database_instance.postgres.settings[0].ip_configuration[0].authorized_networks) == 0
    error_message = "SQL must not allow direct public authorized networks."
  }
  assert {
    condition     = google_storage_bucket.reports.public_access_prevention == "enforced"
    error_message = "Reports must remain private."
  }
  assert {
    condition     = local.common_env.AUTH_MODE == "google" && local.common_env.APP_ENV == "production"
    error_message = "Cloud configuration cannot enable the test identity provider."
  }
}

run "production_plan" {
  command = plan
  variables { environment = "production" }
  assert {
    condition     = google_sql_database_instance.postgres.settings[0].availability_type == "REGIONAL"
    error_message = "Production requires regional HA."
  }
  assert {
    condition     = google_sql_database_instance.postgres.deletion_protection
    error_message = "Database deletion protection must remain enabled."
  }
}
