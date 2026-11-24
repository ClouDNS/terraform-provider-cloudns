data "cloudns_dynamic_url" "home" {
  domain   = "example.com"
  recordid = "123456"
}

output "dynamic_url" {
  value     = data.cloudns_dynamic_url.home.url
  sensitive = true
}
