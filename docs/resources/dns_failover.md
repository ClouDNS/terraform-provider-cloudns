---
page_title: "cloudns_dns_failover Resource - terraform-provider-cloudns"
subcategory: ""
description: |-
  A DNS failover record.
---

# cloudns_dns_failover (Resource)

A DNS failover record.

## Example Usage
In the exmaples we have cloudns_dns_zone.sub-testzone-bg which is a master DNS zone, which you can check in the dns_zone docs


```terraform
# Activation a PING DNS failover check
resource "cloudns_dns_failover" "testzone-bg-http" {
  domain            = cloudns_dns_zone.sub-testzone-bg.domain
  recordid          = cloudns_dns_record.sub-testzone-bg-a["something"].id
  checktype         = "1"
  mainip            = cloudns_dns_record.sub-testzone-bg-a["something"].value
}


# Activating an UDP Failover check
resource "cloudns_dns_failover" "testzone-bg-http" {
  domain            = cloudns_dns_zone.sub-testzone-bg.domain
  recordid          = cloudns_dns_record.sub-testzone-bg-a["something"].id
  checktype         = "9"
  port              = "90"
  mainip            = cloudns_dns_record.sub-testzone-bg-a["something"].value
  depends_on = [ cloudns_dns_zone.sub-testzone-bg ]
}
```

## Schema

### Required

- `domain` (String) The name of the DNS zone (eg: mydomain.com)
- `recordid` (String) The ID of the record for which the failover to be activated (eg: 123456789)
- `checktype` (string) Monitoring check types for this Failover.



### Optional

- `downeventhandler` (String) Event handler if Main IP is down.
- `upeventhandler` (String) Event handler if Main IP is up.
- `mainip` (String) Main IP address which will be monitored.
- `backupip1` (String) First Backup IP address.
- `backupip2` (String) Second Backup IP address.
- `backupip3` (String) Third Backup IP address.
- `backupip4` (String) Fourth Backup IP address.
- `backupip5` (String) Fifth Backup IP address.
- `monitoringregion` (String) Monitoring region or country.
- `checkperiod` (String) Time-frame between each monitoring check.
- `notificationmail` (String) Email notifications settings.
- `host` (String) A host to query.
- `port` (String) A port to query.
- `path` (String) Path for the URL.
- `content` (String) Parameter required for Custom HTTP and Custom HTTPS check types.
- `querytype` (String) Parameter required for DNS check type. It must contain the record type (e.g., A).
- `queryresponse` (String) Parameter required for DNS check type. You must fill in the response of the DNS server for this specific record.
- `latencylimit` (String) Only for Ping monitoring checks. If the latency of the check is above the limit, the check will be marked as DOWN.
- `timeout` (String) Only for Ping monitoring checks. Seconds to wait for a response. Must be between 1 and 5. Default value is 2.
- `checkregion` (String) The region from which the check is monitored (it is only received from API).
- `httprequesttype` (String) Only for HTTP/S checks. The request type will be used for the check. The default value is GET.



### Read-Only

- `id` (String) The ID of this resource.
