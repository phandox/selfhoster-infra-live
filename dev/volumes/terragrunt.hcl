locals {
  dev_vars = read_terragrunt_config(find_in_parent_folders("dev.hcl"))
}

terraform {
  source = "github.com/phandox/selfhoster//infra-modules/volumes"
}

include "root" {
  path = find_in_parent_folders()
}

inputs = merge(
  local.dev_vars.locals,
  {
    name     = "postgres-data-volume"
    size     = 1
    fs_label = "pgsql_data"
  }
)
