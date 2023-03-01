remote_state {
  backend = "gcs"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    skip_bucket_creation        = true
    skip_bucket_versioning      = false
    bucket                      = "ln-gcp-sh-infra-tfstates"
    prefix                      = "do/${path_relative_to_include()}/terraform.tfstate"
    impersonate_service_account = "tf-state@ln-gcp-sh-infra.iam.gserviceaccount.com"
  }
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
  token = var.do_token
}

EOF
}
