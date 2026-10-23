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
    condition     = can(regex("^https://([a-z0-9]([a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9]([a-z0-9-]*[a-z0-9])?$", var.app_origin))
    error_message = "Supply an exact HTTPS DNS origin without path, credentials, port, query, fragment or trailing slash."
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

variable "github_repository_id" {
  type        = string
  description = "Immutable numeric GitHub repository ID; never infer it from a reusable name."
  validation {
    condition     = can(regex("^[1-9][0-9]*$", var.github_repository_id))
    error_message = "Supply the positive numeric GitHub repository ID."
  }
}
variable "github_repository_owner_id" {
  type        = string
  description = "Immutable numeric GitHub owner ID used to prevent namespace reuse."
  validation {
    condition     = can(regex("^[1-9][0-9]*$", var.github_repository_owner_id))
    error_message = "Supply the positive numeric GitHub owner ID."
  }
}
variable "operational_notification_channels" {
  type        = list(string)
  default     = []
  description = "Verified Monitoring channels for API, SQL and worker incidents; separate from billing contacts."
  validation {
    condition     = (var.environment != "production" || length(var.operational_notification_channels) > 0) && alltrue([for channel in var.operational_notification_channels : can(regex("^projects/[^/]+/notificationChannels/[^/]+$", channel))])
    error_message = "Supply valid Monitoring channel resource names; production requires at least one verified operational recipient."
  }
}
