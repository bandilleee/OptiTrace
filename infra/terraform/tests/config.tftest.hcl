mock_provider "google" {}
mock_provider "random" {}

variables {
  project_id                 = "optitrace-test-project"
  app_origin                 = "https://optitrace.example.test"
  engine_image               = "africa-south1-docker.pkg.dev/optitrace-test-project/images/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  web_image                  = "africa-south1-docker.pkg.dev/optitrace-test-project/images/web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  google_client_id           = "test-client.apps.googleusercontent.com"
  google_client_secret       = "test-only-not-a-real-secret"
  identity_platform_api_key  = "test-only-not-a-real-key"
  github_repository          = "example/optitrace"
  github_repository_id       = "123456789"
  github_repository_owner_id = "12345678"
}

run "staging_plan" {
  command = plan
  assert {
    condition     = google_cloud_scheduler_job.recover.schedule == "*/15 * * * *"
    error_message = "Development/staging recovery must use the declared lower-cost cadence."
  }
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
  variables {
    environment                       = "production"
    operational_notification_channels = ["projects/optitrace-test-project/notificationChannels/test"]
  }
  assert {
    condition     = google_cloud_scheduler_job.recover.schedule == "* * * * *"
    error_message = "Production recovery cadence unexpectedly changed."
  }
  assert {
    condition     = google_sql_database_instance.postgres.settings[0].availability_type == "REGIONAL"
    error_message = "Production requires regional HA."
  }
  assert {
    condition     = google_sql_database_instance.postgres.deletion_protection
    error_message = "Database deletion protection must remain enabled."
  }
}

run "reject_origin_with_query" {
  command = plan
  variables { app_origin = "https://optitrace.example.test?next=bad" }
  expect_failures = [var.app_origin]
}
run "reject_repository_name_as_id" {
  command = plan
  variables { github_repository_id = "example/optitrace" }
  expect_failures = [var.github_repository_id]
}
run "reject_production_without_alert_recipient" {
  command = plan
  variables { environment = "production" }
  expect_failures = [var.operational_notification_channels]
}
run "federation_uses_immutable_ids" {
  command = plan
  assert {
    condition     = strcontains(google_iam_workload_identity_pool_provider.github.attribute_condition, "assertion.repository_id == '123456789'") && strcontains(google_iam_workload_identity_pool_provider.github.attribute_condition, "assertion.repository_owner_id == '12345678'")
    error_message = "Federation must bind immutable repository and owner IDs."
  }
}

run "reject_empty_alert_resource" {
  command = plan
  variables { operational_notification_channels = [""] }
  expect_failures = [var.operational_notification_channels]
}
