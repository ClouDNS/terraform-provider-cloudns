package cloudns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDnsZone() *schema.Resource {
	return &schema.Resource{
		Description: "A DNS zone managed by ClouDNS.",

		CreateContext: resourceDnsZoneCreate,
		ReadContext:   resourceDnsZoneRead,
		UpdateContext: resourceDnsZoneUpdate,
		DeleteContext: resourceDnsZoneDelete,
		CustomizeDiff: resourceDnsZoneCustomizeDiff,

		Importer: &schema.ResourceImporter{
			StateContext: resourceDnsZoneImport,
		},

		Schema: map[string]*schema.Schema{
			"domain": {
				Description: "The name of the DNS zone.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"master": {
				Description: "Master IP for slave zone",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"nameserver_type": {
				Description:   "The type of nameservers to use (all/free/premium.)",
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"nameservers"},
				ValidateFunc: func(val any, key string) (warns []string, errs []error) {
					acceptableValues := []string{"all", "free", "premium"}
					if !slices.Contains(acceptableValues, val.(string)) {
						errs = append(errs, fmt.Errorf("%q must be one of %s, got %s", key, strings.Join(acceptableValues, ", "), val))
					}
					return
				},
			},
			"nameservers": {
				Description:   "A set of NS servers to use for this zone.",
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"nameserver_type"},
				Computed:      true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"type": {
				Description: "The type of the DNS zone (master/slave/parked/geodns).",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceDnsZoneCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientConfig := meta.(ClientConfig)
	zoneToCreate := toApiZone(d)

	// Determine (and eventually sort) NS records
	_, nsExist := d.GetOk("nameservers")
	_, nstExist := d.GetOk("nameserver_type")
	if nsExist || nstExist {
		nsList, _ := cloudns.Ns{}.List(clientConfig.apiAccess)
		zoneToCreate.Ns = getNsNames(d, nsList)
		d.Set("nameservers", zoneToCreate.Ns)
	}

	resp := zoneToCreate
	var err error
	if zoneToCreate.Ztype == "master" && hasEmptyNameserverList(d.GetRawConfig()) {
		err = createZoneWithoutNS(ctx, clientConfig, zoneToCreate.Domain)
	} else {
		resp, err = zoneToCreate.Create(&clientConfig.apiAccess)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	if len(zoneToCreate.Ns) == 0 {
		tflog.Debug(ctx, fmt.Sprintf("CREATE DNS zone: %s, type: %s", resp.Domain, resp.Ztype))
	} else {
		tflog.Debug(ctx, fmt.Sprintf("CREATE DNS zone: %s, type: %s, ns: %s", resp.Domain, resp.Ztype, strings.Join(zoneToCreate.Ns, ", ")))
	}

	d.SetId(zoneToCreate.Domain)
	return resourceDnsZoneRead(ctx, d, meta)
}

func resourceDnsZoneCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if d.Get("type") != "master" || !hasEmptyNameserverList(d.GetRawConfig()) {
		return nil
	}
	if _, configured := d.GetOk("nameserver_type"); configured {
		return fmt.Errorf("nameserver_type cannot be combined with an empty nameservers list")
	}
	if d.Id() != "" {
		// An empty list controls the initial records. NS records added later
		// must not cause the whole zone to be replaced.
		return d.Clear("nameservers")
	}
	return nil
}

func hasEmptyNameserverList(config cty.Value) bool {
	if config.IsNull() || !config.IsKnown() {
		return false
	}
	nameservers := config.GetAttr("nameservers")
	return nameservers.IsKnown() && !nameservers.IsNull() && nameservers.LengthInt() == 0
}

func createZoneWithoutNS(ctx context.Context, config ClientConfig, domain string) error {
	fields := url.Values{
		"auth-password": {config.apiAccess.Authpassword},
		"domain-name":   {domain},
		"zone-type":     {"master"},
		// The API uses default NS records when "ns": [] is sent as JSON.
		"ns[]": {""},
	}
	if config.apiAccess.Authid != 0 {
		fields.Set("auth-id", strconv.Itoa(config.apiAccess.Authid))
	} else {
		fields.Set("sub-auth-id", strconv.Itoa(config.apiAccess.Subauthid))
	}
	config.rateLimiter.Take()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.cloudns.net/dns/register.json", strings.NewReader(fields.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	response, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("create DNS zone %s without NS records: %w", domain, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("create DNS zone %s without NS records: HTTP %d", domain, response.StatusCode)
	}
	var result struct {
		Status      string `json:"status"`
		Description string `json:"statusDescription"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("create DNS zone %s without NS records: invalid API response: %w", domain, err)
	}
	if result.Status != "Success" {
		return fmt.Errorf("create DNS zone %s without NS records: API rejected request: %s", domain, result.Description)
	}
	return nil
}

func resourceDnsZoneRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientConfig := meta.(ClientConfig)
	zoneToRead := toApiZone(d)

	clientConfig.rateLimiter.Take()
	zoneRead, err := zoneToRead.Read(&clientConfig.apiAccess)
	if err != nil {
		if isNotFoundErr(err) {
			tflog.Warn(ctx, fmt.Sprintf("DNS zone not found: %s. Removing from state.", zoneToRead.Domain))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if zoneRead.Domain == "" {
		tflog.Warn(ctx, fmt.Sprintf("Received unexpected empty response for DNS zone: %s. Removing from state.", zoneToRead.Domain))
		d.SetId("")
		return nil
	}

	nsRecords, err := getFilteredZoneRecords(zoneToRead, clientConfig, []string{"NS"})
	if err == nil && (zoneRead.Ztype == "master" || len(nsRecords) > 0) {
		zoneRead.Ns = sortNsNames(nsRecords)
	} else if zoneRead.Ztype == "master" && slices.Equal(zoneRead.Ns, []string{""}) {
		zoneRead.Ns = nil
	}

	d.Set("domain", zoneRead.Domain)
	d.Set("type", zoneRead.Ztype)
	d.Set("nameservers", zoneRead.Ns)

	return nil
}

func resourceDnsZoneUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Errorf("Updating DNS zones is not supported.")
}

func resourceDnsZoneDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientConfig := meta.(ClientConfig)
	zoneToDelete := toApiZone(d)

	resp, err := zoneToDelete.Destroy(&clientConfig.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Delete DNS zone: %s, type: %s", resp.Domain, resp.Ztype))

	d.SetId("")
	return nil
}

func resourceDnsZoneImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	clientConfig := meta.(ClientConfig)
	domain := d.Id()

	clientConfig.rateLimiter.Take()
	zoneToRead := cloudns.Zone{Domain: domain}
	zoneRead, err := zoneToRead.Read(&clientConfig.apiAccess)
	if err != nil {
		return nil, err
	}

	if zoneRead.Domain == "" {
		return nil, fmt.Errorf("Zone not found: %#v", domain)
	}

	nsRecords, err := getFilteredZoneRecords(zoneToRead, clientConfig, []string{"NS"})
	if err == nil && (zoneRead.Ztype == "master" || len(nsRecords) > 0) {
		zoneRead.Ns = sortNsNames(nsRecords)
	} else if zoneRead.Ztype == "master" && slices.Equal(zoneRead.Ns, []string{""}) {
		zoneRead.Ns = nil
	}

	err = updateZoneState(d, &zoneRead)
	if err != nil {
		return nil, err
	}
	d.SetId(domain)

	tflog.Debug(ctx, fmt.Sprintf("IMPORT Zone %s", domain))

	return []*schema.ResourceData{d}, nil
}

func updateZoneState(d *schema.ResourceData, zone *cloudns.Zone) error {
	if err := d.Set("domain", zone.Domain); err != nil {
		return err
	}

	if err := d.Set("type", zone.Ztype); err != nil {
		return err
	}

	if zone.Ztype == "slave" && zone.Master != "" {
		if err := d.Set("master", zone.Master); err != nil {
			return err
		}
	}

	if err := d.Set("nameservers", zone.Ns); err != nil {
		return err
	}

	return nil
}

func sortNsNames(uns []string) []string {
	sort.Slice(uns, func(i, j int) bool {
		return uns[i] < uns[j]
	})

	var rns []string
	rns = append(rns, uns...)

	return rns
}

func getNsNames(d *schema.ResourceData, nsList []cloudns.Ns) []string {
	ns, isset := d.GetOk("nameservers")
	if isset {
		var rns []string
		for _, rec := range ns.([]interface{}) {
			rns = append(rns, rec.(string))
		}
		rns = sortNsNames(rns)
		return slices.CompactFunc(rns, strings.EqualFold)
	}

	filter, isset := d.GetOk("nameserver_type")
	if !isset || filter == "all" {
		return nil
	}

	var fns []string
	for _, ns := range nsList {
		if filter == ns.Type {
			fns = append(fns, ns.Name)
		}
	}

	// we compact and sort NS records to avoid creating diffs should there be changes in the API response
	fns = sortNsNames(fns)
	fns = slices.CompactFunc(fns, strings.EqualFold)

	return fns
}

func getFilteredZoneRecords(z cloudns.Zone, c ClientConfig, filter []string) ([]string, error) {
	// TODO: functionality should be moved to the `cloudns-go` repository
	zoneRecords, err := z.List(&c.apiAccess)
	if err != nil && len(zoneRecords) == 0 {
		return nil, fmt.Errorf("found no records zone for %s", z.Domain)
	}

	var rns []string
	for _, rec := range zoneRecords {
		if len(filter) == 0 || slices.Contains(filter, rec.Rtype) {
			rns = append(rns, rec.Record)
		}
	}

	return rns, nil
}

func toApiZone(d *schema.ResourceData) cloudns.Zone {
	domain := d.Get("domain").(string)
	zoneType := d.Get("type").(string)
	master := d.Get("master").(string)

	return cloudns.Zone{
		Domain: domain,
		Ztype:  zoneType,
		Master: master,
		Ns:     []string{},
	}
}

func isNotFoundErr(err error) bool {
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no zones returned")
}
