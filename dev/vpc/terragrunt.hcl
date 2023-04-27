terraform {
  source = "github.com/phandox/selfhoster//infra-modules/vpc"
}

include "root" {
  path = find_in_parent_folders()
}

locals {
  dev_vars = read_terragrunt_config(find_in_parent_folders("dev.hcl"))
}

include "state" {
  path = find_in_parent_folders("state.hcl")
}


inputs = merge(
  local.dev_vars.locals,
  {
    ip_range = "10.0.100.0/24"
  }
)
