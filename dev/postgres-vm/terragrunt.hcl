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

dependency "common-firewall" {
  config_path = "../common-firewall"
  mock_outputs = {
    fw-tags = {
      ssh             = "dummy-tag"
      internet-egress = "dummy-tag"
    }
  }
}

dependency "persistent-volume" {
  config_path = "../volumes"
  mock_outputs = {
    volume_id = "mock-id"
  }
}

terraform {
  source = "github.com/phandox/selfhoster//infra-modules/postgres-vm"
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
    instance_count = 1
    instance_size  = "s-1vcpu-512mb-10gb"
    tags           = [dependency.common-firewall.outputs.fw-tags["ssh"], dependency.common-firewall.outputs.fw-tags["internet-egress"], "psql"]
    vpc            = dependency.vpc.outputs.vpc
    volume_id      = dependency.persistent-volume.outputs.volume_id
  }
)
