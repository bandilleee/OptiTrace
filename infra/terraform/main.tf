locals {
  name = "optitrace-${var.environment}"
  apis = toset(["run.googleapis.com", "sqladmin.googleapis.com", "identitytoolkit.googleapis.com", "secretmanager.googleapis.com", "artifactregistry.googleapis.com", "cloudscheduler.googleapis.com", "iam.googleapis.com", "iamcredentials.googleapis.com", "sts.googleapis.com", "monitoring.googleapis.com", "logging.googleapis.com", "billingbudgets.googleapis.com"])
  # Direct submissions launch immediately; the scheduler only recovers missed/expired work.
  recovery_schedule = var.environment == "production" ? "* * * * *" : "*/15 * * * *"
  common_env = {
    APP_ENV              = "production"
    AUTH_MODE            = "google"
    AUTH_COOKIE_SECURE   = "true"
    APP_ORIGIN           = var.app_origin
    API_PUBLIC_URL       = "${var.app_origin}/api"
    GOOGLE_CLOUD_PROJECT = var.project_id
    GOOGLE_CLIENT_ID     = var.google_client_id
  }
}
resource "google_project_service" "api" {
  for_each           = local.apis
  service            = each.value
  disable_on_destroy = false
}
resource "google_artifact_registry_repository" "images" {
  location      = var.region
  repository_id = local.name
  format        = "DOCKER"
  depends_on    = [google_project_service.api]
}
resource "google_service_account" "api" {
  account_id = "${local.name}-api"
}
resource "google_service_account" "worker" {
  account_id = "${local.name}-worker"
}
resource "google_service_account" "web" {
  account_id = "${local.name}-web"
}
resource "google_service_account" "simulator" {
  account_id = "${local.name}-sim"
}
resource "google_service_account" "migration" {
  account_id = "${local.name}-migration"
}
resource "google_service_account" "scheduler" {
  account_id = "${local.name}-scheduler"
}
resource "google_service_account" "release" {
  account_id = "${local.name}-release"
}
resource "random_password" "app" {
  length  = 40
  special = false
}
resource "random_password" "admin" {
  length  = 40
  special = false
}
resource "google_sql_database_instance" "postgres" {
  name                = "${local.name}-postgres"
  database_version    = "POSTGRES_18"
  region              = var.region
  deletion_protection = true
  settings {
    tier              = "db-custom-1-3840"
    edition           = "ENTERPRISE"
    availability_type = var.environment == "production" ? "REGIONAL" : "ZONAL"
    disk_size         = 10
    disk_autoresize   = true
    ip_configuration {
      ipv4_enabled = true
      # No authorized networks: connections use the authenticated Cloud SQL connector.
      ssl_mode = "ENCRYPTED_ONLY"
    }
    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
      start_time                     = "01:00"
      backup_retention_settings {
        retained_backups = 7
      }
    }
    maintenance_window {
      day  = 7
      hour = 2
    }
    insights_config {
      query_insights_enabled = true
    }
  }
  depends_on = [google_project_service.api]
}
resource "google_sql_database" "app" {
  name     = "optitrace"
  instance = google_sql_database_instance.postgres.name
}
resource "google_sql_user" "app" {
  name     = "optitrace_login"
  instance = google_sql_database_instance.postgres.name
  password = random_password.app.result
}
resource "google_sql_user" "migration" {
  name     = "postgres"
  instance = google_sql_database_instance.postgres.name
  password = random_password.admin.result
}
resource "google_secret_manager_secret" "config" {
  for_each  = toset(["database-url", "migration-database-url", "google-client-secret", "identity-platform-api-key"])
  secret_id = "${local.name}-${each.key}"
  replication {
    auto {
    }
  }
  depends_on = [google_project_service.api]
}
locals {
  secret_payloads = {
    database-url              = "postgres://optitrace_login:${random_password.app.result}@/optitrace?host=/cloudsql/${google_sql_database_instance.postgres.connection_name}&sslmode=disable"
    migration-database-url    = "postgres://postgres:${random_password.admin.result}@/optitrace?host=/cloudsql/${google_sql_database_instance.postgres.connection_name}&sslmode=disable"
    google-client-secret      = var.google_client_secret
    identity-platform-api-key = var.identity_platform_api_key
  }
}
resource "google_secret_manager_secret_version" "config" {
  for_each    = google_secret_manager_secret.config
  secret      = each.value.id
  secret_data = local.secret_payloads[each.key]
}
resource "google_secret_manager_secret_iam_member" "api" {
  for_each  = toset(["database-url", "google-client-secret", "identity-platform-api-key"])
  secret_id = google_secret_manager_secret.config[each.key].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api.email}"
}
resource "google_secret_manager_secret_iam_member" "worker" {
  secret_id = google_secret_manager_secret.config["database-url"].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.worker.email}"
}
resource "google_secret_manager_secret_iam_member" "migration" {
  secret_id = google_secret_manager_secret.config["migration-database-url"].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.migration.email}"
}
resource "google_project_iam_member" "sql_client" {
  for_each = { api = google_service_account.api.email, worker = google_service_account.worker.email, migration = google_service_account.migration.email }
  project  = var.project_id
  role     = "roles/cloudsql.client"
  member   = "serviceAccount:${each.value}"
}
resource "google_project_iam_custom_role" "identity_lookup" {
  role_id     = "optitrace_${var.environment}_identity_lookup"
  title       = "OptiTrace identity status lookup"
  permissions = ["firebaseauth.users.get"]
}
resource "google_project_iam_member" "identity_lookup" {
  project = var.project_id
  role    = google_project_iam_custom_role.identity_lookup.name
  member  = "serviceAccount:${google_service_account.api.email}"
}
resource "google_identity_platform_config" "identity" {
  project            = var.project_id
  authorized_domains = [trimprefix(var.app_origin, "https://")]
  sign_in {
    allow_duplicate_emails = false
  }
  depends_on = [google_project_service.api]
}
resource "google_identity_platform_default_supported_idp_config" "google" {
  enabled       = true
  idp_id        = "google.com"
  client_id     = var.google_client_id
  client_secret = var.google_client_secret
  depends_on    = [google_identity_platform_config.identity]
}
resource "google_cloud_run_v2_service" "simulator" {
  name                = "${local.name}-simulator"
  location            = var.region
  deletion_protection = true
  ingress             = "INGRESS_TRAFFIC_ALL"
  template {
    service_account = google_service_account.simulator.email
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }
    containers {
      image   = var.engine_image
      command = ["/app/simulator"]
      ports {
        container_port = 8090
      }
      env {
        name  = "APP_ENV"
        value = "production"
      }
      env {
        name  = "AUTH_MODE"
        value = "google"
      }
      resources {
        limits = {
          cpu = "1", memory = "256Mi"
        }
      }
    }
  }
  depends_on = [google_project_service.api]
}
resource "google_cloud_run_v2_service_iam_member" "simulator_api" {
  name     = google_cloud_run_v2_service.simulator.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.api.email}"
}
resource "google_cloud_run_v2_service_iam_member" "simulator_worker" {
  name     = google_cloud_run_v2_service.simulator.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.worker.email}"
}
resource "google_cloud_run_v2_service" "api" {
  name                = "${local.name}-api"
  location            = var.region
  deletion_protection = true
  template {
    service_account = google_service_account.api.email
    timeout         = "30s"
    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }
    containers {
      image = var.engine_image
      ports {
        container_port = 8080
      }
      resources {
        limits = {
          cpu = "1", memory = "512Mi"
        }
      }
      dynamic "env" {
        for_each = merge(local.common_env, {
          SIMULATOR_URL = google_cloud_run_v2_service.simulator.uri, SIMULATOR_AUDIENCE = google_cloud_run_v2_service.simulator.uri, BENCHMARK_JOB_NAME = google_cloud_run_v2_job.benchmark.id
          }
        )
        content {
          name  = env.key
          value = env.value
        }
      }
      dynamic "env" {
        for_each = {
          DATABASE_URL = "database-url", GOOGLE_CLIENT_SECRET = "google-client-secret", IDENTITY_PLATFORM_API_KEY = "identity-platform-api-key"
        }
        content {
          name = env.key
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.config[env.value].secret_id
              version = google_secret_manager_secret_version.config[env.value].version
            }
          }
        }
      }
      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }
      startup_probe {
        http_get {
          path = "/healthz"
        }
        initial_delay_seconds = 1
      }
      liveness_probe {
        http_get {
          path = "/healthz"
        }
      }
    }
    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.postgres.connection_name]
      }
    }
  }
  depends_on = [google_secret_manager_secret_iam_member.api, google_project_iam_member.sql_client]
}
resource "google_cloud_run_v2_service_iam_member" "api_ingress" {
  name     = google_cloud_run_v2_service.api.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "allUsers" # The Go API enforces application sessions/API credentials on every tenant route.
}
resource "google_cloud_run_v2_service" "web" {
  name                = "${local.name}-web"
  location            = var.region
  deletion_protection = true
  template {
    service_account = google_service_account.web.email
    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }
    containers {
      image = var.web_image
      ports {
        container_port = 3000
      }
      env {
        name  = "API_BASE_URL"
        value = google_cloud_run_v2_service.api.uri
      }
      resources {
        limits = {
          cpu = "1", memory = "512Mi"
        }
      }
    }
  }
}
resource "google_cloud_run_v2_service_iam_member" "web_ingress" {
  name     = google_cloud_run_v2_service.web.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "allUsers"
}
resource "google_cloud_run_v2_job" "benchmark" {
  name                = "${local.name}-benchmark"
  location            = var.region
  deletion_protection = true
  template {
    template {
      service_account = google_service_account.worker.email
      timeout         = "900s"
      max_retries     = 1
      containers {
        image   = var.engine_image
        command = ["/app/worker"]
        args    = ["-once"]
        env {
          name  = "APP_ENV"
          value = "production"
        }
        env {
          name  = "SIMULATOR_URL"
          value = google_cloud_run_v2_service.simulator.uri
        }
        env {
          name  = "SIMULATOR_AUDIENCE"
          value = google_cloud_run_v2_service.simulator.uri
        }
        env {
          name  = "REPORT_BUCKET"
          value = google_storage_bucket.reports.name
        }
        env {
          name = "DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.config["database-url"].secret_id
              version = google_secret_manager_secret_version.config["database-url"].version
            }
          }
        }
        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }
        resources {
          limits = {
            cpu = "1", memory = "512Mi"
          }
        }
      }
      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [google_sql_database_instance.postgres.connection_name]
        }
      }
    }
  }
  depends_on = [google_secret_manager_secret_iam_member.worker, google_project_iam_member.sql_client]
}
resource "google_cloud_run_v2_job" "migration" {
  name                = "${local.name}-migration"
  location            = var.region
  deletion_protection = true
  template {
    template {
      service_account = google_service_account.migration.email
      timeout         = "300s"
      max_retries     = 0
      containers {
        image   = var.engine_image
        command = ["/app/migrate"]
        env {
          name  = "APP_ENV"
          value = "production"
        }
        env {
          name = "DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.config["migration-database-url"].secret_id
              version = google_secret_manager_secret_version.config["migration-database-url"].version
            }
          }
        }
        env {
          name  = "APP_DATABASE_USER"
          value = google_sql_user.app.name
        }
        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }
      }
      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [google_sql_database_instance.postgres.connection_name]
        }
      }
    }
  }
  depends_on = [google_secret_manager_secret_iam_member.migration, google_sql_user.app]
}
resource "google_cloud_run_v2_job_iam_member" "benchmark_api" {
  name     = google_cloud_run_v2_job.benchmark.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.api.email}"
}
resource "google_cloud_run_v2_job_iam_member" "benchmark_scheduler" {
  name     = google_cloud_run_v2_job.benchmark.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler.email}"
}
resource "google_cloud_scheduler_job" "recover" {
  name      = "${local.name}-recover-benchmarks"
  region    = var.region
  schedule  = local.recovery_schedule
  time_zone = "Africa/Johannesburg"
  http_target {
    uri         = "https://run.googleapis.com/v2/${google_cloud_run_v2_job.benchmark.id}:run"
    http_method = "POST"
    body        = base64encode("{}")
    headers = {
      "Content-Type" = "application/json"
    }
    oauth_token {
      service_account_email = google_service_account.scheduler.email
    }
  }
  depends_on = [google_project_service.api]
}
resource "google_storage_bucket" "reports" {
  name                        = "${var.project_id}-${local.name}-reports"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false
  versioning {
    enabled = true
  }
  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type = "Delete"
    }
  }
}
resource "google_storage_bucket_iam_member" "worker_reports" {
  bucket = google_storage_bucket.reports.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${google_service_account.worker.email}"
}
resource "google_storage_bucket_iam_member" "api_reports" {
  bucket = google_storage_bucket.reports.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.api.email}"
}
resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "${local.name}-github"
  depends_on                = [google_project_service.api]
}
resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  attribute_mapping = {
    "google.subject" = "assertion.sub", "attribute.repository_id" = "assertion.repository_id", "attribute.repository_owner_id" = "assertion.repository_owner_id"
  }
  attribute_condition = "assertion.repository_id == '${var.github_repository_id}' && assertion.repository_owner_id == '${var.github_repository_owner_id}' && assertion.repository == '${var.github_repository}' && assertion.ref == 'refs/heads/main' && assertion.sub == 'repo:${var.github_repository}:environment:${var.environment}'"
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}
resource "google_service_account_iam_member" "release_federation" {
  service_account_id = google_service_account.release.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository_id/${var.github_repository_id}"
}
resource "google_project_iam_member" "release_roles" {
  for_each = toset(["roles/run.developer", "roles/artifactregistry.writer"])
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.release.email}"
}
resource "google_service_account_iam_member" "release_act_as" {
  for_each = {
    api = google_service_account.api.name, web = google_service_account.web.name, worker = google_service_account.worker.name, migration = google_service_account.migration.name, simulator = google_service_account.simulator.name
  }
  service_account_id = each.value
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.release.email}"
}
resource "google_billing_budget" "monthly" {
  count           = var.billing_account == "" ? 0 : 1
  billing_account = var.billing_account
  display_name    = "${local.name} monthly alert"
  budget_filter {
    projects = ["projects/${data.google_project.current.number}"]
  }
  amount {
    specified_amount {
      currency_code = "USD"
      units         = tostring(var.monthly_budget_usd)
    }
  }
  threshold_rules {
    threshold_percent = 0.5
  }
  threshold_rules {
    threshold_percent = 0.9
  }
  threshold_rules {
    threshold_percent = 1.0
  }
  all_updates_rule {
    monitoring_notification_channels = var.budget_notification_channels
  }
}
data "google_project" "current" {
  project_id = var.project_id
}
