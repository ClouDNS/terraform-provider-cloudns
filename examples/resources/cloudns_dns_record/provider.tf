terraform {
  required_providers {
    cloudns = {
      source = "Cloudns/cloudns"
      version = "1.0.0"
    }
  }
}

provider "cloudns" {
  sub_auth_id = xxxx
  password = "xxxx"
  rate_limit = 10
}