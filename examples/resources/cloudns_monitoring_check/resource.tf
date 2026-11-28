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

resource "cloudns_monitoring_check" "dns" {
  name           = "Authoritative DNS"
  check_type     = "dns"
  ip             = "192.0.2.53"
  host           = "www.example.com"
  query_type     = "A"
  query_response = "192.0.2.10"
  check_period   = 600
  enabled        = false
}
