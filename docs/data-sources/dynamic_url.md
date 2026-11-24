---
page_title: "cloudns_dynamic_url Data Source - terraform-provider-cloudns"
subcategory: ""
description: |-
  Looks up the Dynamic DNS URL of an existing A or AAAA record.
---

# cloudns_dynamic_url (Data Source)

Looks up the Dynamic DNS URL of an existing A or AAAA record. The provider checks
the record's Dynamic DNS status first. If it is disabled, the lookup returns an
error without requesting a URL. If Dynamic DNS is disabled concurrently between
the status check and the URL request, the API can enable it again.

Use the [cloudns_dynamic_url resource](../resources/dynamic_url) to enable and
manage Dynamic DNS in Terraform.

## Example Usage

```terraform
data "cloudns_dynamic_url" "home" {
  domain   = "example.com"
  recordid = "123456"
}

output "dynamic_url" {
  value     = data.cloudns_dynamic_url.home.url
  sensitive = true
}
```

## Schema

### Required

- `domain` (String) The name of the DNS zone.
- `recordid` (String) The ID of the DNS record.

### Read-Only

- `id` (String) The DNS zone name and record ID, separated by a slash.
- `url` (String, Sensitive) The Dynamic DNS update URL. This value authorizes
  DNS updates and can be present in Terraform state and saved plans.
