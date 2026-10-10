package cloudns

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	api "github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.uber.org/ratelimit"
)

type dnssecTestTransport func(*http.Request) (*http.Response, error)

func (f dnssecTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type dnssecFake struct {
	enabled, optout, unavailable, missing, denied, failOptOut bool
	pending, reads, activations, deactivations                int
}

func (f *dnssecFake) install(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = dnssecTestTransport(func(r *http.Request) (*http.Response, error) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["domain-name"] != "example.test" || payload["auth-id"] != float64(42) {
			t.Errorf("unexpected payload %#v", payload)
		}
		body := `{"status":"Success"}`
		if f.denied {
			body = `{"status":"Failed","statusDescription":"Invalid authentication"}`
		} else if f.missing {
			body = `{"status":"Failed","statusDescription":"Missing domain-name"}`
		} else {
			switch r.URL.Path {
			case "/dns/is-dnssec-available.json":
				body = "1"
				if f.unavailable {
					body = "0"
				}
			case "/dns/activate-dnssec.json":
				f.enabled = true
				f.activations++
			case "/dns/deactivate-dnssec.json":
				f.enabled = false
				f.optout = false
				f.deactivations++
			case "/dns/set-dnssec-optout.json":
				if f.failOptOut {
					body = `{"status":"Failed","statusDescription":"opt-out failed"}`
				} else {
					f.optout = payload["status"] == float64(1)
				}
			case "/dns/get-dnssec-ds-records.json":
				f.reads++
				body = `{"status":"Failed","statusDescription":"dnssec_not_active"}`
				if f.enabled {
					optout := 0
					if f.optout {
						optout = 1
					}
					data := map[string]interface{}{"status": "1", "optout": optout, "ds": []string{"example.test. 3600 IN DS 123 13 2 ABCD"}, "dnskey": []string{"example.test. 3600 IN DNSKEY 257 3 13 KEY"}, "ds_records": []map[string]string{{"key_tag": "123", "algorithm": "13", "digest_type": "2", "digest": "ABCD"}}}
					if f.pending > 0 {
						f.pending--
						data["ds"] = []string{}
						data["dnskey"] = []string{}
					}
					raw, _ := json.Marshal(data)
					body = string(raw)
				}
			default:
				t.Fatalf("unexpected endpoint %s", r.URL.Path)
			}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
}
func dnssecTestMeta() ClientConfig {
	return ClientConfig{apiAccess: api.Apiaccess{Authid: 42, Authpassword: "dummy"}, rateLimiter: ratelimit.NewUnlimited()}
}
func dnssecTestData(t *testing.T, enabled, optout bool) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceDnsDNSSEC().Schema, map[string]interface{}{"domain": "example.test", "enabled": enabled, "opt_out": optout})
}
func TestDNSSECLifecycle(t *testing.T) {
	f := &dnssecFake{pending: 1}
	f.install(t)
	ctx := context.Background()
	meta := dnssecTestMeta()
	d := dnssecTestData(t, true, true)
	if diags := resourceDnsDNSSECApply(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Id() != "example.test" || !f.enabled || !f.optout || d.Get("ds").(*schema.Set).Len() != 1 || d.Get("dnskey").(*schema.Set).Len() != 1 || len(d.Get("ds_records").([]interface{})) != 1 {
		t.Fatalf("incorrect active state: %#v", d.State())
	}
	if f.reads < 4 {
		t.Fatal("did not wait for asynchronous keys")
	}
	if diags := resourceDnsDNSSECApply(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if f.activations != 1 {
		t.Fatal("re-activated already signed zone")
	}
	// Detect an external opt-out change, then apply the desired value in place.
	f.optout = false
	if diags := resourceDnsDNSSECRead(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Get("opt_out").(bool) {
		t.Fatal("drift not read")
	}
	d.Set("opt_out", true)
	if diags := resourceDnsDNSSECApply(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	d.Set("enabled", false)
	d.Set("opt_out", false)
	if diags := resourceDnsDNSSECApply(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if f.enabled || d.Get("ds").(*schema.Set).Len() != 0 || d.Get("dnskey").(*schema.Set).Len() != 0 || len(d.Get("ds_records").([]interface{})) != 0 {
		t.Fatal("disabled state retained keys")
	}
	if diags := resourceDnsDNSSECDelete(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Id() != "" || f.deactivations != 1 {
		t.Fatal("incorrect deletion")
	}
}
func TestDNSSECImportAndDestroy(t *testing.T) {
	f := &dnssecFake{enabled: true, optout: true}
	f.install(t)
	ctx := context.Background()
	meta := dnssecTestMeta()
	d := schema.TestResourceDataRaw(t, resourceDnsDNSSEC().Schema, nil)
	d.SetId("example.test")
	imported, err := resourceDnsDNSSEC().Importer.StateContext(ctx, d, meta)
	if err != nil || len(imported) != 1 {
		t.Fatalf("import %v", err)
	}
	if diags := resourceDnsDNSSECRead(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Get("domain") != "example.test" || !d.Get("opt_out").(bool) {
		t.Fatal("import state")
	}
	if diags := resourceDnsDNSSECDelete(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if f.enabled || f.deactivations != 1 || d.Id() != "" {
		t.Fatal("destroy did not disable signing")
	}
}
func TestDNSSECFailures(t *testing.T) {
	for _, tc := range []struct {
		name            string
		fake            dnssecFake
		enabled, optout bool
		wantID          string
	}{
		{"unavailable", dnssecFake{unavailable: true}, true, false, ""},
		{"authentication", dnssecFake{denied: true}, true, false, ""},
		{"invalid combination", dnssecFake{}, false, true, ""},
		{"partial activation", dnssecFake{failOptOut: true}, true, true, "example.test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.fake
			f.install(t)
			d := dnssecTestData(t, tc.enabled, tc.optout)
			if diags := resourceDnsDNSSECApply(context.Background(), d, dnssecTestMeta()); !diags.HasError() {
				t.Fatal("expected error")
			}
			if d.Id() != tc.wantID {
				t.Fatalf("id=%q", d.Id())
			}
		})
	}
}
func TestDNSSECReadMissingAndAuth(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "auth", true: "missing"}[missing], func(t *testing.T) {
			f := &dnssecFake{missing: missing, denied: !missing}
			f.install(t)
			d := dnssecTestData(t, true, false)
			d.SetId("example.test")
			diags := resourceDnsDNSSECRead(context.Background(), d, dnssecTestMeta())
			if diags.HasError() == missing {
				t.Fatalf("diags=%v", diags)
			}
			if (d.Id() == "") != missing {
				t.Fatal("incorrect state removal")
			}
		})
	}
}
