---
page_title: "cloudns_monitoring_check Resource - terraform-provider-cloudns"
description: |-
  Manages a standalone ClouDNS Monitoring check.
---

# cloudns_monitoring_check (Resource)

Manages a standalone ClouDNS Monitoring check. Availability of check types,
intervals and regions depends on the Monitoring plan.

## Example Usage

Replace the documentation IP address and hostname with the service you control,
then enable the check.

```terraform
resource "cloudns_monitoring_check" "web" {
  name             = "Website health"
  check_type       = "web"
  ip               = "192.0.2.10"
  host             = "www.example.com"
  port             = 443
  path             = "/health"
  check_period     = 600
  http_status_code = 200
  enabled          = false
}
```

Use `check_type = "dns"` with the nameserver in `ip`, the name to query in
`host`, and the expected `query_type` and `query_response`.

## Configuration

- `name` (Required) Check name, up to 32 characters.
- `check_type` (Required) One of `ping`, `web`, `tcp`, `udp`, `dns`,
  `heartbeat`, `tls`, `smtp`, `streaming`, `imap` or `keyword`.
- `ip` Target IP address or hostname, required except for Heartbeat.
- `ip_type` Address family for hostname targets. Set to `1` for both,
  `2` for IPv4 or `3` for IPv6.
- `check_period` Check interval in seconds.
- `monitoring_region` Region, country code or node ID.
- `status_change_checks` Number of checks required to confirm a status change.
- `enabled` Set to `false` to pause the check. New checks default to active.
- `notification_email` Initial email notification recipient. Required for
  creation with an API sub-user. Changing it replaces the check. Manage later
  notification changes in ClouDNS.

| Check type | Additional required fields | Optional settings |
| --- | --- | --- |
| Ping | | `ping_threshold`, `latency_limit`, `timeout` |
| Web | `host`, `port` | `path`, `http_protocol`, `http_request_type`, `http_status_code`, `timeout`, `content`, `content_match`, `custom_header`, `custom_header_value` |
| TCP, UDP | `port`, `open_port` | |
| DNS | `host`, `query_type`, `query_response` | |
| Heartbeat | | |
| TLS | `host`, `port` | `notification_period`, `cacert` |
| SMTP | `port` | `connection_security` |
| Streaming | `url` | `audio_threshold` |
| IMAP | `port` | `connection_security`, `timeout` |
| Keyword | `host`, `port`, `keyword` | `path`, `http_protocol`, `http_request_type`, `http_status_code`, `custom_header`, `custom_header_value` |

For TCP and UDP, use `open_port = 1` to expect an open port or
`open_port = 0` to expect a closed port. Optional API settings are read
into state. Set a different value explicitly to change a setting.

## Attributes

- `id` The ClouDNS Monitoring check ID.
- `heartbeat_url` Callback URL for Heartbeat checks. Terraform marks this
  attribute as sensitive. Treat it as a credential and protect state and plan
  files. Terraform also marks `custom_header_value` and the Streaming `url`
  as sensitive.

## Import

Import an existing check by its numeric ID and omit `notification_email`
from the resource configuration to avoid replacing the imported check.

```sh
terraform import cloudns_monitoring_check.web 12345
```
