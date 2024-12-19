locals {
  a = {
    "@" : "125.127.127.127",
    "*" : "127.127.127.127",
    "something" : "192.168.0.1",
    "somethingelse" : "192.168.0.2"
  }
  mx = {
    "@" : "mf41.cloudns.net"
  }
  cname = {
    "www" : "@",
  }
  txt = {
    "@" : "v=spf1 include:spf.smtp.relay ~all"
  }

  ns = [
    "pns61.cloudns.net",
    "pns62.cloudns.com",
    "pns63.cloudns.net",
    "pns64.cloudns.uk",
  ]
}
