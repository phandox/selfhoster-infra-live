locals {
  dev_vars = read_terragrunt_config(find_in_parent_folders("dev.hcl"))
}
dependency "vpc" {
  config_path = "../vpc"

  mock_outputs = {
    vpc = {
      id = "dummy-id"
      ip_range = "10.0.10.0/24"
    }
  }
}

dependency "common-firewall" {
  config_path = "../common-firewall"
  mock_outputs = {
    fw-tags = {
      ssh = "dummy-tag"
      internet-egress = "dummy-tag"
    }
  }
}

terraform {
  source = "github.com:phandox/selfhoster/infra-modules//postgres-vm"
}

include "root" {
  path = find_in_parent_folders()
}

inputs = merge(
  local.dev_vars.locals,
  {
    instance_count = 1
    instance_size = "s-1vcpu-512mb-10gb"
    tags = [dependency.common-firewall.outputs.fw-tags["ssh"], dependency.common-firewall.outputs.fw-tags["internet-egress"]]
    vpc = dependency.vpc.outputs.vpc
  }
)