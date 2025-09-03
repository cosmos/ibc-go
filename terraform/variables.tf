# Variables for GitHub Terraform configuration

variable "github_token" {
  description = "GitHub personal access token with repo and admin:org permissions"
  type        = string
  sensitive   = true
}