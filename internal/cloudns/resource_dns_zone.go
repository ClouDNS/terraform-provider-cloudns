package cloudns

import (
	"context"
	"fmt"

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
		return diag.FromErr(err)
	}

	if zoneRead.Domain != "" {
		d.Set("domain", zoneRead.Domain)
		d.Set("type", zoneRead.Ztype)
	}

	d.Set("domain", "")
	d.Set("type", "")
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
