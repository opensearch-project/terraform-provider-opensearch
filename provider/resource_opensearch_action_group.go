package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
)

var openDistroActionGroupSchema = map[string]*schema.Schema{
	"action_group_name": {
		Description: "The name of the security action group.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"allowed_actions": {
		Description: "A list of allowed actions.",
		Type:        schema.TypeSet,
		Required:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Set:         schema.HashString,
	},
	"type": {
		Description: "The type of the action group, either `cluster`, `index` or `kibana`.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"description": {
		Description: "Description of the action group.",
		Type:        schema.TypeString,
		Optional:    true,
	},
}

func resourceOpenSearchActionGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch security action group resource. Action groups are reusable sets of permissions that can be referenced from roles. Please refer to the OpenSearch Access Control documentation for details.",
		Create:      resourceOpensearchOpenDistroActionGroupCreate,
		Read:        resourceOpensearchOpenDistroActionGroupRead,
		Update:      resourceOpensearchOpenDistroActionGroupUpdate,
		Delete:      resourceOpensearchOpenDistroActionGroupDelete,
		Schema:      openDistroActionGroupSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchOpenDistroActionGroupCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroActionGroup(d, m); err != nil {
		log.Printf("[INFO] Failed to create OpenDistroActionGroup: %+v", err)
		return err
	}

	name := d.Get("action_group_name").(string)
	d.SetId(name)
	return resourceOpensearchOpenDistroActionGroupRead(d, m)
}

func resourceOpensearchOpenDistroActionGroupRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetOpenDistroActionGroup(d.Id(), m)

	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] OpenDistroActionGroup (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if err := d.Set("action_group_name", d.Id()); err != nil {
		return fmt.Errorf("error setting action_group_name: %s", err)
	}
	if err := d.Set("allowed_actions", res.AllowedActions); err != nil {
		return fmt.Errorf("error setting allowed_actions: %s", err)
	}
	if err := d.Set("type", res.Type); err != nil {
		return fmt.Errorf("error setting type: %s", err)
	}
	if err := d.Set("description", res.Description); err != nil {
		return fmt.Errorf("error setting description: %s", err)
	}

	return nil
}

func resourceOpensearchOpenDistroActionGroupUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroActionGroup(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroActionGroupRead(d, m)
}

func resourceOpensearchOpenDistroActionGroupDelete(d *schema.ResourceData, m interface{}) error {
	path, err := uritemplates.Expand("/_plugins/_security/api/actiongroups/{name}", map[string]string{
		"name": d.Get("action_group_name").(string),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for action group: %+v", err)
	}

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method:           "DELETE",
		Path:             path,
		RetryStatusCodes: []int{http.StatusConflict, http.StatusInternalServerError},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})

	return err
}

func resourceOpensearchGetOpenDistroActionGroup(actionGroupID string, m interface{}) (ActionGroupBody, error) {
	var err error
	actionGroup := new(ActionGroupBody)

	path, err := uritemplates.Expand("/_plugins/_security/api/actiongroups/{name}", map[string]string{
		"name": actionGroupID,
	})

	if err != nil {
		return *actionGroup, fmt.Errorf("error building URL path for action group: %+v", err)
	}

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return *actionGroup, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return *actionGroup, err
	}
	body = res.Body

	var actionGroupDefinition map[string]ActionGroupBody

	if err := json.Unmarshal(body, &actionGroupDefinition); err != nil {
		return *actionGroup, fmt.Errorf("error unmarshalling action group body: %+v: %+v", err, body)
	}

	*actionGroup = actionGroupDefinition[actionGroupID]

	return *actionGroup, err
}

func resourceOpensearchPutOpenDistroActionGroup(d *schema.ResourceData, m interface{}) (*ActionGroupResponse, error) {
	response := new(ActionGroupResponse)

	actionGroupDefinition := ActionGroupBody{
		AllowedActions: expandStringList(d.Get("allowed_actions").(*schema.Set).List()),
		Type:           d.Get("type").(string),
		Description:    d.Get("description").(string),
	}

	actionGroupJSON, err := json.Marshal(actionGroupDefinition)
	if err != nil {
		return response, fmt.Errorf("body Error : %s", actionGroupJSON)
	}

	path, err := uritemplates.Expand("/_plugins/_security/api/actiongroups/{name}", map[string]string{
		"name": d.Get("action_group_name").(string),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for action group: %+v", err)
	}

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "PUT",
		Path:   path,
		Body:   string(actionGroupJSON),
		// see https://github.com/opendistro-for-
		// elasticsearch/security/issues/1095, this should return a 409, but
		// retry on the 500 as well. We can't parse the message to only retry on
		// the conflict exception because the client doesn't directly
		// expose the error response body
		RetryStatusCodes: []int{http.StatusConflict, http.StatusInternalServerError},
		Retrier: elastic7.NewBackoffRetrier(
			elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
		),
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling action group body: %+v: %+v", err, body)
	}

	return response, nil
}

type ActionGroupResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type ActionGroupBody struct {
	AllowedActions []string `json:"allowed_actions"`
	Type           string   `json:"type,omitempty"`
	Description    string   `json:"description,omitempty"`
	Reserved       bool     `json:"reserved,omitempty"`
	Hidden         bool     `json:"hidden,omitempty"`
	Static         bool     `json:"static,omitempty"`
}
