locals {
  env = read_terragrunt_config(find_in_parent_folders("env.hcl"))
  secret_vars = yamldecode(sops_decrypt_file(find_in_parent_folders("secrets.yaml")))
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<EOF
terraform {
  required_providers {
    digitalocean = {
      source = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

variable do_token {
  type = string
}

provider "digitalocean" {
  token = local.secret_vars.do_token
}

EOF
}
