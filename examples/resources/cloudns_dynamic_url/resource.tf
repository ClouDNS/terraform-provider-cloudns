variable "domain" {
  description = "An existing ClouDNS zone in your account."
  type        = string
}

resource "cloudns_dns_record" "home" {
  zone  = var.domain
  name  = "home"
  type  = "A"
  value = "192.0.2.1"
  ttl   = "60"

  # Let Dynamic DNS update the address without Terraform resetting it.
  lifecycle {
    ignore_changes = [value]
  }
}

output "dynamic_url" {
  value     = cloudns_dynamic_url.home.url
  sensitive = true
}

resource "cloudns_dynamic_url" "home" {
  domain   = cloudns_dns_record.home.zone
  recordid = cloudns_dns_record.home.id
}
