#adding master DNS zone
resource "cloudns_dns_zone" "zone-test" {
  domain     = "zonecreat2io122.co"
  type       = "master"
}

#adding A record with default geo
resource "cloudns_dns_record" "A-record-test" {
  name     = "Arecord"
  zone     = "asdasd.com"
  type     = "A"
  value     = "1.2.3.5"
  ttl      = "3600"
}

#adding slave DNS zone
resource "cloudns_dns_zone" "zone-slave-test" {
  domain     = "slavezonecreatsion21.co"
  type       = "slave"
  master     = "58.52.135.59"
}