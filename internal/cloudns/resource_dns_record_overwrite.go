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

	"github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const defaultGeoDNSLocation = "1"

func sameDNSRecordHost(first, second string) bool {
	if first == "@" {
		first = ""
	}
	if second == "@" {
		second = ""
	}
	return strings.EqualFold(first, second)
}

func findDNSRecordToOverwrite(ctx context.Context, config ClientConfig, desired cloudns.Record) (*cloudns.Record, error) {
	config.rateLimiter.Take()
	records, err := (cloudns.Zone{Domain: desired.Domain}).List(&config.apiAccess)
	if err != nil {
		return nil, fmt.Errorf("list DNS records in %s for allow_overwrite: %w", desired.Domain, err)
	}

	var locations map[string]string
	if desired.GeodnsCode != "" {
		locations, err = listGeoDNSLocations(ctx, config, desired.Domain)
		if err != nil {
			return nil, err
		}
	}
	targetLocation, err := dnsRecordLocation(desired, locations)
	if err != nil {
		return nil, err
	}

	var match *cloudns.Record
	for _, current := range records {
		if !sameDNSRecordHost(current.Host, desired.Host) || !strings.EqualFold(current.Rtype, desired.Rtype) {
			continue
		}
		// An ID in the record listing already identifies the location.
		if current.GeodnsLocation != "" {
			current.GeodnsCode = ""
		}
		if current.GeodnsCode != "" && locations == nil {
			locations, err = listGeoDNSLocations(ctx, config, desired.Domain)
			if err != nil {
				return nil, err
			}
		}
		location, err := dnsRecordLocation(current, locations)
		if err != nil {
			return nil, err
		}
		if location != targetLocation {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple %s DNS records match host %q in zone %s for allow_overwrite; import the intended record with <zone>/<id>", desired.Rtype, desired.Host, desired.Domain)
		}
		if desired.GeodnsCode != "" || desired.GeodnsLocation != "" || current.GeodnsCode != "" {
			current.GeodnsLocation = location
		}
		match = &current
	}
	return match, nil
}

func dnsRecordLocation(record cloudns.Record, locations map[string]string) (string, error) {
	location := record.GeodnsLocation
	if record.GeodnsCode != "" {
		resolved := locations[strings.ToUpper(record.GeodnsCode)]
		if resolved == "" {
			return "", fmt.Errorf("unknown GeoDNS location code %q in zone %s", record.GeodnsCode, record.Domain)
		}
		if location != "" && location != resolved {
			return "", fmt.Errorf("geodnslocation %q and geodnscode %q select different locations in zone %s", location, record.GeodnsCode, record.Domain)
		}
		location = resolved
	}
	if location == "" {
		location = defaultGeoDNSLocation
	}
	return location, nil
}

func listGeoDNSLocations(ctx context.Context, config ClientConfig, zone string) (map[string]string, error) {
	fields := url.Values{
		"domain-name":   {zone},
		"auth-password": {config.apiAccess.Authpassword},
	}
	if config.apiAccess.Authid != 0 {
		fields.Set("auth-id", strconv.Itoa(config.apiAccess.Authid))
	} else {
		fields.Set("sub-auth-id", strconv.Itoa(config.apiAccess.Subauthid))
	}
	config.rateLimiter.Take()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.cloudns.net/dns/get-geodns-locations.json", strings.NewReader(fields.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	response, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("list GeoDNS locations in %s: %w", zone, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list GeoDNS locations in %s: HTTP %d", zone, response.StatusCode)
	}
	var rows []struct {
		ID   json.Number `json:"id"`
		Code string      `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("list GeoDNS locations in %s: %w", zone, err)
	}
	locations := make(map[string]string, len(rows))
	for _, row := range rows {
		locations[strings.ToUpper(row.Code)] = row.ID.String()
	}
	return locations, nil
}

func overwriteDNSRecord(ctx context.Context, d *schema.ResourceData, config ClientConfig, desired, existing cloudns.Record) diag.Diagnostics {
	if _, configured := d.GetOkExists("status"); configured {
		return diag.Errorf("allow_overwrite cannot set status on an existing DNS record %s/%s", desired.Domain, existing.ID)
	}
	// The current SDK sends the wrong CERT type and omits LOC latitude seconds.
	switch desired.Rtype {
	case "CERT", "LOC":
		return diag.Errorf("allow_overwrite cannot safely update %s records; import %s/%s to manage this record", desired.Rtype, desired.Domain, existing.ID)
	}
	if id, err := strconv.Atoi(existing.ID); err != nil || id <= 0 {
		return diag.Errorf("invalid existing DNS record ID %q in zone %s", existing.ID, desired.Domain)
	}
	desired.ID = existing.ID
	desired.Host = existing.Host
	desired.GeodnsLocation = existing.GeodnsLocation
	desired.GeodnsCode = ""
	config.rateLimiter.Take()
	if _, err := desired.Update(&config.apiAccess); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(existing.ID)
	diags := resourceDnsRecordRead(ctx, d, config)
	if d.Id() == "" {
		d.SetId(existing.ID)
		return diag.Errorf("could not read DNS record %s in zone %s after overwrite", existing.ID, desired.Domain)
	}
	return diags
}
