terraform {
  required_providers {
    cloudns = {
      source = "registry.terraform.io/cloudns/cloudns"
      version = "1.0.0"
    }
  }
}

provider "cloudns" {
  auth_id = ****
  password = "******"
  rate_limit = 10
}