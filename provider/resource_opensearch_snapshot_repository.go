package provider

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

func resourceOpensearchSnapshotRepository() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch snapshot repository resource.",
		Create:      resourceOpensearchSnapshotRepositoryCreate,
		Read:        resourceOpensearchSnapshotRepositoryRead,
		Update:      resourceOpensearchSnapshotRepositoryUpdate,
		Delete:      resourceOpensearchSnapshotRepositoryDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The name of the repository.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"type": {
				Description: "The name of the repository backend (required plugins must be installed).",
				Type:        schema.TypeString,
				Required:    true,
			},
			"settings": {
				Description: "The settings map applicable for the backend, see official documentation for plugins.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOpensearchSnapshotRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceOpensearchSnapshotRepositoryUpdate(d, meta)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceOpensearchSnapshotRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var repositoryType string
	var settings map[string]interface{}
	var err error
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	repositoryType, settings, err = getSnapshotRepository(client, id)

	if err != nil {
		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("name", id)
	ds.set("type", repositoryType)
	ds.set("settings", settings)
	return ds.err
}

func getSnapshotRepository(client *OpenSearchClient, id string) (string, map[string]interface{}, error) {
	res, err := client.Client.Snapshot.Repository.Get(context.TODO(), &opensearchapi.SnapshotRepositoryGetReq{
		Repos: []string{id},
	})
	if err != nil {
		return "", make(map[string]interface{}), err
	}

	if repo, ok := res.Repos[id]; ok {
		// Convert settings from map[string]string to map[string]interface{}
		settings := make(map[string]interface{})
		for k, v := range repo.Settings {
			settings[k] = v
		}
		return repo.Type, settings, nil
	}

	return "", make(map[string]interface{}), nil
}

func resourceOpensearchSnapshotRepositoryUpdate(d *schema.ResourceData, meta interface{}) error {
	repositoryType := d.Get("type").(string)
	name := d.Get("name").(string)

	var settings map[string]interface{}

	if v, ok := d.GetOk("settings"); ok {
		settings = v.(map[string]interface{})
	}

	var err error
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	err = createSnapshotRepository(client, name, repositoryType, settings)

	return err
}

func createSnapshotRepository(client *OpenSearchClient, name string, repositoryType string, settings map[string]interface{}) error {
	// Build the request body
	repoBody := map[string]interface{}{
		"type":     repositoryType,
		"settings": settings,
	}

	bodyJSON, err := json.Marshal(repoBody)
	if err != nil {
		return err
	}

	_, err = client.Client.Snapshot.Repository.Create(context.TODO(), opensearchapi.SnapshotRepositoryCreateReq{
		Repo: name,
		Body: bytes.NewReader(bodyJSON),
	})
	return err
}

func resourceOpensearchSnapshotRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	id := d.Id()

	var err error
	client, err := getOpenSearchClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	_, err = client.Client.Snapshot.Repository.Delete(context.TODO(), opensearchapi.SnapshotRepositoryDeleteReq{
		Repos: []string{id},
	})

	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
