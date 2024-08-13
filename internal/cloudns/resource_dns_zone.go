package cloudns

import (
	"context"
	"fmt"
	"strings"

	"github.com/ClouDNS/cloudns-go"
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
			"type": {
				Description: "The type of the DNS zone (master/slave/parked/geodns).",
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
		},
	}
}

func resourceDnsZoneCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientConfig := meta.(ClientConfig)
	zoneToCreate := toApiZone(d)

	resp, err := zoneToCreate.Create(&clientConfig.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("CREATE DNS zone: %s, type: %s", resp.Domain, resp.Ztype))

	d.SetId(zoneToCreate.Domain)
	return resourceDnsZoneRead(ctx, d, meta)
}

func resourceDnsZoneRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientConfig := meta.(ClientConfig)
	lookup := toApiZone(d)

	clientConfig.rateLimiter.Take()
	zoneRead, err := lookup.Read(&clientConfig.apiAccess)
	if err != nil {
		if isNotFoundErr(err) {
			tflog.Warn(ctx, fmt.Sprintf("DNS zone not found: %s. Removing from state.", lookup.Domain))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if zoneRead.Domain == "" {
		tflog.Warn(ctx, fmt.Sprintf("Received unexpected empty response for DNS zone: %s. Removing from state.", lookup.Domain))
		d.SetId("")
		return nil
	}

	d.Set("domain", zoneRead.Domain)
	d.Set("type", zoneRead.Ztype)
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
	config := meta.(ClientConfig)
	domain := d.Id()

	config.rateLimiter.Take()
	zoneRead, err := cloudns.Zone{Domain: domain}.Read(&config.apiAccess)
	if err != nil {
		return nil, err
	}

	if zoneRead.Domain == "" {
		return nil, fmt.Errorf("Zone not found: %#v", domain)
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

	if len(zone.Ns) > 0 {
		if err := d.Set("ns", zone.Ns); err != nil {
			return err
		}
	}

	if zone.Ztype == "slave" && zone.Master != "" {
		if err := d.Set("master", zone.Master); err != nil {
			return err
		}
	}

	return nil
}

func toApiZone(d *schema.ResourceData) cloudns.Zone {
	domain := d.Get("domain").(string)
	zoneType := d.Get("type").(string)
	master := d.Get("master").(string)

	return cloudns.Zone{
		Domain: domain,
		Ztype:  zoneType,
		Master: master,
	}
}

func isNotFoundErr(err error) bool {
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no zones returned")
}
