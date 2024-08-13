package cloudns

import (
	"context"
	"fmt"
	"strings"

	"github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceDnsRecord() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "A simple DNS record.",

		CreateContext: resourceDnsRecordCreate,
		ReadContext:   resourceDnsRecordRead,
		UpdateContext: resourceDnsRecordUpdate,
		DeleteContext: resourceDnsRecordDelete,
		CustomizeDiff: resourceDnsRecordValidate,

		Importer: &schema.ResourceImporter{
			StateContext: resourceDnsRecordImport,
		},

		// Naming **does not** follow the scheme used by ClouDNS, due to how comically misleading and unclear it is
		// see: https://www.cloudns.net/wiki/article/58/ for the relevant "vanilla" schema on ClouDNS side
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the record",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
			},
			"zone": {
				Description: "The zone on which to add the record",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"ttl": {
				Description: "The TTL to assign to the record",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    false,
			},
			"type": {
				Description: "The type of record",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
			},
			"value": {
				Description: "Value of the record",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"priority": {
				Description:      "Priority for MX record",
				Type:             schema.TypeInt,
				Optional:         true,
				ForceNew:         false,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 65535)),
			},
			"weight": {
				Description: "Weight for SRV record",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"port": {
				Description: "Port for SRV record",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"frame": {
				Description: "Frame for WR record - 0 or 1 to disable or enable frame",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"frametitle": {
				Description: "Title if frame is enabled in Web redirects",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"framekeywords": {
				Description: "Keywords if frame is enabled in Web redirects",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"framedescription": {
				Description: "Description if frame is enabled in Web redirects",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"mobilemeta": {
				Description: "Mobile responsive meta tags if Web redirects with frame is enabled. Default value - 0.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"savepath": {
				Description: "0 or 1 for Web redirects",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"redirecttype": {
				Description: "301 or 302 for Web redirects if frame is disabled",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"mail": {
				Description: "E-mail address for RP records",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"txt": {
				Description: "Domain name for TXT record used in RP records",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"algorithm": {
				Description: "Algorithm used to create the SSHFP fingerprint. Required for SSHFP records only.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"fptype": {
				Description: "Type of the SSHFP algorithm. Required for SSHFP records only.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"status": {
				Description: "Set to 1 to create the record active or to 0 to create it inactive. If omitted the record will be created active.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"geodnslocation": {
				Description: "ID of a GeoDNS location for A, AAAA, CNAME, NAPTR or SRV record. The GeoDNS locations can be obtained with List GeoDNS locations",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"geodnscode": {
				Description: "Code of a GeoDNS location for A, AAAA, CNAME, NAPTR or SRV record.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"caaflag": {
				Description: "0 - Non critical or 128 - Critical",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"caatype": {
				Description: "Type of CAA record. The available flags are issue, issuewild, iodef.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"caavalue": {
				Description: "Value of the CAA record.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"tlsausage": {
				Description: "Shows the provided association that will be used.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"tlsaselector": {
				Description: "Specifies which part of the TLS certificate presented by the server will be matched against the association data",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"tlsamatchingtype": {
				Description: "Specifies how the certificate association is presented.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"smimeausage": {
				Description: "Shows the provided association that will be used.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"smimeaselector": {
				Description: "Specifies which part of the TLS certificate presented by the server will be matched against the association data",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"smimeamatchingtype": {
				Description: "Specifies how the certificate association is presented.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"keytag": {
				Description: "A numeric value used for identifying the referenced DS record.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"digesttype": {
				Description: "The cryptographic hash algorithm used to create the Digest value.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"order": {
				Description: "Specifies the order in which multiple NAPTR records must be processed (low to high).",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"pref": {
				Description: "Specifies the order (low to high) in which NAPTR records with equal Order values should be processed.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"flag": {
				Description: "Controls aspects of the rewriting and interpretation of the fields in the record.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"params": {
				Description: "Specifies the service parameters applicable to this delegation path.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"regexp": {
				Description: "Contains a substitution expression that is applied to the original string, held by the client in order to construct the next domain name to lookup.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"replace": {
				Description: "Specifies the next domain name (fully qualified) to query for depending on the potential values found in the flags field.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"certtype": {
				Description: "Type of the Certificate/CRL.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"certkeytag": {
				Description: "A numeric value (0-65535), used to efficiently pick a CERT record.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"certalgorithm": {
				Description: "Identifies the algorithm used to produce a legitimate signature.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    false,
			},
			"latdeg": {
				Description: "A numeric value(0-90), sets the latitude degrees.",
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    false,
			},
			"latmin": {
				Description: "A numeric value(0-59), sets the latitude minutes.",
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    false,
			},
			"latsec": {
				Description: "A numeric value(0-59), sets the latitude seconds.",
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    false,
			},
			"latdir": {
				Description: "Sets the latitude direction. Possible values: N - North, S - South",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"longdeg": {
				Description: "A numeric value(0-180), sets the longitude degrees.",
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    false,
			},
			"longmin": {
				Description: "A numeric value(0-59), sets the longitude minutes.",
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    false,
			},
			"longsec": {
				Description: "A numeric value(0-59), sets the longitude seconds.",
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    false,
			},
			"longdir": {
				Description: "Sets the longitude direction. Possible values: W - West, E - East",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"altitude": {
				Description: "A numeric value(-100000.00 - 42849672.95), sets the altitude in meters.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"size": {
				Description: "A numeric value(0 - 90000000.00), sets the size in meters.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"hprecision": {
				Description: "A numeric value(0 - 90000000.00), sets the horizontal precision in meters.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"vprecision": {
				Description: "A numeric value(0 - 90000000.00), sets the vertical precision in meters.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"cpu": {
				Description: "The CPU of the server.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"os": {
				Description: "The operating system of the server.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
		},
	}
}

func resourceDnsRecordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientConfig := meta.(ClientConfig)
	recordToCreate := toApiRecord(d)

	tflog.Debug(ctx, fmt.Sprintf("CREATE %s.%s %d in %s %s", recordToCreate.Host, recordToCreate.Domain, recordToCreate.TTL, recordToCreate.Rtype, recordToCreate.Record))

	clientConfig.rateLimiter.Take()
	recordCreated, err := recordToCreate.Create(&clientConfig.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(recordCreated.ID)

	return resourceDnsRecordRead(ctx, d, meta)
}

func resourceDnsRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Id() == "" {
		d.SetId("")
		return nil
	}

	config := meta.(ClientConfig)
	lookup := toApiRecord(d)

	tflog.Debug(ctx, fmt.Sprintf("READ Record#%s (%s.%s %d in %s %s)", lookup.ID, lookup.Host, lookup.Domain, lookup.TTL, lookup.Rtype, lookup.Record))

	config.rateLimiter.Take()
	zoneRead, err := cloudns.Zone{Domain: lookup.Domain}.List(&config.apiAccess)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	for _, zoneRecord := range zoneRead {
		wantedId := d.Id()
		actualId := zoneRecord.ID
		if wantedId == actualId {
			err = updateState(d, &zoneRecord)
			if err != nil {
				return diag.FromErr(err)
			}
			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceDnsRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	record := toApiRecord(d)

	tflog.Debug(ctx, fmt.Sprintf("UPDATE %s.%s %d in %s %s", record.Host, record.Domain, record.TTL, record.Rtype, record.Record))

	config.rateLimiter.Take()
	updated, err := record.Update(&config.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(updated.ID)

	return resourceDnsRecordRead(ctx, d, meta)
}

func resourceDnsRecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	record := toApiRecord(d)

	tflog.Debug(ctx, fmt.Sprintf("DELETE %s.%s %d in %s %s", record.Host, record.Domain, record.TTL, record.Rtype, record.Record))

	config.rateLimiter.Take()
	_, err := record.Destroy(&config.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDnsRecordRead(ctx, d, meta)
}

func resourceDnsRecordValidate(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	rtype := d.Get("type").(string)
	_, isPriorityProvided := d.GetOkExists("priority")
	if rtype == "MX" && !isPriorityProvided {
		return fmt.Errorf("Priority is required for MX record")
	}
	return nil
}

func resourceDnsRecordImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(ClientConfig)

	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Bad ID format: %#v. Expected: \"zone/id\"", d.Id())
	}
	zone := parts[0]
	wantedId := parts[1]

	config.rateLimiter.Take()
	zoneRead, err := cloudns.Zone{Domain: zone}.List(&config.apiAccess)
	if err != nil {
		return nil, err
	}

	idx := -1
	for i, zoneRecord := range zoneRead {
		if wantedId == zoneRecord.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("Record not found: %#v", wantedId)
	}

	zoneRecord := zoneRead[idx]
	err = updateState(d, &zoneRecord)
	if err != nil {
		return nil, err
	}
	d.SetId(wantedId)

	tflog.Debug(ctx, fmt.Sprintf("IMPORT %s.%s %d in %s %s", zoneRecord.Host, zoneRecord.Domain, zoneRecord.TTL, zoneRecord.Rtype, zoneRecord.Record))

	return []*schema.ResourceData{d}, nil
}

func updateState(d *schema.ResourceData, zoneRecord *cloudns.Record) error {
	err := d.Set("name", zoneRecord.Host)
	if err != nil {
		return err
	}

	err = d.Set("zone", zoneRecord.Domain)
	if err != nil {
		return err
	}

	err = d.Set("type", zoneRecord.Rtype)
	if err != nil {
		return err
	}

	err = d.Set("value", zoneRecord.Record)
	if err != nil {
		return err
	}

	err = d.Set("ttl", zoneRecord.TTL)
	if err != nil {
		return err
	}

	if zoneRecord.Rtype == "MX" {
		err = d.Set("priority", zoneRecord.Priority)
		if err != nil {
			return err
		}
	}

	return nil
}

func toApiRecord(d *schema.ResourceData) cloudns.Record {
	id := d.Id()
	name := d.Get("name").(string)
	zone := d.Get("zone").(string)
	rtype := d.Get("type").(string)
	value := d.Get("value").(string)
	ttl := d.Get("ttl").(int)
	priority := 0
	frame := ""
	frameTitle := ""
	frameKeywords := ""
	frameDescription := ""
	mobileMeta := 0
	savePath := 0
	redirectType := 301
	weight := 0
	port := 0
	mail := ""
	txt := ""
	algorithm := 0
	fptype := 0
	caaflag := ""
	caatype := ""
	caavalue := ""
	tlsausage := ""
	tlsaselector := ""
	tlsamatchingtype := ""
	smimeausage := ""
	smimeaselector := ""
	smimeamatchingtype := ""
	keytag := 0
	digesttype := 0
	order := ""
	pref := ""
	flag := ""
	params := ""
	regexp := ""
	replace := ""
	certtype := 0
	certkeytag := 0
	certalgorithm := 0
	latdeg := 0.0
	latmin := 0.0
	latdir := ""
	longdeg := 0.0
	longmin := 0.0
	longsec := 0.0
	longdir := ""
	altitude := ""
	size := ""
	hprecision := ""
	vprecision := ""
	cpu := ""
	os := ""

	if rtype == "MX" {
		priority = d.Get("priority").(int)
	} else if rtype == "WR" {
		frame = d.Get("frame").(string)
		frameTitle = d.Get("frametitle").(string)
		frameKeywords = d.Get("framekeywords").(string)
		frameDescription = d.Get("framedescription").(string)
		mobileMeta = d.Get("mobilemeta").(int)
		savePath = d.Get("savepath").(int)
		redirectType = d.Get("redirecttype").(int)
	} else if rtype == "SRV" {
		weight = d.Get("weight").(int)
		priority = d.Get("priority").(int)
		port = d.Get("port").(int)
	} else if rtype == "RP" {
		mail = d.Get("mail").(string)
		txt = d.Get("txt").(string)
	} else if rtype == "SSHFP" {
		algorithm = d.Get("algorithm").(int)
		fptype = d.Get("fptype").(int)
	} else if rtype == "NAPTR" {
		flag = d.Get("flag").(string)
		order = d.Get("order").(string)
		pref = d.Get("pref").(string)
		params = d.Get("params").(string)
		regexp = d.Get("regexp").(string)
		replace = d.Get("replace").(string)
	} else if rtype == "CAA" {
		caaflag = d.Get("caaflag").(string)
		caatype = d.Get("caatype").(string)
		caavalue = d.Get("caavalue").(string)
	} else if rtype == "TLSA" {
		tlsausage = d.Get("tlsausage").(string)
		tlsaselector = d.Get("tlsaselector").(string)
		tlsamatchingtype = d.Get("tlsamatchingtype").(string)
	} else if rtype == "DS" {
		keytag = d.Get("keytag").(int)
		algorithm = d.Get("algorithm").(int)
		digesttype = d.Get("digesttype").(int)
	} else if rtype == "CERT" {
		certtype = d.Get("keytag").(int)
		certkeytag = d.Get("certkeytag").(int)
		certalgorithm = d.Get("certalgorithm").(int)
	} else if rtype == "HINFO" {
		cpu = d.Get("cpu").(string)
		os = d.Get("os").(string)
	} else if rtype == "LOC" {
		latdeg = d.Get("latdeg").(float64)
		latmin = d.Get("latmin").(float64)
		latdir = d.Get("latdir").(string)
		longdeg = d.Get("longdeg").(float64)
		longmin = d.Get("longmin").(float64)
		longsec = d.Get("longsec").(float64)
		longdir = d.Get("longdir").(string)
		altitude = d.Get("altitude").(string)
		size = d.Get("size").(string)
		hprecision = d.Get("hprecision").(string)
		vprecision = d.Get("vprecision").(string)
	} else if rtype == "SMIMEA" {
		smimeausage = d.Get("smimeausage").(string)
		smimeaselector = d.Get("smimeaselector").(string)
		smimeamatchingtype = d.Get("smimeamatchingtype").(string)
	}

	var geodnslocation string
	var geodnscode string
	if v, ok := d.GetOk("geodnslocation"); ok {
		geodnslocation = v.(string)
	}
	if v, ok := d.GetOk("geodnscode"); ok {
		geodnscode = v.(string)
	}

	return cloudns.Record{
		ID:                 id,
		Host:               name,
		Domain:             zone,
		Rtype:              rtype,
		Record:             value,
		TTL:                ttl,
		Priority:           priority,
		Frame:              frame,
		FrameTitle:         frameTitle,
		FrameKeywords:      frameKeywords,
		FrameDescription:   frameDescription,
		MobileMeta:         mobileMeta,
		SavePath:           savePath,
		RedirectType:       redirectType,
		Weight:             weight,
		Port:               port,
		Mail:               mail,
		Txt:                txt,
		Algorithm:          algorithm,
		Fptype:             fptype,
		Flag:               flag,
		Order:              order,
		Pref:               pref,
		Params:             params,
		Regexp:             regexp,
		Replace:            replace,
		CaaFlag:            caaflag,
		CaaType:            caatype,
		CaaValue:           caavalue,
		TlsaUsage:          tlsausage,
		TlsaSelector:       tlsaselector,
		TlsaMatchingType:   tlsamatchingtype,
		KeyTag:             keytag,
		DigestType:         digesttype,
		CertType:           certtype,
		CertKeyTag:         certkeytag,
		CertAlgorithm:      certalgorithm,
		CPU:                cpu,
		OS:                 os,
		LatDeg:             latdeg,
		LatMin:             latmin,
		LatDir:             latdir,
		LongDeg:            longdeg,
		LongMin:            longmin,
		LongSec:            longsec,
		LongDir:            longdir,
		Altitude:           altitude,
		Size:               size,
		HPrecision:         hprecision,
		VPrecision:         vprecision,
		SmimeaUsage:        smimeausage,
		SmimeaSelector:     smimeaselector,
		SmimeaMatchingType: smimeamatchingtype,
		GeodnsLocation:     geodnslocation,
		GeodnsCode:         geodnscode,
	}
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "Missing domain-name")
}
