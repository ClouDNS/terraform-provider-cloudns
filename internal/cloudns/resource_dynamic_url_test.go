package cloudns

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	api "github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.uber.org/ratelimit"
)

func TestResourceDynamicURLSchema(t *testing.T) {
	resource := resourceDynamicUrl()
	urlSchema := resource.Schema["url"]

	if !urlSchema.Computed {
		t.Error("url must be computed")
	}

	if !urlSchema.Sensitive {
		t.Error("url must be sensitive because it acts as a credential")
	}
}

type dynamicURLTestTransport func(*http.Request) (*http.Response, error)

func (f dynamicURLTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDynamicURLLifecycleDoesNotLogSecret(t *testing.T) {
	const fakeURL = "https://example.invalid/dynamic/FIXTURE_SECRET"
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	var endpoints []string
	http.DefaultTransport = dynamicURLTestTransport(func(r *http.Request) (*http.Response, error) {
		endpoints = append(endpoints, r.URL.Path)
		body := `{"url":"` + fakeURL + `"}`
		switch r.URL.Path {
		case "/dns/get-dynamic-url.json":
		case "/dns/disable-dynamic-url.json":
			body = `{"status":"Success"}`
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)
	resource := resourceDynamicUrl()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{"domain": "example.test", "recordid": "123"})
	meta := ClientConfig{apiAccess: api.Apiaccess{Authid: 1, Authpassword: "dummy"}, rateLimiter: ratelimit.NewUnlimited()}
	if diags := resource.CreateContext(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := resource.ReadContext(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Id() != "123" || d.Get("domain") != "example.test" || d.Get("recordid") != "123" || d.Get("url") != fakeURL {
		t.Fatal("incorrect dynamic URL state")
	}
	if d.State().Attributes["url"] != fakeURL {
		t.Fatal("sensitive URL must still be persisted in state")
	}
	if diags := resource.DeleteContext(ctx, d, meta); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Id() != "" || strings.Join(endpoints, ",") != "/dns/get-dynamic-url.json,/dns/get-dynamic-url.json,/dns/disable-dynamic-url.json" {
		t.Fatal("incorrect dynamic URL lifecycle")
	}
	w.Close()
	os.Stdout = oldStdout
	stdout, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "Dynamic URL state set successfully") {
		t.Fatal("debug log capture is inactive")
	}
	if strings.Contains(logs.String(), fakeURL) || strings.Contains(logs.String(), "FIXTURE_SECRET") || len(stdout) != 0 {
		t.Fatal("dynamic URL leaked to debug logs or stdout")
	}
}
