package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/olivere/elastic/uritemplates"

	elastic7 "github.com/olivere/elastic/v7"
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

	if elastic7.IsNotFound(err) {
		log.Printf("[WARN] Channel configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	configId := res.ChannelConfigurationInfos[0]["config_id"].(string)

	log.Printf("[DEBUG] Config ID from API: %v", configId)

	if err := d.Set("config_id", configId); err != nil {
		return err
	  }

	if _, ok := openDistroChannelConfigurationSchema["body"]; ok {
		json, err := json.Marshal(res.ChannelConfigurationInfos[0])
		if err != nil {
			return err
		}
		if err := d.Set("body", json); err != nil {
			return err
		  }
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
	var err error

	path, err := uritemplates.Expand("/_plugins/_notifications/configs/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return fmt.Errorf("error building URL path for channel configuration: %+v", err)
	}

	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "DELETE",
		Path:   path,
	})

	return err
}

func resourceOpensearchOpenDistroGetChannelConfiguration(channelConfigurationID string, m interface{}) (*channelConfigurationReadResponse, error) {
	var err error
	response := new(channelConfigurationReadResponse)

	path, err := uritemplates.Expand("/_plugins/_notifications/configs/{id}", map[string]string{
		"id": channelConfigurationID,
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for channel configuration: %+v", err)
	}

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling channel configuration body: %+v: %+v", err, body)
	}

	normalizeChannelConfiguration(response.ChannelConfigurationInfos[0])

	return response, err
}

func resourceOpensearchOpenDistroPostChannelConfiguration(d *schema.ResourceData, m interface{}) (*channelConfigurationCreationResponse, error) {
	channelConfigurationJSON := d.Get("body").(string)

	var err error
	response := new(channelConfigurationCreationResponse)

	path := "/_plugins/_notifications/configs/"

	var body json.RawMessage
	osClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}
	var res *elastic7.Response
	res, err = osClient.PerformRequest(context.TODO(), elastic7.PerformRequestOptions{
		Method: "POST",
		Path:   path,
		Body:   channelConfigurationJSON,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling channel configuration body: %+v: %+v", err, body)
	}
	return response, nil
}

func resourceOpensearchOpenDistroPutChannelConfiguration(d *schema.ResourceData, m interface{}) (*channelConfigurationCreationResponse, error) {
	channelConfigurationJSON := d.Get("body").(string)

	var err error
	response := new(channelConfigurationCreationResponse)

	path, err := uritemplates.Expand("/_plugins/_notifications/configs/{id}", map[string]string{
		"id": d.Id(),
	})
	if err != nil {
		return response, fmt.Errorf("error building URL path for channel configuration: %+v", err)
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
		Body:   channelConfigurationJSON,
	})
	if err != nil {
		return response, err
	}
	body = res.Body

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
