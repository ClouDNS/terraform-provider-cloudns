package cloudns

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/ClouDNS/cloudns-go"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var monitoringCheckTypes = map[string]int{
	"ping": 17, "web": 18, "tcp": 8, "udp": 9, "dns": 10, "heartbeat": 12,
	"tls": 13, "smtp": 15, "streaming": 16, "imap": 19, "keyword": 20,
}

type monitoringField struct {
	name        string
	description string
	kind        schema.ValueType
	types       []string
	sensitive   bool
	validate    schema.SchemaValidateFunc
}

var monitoringFields = []monitoringField{
	{name: "ip", description: "Target IP address or hostname. Required except for Heartbeat.", kind: schema.TypeString},
	{name: "ip_type", description: "Address family when targeting a hostname. 1 for both, 2 for IPv4, 3 for IPv6.", kind: schema.TypeInt, validate: validation.IntBetween(1, 3)},
	{name: "check_period", description: "Interval between checks in seconds. Availability depends on your plan.", kind: schema.TypeInt, validate: validation.IntInSlice([]int{60, 120, 300, 600, 900, 1200, 1800, 3600, 10800, 21600, 43200, 86400})},
	{name: "monitoring_region", description: "Region, country or node ID from which to monitor.", kind: schema.TypeString},
	{name: "status_change_checks", description: "Number of checks required to confirm a status change.", kind: schema.TypeInt, validate: validation.IntAtLeast(1)},
	{name: "host", description: "Hostname for the protocol request.", kind: schema.TypeString, types: []string{"web", "dns", "tls", "keyword"}},
	{name: "port", description: "Port to check.", kind: schema.TypeInt, types: []string{"web", "tcp", "udp", "tls", "smtp", "imap", "keyword"}, validate: validation.IntBetween(1, 65535)},
	{name: "ping_threshold", description: "Packet loss threshold in percent.", kind: schema.TypeInt, types: []string{"ping"}, validate: validation.IntInSlice([]int{15, 25, 50, 100})},
	{name: "latency_limit", description: "Maximum Ping latency.", kind: schema.TypeFloat, types: []string{"ping"}},
	{name: "timeout", description: "Seconds to wait for a response.", kind: schema.TypeInt, types: []string{"ping", "web", "imap"}, validate: validation.IntBetween(1, 10)},
	{name: "open_port", description: "Expected port state. 1 for open, 0 for closed.", kind: schema.TypeInt, types: []string{"tcp", "udp"}, validate: validation.IntBetween(0, 1)},
	{name: "http_protocol", description: "HTTP protocol for the request.", kind: schema.TypeString, types: []string{"web", "keyword"}, validate: validation.StringInSlice([]string{"http", "https"}, false)},
	{name: "http_request_type", description: "HTTP method for the request.", kind: schema.TypeString, types: []string{"web", "keyword"}, validate: validation.StringInSlice([]string{"GET", "HEAD", "POST", "PUT", "DELETE"}, false)},
	{name: "http_status_code", description: "Expected HTTP status code.", kind: schema.TypeInt, types: []string{"web", "keyword"}, validate: validation.IntBetween(100, 599)},
	{name: "path", description: "HTTP request path.", kind: schema.TypeString, types: []string{"web", "keyword"}},
	{name: "content", description: "Expected HTTP response content.", kind: schema.TypeString, types: []string{"web"}},
	{name: "content_match", description: "How to match the expected response content.", kind: schema.TypeString, types: []string{"web"}, validate: validation.StringInSlice([]string{"exact", "keyword", "regex"}, false)},
	{name: "custom_header", description: "Optional HTTP request header name.", kind: schema.TypeString, types: []string{"web", "keyword"}},
	{name: "custom_header_value", description: "Optional HTTP request header value, which may contain a credential.", kind: schema.TypeString, types: []string{"web", "keyword"}, sensitive: true},
	{name: "query_type", description: "DNS record type to query.", kind: schema.TypeString, types: []string{"dns"}},
	{name: "query_response", description: "Expected DNS query response.", kind: schema.TypeString, types: []string{"dns"}},
	{name: "notification_period", description: "TLS certificate expiry notification schedule.", kind: schema.TypeInt, types: []string{"tls"}, validate: validation.IntInSlice([]int{1, 7, 15, 30})},
	{name: "cacert", description: "Custom root CA certificate for TLS checks.", kind: schema.TypeString, types: []string{"tls"}},
	{name: "connection_security", description: "Mail transport security. 1 for none, 2 for STARTTLS, 3 for TLS.", kind: schema.TypeInt, types: []string{"smtp", "imap"}, validate: validation.IntBetween(1, 3)},
	{name: "url", description: "Streaming service URL.", kind: schema.TypeString, types: []string{"streaming"}, sensitive: true},
	{name: "audio_threshold", description: "Minimum streaming volume, in dB.", kind: schema.TypeFloat, types: []string{"streaming"}, validate: validation.FloatBetween(-91, 0)},
	{name: "keyword", description: "Keyword expected in the HTTP response.", kind: schema.TypeString, types: []string{"keyword"}},
}

