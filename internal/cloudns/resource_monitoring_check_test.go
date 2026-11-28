package cloudns

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	api "github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.uber.org/ratelimit"
)

func TestMonitoringCheckFieldsPreserveZeroAndExcludeOtherTypes(t *testing.T) {
	resource := resourceMonitoringCheck()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "test", "check_type": "tcp", "ip": "192.0.2.10",
		"port": 443, "open_port": 0, "enabled": false, "path": "/previous",
	})
	check := monitoringCheckFromResource(d, 123)
	if check.ID != 123 || check.CheckType != 8 || check.Settings.Get("open_port") != "0" ||
		check.Settings.Get("state") != "0" || check.Settings.Has("path") {
		t.Fatal("TCP request did not preserve explicit zeroes or included stale Web settings")
	}
}

func TestMonitoringCheckLifecycle(t *testing.T) {
	provider := New()()
	if err := provider.InternalValidate(); err != nil {
		t.Fatal(err)
	}
	resource := provider.ResourcesMap["cloudns_monitoring_check"]
	for _, field := range []string{"heartbeat_url", "custom_header_value", "url"} {
		if !resource.Schema[field].Sensitive {
			t.Fatalf("%s must be sensitive", field)
		}
	}
	config := ClientConfig{apiAccess: api.Apiaccess{Subauthid: 42, Authpassword: "fixture-password"}, rateLimiter: ratelimit.NewUnlimited()}
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "test", "check_type": "heartbeat", "check_period": 600,
		"enabled": false, "notification_email": "monitoring@example.invalid",
	})
	ctx := context.Background()
	resultGet := `{"id":"123","name":"test","check_type":"12","state":"0","check_period":"600","heartbeat_url":"https://example.invalid/secret","status":"-1"}`
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	http.DefaultTransport = dynamicURLTestTransport(func(r *http.Request) (*http.Response, error) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(raw))
		if err != nil || r.Method != http.MethodPost || form.Get("sub-auth-id") != "42" {
			t.Fatal("unexpected Monitoring request")
		}
		payload := resultGet
		switch r.URL.Path {
		case "/monitoring/create.json":
			if form.Get("state") != "0" || form.Get("check_period") != "600" || form.Get("type") != "mail" {
				t.Fatal("incorrect create settings")
			}
			payload = `{"status":"Success","id":123}`
		case "/monitoring/get-monitoring-check.json":
			if form.Get("id") != "123" {
				t.Fatal("read must use the resource ID")
			}
		case "/monitoring/update.json":
			if form.Get("id") != "123" || form.Get("check_period") != "600" || form.Get("state") != "0" || form.Has("value") {
				t.Fatal("update did not retain check configuration")
			}
			payload = `{"status":"Success"}`
		case "/monitoring/delete.json":
			payload = `{"status":"Failed","statusDescription":"Invalid monitoring check."}`
		default:
			t.Fatalf("unexpected Monitoring endpoint %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})
	if diags := resource.CreateContext(ctx, d, config); diags.HasError() || d.Id() != "123" || d.Get("heartbeat_url") == "" {
		t.Fatalf("create or refresh failed: %v", diags)
	}
	if diags := resource.UpdateContext(ctx, d, config); diags.HasError() || d.Id() != "123" {
		t.Fatalf("update failed: %v", diags)
	}
	resultGet = `{"status":"Failed","statusDescription":"Restricted access!"}`
	if diags := resource.ReadContext(ctx, d, config); !diags.HasError() || d.Id() != "123" {
		t.Fatal("permission error must preserve the ID")
	}
	resultGet = `{"status":"Failed","statusDescription":"Invalid monitoring check."}`
	if diags := resource.ReadContext(ctx, d, config); diags.HasError() || d.Id() != "" {
		t.Fatal("missing check must be removed from state")
	}
	d.SetId("123")
	if diags := resource.DeleteContext(ctx, d, config); diags.HasError() || d.Id() != "" {
		t.Fatal("deleting an already absent check must succeed")
	}
}
