terraform {
  required_providers {
    cloudns = {
      source = "cloudns/cloudns"
      version = "1.0.0"
    }
  }
}

provider "cloudns" {
  auth_id = xxxx
  password = "xxxx"
  rate_limit = 10
}