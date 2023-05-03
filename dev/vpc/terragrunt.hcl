terraform {
  source = "github.com/phandox/selfhoster//infra-modules/vpc"
}

include "root" {
  path = find_in_parent_folders()
}

locals {
  common_vars = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals.common_vars
}

include "state" {
  path = find_in_parent_folders("state.hcl")
}


inputs = merge(
  local.common_vars,
  {
    ip_range = "10.0.100.0/24"
  }
)
