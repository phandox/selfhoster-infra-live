locals {
  common_vars = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals.common_vars
}

terraform {
  source = "github.com/phandox/selfhoster//infra-modules/volumes?ref=v1.2.0"
}

include "root" {
  path = find_in_parent_folders()
}

include "state" {
  path = find_in_parent_folders("state.hcl")
}

inputs = merge(
  local.common_vars,
  {
    name     = "pgsql-prod-data"
    size     = 1
    fs_label = "pgsql_prod_data"
  }
)
