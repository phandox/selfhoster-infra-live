locals {
  # Terraform state settings
  bucket  = "ln-gcp-sh-infra-prod-tfstates"
  project = "ln-gcp-sh-infra-prod"

  # Prod variables
  common_vars = {
    env    = "prod"
    region = "fra1"
  }
}
