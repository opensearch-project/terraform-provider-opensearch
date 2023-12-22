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

var openDistroUserSchema = map[string]*schema.Schema{
	"username": {
		Description: "The name of the security user.",
		Type:        schema.TypeString,
		Required:    true,
	},
	"password": {
		Description:   "The plain text password for the user, cannot be specified with `password_hash`. Some implementations may enforce a password policy. Invalid passwords may cause a non-descriptive HTTP 400 Bad Request error. For AWS OpenSearch domains \"password must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one digit, and one special character\".",
		Type:          schema.TypeString,
		Optional:      true,
		Sensitive:     true,
		StateFunc:     hashSum,
		ConflictsWith: []string{"password_hash"},
	},
	"password_hash": {
		Description:   "The pre-hashed password for the user, cannot be specified with `password`.",
		Type:          schema.TypeString,
		Optional:      true,
		Sensitive:     true,
		StateFunc:     hashSum,
		ConflictsWith: []string{"password"},
	},
	"backend_roles": {
		Description: "A list of backend roles.",
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"attributes": {
		Description: "A map of arbitrary key value string pairs stored alongside of users.",
		Type:        schema.TypeMap,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	},
	"description": {
		Description: "Description of the user.",
		Type:        schema.TypeString,
		Optional:    true,
	},
}

func resourceOpenSearchUser() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch security user. Please refer to the OpenSearch Access Control documentation for details.",
		Create:      resourceOpensearchOpenDistroUserCreate,
		Read:        resourceOpensearchOpenDistroUserRead,
		Update:      resourceOpensearchOpenDistroUserUpdate,
		Delete:      resourceOpensearchOpenDistroUserDelete,
		Schema:      openDistroUserSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchOpenDistroUserCreate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchPutOpenDistroUser(d, m)

	if err != nil {
		return err
	}

	name := d.Get("username").(string)
	d.SetId(name)
	return resourceOpensearchOpenDistroUserRead(d, m)
}

func resourceOpensearchOpenDistroUserRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchGetOpenDistroUser(d.Id(), m)

	if err != nil {
		if elastic7.IsNotFound(err) {
			log.Printf("[WARN] OdfeUser (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("backend_roles", res.BackendRoles)
	ds.set("attributes", res.Attributes)
	ds.set("description", res.Description)
	return ds.err
}

func resourceOpensearchOpenDistroUserUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPutOpenDistroUser(d, m); err != nil {
		return err
	}

	return resourceOpensearchOpenDistroUserRead(d, m)
}

func resourceOpensearchOpenDistroUserDelete(d *schema.ResourceData, m interface{}) error {
	var err error

	path, err := uritemplates.Expand("/_plugins/_security/api/internalusers/{name}", map[string]string{
		"name": d.Get("username").(string),
	})
	if err != nil {
		return fmt.Errorf("Error building URL path for user: %+v", err)
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

func resourceOpensearchGetOpenDistroUser(userID string, m interface{}) (UserBody, error) {
	var err error
	user := new(UserBody)
	path, err := uritemplates.Expand("/_plugins/_security/api/internalusers/{name}", map[string]string{
		"name": userID,
	})
	log.Printf("The resourceOpensearchGetOpenDistroUser path is " + string(path))
	if err != nil {
		return *user, fmt.Errorf("Error building URL path for user: %+v", err)
	}

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return *user, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})
	log.Printf("The resourceOpensearchGetOpenDistroUser res is " + string(res.Body))
	log.Printf("The resourceOpensearchGetOpenDistroUser res StatusCode is " + fmt.Sprint(res.StatusCode))
	if err != nil {
		return *user, err
	}
	body = res.Body

	var userDefinition map[string]UserBody

	if err := json.Unmarshal(body, &userDefinition); err != nil {
		return *user, fmt.Errorf("Error unmarshalling user body: %+v: %+v", err, body)
	}

	*user = userDefinition[userID]

	return *user, err
}

func resourceOpensearchPutOpenDistroUser(d *schema.ResourceData, m interface{}) (*UserResponse, error) {
	response := new(UserResponse)

	userDefinition := UserBody{
		BackendRoles: d.Get("backend_roles").(*schema.Set).List(),
		Description:  d.Get("description").(string),
		Attributes:   d.Get("attributes").(map[string]interface{}),
	}

	if d.HasChange("password") {
		userDefinition.Password = d.Get("password").(string)
	}
	if d.HasChange("password_hash") {
		userDefinition.PasswordHash = d.Get("password_hash").(string)
	}

	userJSON, err := json.Marshal(userDefinition)
	if err != nil {
		return response, fmt.Errorf("Body Error : %s", userJSON)
	}

	path, err := uritemplates.Expand("/_plugins/_security/api/internalusers/{name}", map[string]string{
		"name": d.Get("username").(string),
	})
	if err != nil {
		return response, fmt.Errorf("Error building URL path for user: %+v", err)
	}

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	log.Printf("[INFO] put opendistro user: %+v", userDefinition)
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "PUT",
		Path:   path,
		Body:   string(userJSON),
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
		e, ok := err.(*elastic7.Error)
		if !ok {
			log.Printf("[INFO] expected error to be of type *elastic.Error")
		} else {
			log.Printf("[INFO] error creating user: %v %v %v", res, res.Body, e)
		}
		return response, err
	}

	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("Error unmarshalling user body: %+v: %+v", err, body)
	}

	return response, nil
}

// UserBody used by the odfe's API
type UserBody struct {
	BackendRoles []interface{}          `json:"backend_roles"`
	Attributes   map[string]interface{} `json:"attributes"`
	Description  string                 `json:"description"`
	Password     string                 `json:"password,omitempty"`
	PasswordHash string                 `json:"hash,omitempty"`
}

// UserResponse sent by the odfe's API
type UserResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}
