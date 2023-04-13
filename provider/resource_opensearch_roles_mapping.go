package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
)

var openDistroRolesMappingSchema = map[string]*schema.Schema{
	"role_name": {
		Description: "The name of the security role.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"backend_roles": {
		Description: "A list of backend roles.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"hosts": {
		Description: "A list of host names.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"users": {
		Description: "A list of users.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"description": {
		Description: "Description of the role mapping.",
		Type:        schema.TypeString,
		Optional:    true,
	},
	"and_backend_roles": {
		Description: "A list of backend roles.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
}

func resourceOpenSearchRolesMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceOpensearchOpenDistroRolesMappingCreate,
		Read:   resourceOpensearchOpenDistroRolesMappingRead,
		Update: resourceOpensearchOpenDistroRolesMappingUpdate,
		Delete: resourceOpensearchOpenDistroRolesMappingDelete,
		Schema: openDistroRolesMappingSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Provides an OpenSearch security role mapping. Please refer to the OpenSearch Access Control documentation for details.",
	}
}

func resourceOpensearchOpenDistroRolesMappingCreate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroRolesMapping(d, m); err != nil {
		log.Printf("[INFO] Failed to put role mapping: %+v", err)
		return err
	}

	name := d.Get("role_name").(string)
	d.SetId(name)
	return resourceOpensearchOpenDistroRolesMappingRead(d, m)
}

func resourceOpensearchOpenDistroRolesMappingRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetOpenDistroRolesMapping(d.Id(), m)

	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] OpenDistroRolesMapping (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if err := d.Set("role_name", d.Id()); err != nil {
		return fmt.Errorf("error setting role_name: %s", err)
	}
	if err := d.Set("backend_roles", res.BackendRoles); err != nil {
		return fmt.Errorf("error setting backend_roles: %s", err)
	}
	if err := d.Set("hosts", res.Hosts); err != nil {
		return fmt.Errorf("error setting hosts: %s", err)
	}
	if err := d.Set("users", res.Users); err != nil {
		return fmt.Errorf("error setting users: %s", err)
	}
	if err := d.Set("description", res.Description); err != nil {
		return fmt.Errorf("error setting description: %s", err)
	}
	if err := d.Set("and_backend_roles", res.AndBackendRoles); err != nil {
		return fmt.Errorf("error setting and_backend_roles: %s", err)
	}

	return nil
}

func resourceOpensearchOpenDistroRolesMappingUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroRolesMapping(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroRolesMappingRead(d, m)
}

func resourceOpensearchOpenDistroRolesMappingDelete(d *schema.ResourceData, m interface{}) error {
	path, err := uritemplates.Expand("/_opendistro/_security/api/rolesmapping/{name}", map[string]string{
		"name": d.Get("role_name").(string),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for role mapping: %+v", err)
	}

	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		_, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
			Method:           "DELETE",
			Path:             path,
			RetryStatusCodes: []int{http.StatusConflict, http.StatusInternalServerError},
			Retrier: elastic7.NewBackoffRetrier(
				elastic7.NewExponentialBackoff(100*time.Millisecond, 30*time.Second),
			),
		})
	default:
		err = errors.New("role mapping resource not implemented prior to v7")
	}

	return err
}

func resourceOpensearchGetOpenDistroRolesMapping(roleID string, m interface{}) (RolesMapping, error) {
	var err error
	var roleMapping = new(RolesMapping)

	path, err := uritemplates.Expand("/_opendistro/_security/api/rolesmapping/{name}", map[string]string{
		"name": roleID,
	})

	if err != nil {
		return *roleMapping, fmt.Errorf("error building URL path for role mapping: %+v", err)
	}
	var body json.RawMessage
	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return *roleMapping, err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		var res *elastic7.Response
		res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
			Method: "GET",
			Path:   path,
		})
		if err != nil {
			return *roleMapping, err
		}
		body = res.Body
	default:
		err = errors.New("role mapping resource not implemented prior to v7")
	}

	if err != nil {
		return *roleMapping, err
	}
	var rolesMappingDefinition map[string]RolesMapping

	if err := json.Unmarshal(body, &rolesMappingDefinition); err != nil {
		return *roleMapping, fmt.Errorf("error unmarshalling role mapping body: %+v: %+v", err, body)
	}

	*roleMapping = rolesMappingDefinition[roleID]

	return *roleMapping, err
}

func resourceOpensearchPutOpenDistroRolesMapping(d *schema.ResourceData, m interface{}) (*RoleMappingResponse, error) {
	var err error
	response := new(RoleMappingResponse)

	rolesMappingDefinition := RolesMapping{
		BackendRoles:    expandStringList(d.Get("backend_roles").(*schema.Set).List()),
		Hosts:           expandStringList(d.Get("hosts").(*schema.Set).List()),
		Users:           expandStringList(d.Get("users").(*schema.Set).List()),
		Description:     d.Get("description").(string),
		AndBackendRoles: expandStringList(d.Get("and_backend_roles").(*schema.Set).List()),
	}
	roleJSON, err := json.Marshal(rolesMappingDefinition)

	if err != nil {
		return response, fmt.Errorf("Body Error : %s", roleJSON)
	}

	path, err := uritemplates.Expand("/_opendistro/_security/api/rolesmapping/{name}", map[string]string{
		"name": d.Get("role_name").(string),
	})

	if err != nil {
		return response, fmt.Errorf("error building URL path for role mapping: %+v", err)
	}

	var body json.RawMessage
	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		var res *elastic7.Response
		res, err = client.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
			Method: "PUT",
			Path:   path,
			Body:   string(roleJSON),
			// see https://github.com/opendistro-for-
			// elasticsearch/security/issues/1095, this should return a 409, but
			// retry on the 500 as well. We can't parse the message to only retry on
			// the conlict exception becaues the client doesn't directly
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
	default:
		return response, errors.New("role mapping resource not implemented prior to v7")
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling role mapping body: %+v: %+v", err, body)
	}

	return response, nil
}

type RoleMappingResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type RolesMapping struct {
	BackendRoles    []string `json:"backend_roles"`
	Hosts           []string `json:"hosts"`
	Users           []string `json:"users"`
	Description     string   `json:"description"`
	AndBackendRoles []string `json:"and_backend_roles"`
}
