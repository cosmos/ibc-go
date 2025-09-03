# GitHub Access Management with Terraform

This Terraform configuration manages GitHub repository access for the Cosmos organization.

## Purpose

This configuration adds `gjermundgaraba` to the `poa` repository in the Cosmos organization with read access.

## Setup

1. **Prerequisites:**
   - Terraform installed (version 1.0+)
   - GitHub personal access token with `repo` and `admin:org` permissions

2. **Configuration:**
   ```bash
   # Copy the example variables file
   cp terraform.tfvars.example terraform.tfvars
   
   # Edit terraform.tfvars and add your GitHub token
   # github_token = "ghp_your_token_here"
   ```

3. **Apply the configuration:**
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## What this configuration does

- Adds `gjermundgaraba` as a collaborator to the `poa` repository
- Grants `pull` permission (read-only access)
- Uses the GitHub Terraform provider to manage access

## Security Notes

- The GitHub token should have minimal required permissions
- Store the token securely and never commit it to version control
- The `terraform.tfvars` file is gitignored to prevent token exposure