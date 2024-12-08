---
page_title: "cloudns_dns_record Resource - terraform-provider-cloudns"
subcategory: ""
description: |-
  A simple DNS record.
---

# cloudns_dns_record (Resource)

A simple DNS record.


## Example Usage

### Adding an A record on the apex of the "something.cloudns.net" zone
```terraform
resource "cloudns_dns_zone" "cloudns-net" {
  domain = "cloudns.net"
  type   = "master"
}

resource "cloudns_dns_record" "some-record" {
  # something.cloudns.net 600 in A 1.2.3.4
  zone  = cloudns_dns_zone.cloudns-net.id
  type  = "A"
  name  = ""
  value = "1.2.3.4"
  ttl   = "600"
}
```


### Adding an A record on the "something.cloudns.net" zone
```terraform
resource "cloudns_dns_zone" "cloudns-net" {
  domain = "cloudns.net"
  type   = "master"
}

resource "cloudns_dns_record" "some-record" {
  # something-else.something.cloudns.net 600 in A 1.2.3.4
  zone  = cloudns_dns_zone.cloudns-net.id
  type  = "A"
  name  = "something-else"
  value = "1.2.3.5"
  ttl   = "600"
}
```


### Adding an MX record on the apex of the "something.cloudns.net" zone
```terraform
resource "cloudns_dns_zone" "cloudns-net" {
  domain = "cloudns.net"
  type   = "master"
}

resource "cloudns_dns_record" "some-record" {
  # something.cloudns.net 600 in MX mail.example.com
  zone     = cloudns_dns_zone.cloudns-net.id
  type     = "MX"
  name     = ""
  value    = "mail.example.com"
  ttl      = "3600"
  priority = "20"
}
```


## Argument Reference

Some more information available in the [API documentation][1].

The following arguments are required:

* `zone` (Required) The domain name of the zone to add the record to.
* `type` (Required) The record type. See valid values below.
* `name` (Required) The hostname of the DNS record.
* `value` (Required) Value of the record, eg. an IP address or an A-record.
* `ttl` (Required) The TTL of the record, in seconds. See valid values below.

The following arguments are optional:

