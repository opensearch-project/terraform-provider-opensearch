package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpensearchCrossClusterReplicationRule() *schema.Resource {
	return &schema.Resource{
		Description:   "Provides an auto-follow replication rule: indexes of the remote (leader) cluster matching the pattern are automatically replicated onto the local (follower) cluster, see the [auto-follow documentation](https://docs.opensearch.org/latest/tuning-your-cluster/replication-plugin/auto-follow/). Send this to the follower cluster. The `leader_alias` must reference a cross-cluster connection, see `opensearch_cross_cluster_connection`.\n\n~> Creating the rule starts replicating the indexes that already match the pattern, in addition to the ones created later. Destroying it only stops *new* indexes from being replicated: the indexes it already created keep replicating and stay read-only until their replication is stopped, e.g. by importing them as `opensearch_cross_cluster_replication` resources and destroying those.\n\n~> The plugin rejects updates of an existing rule (`Exisiting autofollow replication rule cannot be recreated/updated`), so every attribute forces the rule to be deleted and created again.",
		CreateContext: resourceOpensearchCrossClusterReplicationRuleCreate,
		ReadContext:   resourceOpensearchCrossClusterReplicationRuleRead,
		DeleteContext: resourceOpensearchCrossClusterReplicationRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceOpensearchCrossClusterReplicationRuleImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the auto-follow replication rule",
			},
			"leader_alias": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the cross-cluster connection to the leader cluster, see `opensearch_cross_cluster_connection`",
			},
			"pattern": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The index pattern matched against the indexes of the leader cluster, wildcards are supported, e.g. `leader-*`. The plugin rejects updates of an existing rule, so changing it recreates the rule.",
			},
			"use_roles": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "The roles used for all subsequent background replication tasks between the indexes. Required if the security plugin is enabled. Cannot be changed without recreating the rule.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"leader_cluster_role": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The role used to authenticate the replication requests on the leader cluster, e.g. `cross_cluster_replication_leader_full_access`",
						},
						"follower_cluster_role": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The role used to authenticate the replication requests on the follower cluster, e.g. `cross_cluster_replication_follower_full_access`",
						},
					},
				},
			},
		},
	}
}

func resourceOpensearchCrossClusterReplicationRuleCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	leaderAlias := d.Get("leader_alias").(string)
	name := d.Get("name").(string)

	if err := createCrossClusterReplicationRule(ctx, m.(*ProviderConf), d); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(crossClusterReplicationRuleID(leaderAlias, name))

	return resourceOpensearchCrossClusterReplicationRuleRead(ctx, d, m)
}

func resourceOpensearchCrossClusterReplicationRuleRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	conf := m.(*ProviderConf)

	leaderAlias, name, err := parseCrossClusterReplicationRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	rule, err := getCrossClusterReplicationRule(ctx, conf, leaderAlias, name)
	if err != nil {
		var httpErr *HTTPError
		// The auto-follow stats API is the only one exposing the configured
		// rules, and the built-in replication roles do not grant
		// `indices:admin/plugins/replication/autofollow/stats`. Keep whatever
		// is in state rather than reporting the rule as gone.
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden {
			log.Printf("[WARN] not allowed to read the auto-follow stats, keeping the state of the replication rule %q unchanged: %s", d.Id(), err)
			return nil
		}
		return diag.FromErr(err)
	}
	if rule == nil {
		// The rule is a persistent task (`autofollow:<leader_alias>:<name>` in
		// the cluster state), and the stats API only reports the tasks already
		// allocated to a node. The create request is acknowledged as soon as
		// the task is written, so right after one an absent rule means "not
		// started yet", not "deleted" — clearing the ID there makes the apply
		// fail with "Root object was present, but now absent". Every attribute
		// is in the configuration, so keeping the state is correct.
		if d.IsNewResource() {
			log.Printf("[DEBUG] replication rule %q not in the auto-follow stats yet, its task is still being allocated", d.Id())
			return nil
		}

		d.SetId("")
		return nil
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", name)
	ds.set("leader_alias", leaderAlias)
	if pattern, ok := rule["pattern"].(string); ok {
		ds.set("pattern", pattern)
	}
	if ds.err != nil {
		return diag.FromErr(ds.err)
	}

	return nil
}

func resourceOpensearchCrossClusterReplicationRuleDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	conf := m.(*ProviderConf)

	leaderAlias, name, err := parseCrossClusterReplicationRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	body, err := json.Marshal(map[string]any{
		"leader_alias": leaderAlias,
		"name":         name,
	})
	if err != nil {
		return diag.Errorf("failed to marshal delete replication rule payload: %s", err)
	}

	url := conf.rawUrl + "/_plugins/_replication/_autofollow"
	if _, err := performRequestAndParse(ctx, conf.osClient, "DELETE", url, strings.NewReader(string(body)), "delete replication rule"); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceOpensearchCrossClusterReplicationRuleImport(ctx context.Context, d *schema.ResourceData, m any) ([]*schema.ResourceData, error) {
	leaderAlias, name, err := parseCrossClusterReplicationRuleID(d.Id())
	if err != nil {
		return nil, err
	}

	if err := d.Set("leader_alias", leaderAlias); err != nil {
		return nil, err
	}
	if err := d.Set("name", name); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

// ============================================
// ===         Helper functions             ===
// ============================================

// The auto-follow API identifies a rule by the pair (leader alias, name), and
// not every version reports the alias back in the stats, so both are encoded
// in the resource ID.
func crossClusterReplicationRuleID(leaderAlias string, name string) string {
	return fmt.Sprintf("%s/%s", leaderAlias, name)
}

func parseCrossClusterReplicationRuleID(id string) (string, string, error) {
	leaderAlias, name, found := strings.Cut(id, "/")
	if !found || leaderAlias == "" || name == "" {
		return "", "", fmt.Errorf("invalid replication rule ID %q, expected the format \"<leader_alias>/<name>\"", id)
	}

	return leaderAlias, name, nil
}

func createCrossClusterReplicationRule(ctx context.Context, conf *ProviderConf, d *schema.ResourceData) error {
	payload := map[string]any{
		"leader_alias": d.Get("leader_alias").(string),
		"name":         d.Get("name").(string),
		"pattern":      d.Get("pattern").(string),
	}
	if roles := expandCrossClusterReplicationUseRoles(d.Get("use_roles").([]any)); roles != nil {
		payload["use_roles"] = roles
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal create replication rule payload: %s", err)
	}

	url := conf.rawUrl + "/_plugins/_replication/_autofollow"
	_, err = performRequestAndParse(ctx, conf.osClient, "POST", url, strings.NewReader(string(body)), "create replication rule")

	return err
}

// Looks the rule up in the auto-follow stats, the only API exposing the
// configured rules. Returns nil when there is no such rule.
func getCrossClusterReplicationRule(ctx context.Context, conf *ProviderConf, leaderAlias string, name string) (map[string]any, error) {
	url := conf.rawUrl + "/_plugins/_replication/autofollow_stats"
	result, err := performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "get replication rules")
	if err != nil {
		return nil, err
	}

	rules, ok := result["autofollow_stats"].([]any)
	if !ok {
		return nil, nil
	}

	for _, r := range rules {
		rule, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if ruleName, ok := rule["name"].(string); !ok || ruleName != name {
			continue
		}
		// Rule names are only unique per connection, but not every version
		// reports the alias a rule belongs to; only compare it when present.
		if alias, ok := rule["leader_alias"].(string); ok && alias != "" && alias != leaderAlias {
			continue
		}

		return rule, nil
	}

	return nil, nil
}
