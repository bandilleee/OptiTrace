resource "google_monitoring_alert_policy" "api_latency" {
  display_name = "${local.name} API request latency investigation"
  combiner     = "OR"
  documentation {
    content = "Cloud request p95, including cold requests, exceeded 500ms for five minutes. Inspect cold starts, auth, database and adapter latency. This alert is operational evidence, not a synthetic benchmark certificate."
  }
  conditions {
    display_name = "API p95 above 500ms"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND resource.label.service_name = \"${google_cloud_run_v2_service.api.name}\" AND metric.type = \"run.googleapis.com/request_latencies\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 500
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_PERCENTILE_95"
      }
    }
  }
  notification_channels = var.operational_notification_channels
  depends_on            = [google_project_service.api]
}
resource "google_monitoring_alert_policy" "api_errors" {
  display_name = "${local.name} API server errors"
  combiner     = "OR"
  conditions {
    display_name = "API 5xx above 1 percent"
    condition_threshold {
      filter             = "resource.type = \"cloud_run_revision\" AND resource.label.service_name = \"${google_cloud_run_v2_service.api.name}\" AND metric.type = \"run.googleapis.com/request_count\" AND metric.label.response_code_class = \"5xx\""
      denominator_filter = "resource.type = \"cloud_run_revision\" AND resource.label.service_name = \"${google_cloud_run_v2_service.api.name}\" AND metric.type = \"run.googleapis.com/request_count\""
      duration           = "300s"
      comparison         = "COMPARISON_GT"
      threshold_value    = 0.01
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["resource.label.service_name"]
      }
      denominator_aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["resource.label.service_name"]
      }
    }
  }
  notification_channels = var.operational_notification_channels
  depends_on            = [google_project_service.api]
}
resource "google_logging_metric" "benchmark_failures" {
  name   = "${local.name}-benchmark-failures"
  filter = "resource.type=\"cloud_run_job\" AND resource.labels.job_name=\"${google_cloud_run_v2_job.benchmark.name}\" AND (severity>=ERROR OR jsonPayload.msg=\"benchmark failed\")"
  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
  depends_on = [google_project_service.api]
}
resource "google_monitoring_alert_policy" "benchmark_failures" {
  display_name = "${local.name} benchmark failures"
  combiner     = "OR"
  conditions {
    display_name = "A benchmark worker failed"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_job\" AND metric.type = \"logging.googleapis.com/user/${google_logging_metric.benchmark_failures.name}\""
      duration        = "0s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_DELTA"
      }
    }
  }
  notification_channels = var.operational_notification_channels
}
resource "google_monitoring_alert_policy" "database_cpu" {
  display_name = "${local.name} database saturation"
  combiner     = "OR"
  conditions {
    display_name = "SQL CPU above 80 percent"
    condition_threshold {
      filter          = "resource.type = \"cloudsql_database\" AND resource.label.database_id = \"${var.project_id}:${google_sql_database_instance.postgres.name}\" AND metric.type = \"cloudsql.googleapis.com/database/cpu/utilization\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }
  notification_channels = var.operational_notification_channels
  depends_on            = [google_project_service.api]
}
