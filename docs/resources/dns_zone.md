---
page_title: "cloudns_dns_zone Resource - terraform-provider-cloudns"
subcategory: ""
description: |-
  A simple DNS zone.
---

# cloudns_dns_zone (Resource)

A simple DNS zone.


## Example Usage

In the examples below simple master and slave zones are created.

### Standard master zone
```terraform
resource "cloudns_dns_zone" "cloudns-net" {
  domain = "cloudns.net"
  type   = "master"
}
```


### Standard slave zone
```terraform
resource "cloudns_dns_zone" "cloudns-net" {
  domain = "cloudns.net"
  type   = "slave"
  master = "127.0.0.1"
}
```

In the examples below simple master and slave zones are created with custom nameserver setups.

### Standard master zone specifying nameserver type
```terraform
resource "cloudns_dns_zone" "cloudns-net" {
  domain          = "cloudns.net"
  type            = "master"
  nameserver_type = "premium"
}
```


### Standard master zone specifying nameservers
```terraform
resource "cloudns_dns_zone" "cloudns-net" {
  domain      = "cloudns.net"
  type        = "master"
  nameservers = [
    "ns1.domain.com",
    "ns2.domain.net",
    "ns3.domain.org",
  ]
}
```


## Argument Reference

Some more information available in the [API documentation][1].

The following arguments are required:

* `domain` - (Required) The name of the DNS zone (eg: mydomain.com)
* `type` - (Required) The type of the DNS zone. Valid values are `"master"` and `"slave"`

The following arguments are optional:

* `master`- (Optional) The IP of the master server. Required if `type` is `"slave"`.
* `nameserver_type` - (Optional) The type of nameservers to assign to the zone upon creation. Valid values are `"all"`, `"free"`, and `"premium"`. Changing this will force a new resource be created.
* `nameservers` - (Optional) The nameservers to assign to the zone upon creation. Setting this will overwrite the setting of `nameserver_type`. Changing this will force a new resource be created.


## Attribute Reference

* `id` (String) The ID of this resource.


## Import

In Terraform v1.5.0 and later, use an [`import` block][2] to import DNS zones using the `domain`. For example:

```terraform
import {
  to = cloudns_dns_zone.cloudns-net
  id = "cloudns.net"
}
```

Using `terraform import`, import DNS zones using the `domain`. For example:

```console
% terraform import cloudns_dns_zone.cloudns-net cloudns.net
```

[1]: https://www.cloudns.net/wiki/article/48/
[2]: https://developer.hashicorp.com/terraform/language/import
