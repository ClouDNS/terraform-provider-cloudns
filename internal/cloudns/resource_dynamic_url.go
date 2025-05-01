package cloudns

import (
	"context"
	"fmt"

	"github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDynamicUrl() *schema.Resource {
	return &schema.Resource{
		Description: "A dynamic URL for an A or AAAA record managed by ClouDNS.",

		CreateContext: resourceDynamicUrlGetOrCreate,
		ReadContext:   resourceDynamicUrlGetOrCreate,
		DeleteContext: resourceDynamicUrlDelete,

		Schema: map[string]*schema.Schema{
			"domain": {
				Description: "The name of the DNS zone.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"recordid": {
				Description: "The ID of the record for which the dynamic URL should be enabled / the same as the id param",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"url": {
				Description: "The URL to which the dynamic DNS request will be sent.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceDynamicUrlGetOrCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	dynUrl := toApiDynamicUrl(d)

	tflog.Debug(ctx, fmt.Sprintf("Dynamic URL object before read: %+v", dynUrl))

	config.rateLimiter.Take()

	readDynUrl, err := dynUrl.ReadOrCreate(&config.apiAccess)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Dynamic URL object after read: %+v", readDynUrl))
	d.SetId(readDynUrl.RecordId)

	err = updateDynamicUrlState(d, &readDynUrl)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Dynamic URL state set successfully")

	return nil
}

func resourceDynamicUrlDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	dynUrl := toApiDynamicUrl(d)

	tflog.Debug(ctx, fmt.Sprintf("DELETE dynamic URL #%s for Domain: %s", dynUrl.RecordId, dynUrl.Domain))

	config.rateLimiter.Take()

	_, err := dynUrl.Delete(&config.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

func toApiDynamicUrl(d *schema.ResourceData) cloudns.DynamicUrl {
	domain := d.Get("domain").(string)
	recordId := d.Get("recordid").(string)

	return cloudns.DynamicUrl{
		Domain:   domain,
		RecordId: recordId,
	}
}

func updateDynamicUrlState(d *schema.ResourceData, dynUrl *cloudns.DynamicUrlResponse) error {
	err := d.Set("domain", dynUrl.Domain)
	if err != nil {
		return err
	}

	err = d.Set("recordid", dynUrl.RecordId)
	if err != nil {
		return err
	}

	err = d.Set("url", dynUrl.Url)
	if err != nil {
		return err
	}

	return nil
}
