package provider

// NOTE: This resource uses raw HTTP API calls instead of the opensearch-go/v4 SDK
// because the SDK's IngestGetResp struct is missing the 'version' field that the
// OpenSearch API returns. This causes data loss during JSON unmarshaling, which
// triggers non-empty plan errors after terraform refresh.
//
// TODO: When the opensearch-go/v4 SDK is updated to include the 'version' field
// in the IngestGetResp struct (or provides a way to access raw response), migrate
// this resource back to using the SDK's typed API methods for consistency.
//
// Related: SDK_UPGRADE_PLAN.md - Phase 3B - Known Issues

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOpensearchIngestPipeline() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch ingest pipeline resource.",
		Create:      resourceOpensearchIngestPipelineCreate,
		Read:        resourceOpensearchIngestPipelineRead,
		Update:      resourceOpensearchIngestPipelineUpdate,
		Delete:      resourceOpensearchIngestPipelineDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the ingest pipeline",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"body": {
				Description:      "The JSON body of the ingest pipeline",
				Type:             schema.TypeString,
				DiffSuppressFunc: diffSuppressIngestPipeline,
				Required:         true,
				ValidateFunc:     validation.StringIsJSON,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchIngestPipelineCreate(d *schema.ResourceData, meta interface{}) error {

	err := resourceOpensearchPutIngestPipeline(d, meta)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceOpensearchIngestPipelineRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var result string
	var err error
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	result, err = getIngestPipeline(client, id)
	if err != nil {
		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", d.Id())
	ds.set("body", result)
	return ds.err
}

func getIngestPipeline(client *OpenSearchClient, id string) (string, error) {
	path := fmt.Sprintf("/_ingest/pipeline/%s", id)

	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return "", fmt.Errorf("error building GET request: %w", err)
	}

	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return "", fmt.Errorf("error getting ingest pipeline: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("ingest pipeline not found: %s", id)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting ingest pipeline: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response to extract just this pipeline's definition
	var pipelines map[string]json.RawMessage
	if err := json.Unmarshal(body, &pipelines); err != nil {
		return "", fmt.Errorf("error unmarshalling pipelines response: %w", err)
	}

	if pipeline, ok := pipelines[id]; ok {
		return string(pipeline), nil
	}

	return "", fmt.Errorf("ingest pipeline %s not found in response", id)
}

func resourceOpensearchIngestPipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceOpensearchPutIngestPipeline(d, meta)
}

func resourceOpensearchIngestPipelineDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/_ingest/pipeline/%s", id)

	req, err := http.NewRequest("DELETE", client.config.rawUrl+path, nil)
	if err != nil {
		return fmt.Errorf("error building DELETE request: %w", err)
	}

	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return fmt.Errorf("error deleting ingest pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		d.SetId("")
		return nil
	}

	return fmt.Errorf("error deleting ingest pipeline: received status code %d", resp.StatusCode)
}

func resourceOpensearchPutIngestPipeline(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	body := d.Get("body").(string)

	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/_ingest/pipeline/%s", name)

	req, err := http.NewRequest("PUT", client.config.rawUrl+path, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("error building PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return fmt.Errorf("error creating ingest pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error creating ingest pipeline: received status code %d", resp.StatusCode)
}
