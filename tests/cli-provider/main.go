// Test-only executable: the production provider with an isolated HTTP transport.
package main

import (
	"fmt"
	"github.com/ClouDNS/terraform-provider-cloudns/internal/cloudns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"net/http"
	"net/url"
	"os"
)

type isolatedTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t isolatedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "api.cloudns.net" {
		return nil, fmt.Errorf("test transport rejects external host")
	}
	clone := r.Clone(r.Context())
	u := *r.URL
	u.Scheme, u.Host = t.target.Scheme, t.target.Host
	clone.URL = &u
	clone.Host = t.target.Host
	return t.base.RoundTrip(clone)
}
func main() {
	target, err := url.Parse(os.Getenv("CLOUDNS_TEST_API_ENDPOINT"))
	if err != nil || target.Scheme != "http" || target.Hostname() != "127.0.0.1" || target.Port() == "" {
		panic("loopback API endpoint required")
	}
	http.DefaultTransport = isolatedTransport{target, http.DefaultTransport}
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: cloudns.New()})
}
