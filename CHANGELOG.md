## 1.3.0 (2026-10-10)

- Mark the `cloudns_dynamic_url` URL as sensitive and remove full response debug logging (#50).
- Use cloudns-go v1.2.0 to prevent Dynamic DNS URLs from being printed to stdout.
- Add Dynamic DNS URL documentation, an example, and state/logging regression tests (#49).

Sensitive redacts standard Terraform output; the URL remains in Terraform state and saved plans.

## 1.2.0 (2026-10-10)

- Add `cloudns_dns_dnssec` to manage zone signing and NSEC3 opt-out, with DS/DNSKEY outputs and import support.
- Use cloudns-go v1.1.0 for DNSSEC support.
- Update acceptance tests for the current zone API and run live tests only when `TF_ACC=1`.

## 0.1.0 (Unreleased)

BACKWARDS INCOMPATIBILITIES / NOTES:
