package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Replication statuses returned by the `_status` API.
const (
	replicationStatusSyncing       = "SYNCING"
	replicationStatusPaused        = "PAUSED"
	replicationStatusFailed        = "FAILED"
	replicationStatusNotInProgress = "REPLICATION NOT IN PROGRESS"
)

func resourceOpensearchCrossClusterReplication() *schema.Resource {
	return &schema.Resource{
		Description:   "Replicates an index of a remote (leader) cluster onto the local (follower) cluster, see the [cross-cluster replication documentation](https://docs.opensearch.org/latest/tuning-your-cluster/replication-plugin/getting-started/). Send this to the follower cluster. The `leader_alias` must reference a cross-cluster connection, see `opensearch_cross_cluster_connection`.\n\n~> Starting replication creates the follower index; an existing index cannot be converted into a follower index. Destroying this resource stops replication and turns the follower index into a regular, writable index: the index and its documents are **not** deleted. Because of that, replicating onto the same index name again (e.g. recreating the resource) fails with `Cant use same index again for replication` until the index is deleted.",
		CreateContext: resourceOpensearchCrossClusterReplicationCreate,
		ReadContext:   resourceOpensearchCrossClusterReplicationRead,
		UpdateContext: resourceOpensearchCrossClusterReplicationUpdate,
		DeleteContext: resourceOpensearchCrossClusterReplicationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"follower_index": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the index created on the follower cluster",
			},
			"leader_alias": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the cross-cluster connection to the leader cluster, see `opensearch_cross_cluster_connection`",
			},
			"leader_index": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the index to replicate on the leader cluster",
			},
			"use_roles": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "The roles used for all subsequent background replication tasks between the indexes. Required if the security plugin is enabled. Cannot be changed without recreating the replication.",
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
			"settings": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Index settings of the follower index, e.g. `index.number_of_replicas`. Applied with the replication update settings API, only the settings listed here are managed.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"paused": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether replication of the leader index is paused. Note that replication cannot be resumed normally once it has been paused for longer than the retention lease period (12 hours by default), see `force_resume`.",
			},
			"force_resume": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether resuming performs a stop-delete-start cycle, restoring the follower index from the leader, when the retention leases have expired. Only used when `paused` goes from `true` to `false`. Defaults to `false`.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the replication: `SYNCING`, `PAUSED`, `FAILED`, or the bootstrapping status reported while the follower index is being restored from the leader",
			},
			"reason": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The reason of the current status, e.g. `User initiated`",
			},
			"leader_checkpoint": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The checkpoint of the leader index; when it is equal to `follower_checkpoint` the indexes are fully synced",
			},
			"follower_checkpoint": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The checkpoint of the follower index; when it is equal to `leader_checkpoint` the indexes are fully synced",
			},
		},
	}
}

func resourceOpensearchCrossClusterReplicationCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	conf := m.(*ProviderConf)
	followerIndex := d.Get("follower_index").(string)

	payload := map[string]any{
		"leader_alias": d.Get("leader_alias").(string),
		"leader_index": d.Get("leader_index").(string),
	}
	if roles := expandCrossClusterReplicationUseRoles(d.Get("use_roles").([]any)); roles != nil {
		payload["use_roles"] = roles
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return diag.Errorf("failed to marshal start replication payload: %s", err)
	}

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_replication/%s/_start", followerIndex)
	if _, err := performRequestAndParse(ctx, conf.osClient, "PUT", url, strings.NewReader(string(body)), "start replication"); err != nil {
		return diag.FromErr(err)
	}

	// The replication is created at this point: set the ID so that a failure
	// of the operations below leaves a destroyable resource behind.
	d.SetId(followerIndex)

	// The follower index is restored from the leader before replication
	// actually starts, and the plugin rejects updates while bootstrapping.
	if err := waitForCrossClusterReplicationSyncing(ctx, conf, followerIndex, d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.FromErr(err)
	}

	if settings := d.Get("settings").(map[string]any); len(settings) > 0 {
		if err := updateCrossClusterReplicationSettings(ctx, conf, followerIndex, settings); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.Get("paused").(bool) {
		if err := pauseCrossClusterReplication(ctx, conf, followerIndex); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceOpensearchCrossClusterReplicationRead(ctx, d, m)
}

func resourceOpensearchCrossClusterReplicationRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	conf := m.(*ProviderConf)
	followerIndex := d.Id()

	status, err := getCrossClusterReplicationStatus(ctx, conf, followerIndex)
	if err != nil {
		var httpErr *HTTPError
		// The follower index itself is gone.
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	state, _ := status["status"].(string)
	if state == "" || strings.EqualFold(state, replicationStatusNotInProgress) {
		d.SetId("")
		return nil
	}

	ds := &resourceDataSetter{d: d}
	ds.set("follower_index", followerIndex)
	ds.set("status", state)
	ds.set("paused", strings.EqualFold(state, replicationStatusPaused))

	// The leader information is not reported for every status, keep whatever
	// is already in state in that case.
	if leaderAlias, ok := status["leader_alias"].(string); ok && leaderAlias != "" {
		ds.set("leader_alias", leaderAlias)
	}
	if leaderIndex, ok := status["leader_index"].(string); ok && leaderIndex != "" {
		ds.set("leader_index", leaderIndex)
	}
	if reason, ok := status["reason"].(string); ok {
		ds.set("reason", reason)
	}

	if details, ok := status["syncing_details"].(map[string]any); ok {
		if checkpoint, ok := details["leader_checkpoint"].(float64); ok {
			ds.set("leader_checkpoint", int(checkpoint))
		}
		if checkpoint, ok := details["follower_checkpoint"].(float64); ok {
			ds.set("follower_checkpoint", int(checkpoint))
		}
	}

	if ds.err != nil {
		return diag.FromErr(ds.err)
	}

	// Only the settings managed by the configuration are tracked, the follower
	// index has many more.
	managed := d.Get("settings").(map[string]any)
	if len(managed) > 0 {
		settings, err := getCrossClusterReplicationIndexSettings(ctx, conf, followerIndex, managed)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("settings", settings); err != nil {
			return diag.Errorf("error setting settings: %s", err)
		}
	}

	return nil
}

func resourceOpensearchCrossClusterReplicationUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	conf := m.(*ProviderConf)
	followerIndex := d.Id()

	// Resume first: a paused replication rejects setting updates.
	resumed := false
	if d.HasChange("paused") && !d.Get("paused").(bool) {
		if err := resumeCrossClusterReplication(ctx, conf, followerIndex, d.Get("force_resume").(bool)); err != nil {
			return diag.FromErr(err)
		}
		if err := waitForCrossClusterReplicationSyncing(ctx, conf, followerIndex, d.Timeout(schema.TimeoutUpdate)); err != nil {
			return diag.FromErr(err)
		}
		resumed = true
	}

	if d.HasChange("settings") {
		settings := d.Get("settings").(map[string]any)
		// Settings removed from the configuration are no longer managed, they
		// keep their current value on the follower index.
		if len(settings) > 0 {
			if err := updateCrossClusterReplicationSettings(ctx, conf, followerIndex, settings); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("paused") && d.Get("paused").(bool) && !resumed {
		if err := pauseCrossClusterReplication(ctx, conf, followerIndex); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceOpensearchCrossClusterReplicationRead(ctx, d, m)
}

func resourceOpensearchCrossClusterReplicationDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	conf := m.(*ProviderConf)
	followerIndex := d.Id()

	status, err := getCrossClusterReplicationStatus(ctx, conf, followerIndex)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	if state, _ := status["status"].(string); strings.EqualFold(state, replicationStatusNotInProgress) {
		return nil
	}

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_replication/%s/_stop", followerIndex)
	if _, err := performRequestAndParse(ctx, conf.osClient, "POST", url, strings.NewReader("{}"), "stop replication"); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// ============================================
// ===         Helper functions             ===
// ============================================

func expandCrossClusterReplicationUseRoles(useRoles []any) map[string]any {
	if len(useRoles) == 0 || useRoles[0] == nil {
		return nil
	}

	roles := useRoles[0].(map[string]any)

	return map[string]any{
		"leader_cluster_role":   roles["leader_cluster_role"].(string),
		"follower_cluster_role": roles["follower_cluster_role"].(string),
	}
}

func getCrossClusterReplicationStatus(ctx context.Context, conf *ProviderConf, followerIndex string) (map[string]any, error) {
	url := conf.rawUrl + fmt.Sprintf("/_plugins/_replication/%s/_status", followerIndex)

	return performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "get replication status")
}

func pauseCrossClusterReplication(ctx context.Context, conf *ProviderConf, followerIndex string) error {
	url := conf.rawUrl + fmt.Sprintf("/_plugins/_replication/%s/_pause", followerIndex)
	_, err := performRequestAndParse(ctx, conf.osClient, "POST", url, strings.NewReader("{}"), "pause replication")

	return err
}

func resumeCrossClusterReplication(ctx context.Context, conf *ProviderConf, followerIndex string, forceResume bool) error {
	body := "{}"
	if forceResume {
		body = `{"force_resume":true}`
	}

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_replication/%s/_resume", followerIndex)
	_, err := performRequestAndParse(ctx, conf.osClient, "POST", url, strings.NewReader(body), "resume replication")

	return err
}

func updateCrossClusterReplicationSettings(ctx context.Context, conf *ProviderConf, followerIndex string, settings map[string]any) error {
	body, err := json.Marshal(map[string]any{"settings": settings})
	if err != nil {
		return fmt.Errorf("failed to marshal replication settings payload: %s", err)
	}

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_replication/%s/_update", followerIndex)
	_, err = performRequestAndParse(ctx, conf.osClient, "PUT", url, strings.NewReader(string(body)), "update replication settings")

	return err
}

// Returns the current value of the given index settings of the follower index,
// keyed like the managed settings.
func getCrossClusterReplicationIndexSettings(ctx context.Context, conf *ProviderConf, followerIndex string, managed map[string]any) (map[string]any, error) {
	url := conf.rawUrl + fmt.Sprintf("/%s/_settings?flat_settings=true", followerIndex)
	result, err := performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "get follower index settings")
	if err != nil {
		return nil, err
	}

	index, ok := result[followerIndex].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("index %q not found in the settings response", followerIndex)
	}
	settings, ok := index["settings"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("no settings returned for index %q", followerIndex)
	}

	current := make(map[string]any, len(managed))
	for key := range managed {
		if value, ok := settings[key]; ok {
			current[key] = fmt.Sprintf("%v", value)
		}
	}

	return current, nil
}

// Waits until the replication left the bootstrapping phase, i.e. until the
// follower index has been restored from the leader and the plugin accepts
// pause and settings updates.
func waitForCrossClusterReplicationSyncing(ctx context.Context, conf *ProviderConf, followerIndex string, timeout time.Duration) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		status, err := getCrossClusterReplicationStatus(ctx, conf, followerIndex)
		if err != nil {
			return retry.NonRetryableError(err)
		}

		state, _ := status["status"].(string)
		switch {
		case strings.EqualFold(state, replicationStatusSyncing), strings.EqualFold(state, replicationStatusPaused):
			return nil
		case strings.EqualFold(state, replicationStatusFailed):
			reason, _ := status["reason"].(string)
			return retry.NonRetryableError(fmt.Errorf("replication of index %q failed: %s", followerIndex, reason))
		default:
			log.Printf("[INFO] waiting for replication of index %q, current status: %s", followerIndex, state)
			return retry.RetryableError(fmt.Errorf("replication of index %q is in status %q, waiting for %q", followerIndex, state, replicationStatusSyncing))
		}
	})
}
