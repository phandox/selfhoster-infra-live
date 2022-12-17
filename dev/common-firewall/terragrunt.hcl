terraform {
  source = "github.com:phandox/selfhoster/infra-modules//common-firewall"
}

include "root" {
  path = find_in_parent_folders()
}