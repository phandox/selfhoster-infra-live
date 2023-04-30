locals {
  env = read_terragrunt_config(find_in_parent_folders("env.hcl"))
}


#remote_state {
#  backend = "gcs"
#  generate = {
#    path      = "backend.tf"
#    if_exists = "overwrite_terragrunt"
#  }
#  config = {
#    bucket                      = local.env.locals.bucket
#    project                     = local.env.locals.project
##    impersonate_service_account = local.env.locals.tf_sa
#    location                    = "europe-west3"
#    prefix                      = "do/${path_relative_to_include()}/terraform.tfstate"
#  }
#}

generate "backend" {
  path = "backend.tf"
  if_exists = "overwrite_terragrunt"
  contents = <<EOF
terraform {
  backend "gcs" {
    bucket  = "ln-gcp-sh-infra-tfstates"
    impersonate_service_account = "tf-state@ln-gcp-sh-infra.iam.gserviceaccount.com"
    prefix = "do/${path_relative_to_include()}/terraform.tfstate"
  }
}
EOF
}
