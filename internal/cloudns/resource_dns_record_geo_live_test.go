package cloudns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// This uses the provider's SDK diff/apply callbacks in process. The separate
// tests/geodns_ttl_live.py exercises the Terraform CLI/plugin handshake as well.
func TestLiveGeoDNSTTLPreservation(t *testing.T) {
	if os.Getenv("CLOUDNS_GEO_TTL_TEST") != "1" {
		t.Skip("live GeoDNS checks require explicit opt-in")
	}
	domain := os.Getenv("CLOUDNS_GEO_TEST_DOMAIN")
	if domain == "" || os.Getenv(EnvVarSubAuthId) == "" || os.Getenv(EnvVarPassword) == "" || os.Getenv(EnvVarAuthId) != "" {
		t.Fatal("dedicated GeoDNS zone and sub-user credentials required")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	api := func(endpoint string, fields url.Values, result interface{}) error {
		if fields == nil {
			fields = url.Values{}
		}
		fields.Set("sub-auth-id", os.Getenv(EnvVarSubAuthId))
		fields.Set("auth-password", os.Getenv(EnvVarPassword))
		fields.Set("domain-name", domain)
		response, err := client.PostForm("https://api.cloudns.net/dns/"+endpoint+".json", fields)
		if err != nil {
			return fmt.Errorf("%s transport error", endpoint)
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			return fmt.Errorf("%s HTTP %d", endpoint, response.StatusCode)
		}
		return json.NewDecoder(response.Body).Decode(result)
	}
	readRows := func() map[string]map[string]interface{} {
		var rows map[string]map[string]interface{}
		if err := api("records", url.Values{"include-notes": {"1"}}, &rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	baseline := readRows()
	prefix := "terraform-geo-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	for _, row := range baseline {
		if strings.Contains(row["host"].(string), prefix) {
			t.Fatal("UUID hostname collision")
		}
	}
	// Discover owned names even if a create response had an ambiguous outcome.
	t.Cleanup(func() {
		rows := readRows()
		for id, row := range rows {
			if strings.Contains(row["host"].(string), prefix) {
				if _, exists := baseline[id]; exists {
					t.Fatal("existing ID is ineligible for cleanup")
				}
				var response map[string]interface{}
				if err := api("delete-record", url.Values{"record-id": {id}}, &response); err != nil || response["status"] != "Success" {
					t.Errorf("cleanup unsuccessful for owned ID %s", id)
				}
			}
		}
		final := readRows()
		if !reflect.DeepEqual(final, baseline) {
			t.Error("zone snapshot differs after cleanup (values withheld)")
		} else {
			t.Logf("cleanup: all %d original rows unchanged, no temporary records", len(baseline))
		}
	})
	p := New()()
	if diags := p.Configure(context.Background(), terraform.NewResourceConfigRaw(nil)); diags.HasError() {
		t.Fatal("provider configuration failed")
	}
	meta := p.Meta()
	r := resourceDnsRecord()
	cases := []struct{ recordType, code, location string }{
		{"A", "BG", "30"}, {"AAAA", "BG", "30"}, {"ALIAS", "BG", "30"},
		{"CNAME", "BG", "30"}, {"SRV", "BG", "30"}, {"NAPTR", "BG", "30"},
		{"A", "DEFAULT", "1"}, {"A", "EUR", "5"}, {"A", "US-1", "259"}, {"A", "IMPORT", "30"},
	}
	for index, test := range cases {
		t.Run(test.recordType+"-"+test.code, func(t *testing.T) {
			ctx := context.Background()
			config := map[string]interface{}{"zone": domain, "name": fmt.Sprintf("%s-%d", prefix, index),
				"type": test.recordType, "value": "target.example.org", "ttl": 300, "geodnscode": test.code}
			switch test.recordType {
			case "A":
				config["value"] = "192.0.2.1"
			case "AAAA":
				config["value"] = "2001:db8::1"
			case "SRV":
				config["name"] = "_probe._tcp." + config["name"].(string)
				config["priority"], config["weight"], config["port"] = 10, 20, 443
			case "NAPTR":
				config["value"] = ""
				config["order"], config["pref"], config["flag"] = "10", "20", "S"
				config["params"], config["regexp"], config["replace"] = "SIP+D2U", "", "target.example.org"
			}
			d := schema.TestResourceDataRaw(t, r.Schema, config)
			if test.code == "IMPORT" {
				delete(config, "geodnscode")
				d = schema.TestResourceDataRaw(t, r.Schema, config)
				var created struct {
					Status string `json:"status"`
					Data   struct {
						ID json.Number `json:"id"`
					} `json:"data"`
				}
				if err := api("add-record", url.Values{"record-type": {"A"}, "host": {config["name"].(string)}, "record": {"192.0.2.1"}, "ttl": {"300"}, "geodns-code": {"BG"}}, &created); err != nil || created.Data.ID == "" {
					t.Fatal("owned import probe creation failed")
				}
				d.SetId(domain + "/" + created.Data.ID.String())
				if _, err := resourceDnsRecordImport(ctx, d, meta); err != nil {
					t.Fatal(err)
				}
			} else if diags := resourceDnsRecordCreate(ctx, d, meta); diags.HasError() {
				t.Fatal(diags)
			}
			id := d.Id()
			if _, exists := baseline[id]; exists || id == "" || d.Get("geodnslocation") != test.location {
				t.Fatal("owned ID/location not captured")
			}
			before := readRows()
			config["ttl"] = 3600
			prior := d.State()
			diff, err := r.Diff(ctx, prior, terraform.NewResourceConfigRaw(config), meta)
			if err != nil || diff == nil || diff.RequiresNew() || diff.Destroy {
				t.Fatalf("invalid TTL diff: %v", err)
			}
			for key, attr := range diff.Attributes {
				if key != "ttl" && !attr.NewComputed {
					t.Fatalf("unexpected non-TTL diff: %s", key)
				}
			}
			updated, diags := r.Apply(ctx, prior, diff, meta)
			if diags.HasError() || updated.ID != id {
				t.Fatalf("TTL apply failed: %v", diags)
			}
			after := readRows()
			for originalID, row := range baseline {
				if !reflect.DeepEqual(after[originalID], row) {
					t.Fatal("original zone row changed (values withheld)")
				}
			}
			if after[id]["ttl"] != "3600" {
				t.Fatal("TTL did not change")
			}
			oldRow, newRow := before[id], after[id]
			delete(oldRow, "ttl")
			delete(newRow, "ttl")
			if !reflect.DeepEqual(oldRow, newRow) {
				t.Fatal("non-TTL test record fields changed")
			}
			diff, err = r.Diff(ctx, updated, terraform.NewResourceConfigRaw(config), meta)
			if err != nil || (diff != nil && !diff.Empty()) {
				t.Fatalf("post-apply diff did not converge: %v", err)
			}
			if index == 0 {
				config["geodnscode"] = "EUR"
				diff, err = r.Diff(ctx, updated, terraform.NewResourceConfigRaw(config), meta)
				if err != nil {
					t.Fatal(err)
				}
				moved, diags := r.Apply(ctx, updated, diff, meta)
				if diags.HasError() || moved.ID != id || moved.Attributes["geodnslocation"] != "5" {
					t.Fatalf("explicit code change failed: %v", diags)
				}
				diff, err = r.Diff(ctx, moved, terraform.NewResourceConfigRaw(config), meta)
				if err != nil || (diff != nil && !diff.Empty()) {
					t.Fatal("explicit code change did not converge")
				}
			}
		})
		if t.Failed() {
			break
		}
	}
}
