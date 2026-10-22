output "web_url" {
  value = google_cloud_run_v2_service.web.uri
}
output "api_url" {
  value = google_cloud_run_v2_service.api.uri
}
output "migration_job" {
  value = google_cloud_run_v2_job.migration.name
}
output "benchmark_job" {
  value = google_cloud_run_v2_job.benchmark.name
}
output "report_bucket" {
  value = google_storage_bucket.reports.name
}
output "workload_identity_provider" {
  value = google_iam_workload_identity_pool_provider.github.name
}
output "release_service_account" {
  value = google_service_account.release.email
}
