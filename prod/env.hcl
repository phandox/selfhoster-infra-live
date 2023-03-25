locals {
  # Terraform state settings
  bucket  = "ln-gcp-sh-infra-prod-tfstates"
  project = "ln-gcp-sh-infra-prod"
  tf_sa   = "tf-state@ln-gcp-sh-infra-prod.iam.gserviceaccount.com"

  # Prod variables
  common_vars = {
    env    = "prod"
    region = "fra1"
  }
}
