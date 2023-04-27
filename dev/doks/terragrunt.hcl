terraform {
  source = "github.com/phandox/selfhoster//infra-modules/doks"
}

locals {
  dev_vars = read_terragrunt_config(find_in_parent_folders("dev.hcl"))
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

include "state" {
  path = find_in_parent_folders("state.hcl")
}

inputs = merge(
  local.dev_vars.locals,
  {
    cluster_name = "doks-fra1-001"
    vpc_uuid     = dependency.vpc.outputs.vpc.id
    size         = "s-2vcpu-2gb"
  }
)
