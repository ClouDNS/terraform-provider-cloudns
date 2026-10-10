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
