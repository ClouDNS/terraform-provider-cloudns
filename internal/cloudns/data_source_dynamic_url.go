package cloudns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDynamicURL() *schema.Resource {
	return &schema.Resource{
		Description: "Looks up the Dynamic DNS URL of an existing A or AAAA record with Dynamic DNS enabled.",
		ReadContext: dataSourceDynamicURLRead,
		Schema: map[string]*schema.Schema{
			"domain": {
				Description:  "The name of the DNS zone.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"recordid": {
				Description:  "The ID of the DNS record.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"url": {
				Description: "The Dynamic DNS update URL. Treat this value as a credential.",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func dataSourceDynamicURLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	domain := d.Get("domain").(string)
	recordID := d.Get("recordid").(string)
	fields := url.Values{
		"domain-name":   {domain},
		"record-id":     {recordID},
		"auth-password": {config.apiAccess.Authpassword},
	}
	if config.apiAccess.Authid != 0 {
		fields.Set("auth-id", strconv.Itoa(config.apiAccess.Authid))
	} else {
		fields.Set("sub-auth-id", strconv.Itoa(config.apiAccess.Subauthid))
	}

	record, err := readDynamicURLAPI(ctx, config, "get-record", fields)
	if err != nil {
		return diag.Errorf("Check Dynamic DNS status for record %s in zone %s: %s", recordID, domain, err)
	}
	if record.ID != recordID {
		return diag.Errorf("ClouDNS did not return record %s in zone %s", recordID, domain)
	}
	if record.DynamicURLStatus == nil || (*record.DynamicURLStatus != 0 && *record.DynamicURLStatus != 1) {
		return diag.Errorf("ClouDNS did not return a recognized Dynamic DNS status for record %s in zone %s", recordID, domain)
	}
	if *record.DynamicURLStatus == 0 {
		return diag.Errorf("Dynamic DNS is disabled for record %s in zone %s", recordID, domain)
	}

	// This API call may activate Dynamic DNS if it is disabled after the status check.
	result, err := readDynamicURLAPI(ctx, config, "get-dynamic-url", fields)
	if err != nil {
		return diag.Errorf("Get Dynamic DNS URL for record %s in zone %s: %s", recordID, domain, err)
	}
	updateURL, err := url.ParseRequestURI(result.URL)
	if err != nil || updateURL.Host == "" || (updateURL.Scheme != "https" && updateURL.Scheme != "http") {
		return diag.Errorf("ClouDNS did not return a valid Dynamic DNS URL for record %s in zone %s", recordID, domain)
	}
	if err := d.Set("url", result.URL); err != nil {
		return diag.Errorf("Could not store the Dynamic DNS URL for record %s in zone %s", recordID, domain)
	}
	d.SetId(domain + "/" + recordID)
	return nil
}

type dynamicURLAPIResponse struct {
	Status           json.RawMessage `json:"status"`
	ID               string          `json:"id"`
	DynamicURLStatus *int            `json:"dynamicurl_status"`
	URL              string          `json:"url"`
}

func readDynamicURLAPI(ctx context.Context, config ClientConfig, endpoint string, fields url.Values) (dynamicURLAPIResponse, error) {
	var result dynamicURLAPIResponse
	config.rateLimiter.Take()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.cloudns.net/dns/"+endpoint+".json", strings.NewReader(fields.Encode()))
	if err != nil {
		return result, fmt.Errorf("could not build the API request")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		return result, fmt.Errorf("API request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("API returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("API returned an invalid response")
	}
	if string(result.Status) == `"Failed"` {
		return result, fmt.Errorf("API rejected the request; verify credentials and permissions and that the record exists")
	}
	return result, nil
}