- `priority` (Optional) Priority for MX or SRV record.
- `weight` (Optional) Weight for SRV record.
- `port` (Optional) Port for SRV record.
- `frame` (Optional) Frame for WR record. Valid values are `0` (disabled,) or `1` (enabled.)
- `frametitle` (Optional) Title if frame is enabled in Web redirects.
- `framekeywords` (Optional) Keywords if frame is enabled in Web redirects.
- `framedescription` (Optional) Description if frame is enabled in Web redirects.
- `mobilemeta` (Optional) Mobile responsive meta tags if Web redirects with frame is enabled. Default value - 0.
- `savepath` (Optional) 0 or 1 for Web redirects.
- `redirecttype` (Optional) Web redirects type if frame is disabled. Valid values are `301` or `302`.
- `mail` (Optional) E-mail address for RP records.
- `txt` (Optional) Domain name for TXT record used in RP records.
- `algorithm` (Optional) Algorithm used to create the SSHFP fingerprint. Required for SSHFP records only.
- `fptype` (Optional) Type of the SSHFP algorithm. Required for SSHFP records only.
- `status` (Optional) The status of the record being created. Valid values are `0` (inactive) or `1` (active.) If omitted the record will be created active.
- `geodnslocation` (Optional) ID of a GeoDNS location for `A`, `AAAA`, `CNAME`, `NAPTR`, or `SRV` record types.
- `geodnscode` (Optional) Code of a GeoDNS location for `A`, `AAAA`, `CNAME`, `NAPTR`, or `SRV` record types.
- `caaflag` (Optional) 0 - Non critical or 128 - Critical.
- `caatype` (Optional) Type of CAA record. The available values are `"issue"`, `"issuewild"`,  and `"iodef"`.
- `caavalue` (Optional) Value of the CAA record.
- `tlsausage` (Optional) Shows the provided association that will be used.
- `tlsaselector` (Optional) Specifies which part of the TLS certificate presented by the server will be matched against the association data.
- `tlsamatchingtype` (Optional) Specifies how the certificate association is presented.
- `smimeausage` (Optional) Shows the provided association that will be used.
- `smimeaselector` (Optional) Specifies which part of the TLS certificate presented by the server will be matched against the association data.
- `smimeamatchingtype` (Optional) Specifies how the certificate association is presented.
- `keytag` (Optional) A numeric value used for identifying the referenced DS record.
- `digesttype` (Optional) The cryptographic hash algorithm used to create the Digest value.
- `order` (Optional) Specifies the order in which multiple NAPTR records must be processed (low to high).
- `pref` (Optional) Specifies the order (low to high) in which NAPTR records with equal `order` values should be processed.
- `flag` (Optional) Controls aspects of the rewriting and interpretation of the fields in the record.
- `params` (Optional) Specifies the service parameters applicable to this delegation path.
- `regexp` (Optional) Contains a substitution expression that is applied to the original string, held by the client in order to construct the next domain name to lookup.
- `replace` (Optional) Specifies the next domain name (fully qualified) to query for depending on the potential values found in the flags field.
- `certtype` (Optional) Type of the Certificate/CRL.
- `certkeytag` (Optional) A numeric value (0-65535), used to efficiently pick a CERT record.
- `certalgorithm` (Optional) Identifies the algorithm used to produce a legitimate signature.
- `latdeg` (Optional) A numeric value (0-90), sets the latitude degrees.
- `latmin` (Optional) A numeric value (0-59), sets the latitude minutes.
- `latsec` (Optional) A numeric value (0-59), sets the latitude seconds.
- `latdir` (Optional) Sets the latitude direction. Valid values are `"N"` (North,) or `"S"` (South.)
- `longdeg` (Optional) A numeric value (0-180), sets the longitude degrees.
- `longmin` (Optional) A numeric value (0-59), sets the longitude minutes.
- `longsec` (Optional) A numeric value (0-59), sets the longitude seconds.
- `longdir` (Optional) Sets the longitude direction. Valid values are `"W"` (West,) or `"E"` (East.)
- `altitude` (Optional) A numeric value (-100000.00 - 42849672.95), sets the altitude in meters.
- `size` (Optional) A numeric value (0 - 90000000.00), sets the size in meters.
- `hprecision` (Optional) A numeric value (0 - 90000000.00), sets the horizontal precision in meters.
- `vprecision` (Optional) A numeric value (0 - 90000000.00), sets the vertical precision in meters.
- `cpu` (Optional) The CPU of the server.
- `os` (Optional) The operating system of the server.


### Valid TTL values

The following values are valid TTL values of DNS records:

* 60 (1 minute)
* 300 (5 minutes)
* 900 (15 minutes)
* 1800 (30 minutes)
* 3600 (1 hour)
* 21600 (6 hours)
* 43200 (12 hours)
* 83400 (1 day)
* 172800 (2 days)
* 259200 (3 days)
* 604800 (1 week)
* 1209600 (2 weeks)
* 2592000 (1 month)

### Valid type values

The following values are valid for type values of DNS records:

* A
* AAAA
* ALIAS
* CAA
* CERT
* CNAME
* DNAME
* DS
* HINFO
* LOC
* MX
* NAPTR
* NS
* OPENPGPKEY
* PTR
* RP
* SMIMEA
* SPF
* SRV
* SSHFP
* TLSA
* TXT
* WR


## Attribute Reference

* `id` The ID of this resource.


## Import

In Terraform v1.5.0 and later, use an [`import` block][2] to import DNS records using their ID. For example:

```terraform
import {
  to = cloudns_dns_record.cloudns-net-record
  id = "cloudns.net/123456789"
}
```

Using `terraform import`, import DNS zones using the `domain`. For example:

```console
% terraform import cloudns_dns_record.cloudns-net-record cloudns.net/123456789
```
[1]: https://www.cloudns.net/wiki/article/58/
[2]: https://developer.hashicorp.com/terraform/language/import
