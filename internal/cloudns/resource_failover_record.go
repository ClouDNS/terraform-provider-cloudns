package cloudns

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDnsFailover() *schema.Resource {
	return &schema.Resource{
		Description: "A DNS failover record managed by ClouDNS.",

		CreateContext: resourceDnsFailoverCreate,
		ReadContext:   resourceDnsFailoverRead,
		UpdateContext: resourceDnsFailoverUpdate,
		DeleteContext: resourceDnsFailoverDelete,

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
			"recordid": {
				Description: "The ID of the record for which the failover to be activated / the same as the id param",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"checktype": {
				Description: "Monitoring check types for this Failover.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
			},
			"downeventhandler": {
				Description: "Event handler if Main IP is down.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"upeventhandler": {
				Description: "Event handler if Main IP is up.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"mainip": {
				Description: "Main IP address which will be monitored.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    false,
			},
			"backupip1": {
				Description: "First Backup IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"backupip2": {
				Description: "Second Backup IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"backupip3": {
				Description: "Third Backup IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"backupip4": {
				Description: "Fourth Backup IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"backupip5": {
				Description: "Fifth Backup IP address.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"monitoringregion": {
				Description: "Monitoring region or country.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
			},
			"checkperiod": {
				Description: "Time-frame between each monitoring check.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"notificationmail": {
				Description: "Email notifications settings.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"host": {
				Description: "A host to query.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"port": {
				Description: "A port to query.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"path": {
				Description: "Path for the URL",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"content": {
				Description: "Parameter required for Custom HTTP and Custom HTTPS check types",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"querytype": {
				Description: "Parameter required for DNS check type. It must contain the record type (e.g. A).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"queryresponse": {
				Description: "Parameter required for DNS check type. You must fill in the response of the DNS server for this specific record.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"latencylimit": {
				Description: "Only for Ping monitoring checks. If the latency of the check is above the limit, the check will be marked as DOWN.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"timeout": {
				Description: "Only for Ping monitoring checks. Seconds to wait for a response. Must be between 1 and 5. Default value is 2.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"checkregion": {
				Description: "The region from which the check is monitored(it is only received from API)",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"httprequesttype": {
				Description: "Only for HTTP/S checks. The request type will be used for the check. The default value is GET. Possible values:",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
}

func resourceDnsFailoverCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clientConfig := meta.(ClientConfig)
	failoverToCreate := toApiFailover(d)
	tflog.Debug(ctx, fmt.Sprintf("Failover data to create: %+v", failoverToCreate))

	resp, err := failoverToCreate.Create(&clientConfig.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Response from Create API: %+v", resp))

	return resourceDnsFailoverRead(ctx, d, meta)
}

func resourceDnsFailoverUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	failover := toApiFailover(d)

	tflog.Debug(ctx, fmt.Sprintf("Update Failover for Domain: %s", failover.Domain))

	config.rateLimiter.Take()
	_, err := failover.Update(&config.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDnsFailoverRead(ctx, d, meta)
}

func resourceDnsFailoverRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	failover := toApiFailover(d)

	tflog.Debug(ctx, fmt.Sprintf("Failover object before read: %+v", failover))

	config.rateLimiter.Take()

	readFailover, err := failover.Read(&config.apiAccess)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Failover object after read: %+v", readFailover))
	d.SetId(readFailover.RecordId)

	err = updateFailoverState(d, &readFailover)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Failover state set successfully"))

	return nil
}

func resourceDnsFailoverDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	failover := toApiFailover(d)

	tflog.Debug(ctx, fmt.Sprintf("DELETE Failover #%s for Domain: %s", failover.RecordId, failover.Domain))

	config.rateLimiter.Take()

	_, err := failover.Delete(&config.apiAccess)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

func toApiFailover(d *schema.ResourceData) cloudns.Failover {
	domain := d.Get("domain").(string)
	recordId := d.Get("recordid").(string)
	failoverType := d.Get("checktype").(string)
	downEventHandler := d.Get("downeventhandler").(string)
	upEventHandler := d.Get("upeventhandler").(string)
	mainIp := d.Get("mainip").(string)
	backupIp1 := d.Get("backupip1").(string)
	backupIp2 := d.Get("backupip2").(string)
	backupIp3 := d.Get("backupip3").(string)
	backupIp4 := d.Get("backupip4").(string)
	backupIp5 := d.Get("backupip5").(string)
	monitoringRegion := d.Get("monitoringregion").(string)
	checkPeriod := d.Get("checkperiod").(string)
	notificationMail := d.Get("notificationmail").(string)
	host := d.Get("host").(string)
	portStr := d.Get("port").(string)
	path := d.Get("path").(string)
	content := d.Get("content").(string)
	querytype := d.Get("querytype").(string)
	queryresponse := d.Get("queryresponse").(string)
	checkRegion := d.Get("checkregion").(string)
	latencyLimit, err := d.Get("latencyLimit").(string)
	if err {
		latencyLimit = ""
	}
	timeout := d.Get("timeout").(string)
	httprequesttype := d.Get("httprequesttype").(string)

	var port cloudns.CustomPort
	if portInt, err := strconv.Atoi(portStr); err == nil {
		port = cloudns.CustomPort(portInt)
	} else {
		if failoverType == "4" || failoverType == "6" {
			port = 80
		}

		if failoverType == "5" || failoverType == "7" {
			port = 443
		}
	}

	var checkSettings = cloudns.CheckSettings{
		Host:            host,
		Port:            port,
		Path:            path,
		Content:         content,
		QueryType:       querytype,
		QueryResponse:   queryresponse,
		LatencyLimit:    latencyLimit,
		Timeout:         timeout,
		HttpRequestType: httprequesttype,
	}

	return cloudns.Failover{
		Domain:           domain,
		RecordId:         recordId,
		FailoverType:     failoverType,
		DownEventHandler: downEventHandler,
		UpEventHandler:   upEventHandler,
		MainIP:           mainIp,
		BackupIp1:        backupIp1,
		BackupIp2:        backupIp2,
		BackupIp3:        backupIp3,
		BackupIp4:        backupIp4,
		BackupIp5:        backupIp5,
		MonitoringRegion: monitoringRegion,
		CheckSettings:    checkSettings,
		CheckPeriod:      checkPeriod,
		NotificationMail: notificationMail,
		CheckRegion:      checkRegion,
	}
}

func updateFailoverState(d *schema.ResourceData, failover *cloudns.Failover) error {
	err := d.Set("domain", failover.Domain)
	if err != nil {
		return err
	}

	err = d.Set("recordid", failover.RecordId)
	if err != nil {
		return err
	}

	err = d.Set("checktype", failover.FailoverType)
	if err != nil {
		return err
	}

	err = d.Set("downeventhandler", failover.DownEventHandler)
	if err != nil {
		return err
	}

	err = d.Set("upeventhandler", failover.UpEventHandler)
	if err != nil {
		return err
	}

	err = d.Set("mainip", failover.MainIP)
	if err != nil {
		return err
	}

	err = d.Set("backupip1", failover.BackupIp1)
	if err != nil {
		return err
	}

	err = d.Set("backupip2", failover.BackupIp2)
	if err != nil {
		return err
	}

	err = d.Set("backupip3", failover.BackupIp3)
	if err != nil {
		return err
	}

	err = d.Set("backupip4", failover.BackupIp4)
	if err != nil {
		return err
	}

	err = d.Set("backupip5", failover.BackupIp5)
	if err != nil {
		return err
	}

	err = d.Set("monitoringregion", failover.MonitoringRegion)
	if err != nil {
		return err
	}

	err = d.Set("checkperiod", failover.CheckPeriod)
	if err != nil {
		return err
	}

	err = d.Set("notificationmail", failover.NotificationMail)
	if err != nil {
		return err
	}

	err = d.Set("host", failover.CheckSettings.Host)
	if err != nil {
		return err
	}

	if err := d.Set("port", strconv.Itoa(int(failover.CheckSettings.Port))); err != nil {
		return err
	}

	err = d.Set("path", failover.CheckSettings.Path)
	if err != nil {
		return err
	}

	err = d.Set("content", failover.CheckSettings.Content)
	if err != nil {
		return err
	}

	err = d.Set("querytype", failover.CheckSettings.QueryType)
	if err != nil {
		return err
	}

	err = d.Set("queryresponse", failover.CheckSettings.QueryResponse)
	if err != nil {
		return err
	}

	err = d.Set("latencylimit", failover.CheckSettings.LatencyLimit)
	if err != nil {
		return err
	}

	err = d.Set("timeout", failover.CheckSettings.Timeout)
	if err != nil {
		return err
	}

	err = d.Set("checkregion", failover.CheckRegion)
	if err != nil {
		return err
	}

	err = d.Set("httprequesttype", failover.CheckSettings.HttpRequestType)
	if err != nil {
		return err
	}

	return nil
}