func resourceMonitoringCheck() *schema.Resource {
	fields := map[string]*schema.Schema{
		"name": {Type: schema.TypeString, Required: true, Description: "Monitoring check name.",
			ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9 .-]{1,32}$`), "use 1 to 32 letters, digits, spaces, dots or hyphens")},
		"check_type": {Type: schema.TypeString, Required: true, Description: "Monitoring check type.",
			ValidateFunc: validation.StringInSlice([]string{"ping", "web", "tcp", "udp", "dns", "heartbeat", "tls", "smtp", "streaming", "imap", "keyword"}, false)},
		"enabled":       {Type: schema.TypeBool, Optional: true, Computed: true, Description: "Whether the check is active. New checks default to active."},
		"heartbeat_url": {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "Heartbeat callback URL. The URL is sensitive and is stored in state."},
		"notification_email": {
			Type: schema.TypeString, Optional: true, ForceNew: true,
			Description: "Initial email notification recipient. Required for creation with an API sub-user and applied only at creation.",
			ValidateFunc: func(value interface{}, _ string) ([]string, []error) {
				valueString := value.(string)
				address, err := mail.ParseAddress(valueString)
				if err != nil || address.Address != valueString {
					return nil, []error{errors.New("notification_email must be an email address")}
				}
				return nil, nil
			},
		},
	}
	for _, field := range monitoringFields {
		fields[field.name] = &schema.Schema{
			Type: field.kind, Optional: true, Computed: true,
			Description: field.description, Sensitive: field.sensitive, ValidateFunc: field.validate,
		}
	}
	return &schema.Resource{
		Description:   "Manages a standalone ClouDNS Monitoring check.",
		Schema:        fields,
		CreateContext: resourceMonitoringCheckCreate,
		ReadContext:   resourceMonitoringCheckRead,
		UpdateContext: resourceMonitoringCheckUpdate,
		DeleteContext: resourceMonitoringCheckDelete,
		CustomizeDiff: validateMonitoringCheck,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func monitorFieldApplies(field monitoringField, checkType string) bool {
	return len(field.types) == 0 || slices.Contains(field.types, checkType)
}

func validateMonitoringCheck(_ context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	if !diff.NewValueKnown("check_type") {
		return nil
	}
	checkType := diff.Get("check_type").(string)
	config := diff.GetRawConfig()
	if config.IsNull() {
		return nil
	}
	for _, field := range monitoringFields {
		if !monitorFieldApplies(field, checkType) && !config.GetAttr(field.name).IsNull() {
			return fmt.Errorf("%s cannot be set for a %s monitoring check", field.name, checkType)
		}
	}
	required := []string{"ip"}
	switch checkType {
	case "heartbeat":
		required = nil
	case "web":
		required = append(required, "host", "port")
	case "keyword":
		required = append(required, "host", "port", "keyword")
	case "tcp", "udp":
		required = append(required, "port", "open_port")
	case "dns":
		required = append(required, "host", "query_type", "query_response")
	case "tls":
		required = append(required, "host", "port")
	case "smtp", "imap":
		required = append(required, "port")
	case "streaming":
		required = append(required, "url")
	}
	for _, field := range required {
		value := config.GetAttr(field)
		if value.IsNull() || (value.IsKnown() && value.Type().Equals(cty.String) && strings.TrimSpace(value.AsString()) == "") {
			return fmt.Errorf("%s is required for a %s monitoring check", field, checkType)
		}
	}
	if _, configured := diff.GetOk("notification_email"); !configured && diff.NewValueKnown("notification_email") && diff.Id() == "" {
		if config, ok := meta.(ClientConfig); ok && config.apiAccess.Subauthid != 0 {
			return errors.New("notification_email is required when creating a check using a ClouDNS API sub-user")
		}
	}
	if timeout, ok := diff.GetOk("timeout"); ok && !config.GetAttr("timeout").IsNull() {
		seconds := timeout.(int)
		if (checkType == "web" && seconds < 4) || (checkType != "web" && seconds > 5) {
			return fmt.Errorf("invalid timeout for a %s monitoring check", checkType)
		}
	}
	if checkType == "imap" && diff.Get("connection_security") == 2 && !config.GetAttr("connection_security").IsNull() {
		return errors.New("IMAP supports connection_security 1 (none) or 3 (TLS)")
	}
	return nil
}

func monitoringCheckFromResource(d *schema.ResourceData, id int) cloudns.MonitoringCheck {
	checkType := d.Get("check_type").(string)
	check := cloudns.MonitoringCheck{ID: id, Name: d.Get("name").(string), CheckType: monitoringCheckTypes[checkType], Settings: url.Values{}}
	config := d.GetRawConfig()
	for _, field := range monitoringFields {
		if !monitorFieldApplies(field, checkType) {
			continue
		}
		value, set := d.GetOkExists(field.name)
		configured := !config.IsNull() && !config.GetAttr(field.name).IsNull()
		if set || configured {
			check.Settings.Set(field.name, fmt.Sprint(value))
		}
	}
	if value, exists := d.GetOkExists("enabled"); exists {
		state := "0"
		if value.(bool) {
			state = "1"
		}
		check.Settings.Set("state", state)
	}
	if slices.Contains([]string{"web", "keyword"}, checkType) && !check.Settings.Has("http_status_code") {
		// The API documents this field as required on update.
		check.Settings.Set("http_status_code", "200")
	}
	return check
}

func resourceMonitoringCheckCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(ClientConfig)
	notification := d.Get("notification_email").(string)
	if config.apiAccess.Subauthid != 0 && notification == "" {
		return diag.Errorf("notification_email is required when creating a check using a ClouDNS API sub-user")
	}
	config.rateLimiter.Take()
	check, err := config.apiAccess.CreateMonitoringCheck(ctx, monitoringCheckFromResource(d, 0), notification)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(check.ID))
	return refreshMonitoringCheckAfterWrite(ctx, d, meta)
}

func resourceMonitoringCheckRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id, err := monitoringResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	config := meta.(ClientConfig)
	config.rateLimiter.Take()
	check, err := config.apiAccess.ReadMonitoringCheck(ctx, id)
	if errors.Is(err, cloudns.ErrMonitoringCheckNotFound) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(setMonitoringCheckState(d, check))
}

func setMonitoringCheckState(d *schema.ResourceData, check cloudns.MonitoringCheck) error {
	var checkType string
	for name, id := range monitoringCheckTypes {
		if id == check.CheckType {
			checkType = name
			break
		}
	}
	if checkType == "" {
		return fmt.Errorf("unsupported Monitoring check type %d", check.CheckType)
	}
	state := check.Settings.Get("state")
	if state != "0" && state != "1" {
		return errors.New("Monitoring read returned an invalid enabled state")
	}
	fields := map[string]interface{}{"name": check.Name, "check_type": checkType, "enabled": state == "1", "heartbeat_url": check.HeartbeatURL}
	for _, field := range monitoringFields {
		raw := check.Settings.Get(field.name)
		if !monitorFieldApplies(field, checkType) || raw == "" {
			fields[field.name] = nil
			continue
		}
		var err error
		switch field.kind {
		case schema.TypeInt:
			fields[field.name], err = strconv.Atoi(raw)
		case schema.TypeFloat:
			fields[field.name], err = strconv.ParseFloat(raw, 64)
		default:
			fields[field.name] = raw
		}
		if err != nil {
			return fmt.Errorf("Monitoring read returned an invalid %s", field.name)
		}
	}
	for key, value := range fields {
		if err := d.Set(key, value); err != nil {
			return fmt.Errorf("could not set Monitoring check attribute %s", key)
		}
	}
	return nil
}

func resourceMonitoringCheckUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id, err := monitoringResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	config := meta.(ClientConfig)
	config.rateLimiter.Take()
	if err := config.apiAccess.UpdateMonitoringCheck(ctx, monitoringCheckFromResource(d, id)); err != nil {
		return diag.FromErr(err)
	}
	return refreshMonitoringCheckAfterWrite(ctx, d, meta)
}

func refreshMonitoringCheckAfterWrite(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id := d.Id()
	diagnostics := resourceMonitoringCheckRead(ctx, d, meta)
	if d.Id() == "" {
		d.SetId(id)
		return diag.Errorf("Monitoring check %s was written but could not be read back", id)
	}
	return diagnostics
}

func resourceMonitoringCheckDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id, err := monitoringResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	config := meta.(ClientConfig)
	config.rateLimiter.Take()
	if err := config.apiAccess.DeleteMonitoringCheck(ctx, id); err != nil && !errors.Is(err, cloudns.ErrMonitoringCheckNotFound) {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func monitoringResourceID(raw string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, errors.New("Monitoring check ID must be a positive integer")
	}
	return id, nil
}
