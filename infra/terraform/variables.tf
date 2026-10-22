variable "project_id" {
  type        = string
  description = "Existing approved Google Cloud project, never inferred from local gcloud defaults."
}
variable "region" {
  type    = string
  default = "africa-south1"
}
variable "environment" {
  type    = string
  default = "staging"
  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "Use staging or production."
  }
}
variable "app_origin" {
  type        = string
  description = "Exact HTTPS frontend origin registered with Google OAuth; no trailing slash."
  validation {
    condition     = can(regex("^https://[^/]+$", var.app_origin))
    error_message = "An exact HTTPS origin is required."
  }
}
variable "engine_image" {
  type        = string
  description = "Reviewed Artifact Registry image by sha256 digest."
  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.engine_image))
    error_message = "Use an immutable image digest."
  }
}
variable "web_image" {
  type = string
  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.web_image))
    error_message = "Use an immutable image digest."
  }
}
variable "google_client_id" {
  type = string
}
variable "google_client_secret" {
  type      = string
  sensitive = true
}
variable "identity_platform_api_key" {
  type      = string
  sensitive = true
}
variable "github_repository" {
  type        = string
  description = "Exact owner/repository trusted for protected release workflows."
  validation {
    condition     = can(regex("^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$", var.github_repository))
    error_message = "Supply owner/repository."
  }
}
variable "billing_account" {
  type        = string
  default     = ""
  description = "Optional billing account for notification budget. Budget is an alert, not a spend cap."
}
variable "monthly_budget_usd" {
  type    = number
  default = 100
}
variable "budget_notification_channels" {
  type    = list(string)
  default = []
}
