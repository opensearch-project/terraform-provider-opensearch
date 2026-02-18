package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var openDistroChannelConfigurationSchema = map[string]*schema.Schema{
	"body": {
		Description:      "The channel configuration document",
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: diffSuppressChannelConfiguration,
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
		ValidateFunc: validation.StringIsJSON,
	},
}

func resourceOpenSearchChannelConfiguration() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch channel configuration. Please refer to the OpenSearch channel configuration documentation for details.",
		Create:      resourceOpensearchOpenDistroChannelConfigurationCreate,
		Read:        resourceOpensearchOpenDistroChannelConfigurationRead,
		Update:      resourceOpensearchOpenDistroChannelConfigurationUpdate,
		Delete:      resourceOpensearchOpenDistroChannelConfigurationDelete,
		Schema:      openDistroChannelConfigurationSchema,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchOpenDistroChannelConfigurationCreate(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchOpenDistroPostChannelConfiguration(d, m)

	if err != nil {
		log.Printf("[INFO] Failed to put channel configuration: %+v", err)
		return err
	}

	d.SetId(res.ID)
	log.Printf("[INFO] Object ID: %s", d.Id())

	return resourceOpensearchOpenDistroChannelConfigurationRead(d, m)
}

func resourceOpensearchOpenDistroChannelConfigurationRead(d *schema.ResourceData, m interface{}) error {
	res, err := resourceOpensearchOpenDistroGetChannelConfiguration(d.Id(), m)

	if isNotFound(err) {
		log.Printf("[WARN] Channel configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	d.SetId(res.ChannelConfigurationInfos[0]["config_id"].(string))

	channelConfigurationJson, err := json.Marshal(res.ChannelConfigurationInfos[0])
	if err != nil {
		return err
	}
	channelConfigurationJsonNormalized, err := structure.NormalizeJsonString(string(channelConfigurationJson))
	if err != nil {
		return err
	}
	err = d.Set("body", channelConfigurationJsonNormalized)
	return err
}

func resourceOpensearchOpenDistroChannelConfigurationUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := resourceOpensearchOpenDistroPutChannelConfiguration(d, m)

	if err != nil {
		return err
	}

	return resourceOpensearchOpenDistroChannelConfigurationRead(d, m)
}

func resourceOpensearchOpenDistroChannelConfigurationDelete(d *schema.ResourceData, m interface{}) error {
	path := fmt.Sprintf("/_plugins/_notifications/configs/%s", d.Id())

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	// Build request
	req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
	if err != nil {
		return fmt.Errorf("error building DELETE request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return fmt.Errorf("error deleting channel configuration: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting channel configuration: received status code %d", resp.StatusCode)
}

func resourceOpensearchOpenDistroGetChannelConfiguration(channelConfigurationID string, m interface{}) (*channelConfigurationReadResponse, error) {
	response := new(channelConfigurationReadResponse)

	path := fmt.Sprintf("/_plugins/_notifications/configs/%s", channelConfigurationID)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return response, fmt.Errorf("error building GET request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error getting channel configuration: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return response, fmt.Errorf("channel configuration not found: %s", channelConfigurationID)
	}

	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("error getting channel configuration: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling channel configuration body: %+v: %+v", err, body)
	}

	normalizeChannelConfiguration(response.ChannelConfigurationInfos[0])

	return response, nil
}

func resourceOpensearchOpenDistroPostChannelConfiguration(d *schema.ResourceData, m interface{}) (*channelConfigurationCreationResponse, error) {
	channelConfigurationJSON := d.Get("body").(string)

	response := new(channelConfigurationCreationResponse)
	path := "/_plugins/_notifications/configs/"

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("POST", client.config.rawUrl+path, strings.NewReader(channelConfigurationJSON))
	if err != nil {
		return response, fmt.Errorf("error building POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error posting channel configuration: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error posting channel configuration: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling channel configuration body: %+v: %+v", err, body)
	}
	return response, nil
}

func resourceOpensearchOpenDistroPutChannelConfiguration(d *schema.ResourceData, m interface{}) (*channelConfigurationCreationResponse, error) {
	channelConfigurationJSON := d.Get("body").(string)

	response := new(channelConfigurationCreationResponse)
	path := fmt.Sprintf("/_plugins/_notifications/configs/%s", d.Id())

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(channelConfigurationJSON))
	if err != nil {
		return response, fmt.Errorf("error building PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return response, fmt.Errorf("error putting channel configuration: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error putting channel configuration: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling channel configuration body: %+v: %+v", err, body)
	}

	return response, nil
}

type channelConfigurationCreationResponse struct {
	ID string `json:"config_id"`
}

type channelConfigurationReadResponse struct {
	ChannelConfigurationInfos []map[string]interface{} `json:"config_list"`
}
