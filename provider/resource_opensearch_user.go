package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
		if isNotFound(err) {
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

	username := d.Get("username").(string)
	path := fmt.Sprintf("/_plugins/_security/api/internalusers/%s", username)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	// Build request
	req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
	if err != nil {
		return fmt.Errorf("error building DELETE request: %w", err)
	}

	// Execute request with retry logic
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		resp, err = client.Client.Client.Perform(req)
		if err == nil && resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting user: received status code %d", resp.StatusCode)
}

func resourceOpensearchGetOpenDistroUser(userID string, m interface{}) (UserBody, error) {
	var err error
	user := new(UserBody)
	path := fmt.Sprintf("/_plugins/_security/api/internalusers/%s", userID)

	log.Printf("The resourceOpensearchGetOpenDistroUser path is %s", path)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return *user, err
	}

	// Build request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return *user, fmt.Errorf("error building GET request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return *user, fmt.Errorf("error getting user: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("The resourceOpensearchGetOpenDistroUser res StatusCode is %d", resp.StatusCode)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return *user, fmt.Errorf("error reading response body: %w", err)
	}

	log.Printf("The resourceOpensearchGetOpenDistroUser res is %s", string(body))

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return *user, fmt.Errorf("user not found: %s", userID)
	}

	if resp.StatusCode != http.StatusOK {
		return *user, fmt.Errorf("error getting user: received status code %d, body: %s", resp.StatusCode, string(body))
	}

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

	username := d.Get("username").(string)
	path := fmt.Sprintf("/_plugins/_security/api/internalusers/%s", username)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] put opendistro user: %+v", userDefinition)

	// Execute request with retry logic
	// see https://github.com/opendistro-for-elasticsearch/security/issues/1095
	var resp *http.Response
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter to avoid thundering herd
			backoff := time.Duration(attempt*200) * time.Millisecond
			time.Sleep(backoff)
		}

		// Build request (must recreate for each attempt as body can't be reused)
		req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(string(userJSON)))
		if err != nil {
			return response, fmt.Errorf("error building PUT request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Client.Client.Perform(req)
		if err == nil {
			// Check if we should retry - 409 CONFLICT is expected for concurrent security config updates
			if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusInternalServerError {
				break
			}
			// For 409/500, close body and retry
			resp.Body.Close()
		}
	}

	if err != nil {
		log.Printf("[INFO] error creating user: %v", err)
		return response, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error creating user: received status code %d, body: %s", resp.StatusCode, string(body))
	}

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
