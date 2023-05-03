locals {
  env = read_terragrunt_config(find_in_parent_folders("env.hcl"))
}


## Can't use impersonation when called from CI - see related issues:
## https://github.com/gruntwork-io/terragrunt/pull/2052
## https://github.com/gruntwork-io/terragrunt/issues/1997
#remote_state {
#  backend = "gcs"
#  generate = {
#    path      = "backend.tf"
#    if_exists = "overwrite_terragrunt"
#  }
#  config = {
#    bucket                      = local.env.locals.bucket
#    project                     = local.env.locals.project
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
    bucket  = "ln-gcp-sh-infra-prod-tfstates"
    prefix = "do/${path_relative_to_include()}/terraform.tfstate"
  }
}
EOF
}
