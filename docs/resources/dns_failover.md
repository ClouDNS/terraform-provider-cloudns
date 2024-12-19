---
page_title: "cloudns_dns_failover Resource - terraform-provider-cloudns"
subcategory: ""
description: |-
  A DNS failover record.
---

# cloudns_dns_failover (Resource)

A DNS failover record.


## Example Usage
In the exmaples we have cloudns_dns_zone.sub-cloudns-net which is a master DNS zone, which you can check in the dns_zone docs


### Activation a PING DNS failover check
```terraform
resource "cloudns_dns_failover" "cloudns-net-http" {
  domain    = cloudns_dns_zone.sub-cloudns-net.domain
  recordid  = cloudns_dns_record.sub-cloudns-net-a["www"].id
  checktype = "1"
  mainip    = cloudns_dns_record.sub-cloudns-net-a["www"].value
}
```

### Activating an UDP Failover check
```terraform
resource "cloudns_dns_failover" "cloudns-net-http" {
  domain    = cloudns_dns_zone.sub-cloudns-net.domain
  recordid  = cloudns_dns_record.sub-cloudns-net-a["www"].id
  checktype = "9"
  port      = "90"
  mainip    = cloudns_dns_record.sub-cloudns-net-a["www"].value
}
```


## Argument Reference

Some more information available in the [API documentation][1].

The following arguments are required:

* `domain` (Required) The name of the DNS zone (eg: mydomain.com)
* `recordid` (Required) The ID of the record for which the failover to be activated (eg: 123456789)
* `checktype` (Required) Monitoring check types for this Failover.

The following arguments are optional:

* `downeventhandler` (Optional) Event handler if Main IP is down.
* `upeventhandler` (Optional) Event handler if Main IP is up.
* `mainip` (Optional) Main IP address which will be monitored.
* `backupip1` (Optional) First Backup IP address.
* `backupip2` (Optional) Second Backup IP address.
* `backupip3` (Optional) Third Backup IP address.
* `backupip4` (Optional) Fourth Backup IP address.
* `backupip5` (Optional) Fifth Backup IP address.
* `monitoringregion` (Optional) Monitoring region or country.
* `checkperiod` (Optional) Time-frame between each monitoring check.
* `notificationmail` (Optional) Email notifications settings.
* `host` (Optional) A host to query.
* `port` (Optional) A port to query.
* `path` (Optional) Path for the URL.
* `content` (Optional) Parameter required for Custom HTTP and Custom HTTPS check types.
* `querytype` (Optional) Parameter required for DNS check type. It must contain the record type (e.g., A).
* `queryresponse` (Optional) Parameter required for DNS check type. You must fill in the response of the DNS server for this specific record.
* `latencylimit` (Optional) Only for Ping monitoring checks. If the latency of the check is above the limit, the check will be marked as DOWN.
* `timeout` (Optional) Only for Ping monitoring checks. Seconds to wait for a response. Must be between 1 and 5. Default value is 2.
* `checkregion` (Optional) The region from which the check is monitored (it is only received from API).
* `httprequesttype` (Optional) Only for HTTP/S checks. The request type will be used for the check. The default value is GET.


## Attribute Reference

* `id` The ID of this resource.


[1]: https://www.cloudns.net/wiki/article/272/
