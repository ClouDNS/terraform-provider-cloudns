# https://www.cloudns.net/wiki/article/516/
resource "cloudns_dns_zone" "somedomain-com" {
  domain = "somedomain1.com"
  type   = "master"
}

resource "cloudns_dns_record" "somedomain-com-a" {
  for_each   = local.a
  name       = each.key
  zone       = cloudns_dns_zone.somedomain-com.domain
  type       = "A"
  value      = each.value
  ttl        = "600"
  depends_on = [cloudns_dns_zone.somedomain-com]
}

resource "cloudns_dns_record" "somedomain-com-mx" {
  for_each   = local.mx
  name       = each.key
  zone       = cloudns_dns_zone.somedomain-com.domain
  type       = "MX"
  value      = each.value
  ttl        = "600"
  depends_on = [cloudns_dns_zone.somedomain-com]
  priority   = ((index(keys(local.mx), each.key) + 1) * 10)
}

resource "cloudns_dns_record" "somedomain-com-cname" {
  for_each   = local.cname
  name       = each.key
  zone       = cloudns_dns_zone.somedomain-com.domain
  type       = "CNAME"
  value      = each.value
  ttl        = "600"
  depends_on = [cloudns_dns_zone.somedomain-com]
}

resource "cloudns_dns_record" "somedomain-com-txt" {
  for_each   = local.txt
  name       = each.key
  zone       = cloudns_dns_zone.somedomain-com.domain
  type       = "TXT"
  value      = each.value
  ttl        = "600"
  depends_on = [cloudns_dns_zone.somedomain-com]
}

resource "cloudns_dns_zone" "someotherdomain1-com" {
  domain          = "someotherdomain1.com"
  type            = "master"
  nameserver_type = "premium"
}

resource "cloudns_dns_zone" "someotherdomain2-com" {
  domain      = "someotherdomain2.com"
  type        = "master"
  nameservers = local.ns
}
