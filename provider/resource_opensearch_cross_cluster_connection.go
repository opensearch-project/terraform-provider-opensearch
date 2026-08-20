package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Kinds of the remote cluster settings managed by this resource, used to
// serialize values to the cluster settings API and parse them back (the
// settings API returns every value as a string when `flat_settings=true`).
const (
	crossClusterSettingString = "string"
	crossClusterSettingInt    = "int"
	crossClusterSettingBool   = "bool"
	crossClusterSettingList   = "list"
)

type crossClusterConnectionSetting struct {
	// attr is the Terraform attribute name.
	attr string
	// suffix is appended to `cluster.remote.<alias>.` to build the setting key.
	suffix string
	kind   string
}

// The `cluster.remote.<alias>.*` persistent cluster settings managed by
// opensearch_cross_cluster_connection. Every one of them is written on each
// create/update (unset attributes are written as `null` so that removing an
// attribute from the configuration resets the setting).
var crossClusterConnectionSettings = []crossClusterConnectionSetting{
	{attr: "mode", suffix: "mode", kind: crossClusterSettingString},
	{attr: "seeds", suffix: "seeds", kind: crossClusterSettingList},
	{attr: "node_connections", suffix: "node_connections", kind: crossClusterSettingInt},
	{attr: "proxy_address", suffix: "proxy_address", kind: crossClusterSettingString},
	{attr: "proxy_socket_connections", suffix: "proxy_socket_connections", kind: crossClusterSettingInt},
	{attr: "server_name", suffix: "server_name", kind: crossClusterSettingString},
	{attr: "skip_unavailable", suffix: "skip_unavailable", kind: crossClusterSettingBool},
	{attr: "transport_compress", suffix: "transport.compress", kind: crossClusterSettingBool},
	{attr: "transport_ping_schedule", suffix: "transport.ping_schedule", kind: crossClusterSettingString},
}

func resourceOpensearchCrossClusterConnection() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides a cross-cluster connection, i.e. the `cluster.remote.<alias>.*` persistent cluster settings that register a remote (leader) cluster on the local (follower) cluster. This is the first step of a cross-cluster replication setup: the alias defined here is the `leader_alias` of `opensearch_cross_cluster_replication` and `opensearch_cross_cluster_replication_rule`. Send this to the follower cluster.",
		CreateContext: resourceOpensearchCrossClusterConnectionCreate,
		ReadContext:   resourceOpensearchCrossClusterConnectionRead,
		UpdateContext: resourceOpensearchCrossClusterConnectionUpdate,
		DeleteContext: resourceOpensearchCrossClusterConnectionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: resourceOpensearchCrossClusterConnectionCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The alias of the cross-cluster connection, used as `leader_alias` when starting replication",
			},
			"mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "sniff",
				Description:  "The connection mode, either `sniff` (the follower connects to the seed nodes and discovers the leader cluster's nodes, requires the leader nodes publish addresses to be reachable) or `proxy` (all connections go through a single proxy address). Defaults to `sniff`.",
				ValidateFunc: validation.StringInSlice([]string{"sniff", "proxy"}, false),
			},
			"seeds": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "The transport addresses (`host:port`, the transport port is 9300 by default) of the seed nodes of the leader cluster. Required and only valid in `sniff` mode.",
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"proxy_address", "proxy_socket_connections", "server_name"},
			},
			"node_connections": {
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "The number of gateway nodes of the leader cluster to connect to. Only valid in `sniff` mode, defaults to `3` on the server side.",
				ValidateFunc:  validation.IntAtLeast(1),
				ConflictsWith: []string{"proxy_address", "proxy_socket_connections", "server_name"},
			},
			"proxy_address": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The address (`host:port`) of the proxy forwarding connections to the leader cluster. Required and only valid in `proxy` mode.",
				ConflictsWith: []string{"seeds", "node_connections"},
			},
			"proxy_socket_connections": {
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "The number of socket connections opened to the proxy address. Only valid in `proxy` mode, defaults to `18` on the server side.",
				ValidateFunc:  validation.IntAtLeast(1),
				ConflictsWith: []string{"seeds", "node_connections"},
			},
			"server_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "The server name passed as the TLS server name indication (SNI) of the proxy connections. Only valid in `proxy` mode.",
				ConflictsWith: []string{"seeds", "node_connections"},
			},
			"skip_unavailable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to ignore the remote cluster when it is unavailable instead of failing requests targeting it. Defaults to `false`.",
			},
			"transport_compress": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to compress the requests sent to the remote cluster. Defaults to `false`.",
			},
			"transport_ping_schedule": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The interval at which application level pings are sent on the connections to the remote cluster, e.g. `30s`. Defaults to `-1` (disabled) on the server side.",
			},
		},
	}
}

func resourceOpensearchCrossClusterConnectionCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, m any) error {
	return validateCrossClusterConnectionMode(d)
}

// Implemented by both *schema.ResourceDiff and *schema.ResourceData, so that
// the validation below can run at plan time and be unit tested.
type crossClusterConnectionConfig interface {
	Get(key string) any
	GetOk(key string) (any, bool)
}

// The connection settings of a mode are rejected by the cluster when another
// mode is configured; catch the missing address of the configured mode early,
// the conflicting ones are handled by ConflictsWith.
func validateCrossClusterConnectionMode(d crossClusterConnectionConfig) error {
	switch d.Get("mode").(string) {
	case "proxy":
		if v, ok := d.GetOk("proxy_address"); !ok || v.(string) == "" {
			return fmt.Errorf("proxy_address is required when mode is \"proxy\"")
		}
	default:
		if v, ok := d.GetOk("seeds"); !ok || len(v.([]any)) == 0 {
			return fmt.Errorf("seeds is required when mode is \"sniff\"")
		}
	}

	return nil
}

func resourceOpensearchCrossClusterConnectionCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	name := d.Get("name").(string)

	if err := putCrossClusterConnectionSettings(ctx, m.(*ProviderConf), crossClusterConnectionPayload(d, name), "create cross-cluster connection"); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)
	return resourceOpensearchCrossClusterConnectionRead(ctx, d, m)
}

func resourceOpensearchCrossClusterConnectionRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	conf := m.(*ProviderConf)
	name := d.Id()

	settings, err := getCrossClusterConnectionSettings(ctx, conf, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(settings) == 0 {
		d.SetId("")
		return nil
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", name)

	for _, setting := range crossClusterConnectionSettings {
		raw, ok := settings[setting.suffix]
		if !ok {
			continue
		}

		switch setting.kind {
		case crossClusterSettingList:
			values, err := crossClusterSettingToStringList(raw)
			if err != nil {
				return diag.Errorf("error reading setting %q of cross-cluster connection %q: %s", setting.suffix, name, err)
			}
			ds.set(setting.attr, values)
		case crossClusterSettingInt:
			value, err := strconv.Atoi(fmt.Sprintf("%v", raw))
			if err != nil {
				return diag.Errorf("error reading setting %q of cross-cluster connection %q: %s", setting.suffix, name, err)
			}
			ds.set(setting.attr, value)
		case crossClusterSettingBool:
			value, err := strconv.ParseBool(fmt.Sprintf("%v", raw))
			if err != nil {
				return diag.Errorf("error reading setting %q of cross-cluster connection %q: %s", setting.suffix, name, err)
			}
			ds.set(setting.attr, value)
		default:
			ds.set(setting.attr, fmt.Sprintf("%v", raw))
		}
	}

	if ds.err != nil {
		return diag.FromErr(ds.err)
	}

	return nil
}

func resourceOpensearchCrossClusterConnectionUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if err := putCrossClusterConnectionSettings(ctx, m.(*ProviderConf), crossClusterConnectionPayload(d, d.Id()), "update cross-cluster connection"); err != nil {
		return diag.FromErr(err)
	}

	return resourceOpensearchCrossClusterConnectionRead(ctx, d, m)
}

func resourceOpensearchCrossClusterConnectionDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	name := d.Id()

	// Removing a remote cluster is done by resetting all of its settings.
	settings := make(map[string]any, len(crossClusterConnectionSettings))
	for _, setting := range crossClusterConnectionSettings {
		settings[crossClusterConnectionSettingKey(name, setting.suffix)] = nil
	}

	if err := putCrossClusterConnectionSettings(ctx, m.(*ProviderConf), settings, "delete cross-cluster connection"); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// ============================================
// ===         Helper functions             ===
// ============================================

func crossClusterConnectionSettingKey(name string, suffix string) string {
	return fmt.Sprintf("cluster.remote.%s.%s", name, suffix)
}

// Builds the flat `cluster.remote.<alias>.*` settings from the resource data.
// Attributes that are not set are explicitly written as `null` so that
// removing an attribute from the configuration resets the setting.
func crossClusterConnectionPayload(d *schema.ResourceData, name string) map[string]any {
	settings := make(map[string]any, len(crossClusterConnectionSettings))

	for _, setting := range crossClusterConnectionSettings {
		key := crossClusterConnectionSettingKey(name, setting.suffix)
		settings[key] = nil

		switch setting.kind {
		case crossClusterSettingList:
			if v, ok := d.GetOk(setting.attr); ok {
				if list := expandStringList(v.([]any)); len(list) > 0 {
					settings[key] = list
				}
			}
		case crossClusterSettingInt:
			if v, ok := d.GetOk(setting.attr); ok {
				settings[key] = v.(int)
			}
		case crossClusterSettingBool:
			// Booleans always have a default matching the server side default,
			// so they can always be written.
			settings[key] = d.Get(setting.attr).(bool)
		default:
			if v, ok := d.GetOk(setting.attr); ok && v.(string) != "" {
				settings[key] = v.(string)
			}
		}
	}

	return settings
}

func putCrossClusterConnectionSettings(ctx context.Context, conf *ProviderConf, settings map[string]any, operation string) error {
	body, err := json.Marshal(map[string]any{"persistent": settings})
	if err != nil {
		return fmt.Errorf("failed to marshal %s payload: %s", operation, err)
	}

	url := conf.rawUrl + "/_cluster/settings"
	_, err = performRequestAndParse(ctx, conf.osClient, "PUT", url, strings.NewReader(string(body)), operation)

	return err
}

// Returns the persistent settings of the given remote cluster alias, keyed by
// the setting suffix (e.g. `seeds`). An empty map means the connection does
// not exist.
func getCrossClusterConnectionSettings(ctx context.Context, conf *ProviderConf, name string) (map[string]any, error) {
	url := conf.rawUrl + "/_cluster/settings?flat_settings=true"
	result, err := performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "read cross-cluster connection")
	if err != nil {
		return nil, err
	}

	persistent, ok := result["persistent"].(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}

	prefix := fmt.Sprintf("cluster.remote.%s.", name)
	settings := map[string]any{}
	for key, value := range persistent {
		if suffix, found := strings.CutPrefix(key, prefix); found {
			settings[suffix] = value
		}
	}

	return settings, nil
}

func crossClusterSettingToStringList(raw any) ([]string, error) {
	switch value := raw.(type) {
	case []any:
		list := make([]string, 0, len(value))
		for _, item := range value {
			list = append(list, fmt.Sprintf("%v", item))
		}
		return list, nil
	case string:
		// A single valued list is returned as a plain string by some versions.
		return []string{value}, nil
	default:
		return nil, fmt.Errorf("expected a list of strings, got %T", raw)
	}
}
