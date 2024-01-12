package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testAccOpensearchIndex = `
resource "opensearch_index" "test" {
  name               = "terraform-test"
  number_of_shards   = 1
  number_of_replicas = 1
}
`

	testOpensearchIndexImport = `
resource "opensearch_index" "test1import" {
  name               = "terraform-test1import"
  number_of_shards   = 1
  number_of_replicas = 1
  mappings = jsonencode(
    {
      "properties" : {
        "name" : {
          "type" : "text"
        }
      }
    }
  )
}
`

	testAccOpensearchIndexUpdate1 = `
resource "opensearch_index" "test" {
  name                                  = "terraform-test"
  number_of_shards                      = 1
  number_of_replicas                    = 2
  number_of_routing_shards              = 1
  routing_partition_size                = 1
  refresh_interval                      = "10s"
  max_result_window                     = 1000
  max_refresh_listeners                 = 10
  blocks_read_only                      = false
  blocks_read                           = false
  blocks_write                          = false
  blocks_metadata                       = false
  search_slowlog_threshold_query_warn   = "5s"
  search_slowlog_threshold_fetch_warn   = "5s"
  search_slowlog_level                  = "warn"
  indexing_slowlog_threshold_index_warn = "5s"
  indexing_slowlog_level                = "warn"
}
`
	testAccOpensearchIndexDefaultShardsReplicas = `
resource "opensearch_index" "testdefaultshardsreplicas" {
  name = "terraform-testdefaultshardsreplicas"
}
`
	testAccOpensearchIndexAnalysis = `
resource "opensearch_index" "test" {
  name               = "terraform-test"
  number_of_shards   = 1
  number_of_replicas = 1
  analysis_analyzer = jsonencode({
    default = {
      filter = [
        "lowercase",
        "asciifolding",
      ]
      tokenizer = "standard"
    }
    full_text_search = {
      filter = [
        "lowercase",
        "asciifolding",
      ]
      tokenizer = "custom_ngram_tokenizer"
    }
  })
  analysis_tokenizer = jsonencode({
    custom_ngram_tokenizer = {
      max_gram = "4"
      min_gram = "3"
      type     = "ngram"
    }
  })
  analysis_filter = jsonencode({
    my_filter_shingle = {
      type             = "shingle"
      max_shingle_size = 2
      min_shingle_size = 2
      output_unigrams  = false
    }
  })
  analysis_char_filter = jsonencode({
    my_char_filter_apostrophe = {
      type     = "mapping"
      mappings = ["'=>"]
    }
  })
  analysis_normalizer = jsonencode({
    my_normalizer = {
      type   = "custom"
      filter = ["lowercase", "asciifolding"]
    }
  })
}
`
	testAccOpensearchIndexInvalid = `
resource "opensearch_index" "test" {
  name               = "terraform-test"
  number_of_shards   = 1
  number_of_replicas = 1
  mappings           = <<EOF
{
  "people": {
    "_all": {
      "enabled": "true"
    },
    "properties": {
      "email": {
        "type": "text"
      }
    }
  }
}
EOF
}
`
	testAccOpensearchMappingWithDocType = `
resource "opensearch_index" "test_doctype" {
  name               = "terraform-test"
  number_of_replicas = "1"
  mappings = jsonencode(
    {
      "properties" : {
        "name" : {
          "type" : "text"
        }
      }
    }
  )
}
`
	testAccOpensearchIndexUpdateForceDestroy = `
resource "opensearch_index" "test" {
  name               = "terraform-test"
  number_of_shards   = 1
  number_of_replicas = 2
  force_destroy      = true
}
`
	testAccOpensearchIndexDateMath = `
resource "opensearch_index" "test_date_math" {
  name = "<terraform-test-{now/y{yyyy}}-000001>"
  # name = "%3Ctest-%7Bnow%2Fy%7Byyyy%7D%7D-000001%3E"
  number_of_shards   = 1
  number_of_replicas = 1
}
`

	testAccOpensearchIndexWithSimilarityConfig = `
resource "opensearch_index" "test_similarity_config" {
  name               = "terraform-test-update-similarity-module"
  number_of_shards   = 1
  number_of_replicas = 1
  index_similarity_default = jsonencode({
    "type" : "BM25",
    "b" : 0.25,
    "k1" : 1.2
  })
}
`

	testAccOpensearchIndexWithKNNConfig = `
resource "opensearch_index" "test_knn_config" {
  name               = "terraform-test-update-knn-module"
  number_of_shards   = 1
  number_of_replicas = 1
  index_knn          = true
}
`

	testAccOpensearchIndexWithKNNAlgoParamEfSearchConfig = `
resource "opensearch_index" "test_knn_algo_param_ef_search_config" {
  name                           = "terraform-test-update-knn-algo-param-ef-search-module"
  number_of_shards               = 1
  number_of_replicas             = 1
  index_knn                      = true
  index_knn_algo_param_ef_search = 600
}
`

	testAccOpensearchIndexRolloverAliasOpendistro = `
resource opensearch_ism_policy "test" {
  policy_id = "test"
  body      = <<EOF
{
  "policy": {
    "description": "Terraform Test",
    "default_state": "hot",
    "states": [
      {
        "name": "hot",
        "actions": [
          {
            "retry": {
              "count": 3,
              "backoff": "exponential",
              "delay": "1m"
            },
            "rollover": {
              "min_index_age": "30d",
              "min_primary_shard_size": "50gb"
            }
          }
        ],
        "transitions": []
      }
    ]
  }
}
  EOF
}

resource "opensearch_index_template" "test" {
  name = "terraform-test"
  body = <<EOF
  {
	"index_patterns": ["terraform-test-*"],
	"template": {
	"settings": {
		"plugins": {
		  "index_state_management": {
			"policy_id": "${opensearch_ism_policy.test.policy_id}",
			"rollover_alias": "terraform-test"
		  }
	  }
	}
  }
  }
  EOF
}

resource "opensearch_index" "test" {
  name               = "terraform-test-000001"
  number_of_shards   = 1
  number_of_replicas = 1
  aliases = jsonencode({
    "terraform-test" = {
      "is_write_index" = true
    }
  })

  depends_on = [opensearch_index_template.test]
}
`
)

