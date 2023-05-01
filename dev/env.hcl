locals {
  bucket  = "ln-gcp-sh-infra-tfstates"
  project = "ln-gcp-sh-infra"
  tf_sa   = "tf-state@ln-gcp-sh-infra.iam.gserviceaccount.com"
  secret_vars = yamldecode(sops_decrypt_file(find_in_parent_folders("secrets.yaml")))

  common_vars = {
    env    = "dev"
    region = "fra1"
    do_token = local.secret_vars.do_token
  }
}
