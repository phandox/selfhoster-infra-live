locals {
  common_vars = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals.common_vars
}

terraform {
  source = "github.com/phandox/selfhoster//infra-modules/common-firewall"
}

include "root" {
  path = find_in_parent_folders()
}

include "state" {
  path = find_in_parent_folders("state.hcl")
}

inputs = local.common_vars
