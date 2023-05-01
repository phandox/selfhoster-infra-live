locals {
#  secret_vars = yamldecode(sops_decrypt_file(find_in_parent_folders("secrets.yaml")))
  # Terraform state settings
  bucket  = "ln-gcp-sh-infra-prod-tfstates"
  project = "ln-gcp-sh-infra-prod"

  # Prod variables
  common_vars = {
    env    = "prod"
    region = "fra1"
#    do_token = local.secret_vars.do_token
  }
}
