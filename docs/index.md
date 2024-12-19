---
page_title: "cloudns Provider"
subcategory: ""
description: |-
  A simple provider for maintaining DNS zones and records.  
---

# cloudns Provider

Use the ClouDNS provider to interact with the ClouDNS API.
Currently the provider supports maintaining DNS zones, records, and failover records.


## Example Usage

Terraform 0.13 and later:
```terraform
terraform {
  required_providers {
    cloudns = {
      source = "Cloudns/cloudns"
      version = "~>1.0.0"
    }
  }
}

# Configure the ClouDNS provider
provider "cloudns" {
  # Optional, ClouDNS currently maxxes out at 20 requests per second per ip. Defaults to 5.
  rate_limit = 5
}

# Create a DNS zone
resource "cloudns_dns_zone" "example_com" {
  domain          = "example.com"
  type            = "master"
  nameserver_type = "premium"
}          
```

Terraform 0.12 and earlier:
```terraform
# Configure the ClouDNS provider
provider "cloudns" {
  # Optional, ClouDNS currently maxxes out at 20 requests per second per ip. Defaults to 5.
  rate_limit = 5
}

# Create a DNS zone
resource "cloudns_dns_zone" "example_com" {
  domain          = "example.com"
  type            = "master"
  nameserver_type = "premium"
}
```

## Authentication and Configuration

In order to use the provider you have to create either an API user or an API sub-user. You can do this in the [API settings][1].


### Provider Configuration

!> **Warning:** Hard-coded credentials are not recommended in any Terraform configuration and risks secret leakage should this file ever be committed to a public version control system.

Credentials can be provided by adding an `auth_id` or `sub_auth_id` along with `password` to the `cloudns` provider block.

Usage:

```terraform
provider "cloudns" {
  auth_id  = 1234
  password = "verysecret"
}
```

## ClouDNS Configuration Reference

|Setting|Provider|[Environment Variable][envvars]|
|-------|--------|-------------------------------|
|Auth ID|`auth_id`|`CLOUDNS_AUTH_ID`|
|Sub-auth ID|`sub_auth_id`|`CLOUDNS_SUB_AUTH_ID`|
|Password|`password`|`CLOUDNS_PASSWORD`|
|Rate limit|`rate_limit`|N/A|
