---
page_title: "cloudns_dns_dnssec Resource - terraform-provider-cloudns"
subcategory: ""
description: |-
  Manages DNSSEC signing for an existing ClouDNS zone.
---

# cloudns_dns_dnssec (Resource)

Manages DNSSEC signing for an existing ClouDNS zone. Publish DS records at the
registrar separately, and remove them before disabling or destroying this resource.

Use one DNSSEC resource per existing zone. Enabling signing waits for DS and
DNSKEY records to be generated, with a default create/update timeout of five
minutes. Read refreshes signing, opt-out and key data from ClouDNS, including
changes made outside Terraform. `opt_out = true` requires `enabled = true`.

This resource manages zone signing only. Publish the exported DS (and DNSKEY,
if required) at your registrar to establish the chain of trust. Before setting
`enabled = false` or destroying this resource, remove DS records at the registrar
and wait for cached DS records to expire. Destroy disables signing without
removing the DNS zone. Avoid `create_before_destroy` when replacing this resource
with another resource managing the same zone.

## Example Usage

```terraform
resource "cloudns_dns_zone" "example" {
  domain          = "example.com"
  type            = "master"
  nameserver_type = "premium"
}

resource "cloudns_dns_dnssec" "example" {
  domain  = cloudns_dns_zone.example.id
  enabled = true
  opt_out = false
}

output "dnssec_ds" {
  value = cloudns_dns_dnssec.example.ds
}

output "dnssec_ds_records" {
  value = cloudns_dns_dnssec.example.ds_records
}

output "dnssec_dnskey" {
  value = cloudns_dns_dnssec.example.dnskey
}
```

## Schema

### Required

- `domain` (String) Existing DNS zone name. Can reference `cloudns_dns_zone.example.id`. Changing it replaces the resource.

### Optional

- `enabled` (Boolean) Whether DNSSEC signing is enabled. Defaults to `true`.
- `opt_out` (Boolean) Whether NSEC3 opt-out is enabled. Requires `enabled = true`. Defaults to `false`.
- `timeouts` (Block) Create and update timeouts (default: `5m`).

### Read-Only

- `id` (String) DNS zone name.
- `ds` (Set of String) DS records in zone-file format, to publish at the registrar.
- `dnskey` (Set of String) DNSKEY records in zone-file format.
- `ds_records` (List of Object) Structured DS parameters for registrar integrations: `key_tag`, `algorithm`, `algorithm_name`, `digest_type`, `digest_type_name`, and `digest` (all strings).

## Import

Import using the DNS zone name. Import reads the current enabled and opt-out state.
Set those values in your configuration to avoid changing signing on the next apply.

```shell
terraform import cloudns_dns_dnssec.example example.com
```
