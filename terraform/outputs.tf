# Outputs for GitHub Terraform configuration

output "collaborator_added" {
  description = "Confirmation that gjermundgaraba was added to poa repository"
  value       = "User ${github_repository_collaborator.gjermundgaraba_poa_access.username} added to ${github_repository_collaborator.gjermundgaraba_poa_access.repository} with ${github_repository_collaborator.gjermundgaraba_poa_access.permission} permission"
}