## 1.2.0 (2026-10-10)

- Add `cloudns_dns_dnssec` to manage zone signing and NSEC3 opt-out, with DS/DNSKEY outputs and import support.
- Use cloudns-go v1.1.0 for DNSSEC support.
- Update acceptance tests for the current zone API and run live tests only when `TF_ACC=1`.

## 0.1.0 (Unreleased)

BACKWARDS INCOMPATIBILITIES / NOTES:
