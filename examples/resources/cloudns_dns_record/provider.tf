terraform {
  required_providers {
    cloudns = {
      source = "registry.terraform.io/ClouDNS/terraform-provider-cloudns"
      version = "1.0.0"
    }
  }
}

provider "cloudns" {
  sub_auth_id = 20821
  password = "123456"
  rate_limit = 10
}