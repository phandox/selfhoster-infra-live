locals {
  common_vars = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals.common_vars
}

terraform {
  source = "github.com/phandox/selfhoster//infra-modules/vpc?ref=v1.2.0"
}

include "root" {
  path = find_in_parent_folders()
}

inputs = merge(
  local.common_vars,
  {
    ip_range = "10.0.200.0/24"
  }
)
