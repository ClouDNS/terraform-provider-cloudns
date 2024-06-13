terraform {
  required_providers {
    cloudns = {
      source = "cloudns/cloudns"
      version = "1.0.0"
    }
  }
}

provider "cloudns" {
  auth_id = 21670
  password = "123456"
  rate_limit = 10
}