# Terraform configuration for managing GitHub access in Cosmos organization

terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }
  }
}

# Configure the GitHub Provider
provider "github" {
  token        = var.github_token
  organization = "cosmos"
}

# Add gjermundgaraba as a collaborator to the poa repository with read access
resource "github_repository_collaborator" "gjermundgaraba_poa_access" {
  repository = "poa"
  username   = "gjermundgaraba"
  permission = "pull"  # Read-only access
}