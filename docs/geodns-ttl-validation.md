# GeoDNS TTL preservation validation

Validated on 2026-10-11 using Go 1.25.8, the provider's existing cloudns-go
v1.2.0 dependency and a dedicated sub-user in a user-provided GeoDNS zone.
No credentials or existing zone contents are included here.

## Result

The final in-process provider integration run passed all ten cases in 52.5
seconds:

| Record | Location/configuration | TTL 300 to 3600 |
| --- | --- | --- |
| A, AAAA, ALIAS, CNAME, SRV, NAPTR | BG, configured by code | Passed for all six types |
| A | DEFAULT, EUR, US-1, configured by code | Passed for all three locations |
| Imported A | Existing BG location, both GeoDNS arguments omitted | Passed |

Each record used a collision-checked UUID hostname. Creation/import captured
the numeric location ID in state. The real Terraform SDK `Resource.Diff` and
`Resource.Apply` methods then exercised the provider update path. No replacement
was planned. Read-back compared every returned field except TTL, including ID,
location ID/name/code and structured SRV/NAPTR fields. Every case produced an
empty subsequent SDK diff.

An intentional BG-to-EUR code change also preserved the ID and converged. A
unit regression verifies that a code change plans the new location ID as computed
rather than promising the previous ID to Terraform.

All 30 original API rows matched the initial snapshot. Cleanup reconciled only
the test UUID names, excluded all original IDs and left no temporary records.
An earlier run passed nine cases but exposed a test-helper JSON type mismatch
for the numeric create-response ID. The helper was corrected; the clean final
run passed all ten cases, including import.

## Offline regression coverage

`go test ./... -count=1` passes with live checks disabled. The new tests cover:

- Current API location selection by exact record ID for all six GeoDNS types,
  including a same-name sibling and stale state/code.
- Refresh/import storing the existing location.
- API read errors and missing IDs preventing any update.
- Explicit location-ID and location-code changes.
- A new location code planning a new computed ID.
- Removing an optional code preserving the existing location.

The main-branch implementation failed the new TTL/location regressions for all
six types in an isolated baseline worktree. The existing Go library already
serializes `geodns-location`; no library or dependency-version change is required.

## CLI boundary and reproduction

A compiled provider and Terraform 1.14.0 were prepared for a separate CLI
plan/apply/plan check. This environment refused the Unix socket needed for the
Terraform plugin handshake (`socket: operation not permitted`). That attempt
created only a temporary import probe, which was removed; all original records
were unchanged. No successful Terraform CLI acceptance run is claimed here.

The successful live run invokes the actual provider callbacks and SDK diff/apply
in process, without an IPC listener. It establishes API preservation and SDK
convergence, but does not replace the outstanding CLI protocol/acceptance check.

For the in-process live checks, set `CLOUDNS_GEO_TTL_TEST=1`,
`CLOUDNS_GEO_TEST_DOMAIN`, `CLOUDNS_SUB_AUTH_ID` and `CLOUDNS_PASSWORD`, unset
`CLOUDNS_AUTH_ID`, and run:

```sh
go test ./internal/cloudns -run '^TestLiveGeoDNSTTLPreservation$' -count=1 -v
```

For CLI validation in an environment permitting plugin sockets, build the
provider and additionally set `CLOUDNS_TF_DEV_PROVIDER_DIR` to its directory and
optionally `TERRAFORM_BINARY`. Then run:

```sh
python3 tests/geodns_ttl_live.py
```

Both tests modify only their own new UUID records. Never run the regular zone
or DNSSEC acceptance suite against this existing-record test zone. The SDK test
reconciles ambiguous creates during cleanup by their UUID name; an unavailable
API can still require subsequent read-back and cleanup of test rows.

The provider update adds one API read when no explicit new location is requested.
If that read fails or the ID is missing, it sends no modification. This change
does not add support for any other previously unsupported GeoDNS feature.
