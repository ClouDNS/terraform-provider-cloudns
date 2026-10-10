package cloudns

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	api "github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceDnsDNSSEC() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages DNSSEC signing for an existing ClouDNS zone. Publish DS records at the registrar separately, and remove them before disabling or destroying this resource.",
		CreateContext: resourceDnsDNSSECApply, UpdateContext: resourceDnsDNSSECApply, ReadContext: resourceDnsDNSSECRead, DeleteContext: resourceDnsDNSSECDelete,
		Timeouts: &schema.ResourceTimeout{Create: schema.DefaultTimeout(5 * time.Minute), Update: schema.DefaultTimeout(5 * time.Minute)},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			if d.NewValueKnown("enabled") && d.NewValueKnown("opt_out") && !d.Get("enabled").(bool) && d.Get("opt_out").(bool) {
				return fmt.Errorf("opt_out requires enabled = true")
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"domain":  {Type: schema.TypeString, Required: true, ForceNew: true, ValidateFunc: validation.StringIsNotWhiteSpace, Description: "Existing DNS zone name. Can reference cloudns_dns_zone.example.id."},
			"enabled": {Type: schema.TypeBool, Optional: true, Default: true, Description: "Whether DNSSEC signing is enabled. Defaults to true."},
			"opt_out": {Type: schema.TypeBool, Optional: true, Default: false, Description: "Whether NSEC3 opt-out is enabled. Requires enabled = true. Defaults to false."},
			"ds":      {Type: schema.TypeSet, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "DS records in zone-file format, to publish at the registrar."},
			"dnskey":  {Type: schema.TypeSet, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "DNSKEY records in zone-file format."},
			"ds_records": {Type: schema.TypeList, Computed: true, Description: "Structured DS parameters for registrar integrations.", Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"key_tag": {Type: schema.TypeString, Computed: true}, "algorithm": {Type: schema.TypeString, Computed: true}, "algorithm_name": {Type: schema.TypeString, Computed: true}, "digest_type": {Type: schema.TypeString, Computed: true}, "digest_type_name": {Type: schema.TypeString, Computed: true}, "digest": {Type: schema.TypeString, Computed: true},
			}}},
		},
	}
}

func resourceDnsDNSSECApply(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(ClientConfig)
	z := api.Zone{Domain: d.Get("domain").(string)}
	timeout := d.Timeout(schema.TimeoutUpdate)
	if d.Id() == "" {
		timeout = d.Timeout(schema.TimeoutCreate)
	}
	enabled, optout := d.Get("enabled").(bool), d.Get("opt_out").(bool)
	if !enabled && optout {
		return diag.Errorf("opt_out requires enabled = true")
	}
	c.rateLimiter.Take()
	current, err := z.DNSSEC(ctx, &c.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}
	if enabled {
		if !current.Enabled {
			c.rateLimiter.Take()
			available, err := z.DNSSECAvailable(ctx, &c.apiAccess)
			if err != nil {
				return diag.FromErr(err)
			}
			if !available {
				return diag.Errorf("DNSSEC is not available for zone %s", z.Domain)
			}
			c.rateLimiter.Take()
			if err := z.ActivateDNSSEC(ctx, &c.apiAccess); err != nil {
				return diag.FromErr(err)
			}
		}
		// Preserve ownership if key generation or setting opt-out fails after activation.
		d.SetId(z.Domain)
		err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
			c.rateLimiter.Take()
			details, err := z.DNSSEC(ctx, &c.apiAccess)
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if !details.Enabled || len(details.DS) == 0 || len(details.DNSKEY) == 0 {
				return retry.RetryableError(fmt.Errorf("waiting for DNSSEC keys for %s", z.Domain))
			}
			current = details
			return nil
		})
		if err != nil {
			return diag.FromErr(err)
		}
		if current.OptOut != optout {
			c.rateLimiter.Take()
			if err := z.SetDNSSECOptOut(ctx, &c.apiAccess, optout); err != nil {
				return diag.FromErr(err)
			}
		}
	} else if current.Enabled {
		c.rateLimiter.Take()
		if err := z.DeactivateDNSSEC(ctx, &c.apiAccess); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(z.Domain)
	// Wait for read-back to match requested flags before returning Terraform state.
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		c.rateLimiter.Take()
		details, err := z.DNSSEC(ctx, &c.apiAccess)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		if details.Enabled != enabled || details.OptOut != optout {
			return retry.RetryableError(fmt.Errorf("waiting for DNSSEC settings for %s", z.Domain))
		}
		current = details
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return dnssecSetState(d, z.Domain, current)
}

func resourceDnsDNSSECRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(ClientConfig)
	z := api.Zone{Domain: d.Id()}
	c.rateLimiter.Take()
	details, err := z.DNSSEC(ctx, &c.apiAccess)
	if err != nil {
		if err.Error() == "DNSSEC: Missing domain-name" {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return dnssecSetState(d, z.Domain, details)
}

func dnssecSetState(d *schema.ResourceData, domain string, details api.DNSSECDetails) diag.Diagnostics {
	records := make([]interface{}, 0, len(details.DSRecords))
	dsRecords := append([]api.DNSSECDSRecord(nil), details.DSRecords...)
	sort.Slice(dsRecords, func(i, j int) bool {
		a, b := dsRecords[i], dsRecords[j]
		return strings.Join([]string{a.KeyTag, a.Algorithm, a.DigestType, a.Digest}, ":") < strings.Join([]string{b.KeyTag, b.Algorithm, b.DigestType, b.Digest}, ":")
	})
	for _, r := range dsRecords {
		records = append(records, map[string]interface{}{"key_tag": r.KeyTag, "algorithm": r.Algorithm, "algorithm_name": r.AlgorithmName, "digest_type": r.DigestType, "digest_type_name": r.DigestTypeName, "digest": r.Digest})
	}
	for k, v := range map[string]interface{}{"domain": domain, "enabled": details.Enabled, "opt_out": details.OptOut, "ds": details.DS, "dnskey": details.DNSKEY, "ds_records": records} {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func resourceDnsDNSSECDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(ClientConfig)
	z := api.Zone{Domain: d.Id()}
	c.rateLimiter.Take()
	details, err := z.DNSSEC(ctx, &c.apiAccess)
	if err != nil {
		if err.Error() == "DNSSEC: Missing domain-name" {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if details.Enabled {
		c.rateLimiter.Take()
		if err := z.DeactivateDNSSEC(ctx, &c.apiAccess); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId("")
	return nil
}
