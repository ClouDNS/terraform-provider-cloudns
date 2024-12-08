#adding A record with geo
resource "cloudns_dns_record" "A-record-test-geo" {
  name           = "A1record"
  zone           = "asdasd.com"
  geodnslocation = "5"
  type           = "A"
  value          = "1.2.3.5"
  ttl            = "3600"
}

#adding A record with default geo
resource "cloudns_dns_record" "A-record-test" {
  name  = "A2record"
  zone  = "asdasd.com"
  type  = "A"
  value = "1.2.3.5"
  ttl   = "3600"
}

# Adding WR record
resource "cloudns_dns_record" "WR-record-tes2t" {
  name             = "webredirect.te2st2"
  zone             = "asdasd.com"
  type             = "WR"
  value            = "https://cloudns.net"
  ttl              = "3600"
  frame            = 1
  frametitle       = "someshit"
  framekeywords    = "somekeywords"
  framedescription = "description"
  mobilemeta       = "1"
  savepath         = "1"
  redirecttype     = "302"
}

# Adding SRV record needs to be fixed

#adding RP record
resource "cloudns_dns_record" "RP-record-test" {
  name = "rprecord"
  zone = "asdasd.com"
  type = "RP"
  mail = "venelin@cloudns.net"
  txt  = "someshit.com"
  ttl  = "3600"
}

#adding SSHFP record
resource "cloudns_dns_record" "SSHFP-record" {
  name      = "rpr23ecord"
  zone      = "asdasd.com"
  type      = "SSHFP"
  algorithm = "1"
  fptype    = "1"
  value     = "9fd1935a5739a39fe6c79f2754076880c7d79bd3"
  ttl       = "3600"
}

#adding NAPTR record
resource "cloudns_dns_record" "NAPTR-record-test" {
  name   = "naptr"
  zone   = "asdasd.com"
  type   = "NAPTR"
  order  = "2"
  pref   = "1"
  flag   = "S"
  params = "someshit"
  regexp = "laina"
  ttl    = "3600"
}

#adding CAA record
resource "cloudns_dns_record" "CAA-record-test" {
  name     = ""
  zone     = "asdasd.com"
  type     = "CAA"
  caaflag  = "0"
  caatype  = "issuewild"
  caavalue = "9fd1935a5739a39fe6c79f2754076880c7d79bd3"
  ttl      = "3600"
}

#adding TLSA record
resource "cloudns_dns_record" "TLSA-record-test" {
  name             = "_80._sip"
  zone             = "asdasd.com"
  type             = "TLSA"
  tlsausage        = "1"
  tlsaselector     = "1"
  tlsamatchingtype = "1"
  value            = "764129429D318DA37504F04DCDDBC0CCA556EC73423DB0DA0DD2307359DAAAE0"
  ttl              = "3600"
}

#adding NS record
resource "cloudns_dns_record" "NS-record-test" {
  name  = "ns"
  zone  = "asdasd.com"
  type  = "NS"
  value = "ns21.cloudns.net"
  ttl   = "3600"
}

#adding DS record
resource "cloudns_dns_record" "DS-record-test" {
  name       = "ns"
  zone       = "asdasd.com"
  type       = "DS"
  keytag     = "111"
  algorithm  = "13"
  digesttype = "3"
  value      = "1BE27F63E3D0EA37782E227EF9DD883D0BC2F4F8445C1BE48ABF1A59442B57A2"
  ttl        = "3600"
}

#adding PTR record
resource "cloudns_dns_record" "PTR-record-test" {
  name  = "1.3.5"
  zone  = "asdasd.com"
  type  = "PTR"
  value = "domain.com"
  ttl   = "3600"
}

#adding HINFO record
resource "cloudns_dns_record" "HINFO-record-test" {
  name = "information"
  zone = "asdasd.com"
  type = "HINFO"
  cpu  = "Intel"
  os   = "Windows"
  ttl  = "3600"
}

#adding LOC record
resource "cloudns_dns_record" "LOC-record-test" {
  name       = "information"
  zone       = "asdasd.com"
  type       = "LOC"
  latdeg     = "12"
  latmin     = "13"
  latsec     = "14"
  latdir     = "S"
  longdeg    = "16"
  longmin    = "17"
  longsec    = "18"
  longdir    = "E"
  altitude   = "12.50"
  size       = "17.50"
  hprecision = "212.20"
  vprecision = "312.20"
  ttl        = "3600"
}

#adding DNAME record
resource "cloudns_dns_record" "DNAME-record-test" {
  name  = "DNAME"
  zone  = "asdasd.com"
  type  = "DNAME"
  value = "somedomain.com"
  ttl   = "3600"
}

#adding SMIMEA record
resource "cloudns_dns_record" "SMIMEA-record-test" {
  zone               = "asdasd.com"
  type               = "SMIMEA"
  name               = "SMIMEA"
  smimeausage        = "0"
  smimeaselector     = "1"
  smimeamatchingtype = "1"
  value              = "1234"
  ttl                = "3600"
}

#adding SRV record
resource "cloudns_dns_record" "SRV-record-test" {
  name     = "_sip._tcp"
  zone     = "asdasd.com"
  type     = "SRV"
  priority = "12"
  weight   = "12"
  port     = "80"
  value    = "somedomain.com"
  ttl      = "3600"
}

#testing RP record
resource "cloudns_dns_record" "RP-record-test-import" {
  name = "rp23record"
  zone = "asdasd.com"
  type = "RP"
  mail = "venelin@cloudns.net"
  txt  = "someshit.com"
  ttl  = "3600"
}