terraform {
  required_providers {
    cloudns = {
      source = "Cloudns/cloudns"
      version = "1.0.0"
    }
  }
}

provider "cloudns" {
  sub_auth_id = 20821
  password = "123456"
  rate_limit = 10
}