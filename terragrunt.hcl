generate "backend" {
  path = "backend.tf"
  if_exists = "overwrite_terragrunt"
  contents = <<EOF
terraform {
  backend "gcs" {
    bucket = "tf-states-org-test-luknagy-com"
    prefix = "do/${path_relative_to_include()}/terraform.tfstate"
    impersonate_service_account = "terraform-state-holder@booming-primer-369213.iam.gserviceaccount.com"
  }
}
EOF
}

generate "provider" {
  path = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents = <<EOF
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
  token = var.do_token
}

EOF
}