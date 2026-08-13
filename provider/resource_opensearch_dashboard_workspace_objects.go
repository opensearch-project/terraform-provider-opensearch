package provider

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpensearchDashboardWorkspaceObjects() *schema.Resource {
	return &schema.Resource{
		Description: "Associates existing saved objects with an OpenSearch Dashboards workspace.\n\nThis resource uses the OpenSearch Dashboards API, so the provider must be configured with `dashboards_url`.",
		Create:      resourceOpensearchDashboardWorkspaceObjectsCreate,
		Read:        resourceOpensearchDashboardWorkspaceObjectsRead,
		Update:      resourceOpensearchDashboardWorkspaceObjectsUpdate,
		Delete:      resourceOpensearchDashboardWorkspaceObjectsDelete,
		Schema: map[string]*schema.Schema{
			"workspace_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Workspace ID to associate saved objects with.",
			},
			"saved_object": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem:        dashboardWorkspaceSavedObjectSchema(),
				Description: "Saved objects to associate with the workspace.",
			},
		},
	}
}

func dashboardWorkspaceSavedObjectSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Saved object type, such as index-pattern, config, or dashboard.",
			},
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Saved object ID.",
			},
		},
	}
}

func resourceOpensearchDashboardWorkspaceObjectsCreate(d *schema.ResourceData, meta interface{}) error {
	if err := associateDashboardWorkspaceObjects(d, meta, "/api/workspaces/_associate"); err != nil {
		return err
	}
	d.SetId(d.Get("workspace_id").(string))
	return resourceOpensearchDashboardWorkspaceObjectsRead(d, meta)
}

func resourceOpensearchDashboardWorkspaceObjectsRead(d *schema.ResourceData, meta interface{}) error {
	_, err := getDashboardWorkspace(d.Get("workspace_id").(string), meta)
	if err != nil {
		if errors.Is(err, errDashboardsNotFound) {
			d.SetId("")
			return nil
		}
		return err
	}
	return nil
}

func resourceOpensearchDashboardWorkspaceObjectsUpdate(d *schema.ResourceData, meta interface{}) error {
	oldObjects, newObjects := d.GetChange("saved_object")
	oldSet := oldObjects.(*schema.Set)
	newSet := newObjects.(*schema.Set)

	removed := oldSet.Difference(newSet)
	if removed.Len() > 0 {
		if err := dashboardWorkspaceObjectsRequest(meta.(*ProviderConf), "/api/workspaces/_dissociate", d.Get("workspace_id").(string), removed); err != nil {
			return err
		}
	}

	added := newSet.Difference(oldSet)
	if added.Len() > 0 {
		if err := dashboardWorkspaceObjectsRequest(meta.(*ProviderConf), "/api/workspaces/_associate", d.Get("workspace_id").(string), added); err != nil {
			return err
		}
	}

	return resourceOpensearchDashboardWorkspaceObjectsRead(d, meta)
}

func resourceOpensearchDashboardWorkspaceObjectsDelete(d *schema.ResourceData, meta interface{}) error {
	if err := associateDashboardWorkspaceObjects(d, meta, "/api/workspaces/_dissociate"); err != nil {
		if errors.Is(err, errDashboardsNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func associateDashboardWorkspaceObjects(d *schema.ResourceData, meta interface{}, path string) error {
	return dashboardWorkspaceObjectsRequest(
		meta.(*ProviderConf),
		path,
		d.Get("workspace_id").(string),
		d.Get("saved_object").(*schema.Set),
	)
}

func dashboardWorkspaceObjectsRequest(conf *ProviderConf, path, workspaceID string, objects *schema.Set) error {
	request := dashboardWorkspaceObjectsRequestBody{
		WorkspaceID:  workspaceID,
		SavedObjects: expandDashboardWorkspaceSavedObjects(objects),
	}

	var response dashboardWorkspaceObjectsResponse
	if err := dashboardsRequest(conf, http.MethodPost, path, request, &response); err != nil {
		return err
	}
	if !response.Success {
		return fmt.Errorf("unexpected workspace saved objects response: %+v", response)
	}
	return nil
}

func expandDashboardWorkspaceSavedObjects(objects *schema.Set) []dashboardWorkspaceSavedObject {
	savedObjects := make([]dashboardWorkspaceSavedObject, 0, objects.Len())
	for _, item := range objects.List() {
		object := item.(map[string]interface{})
		savedObjects = append(savedObjects, dashboardWorkspaceSavedObject{
			Type: object["type"].(string),
			ID:   object["id"].(string),
		})
	}
	return savedObjects
}

type dashboardWorkspaceObjectsRequestBody struct {
	WorkspaceID  string                          `json:"workspaceId"`
	SavedObjects []dashboardWorkspaceSavedObject `json:"savedObjects"`
}

type dashboardWorkspaceSavedObject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type dashboardWorkspaceObjectsResponse struct {
	Success bool `json:"success"`
}
