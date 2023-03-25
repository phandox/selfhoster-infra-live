terraform {
  source = "github.com/phandox/selfhoster//infra-modules/doks?ref=v1.2.0"
}

locals {
  common_vars = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals.common_vars
}

dependency "vpc" {
  config_path = "../vpc"

  mock_outputs = {
    vpc = {
      id       = "dummy-id"
      ip_range = "10.0.10.0/24"
    }
  }
}

include "root" {
  path = find_in_parent_folders()
}

inputs = merge(
  local.common_vars,
  {
    cluster_name = "doks-fra1-${local.common_vars.env}-001"
    vpc_uuid     = dependency.vpc.outputs.vpc.id
    size         = "s-2vcpu-2gb"
  }
)