func TestAccOpensearchIndex(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndex,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.test"),
				),
			},
			{
				Config: testAccOpensearchIndexUpdate1,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexUpdated("opensearch_index.test"),
				),
			},
			{
				Config: testAccOpensearchIndexUpdateForceDestroy,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexUpdated("opensearch_index.test"),
				),
			},
		},
	})
}

func TestAccOpensearchIndexDefaultShardsReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndexDefaultShardsReplicas,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.testdefaultshardsreplicas"),
				),
			},
			{
				Config: testAccOpensearchIndexDefaultShardsReplicas,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.testdefaultshardsreplicas"),
				),
			},
		},
	})
}

func TestAccOpensearchIndex_rolloverAliasOpendistro(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool = true

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("Opendistro index policies only supported on ES 7")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: checkOpensearchIndexRolloverAliasDestroy(testAccOpendistroProvider, "terraform-test"),
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndexRolloverAliasOpendistro,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexRolloverAliasExists(testAccOpendistroProvider, "terraform-test"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				ResourceName:      "opensearch_index.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"aliases",       // not handled by this provider
					"force_destroy", // not returned from the API
				},
				ImportStateCheck:   checkOpensearchIndexRolloverAliasState("terraform-test"),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func checkOpensearchIndexRolloverAliasExists(provider *schema.Provider, alias string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meta := provider.Meta()

		var count int
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		r, err := osClient.CatAliases().Alias(alias).Do(context.TODO())
		if err != nil {
			return err
		}
		count = len(r)

		if count == 0 {
			return fmt.Errorf("rollover alias %q not found", alias)
		}

		return nil
	}
}

func checkOpensearchIndexRolloverAliasState(alias string) resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected 1 state: %+v", s)
		}
		rs := s[0]
		if rs.Attributes["rollover_alias"] != alias {
			return fmt.Errorf("expected rollover alias %q got %q", alias, rs.Attributes["rollover_alias"])
		}

		return nil
	}
}

func checkOpensearchIndexRolloverAliasDestroy(provider *schema.Provider, alias string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meta := provider.Meta()

		var count int
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		r, err := osClient.CatAliases().Alias(alias).Do(context.TODO())
		if err != nil {
			return err
		}
		count = len(r)

		if count > 0 {
			return fmt.Errorf("rollover alias %q still exists", alias)
		}

		return nil
	}
}

func TestAccOpensearchIndexAnalysis(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndexAnalysis,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.test"),
				),
			},
		},
	})
}

func TestAccOpensearchIndex_handleInvalid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchIndexInvalid,
				ExpectError: regexp.MustCompile("Failed to parse mapping"),
			},
		},
	})
}

func TestAccOpensearchIndex_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testOpensearchIndexImport,
			},
			{
				ResourceName:      "opensearch_index.test1import",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					// not returned from the API
					"force_destroy",
				},
			},
		},
	})
}

func TestAccOpensearchIndex_dateMath(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndexDateMath,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.test_date_math"),
				),
			},
		},
	})
}

func TestAccOpensearchIndex_similarityConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndexWithSimilarityConfig,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.test_similarity_config"),
				),
			},
		},
	})
}

func TestAccOpensearchIndex_knnConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndexWithKNNConfig,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.test_knn_config"),
				),
			},
		},
	})
}

func TestAccOpensearchIndex_knnAlgoParamEFSearchConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchIndexWithKNNAlgoParamEfSearchConfig,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.test_knn_algo_param_ef_search_config"),
				),
			},
		},
	})
}

func TestAccOpensearchIndex_doctype(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var config string = testAccOpensearchMappingWithDocType

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkOpensearchIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					checkOpensearchIndexExists("opensearch_index.test_doctype"),
				),
			},
		},
	})
}

func checkOpensearchIndexExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("index ID not set")
		}

		meta := testAccProvider.Meta()

		var err error
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = osClient.IndexGetSettings(rs.Primary.ID).Do(context.TODO())

		return err
	}
}

func checkOpensearchIndexUpdated(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("index ID not set")
		}

		meta := testAccProvider.Meta()
		var settings map[string]interface{}

		var err error
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		resp, err := osClient.IndexGetSettings(rs.Primary.ID).Do(context.TODO())
		if err != nil {
			return err
		}
		settings = resp[rs.Primary.ID].Settings["index"].(map[string]interface{})

		r, ok := settings["number_of_replicas"]
		if ok {
			if ir := r.(string); ir != "2" {
				return fmt.Errorf("expected 2 got %s", ir)
			}
			return nil
		}

		return errors.New("field not found")
	}
}

func checkOpensearchIndexDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_index" {
			continue
		}

		meta := testAccProvider.Meta()

		var err error
		osClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		_, err = osClient.IndexGetSettings(rs.Primary.ID).Do(context.TODO())

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("index %q still exists", rs.Primary.ID)
	}

	return nil
}
