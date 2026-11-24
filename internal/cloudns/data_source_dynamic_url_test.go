package cloudns

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	api "github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.uber.org/ratelimit"
)

func TestDataSourceDynamicURLChecksStatusBeforeRequestingURL(t *testing.T) {
	const fakeURL = "https://example.invalid/dynamicURL/?q=FIXTURE_SECRET"
	const activeRecord = `{"id":"123","status":1,"dynamicurl_status":1}`
	const getRecord = "/dns/get-record.json"
	const getURL = "/dns/get-dynamic-url.json"

	provider := New()()
	resource := provider.DataSourcesMap["cloudns_dynamic_url"]
	if resource == nil || !resource.Schema["url"].Sensitive || !resource.Schema["url"].Computed {
		t.Fatal("data source must be registered with a sensitive computed URL")
	}

	for _, tc := range []struct {
		name          string
		recordBody    string
		urlBody       string
		wantEndpoints []string
		wantError     string
	}{
		{"active", activeRecord, `{"url":"` + fakeURL + `"}`, []string{getRecord, getURL}, ""},
		{"disabled", `{"id":"123","status":1,"dynamicurl_status":0}`, "", []string{getRecord}, "Dynamic DNS is disabled"},
		{"missing status", `{"id":"123","status":1}`, "", []string{getRecord}, "recognized Dynamic DNS status"},
		{"unknown status", `{"id":"123","status":1,"dynamicurl_status":2}`, "", []string{getRecord}, "recognized Dynamic DNS status"},
		{"wrong record", `{"id":"456","dynamicurl_status":1}`, "", []string{getRecord}, "did not return record"},
		{"record API failure", `{"status":"Failed","statusDescription":"FIXTURE_SECRET"}`, "", []string{getRecord}, "API rejected"},
		{"URL API failure", activeRecord, `{"status":"Failed","statusDescription":"FIXTURE_SECRET"}`, []string{getRecord, getURL}, "API rejected"},
		{"URL missing", activeRecord, `{"status":"Success"}`, []string{getRecord, getURL}, "valid Dynamic DNS URL"},
		{"invalid URL", activeRecord, `{"url":"FIXTURE_SECRET"}`, []string{getRecord, getURL}, "valid Dynamic DNS URL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var endpoints []string
			oldTransport := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = oldTransport })
			http.DefaultTransport = dynamicURLTestTransport(func(r *http.Request) (*http.Response, error) {
				endpoints = append(endpoints, r.URL.Path)
				if r.Method != http.MethodPost || r.URL.Host != "api.cloudns.net" || r.URL.RawQuery != "" {
					t.Fatal("wrong API request method or destination")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				fields, err := url.ParseQuery(string(body))
				if err != nil || fields.Get("domain-name") != "example.test" || fields.Get("record-id") != "123" ||
					fields.Get("sub-auth-id") != "10" || fields.Get("auth-id") != "" || fields.Get("auth-password") != "fixture-password" {
					t.Fatal("wrong API request parameters")
				}
				payload := tc.recordBody
				if r.URL.Path == getURL {
					payload = tc.urlBody
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload))}, nil
			})

			d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{"domain": "example.test", "recordid": "123"})
			config := ClientConfig{apiAccess: api.Apiaccess{Subauthid: 10, Authpassword: "fixture-password"}, rateLimiter: ratelimit.NewUnlimited()}
			diags := resource.ReadContext(context.Background(), d, config)
			if !slices.Equal(endpoints, tc.wantEndpoints) {
				t.Fatalf("API calls = %v, want %v", endpoints, tc.wantEndpoints)
			}
			if tc.wantError != "" {
				if !diags.HasError() || !strings.Contains(diags[0].Summary, tc.wantError) {
					t.Fatalf("expected diagnostic containing %q", tc.wantError)
				}
				if strings.Contains(diags[0].Summary+diags[0].Detail, "FIXTURE_SECRET") || d.Id() != "" || d.Get("url") != "" {
					t.Fatal("failed lookup exposed a secret or populated state")
				}
				return
			}
			if diags.HasError() || d.Id() != "example.test/123" || d.Get("url") != fakeURL {
				t.Fatal("successful lookup did not populate the expected state")
			}
		})
	}
}
