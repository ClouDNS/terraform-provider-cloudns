package cloudns

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func geoRecordData(t *testing.T, recordType, location, code string, changes map[string]string) *schema.ResourceData {
	t.Helper()
	attrs := map[string]string{"id": "2", "zone": "example.test", "name": "probe", "type": recordType,
		"value": "192.0.2.2", "ttl": "300", "geodnslocation": location, "geodnscode": code}
	diff := &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{}}
	for key, value := range changes {
		diff.Attributes[key] = &terraform.ResourceAttrDiff{Old: attrs[key], New: value}
	}
	d, err := schema.InternalMap(resourceDnsRecord().Schema).Data(&terraform.InstanceState{ID: "2", Attributes: attrs}, diff)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func installGeoRecordAPI(t *testing.T, recordType, location, failure string, missing bool) *[]map[string]interface{} {
	t.Helper()
	writes := []map[string]interface{}{}
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = dnssecTestTransport(func(r *http.Request) (*http.Response, error) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		body := `{"status":"Success"}`
		switch r.URL.Path {
		case "/dns/records.json":
			rows := map[string]interface{}{"1": map[string]interface{}{"id": "1", "type": recordType, "host": "probe", "record": "192.0.2.1", "ttl": "300", "geodns-location": "5"}}
			if !missing {
				rows["2"] = map[string]interface{}{"id": "2", "type": recordType, "host": "probe", "record": "192.0.2.2", "ttl": "3600", "geodns-location": location}
			}
			raw, _ := json.Marshal(rows)
			body = string(raw)
			if failure != "" {
				body = `{"status":"Failed","statusDescription":"` + failure + `"}`
			}
		case "/dns/mod-record.json":
			writes = append(writes, payload)
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	return &writes
}

func TestGeoDNSTTLUpdatePreservesLocationByID(t *testing.T) {
	for _, recordType := range []string{"A", "AAAA", "ALIAS", "CNAME", "SRV", "NAPTR"} {
		t.Run(recordType, func(t *testing.T) {
			writes := installGeoRecordAPI(t, recordType, "30", "", false)
			// Even stale state and an unchanged code must not overwrite the API location.
			d := geoRecordData(t, recordType, "5", "EUR", map[string]string{"ttl": "3600"})
			if diags := resourceDnsRecordUpdate(context.Background(), d, dnssecTestMeta()); diags.HasError() {
				t.Fatal(diags)
			}
			if len(*writes) != 1 || (*writes)[0]["record-id"] != float64(2) || (*writes)[0]["geodns-location"] != "30" || (*writes)[0]["ttl"] != float64(3600) {
				t.Fatalf("incorrect update: %#v", *writes)
			}
			if _, exists := (*writes)[0]["geodns-code"]; exists {
				t.Fatal("stale location code was sent")
			}
			if d.Id() != "2" || d.Get("geodnslocation") != "30" {
				t.Fatal("ID/location not retained in state")
			}
		})
	}
}

func TestGeoDNSRefreshAndImportCaptureLocation(t *testing.T) {
	installGeoRecordAPI(t, "A", "30", "", false)
	d := geoRecordData(t, "A", "", "", nil)
	if diags := resourceDnsRecordRead(context.Background(), d, dnssecTestMeta()); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Get("geodnslocation") != "30" {
		t.Fatal("refresh did not capture location")
	}
	d.SetId("example.test/2")
	if _, err := resourceDnsRecordImport(context.Background(), d, dnssecTestMeta()); err != nil {
		t.Fatal(err)
	}
	if d.Id() != "2" || d.Get("geodnslocation") != "30" {
		t.Fatal("import did not capture ID/location")
	}
}

func TestGeoDNSUpdateReadFailurePreventsWrites(t *testing.T) {
	for _, failure := range []string{"Invalid authentication", "Request blocked"} {
		t.Run(failure, func(t *testing.T) {
			writes := installGeoRecordAPI(t, "A", "30", failure, false)
			d := geoRecordData(t, "A", "", "", map[string]string{"ttl": "3600"})
			if diags := resourceDnsRecordUpdate(context.Background(), d, dnssecTestMeta()); !diags.HasError() || len(*writes) != 0 {
				t.Fatal("read failure did not prevent update")
			}
		})
	}
	writes := installGeoRecordAPI(t, "A", "30", "", true)
	d := geoRecordData(t, "A", "", "", map[string]string{"ttl": "3600"})
	if diags := resourceDnsRecordUpdate(context.Background(), d, dnssecTestMeta()); !diags.HasError() || len(*writes) != 0 {
		t.Fatal("missing ID did not prevent update")
	}
}

func TestGeoDNSExplicitLocationChangesRemainPossible(t *testing.T) {
	for _, attr := range []string{"geodnslocation", "geodnscode"} {
		t.Run(attr, func(t *testing.T) {
			writes := installGeoRecordAPI(t, "A", "5", "", false)
			value, apiKey, omitted := "5", "geodns-location", "geodns-code"
			if attr == "geodnscode" {
				value, apiKey, omitted = "EUR", "geodns-code", "geodns-location"
			}
			d := geoRecordData(t, "A", "30", "BG", map[string]string{attr: value})
			if diags := resourceDnsRecordUpdate(context.Background(), d, dnssecTestMeta()); diags.HasError() {
				t.Fatal(diags)
			}
			if len(*writes) != 1 || (*writes)[0][apiKey] != value {
				t.Fatalf("requested location not sent: %#v", *writes)
			}
			if _, exists := (*writes)[0][omitted]; exists {
				t.Fatal("previous location was also sent")
			}
		})
	}
}

func TestGeoDNSCodeChangePlansNewLocationAsComputed(t *testing.T) {
	d := geoRecordData(t, "A", "30", "BG", nil)
	config := terraform.NewResourceConfigRaw(map[string]interface{}{
		"zone": "example.test", "name": "probe", "type": "A", "value": "192.0.2.2",
		"ttl": 300, "geodnscode": "EUR",
	})
	diff, err := resourceDnsRecord().Diff(context.Background(), d.State(), config, dnssecTestMeta())
	if err != nil {
		t.Fatal(err)
	}
	attr := diff.Attributes["geodnslocation"]
	if attr == nil || !attr.NewComputed {
		t.Fatal("code change must not promise the previous computed location ID")
	}
}

func TestGeoDNSRemovingCodePreservesExistingLocation(t *testing.T) {
	writes := installGeoRecordAPI(t, "A", "30", "", false)
	d := geoRecordData(t, "A", "30", "BG", map[string]string{"ttl": "3600", "geodnscode": ""})
	if diags := resourceDnsRecordUpdate(context.Background(), d, dnssecTestMeta()); diags.HasError() {
		t.Fatal(diags)
	}
	if len(*writes) != 1 || (*writes)[0]["geodns-location"] != "30" {
		t.Fatal("removing optional code reset the existing location")
	}
}
