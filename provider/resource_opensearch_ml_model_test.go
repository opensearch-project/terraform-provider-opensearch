package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testModelURL              = "https://artifacts.opensearch.org/models/ml-models/huggingface/sentence-transformers/paraphrase-MiniLM-L3-v2/1.0.2/torch_script/sentence-transformers_paraphrase-MiniLM-L3-v2-1.0.2-torch_script.zip"
	testModelContentHashValue = "843d3246ed04369593f1c54f2be92dc9878d60d5610b89617c585619e9f162d0"
	mlModelTestAWSAccessKey   = "AKIAIOSFODNN7EXAMPLE"
	mlModelTestAWSSecretKey   = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

// Map of resource IDS to verify resources are not recreated on updates.
// Note: This global map is shared across tests.
var savedMLModelIDs = make(map[string]string)

func testSaveOpensearchMLModelID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		savedMLModelIDs[name] = resourceID
		return nil
	}
}

func testCheckOpensearchMLModelIDEqualsSavedID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		currentID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}

		savedID, ok := savedMLModelIDs[name]
		if !ok {
			return fmt.Errorf("resource with name '%s' not found in savedMLModelIDs", name)
		}

		if savedID != currentID {
			return fmt.Errorf("ID of ML Model with name %s does not match original resource. ID of original resource: %s. ID of resource after update: %s", name, savedID, currentID)
		}

		return nil
	}
}

func TestAccOpensearchMLModel_OSProvidedPretrained(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelConfig_OSProvidedPretrained_WithOptional(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.os_pretrained_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model.os_pretrained_with_optional", "name", "amazon/neural-sparse/opensearch-neural-sparse-encoding-doc-v2-mini"),
					resource.TestCheckResourceAttr("opensearch_ml_model.os_pretrained_with_optional", "description", "ML Model for OpenSearch-provided pretrained sparse encoding model"),
					resource.TestCheckResourceAttr("opensearch_ml_model.os_pretrained_with_optional", "version", "1.0.0"),
					resource.TestCheckResourceAttr("opensearch_ml_model.os_pretrained_with_optional", "model_format", "TORCH_SCRIPT"),
					resource.TestCheckResourceAttr("opensearch_ml_model.os_pretrained_with_optional", "function_name", "SPARSE_ENCODING"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.os_pretrained_with_optional", "model_group_id"),
					resource.TestCheckResourceAttr("opensearch_ml_model.os_pretrained_with_optional", "is_enabled", "true"),
					testCheckOpensearchMLModelIsEnabledInAPI("opensearch_ml_model.os_pretrained_with_optional", true),
				),
			},
			{
				ResourceName:      "opensearch_ml_model.os_pretrained_with_optional",
				ImportState:       true,
				ImportStateVerify: true,
				// These fields are not returned by the OpenSearch API for OS-provided pretrained models:
				//   - description: OpenSearch does not persist description for OS-provided models in the model metadata,
				//     though it does return it for custom and third-party models
				//   - version
				//   - deploy_after_registering: attribute is only used by Provider, not by the OpenSearch API
				//   - wait_for_predict: attribute is only used by Provider, not by the OpenSearch API
				//   - predict_probe_body: attribute is only used by Provider, not by the OpenSearch API
				//   - is_enabled: register ignores the field; the API only echoes it once an Update (PUT) has
				//     persisted it. The provider always issues that PUT after register, but SPARSE_ENCODING
				//     models reject it (403), so here the field is never persisted nor returned by GET and is
				//     left to whatever the import writes (the zero value). Other model types persist it, which
				//     is why their import steps do not ignore is_enabled.
				ImportStateVerifyIgnore: []string{"description", "version", "deploy_after_registering", "wait_for_predict", "predict_probe_body", "is_enabled"},
			},
			// Note: OS-provided pretrained sparse encoding models cannot be updated via OpenSearch API
			// The API returns 403: "The function category SPARSE_ENCODING is not supported at this time"
		},
	})
}

func TestAccOpensearchMLModel_Custom(t *testing.T) {
	defer func() {
		delete(savedMLModelIDs, "opensearch_ml_model.custom_with_optional")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelConfig_Custom_WithOptional(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.custom_with_optional"),
					testSaveOpensearchMLModelID("opensearch_ml_model.custom_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "name", "custom_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "description", "ML Model for custom model with optional attributes"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.custom_with_optional", "model_group_id"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "version", "1.0.2"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_format", "TORCH_SCRIPT"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "function_name", "TEXT_EMBEDDING"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "url", testModelURL),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_content_hash_value", testModelContentHashValue),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.model_type", "bert"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.embedding_dimension", "384"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.framework_type", "SENTENCE_TRANSFORMERS"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.pooling_mode", "MEAN"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.all_config", "{\"_name_or_path\": \"sentence-transformers/paraphrase-MiniLM-L3-v2\", \"architectures\": [\"BertModel\"], \"attention_probs_dropout_prob\": 0.1, \"classifier_dropout\": null, \"gradient_checkpointing\": false, \"hidden_act\": \"gelu\", \"hidden_dropout_prob\": 0.1, \"hidden_size\": 384, \"initializer_range\": 0.02, \"intermediate_size\": 1536, \"layer_norm_eps\": 1e-12, \"max_position_embeddings\": 512, \"model_type\": \"bert\", \"num_attention_heads\": 12, \"num_hidden_layers\": 3, \"pad_token_id\": 0, \"position_embedding_type\": \"absolute\", \"torch_dtype\": \"float32\", \"transformers_version\": \"4.49.0\", \"type_vocab_size\": 2, \"use_cache\": true, \"vocab_size\": 30522}"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.normalize_result", "false"),
					// Note: 'additional_config' is not returned by the API and cannot be tested here.
					// Unlike root-level fields like 'url' and 'version' which persist in state when not explicitly set by Read,
					// 'additional_config' is nested in 'model_config' which IS explicitly set by Read.
					// When d.Set("model_config", ...) is called, it overwrites the entire block with only API-returned fields,
					// clearing 'additional_config' from state. This is correct behavior - state should reflect API reality.
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "rate_limiter.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "rate_limiter.0.limit", "4"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "rate_limiter.0.unit", "SECONDS"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "interface.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "interface.0.input", "{\"properties\":{\"parameters\":{\"properties\":{\"messages\":{\"type\":\"string\",\"description\":\"Test description field\"}}}}}"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "interface.0.output", ""),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "is_enabled", "false"),
					testCheckOpensearchMLModelIsEnabledInAPI("opensearch_ml_model.custom_with_optional", false),
				),
			},
			{
				Config: testAccOpensearchMLModelConfig_Custom_WithOptional_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.custom_with_optional"),
					testCheckOpensearchMLModelIDEqualsSavedID("opensearch_ml_model.custom_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "name", "custom_with_optional_updated_name"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "description", "Updated description for ML Model for custom model with optional attributes"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.custom_with_optional", "model_group_id"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "version", "1.0.2"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_format", "TORCH_SCRIPT"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "function_name", "TEXT_EMBEDDING"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "url", testModelURL),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_content_hash_value", testModelContentHashValue),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.model_type", "bert"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.embedding_dimension", "384"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.framework_type", "SENTENCE_TRANSFORMERS"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.pooling_mode", "MEAN"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.all_config", "{\"updated\": true}"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "model_config.0.normalize_result", "true"),
					// Note: 'additional_config' is not returned by the API and cannot be tested here.
					// Unlike root-level fields like 'url' and 'version' which persist in state when not explicitly set by Read,
					// 'additional_config' is nested in 'model_config' which IS explicitly set by Read.
					// When d.Set("model_config", ...) is called, it overwrites the entire block with only API-returned fields,
					// clearing 'additional_config' from state. This is correct behavior - state should reflect API reality.
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "rate_limiter.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "rate_limiter.0.limit", "8"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "rate_limiter.0.unit", "MINUTES"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "interface.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "interface.0.input", "{\"properties\":{\"parameters\":{\"properties\":{\"text\":{\"type\":\"string\",\"description\":\"Updated text field\"}}}}}"),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "interface.0.output", ""),
					resource.TestCheckResourceAttr("opensearch_ml_model.custom_with_optional", "is_enabled", "true"),
					testCheckOpensearchMLModelIsEnabledInAPI("opensearch_ml_model.custom_with_optional", true),
				),
			},
			{
				ResourceName:      "opensearch_ml_model.custom_with_optional",
				ImportState:       true,
				ImportStateVerify: true,
				// These fields are not returned by the OpenSearch API for custom models:
				//   - url
				//   - version
				//   - model_config.0.additional_config (nested field)
				//   - deploy_after_registering: attribute is only used by Provider, not by the OpenSearch API
				//   - wait_for_predict: attribute is only used by Provider, not by the OpenSearch API
				//   - predict_probe_body: attribute is only used by Provider, not by the OpenSearch API
				ImportStateVerifyIgnore: []string{"url", "version", "model_config.0.additional_config.#", "model_config.0.additional_config.0.%", "model_config.0.additional_config.0.space_type", "deploy_after_registering", "wait_for_predict", "predict_probe_body"},
			},
		},
	})
}

func TestAccOpensearchMLModel_ThirdParty(t *testing.T) {
	defer func() {
		delete(savedMLModelIDs, "opensearch_ml_model.thirdparty_with_optional")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelConfig_ThirdParty_Minimal(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.thirdparty_minimal"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_minimal", "name", "thirdparty_minimal"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_minimal", "function_name", "REMOTE"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.thirdparty_minimal", "model_group_id"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.thirdparty_minimal", "connector_id"),
					// When is_enabled is omitted it must default to true (schema Default).
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_minimal", "is_enabled", "true"),
					testCheckOpensearchMLModelIsEnabledInAPI("opensearch_ml_model.thirdparty_minimal", true),
				),
			},
			{
				Config: testAccOpensearchMLModelConfig_ThirdParty_WithOptional(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.thirdparty_with_optional"),
					testSaveOpensearchMLModelID("opensearch_ml_model.thirdparty_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "name", "thirdparty_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "description", "ML Model for third-party model with optional attributes"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.thirdparty_with_optional", "model_group_id"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.thirdparty_with_optional", "connector_id"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "function_name", "REMOTE"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "rate_limiter.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "rate_limiter.0.limit", "4"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "rate_limiter.0.unit", "SECONDS"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "interface.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "interface.0.input", "{\"properties\":{\"parameters\":{\"properties\":{\"messages\":{\"type\":\"string\",\"description\":\"Test description field\"}}}}}"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "guardrails.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "guardrails.0.type", "local_regex"),
					// Note: 'input_guardrail' is not returned by the API and cannot be tested here.
					// Similar to 'additional_config' in model_config, this nested field is cleared when d.Set("guardrails", ...)
					// overwrites the entire block with only API-returned fields. This is correct behavior - state should reflect API reality.
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "is_enabled", "false"),
					testCheckOpensearchMLModelIsEnabledInAPI("opensearch_ml_model.thirdparty_with_optional", false),
				),
			},
			{
				Config: testAccOpensearchMLModelConfig_ThirdParty_WithOptional_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.thirdparty_with_optional"),
					testCheckOpensearchMLModelIDEqualsSavedID("opensearch_ml_model.thirdparty_with_optional"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "name", "thirdparty_with_optional_updated"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "description", "Updated description of ML Model for third-party model with optional attributes"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.thirdparty_with_optional", "model_group_id"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.thirdparty_with_optional", "connector_id"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "function_name", "REMOTE"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "rate_limiter.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "rate_limiter.0.limit", "10"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "rate_limiter.0.unit", "MINUTES"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "interface.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "interface.0.input", "{\"properties\":{\"parameters\":{\"properties\":{\"prompt\":{\"type\":\"string\",\"description\":\"Updated input field\"}}}}}"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "guardrails.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "guardrails.0.type", "local_regex"),
					// Note: 'input_guardrail' is not returned by the API and cannot be tested here.
					// Similar to 'additional_config' in model_config, this nested field is cleared when d.Set("guardrails", ...)
					// overwrites the entire block with only API-returned fields. This is correct behavior - state should reflect API reality.
					resource.TestCheckResourceAttr("opensearch_ml_model.thirdparty_with_optional", "is_enabled", "true"),
					testCheckOpensearchMLModelIsEnabledInAPI("opensearch_ml_model.thirdparty_with_optional", true),
				),
			},
			{
				ResourceName:      "opensearch_ml_model.thirdparty_with_optional",
				ImportState:       true,
				ImportStateVerify: true,
				// These fields are not returned by the OpenSearch API:
				//   - deploy_after_registering: attribute is only used by Provider, not by the OpenSearch API
				//   - wait_for_predict: attribute is only used by Provider, not by the OpenSearch API
				//   - predict_probe_body: attribute is only used by Provider, not by the OpenSearch API
				ImportStateVerifyIgnore: []string{"deploy_after_registering", "wait_for_predict", "predict_probe_body"},
			},
		},
	})
}

func TestAccOpensearchMLModel_GuardrailsModelType(t *testing.T) {
	defer func() {
		delete(savedMLModelIDs, "opensearch_ml_model.guardrails_model_type")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelConfig_ThirdParty_WithGuardrailsTypeModel(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.guardrails_type_model"),
					testSaveOpensearchMLModelID("opensearch_ml_model.guardrails_type_model"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "name", "guardrails_type_model"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "description", "ML Model with model-based guardrails"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.guardrails_type_model", "model_group_id"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.guardrails_type_model", "connector_id"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "function_name", "REMOTE"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "guardrails.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "guardrails.0.type", "model"),
					// Note: 'model_id' and 'response_validation_regex' are not returned by the API and cannot be tested here.
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "is_enabled", "false"),
				),
			},
			{
				Config: testAccOpensearchMLModelConfig_ThirdParty_WithGuardrailsTypeModel_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.guardrails_type_model"),
					testCheckOpensearchMLModelIDEqualsSavedID("opensearch_ml_model.guardrails_type_model"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "name", "guardrails_type_model_updated"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "description", "Updated ML Model with model-based guardrails"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.guardrails_type_model", "model_group_id"),
					resource.TestCheckResourceAttrSet("opensearch_ml_model.guardrails_type_model", "connector_id"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "function_name", "REMOTE"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "guardrails.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "guardrails.0.type", "model"),
					// Note: 'model_id' and 'response_validation_regex' are not returned by the API and cannot be tested here.
					resource.TestCheckResourceAttr("opensearch_ml_model.guardrails_type_model", "is_enabled", "true"),
				),
			},
			{
				ResourceName:      "opensearch_ml_model.guardrails_type_model",
				ImportState:       true,
				ImportStateVerify: true,
				// These fields are not returned by the OpenSearch API:
				//   - deploy_after_registering: attribute is only used by Provider, not by the OpenSearch API
				//   - wait_for_predict: attribute is only used by Provider, not by the OpenSearch API
				//   - predict_probe_body: attribute is only used by Provider, not by the OpenSearch API
				ImportStateVerifyIgnore: []string{"deploy_after_registering", "wait_for_predict", "predict_probe_body"},
			},
		},
	})
}

func TestAccOpensearchMLModel_CustomModel_MissingRequiredFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchMLModelConfig_CustomModel_MissingURL(),
				ExpectError: regexp.MustCompile("This model is not in the pre-trained model list"),
			},
			{
				Config:      testAccOpensearchMLModelConfig_CustomModel_MissingModelConfig(),
				ExpectError: regexp.MustCompile("model config is null"),
			},
		},
	})
}

func TestAccOpensearchMLModel_ThirdPartyModel_MissingConnectorID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchMLModelConfig_ThirdPartyModel_MissingConnectorID(),
				ExpectError: regexp.MustCompile("You must provide connector content when creating a remote model without connector id"),
			},
		},
	})
}

func TestAccOpensearchMLModel_DeployAfterRegistering(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLModelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLModelConfig_WithDeploy(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.with_deploy"),
					resource.TestCheckResourceAttr("opensearch_ml_model.with_deploy", "name", "test_model_with_deploy"),
					resource.TestCheckResourceAttr("opensearch_ml_model.with_deploy", "deploy_after_registering", "true"),
					testCheckOpensearchMLModelDeploymentStatus("opensearch_ml_model.with_deploy", "DEPLOYED"),
				),
			},
			{
				Config: testAccOpensearchMLModelConfig_WithoutDeploy(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLModelExists("opensearch_ml_model.without_deploy"),
					resource.TestCheckResourceAttr("opensearch_ml_model.without_deploy", "name", "test_model_without_deploy"),
					resource.TestCheckResourceAttr("opensearch_ml_model.without_deploy", "deploy_after_registering", "false"),
					testCheckOpensearchMLModelDeploymentStatus("opensearch_ml_model.without_deploy", "REGISTERED"),
				),
			},
		},
	})
}

func testCheckOpensearchMLModelExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		_, apiError := getMLModelFromAPI(context.Background(), conf, resourceID)
		return apiError
	}
}

func testCheckOpensearchMLModelIsEnabledInAPI(name string, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		model, apiError := getMLModelFromAPI(context.Background(), conf, resourceID)
		if apiError != nil {
			return apiError
		}

		actual := true // absent means enabled (cluster default)
		if v, ok := model["is_enabled"].(bool); ok {
			actual = v
		}
		if actual != expected {
			return fmt.Errorf("model %s: expected is_enabled=%t in cluster, got %t (raw: %v)", name, expected, actual, model["is_enabled"])
		}
		return nil
	}
}

func testCheckOpensearchMLModelDeploymentStatus(name string, expectedStatus string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		model, apiError := getMLModelFromAPI(context.Background(), conf, resourceID)
		if apiError != nil {
			return apiError
		}

		modelState, ok := model["model_state"].(string)
		if !ok {
			return fmt.Errorf("model_state not found in API response")
		}

		if modelState != expectedStatus {
			return fmt.Errorf("expected model_state to be '%s', got '%s'", expectedStatus, modelState)
		}

		return nil
	}
}

func testCheckOpensearchMLModelDestroy(s *terraform.State) error {
	resourceType := "opensearch_ml_model"
	for _, rs := range s.RootModule().Resources {
		if rs.Type != resourceType {
			continue
		}

		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		_, apiError := getMLModelFromAPI(context.Background(), conf, rs.Primary.ID)

		// If there's no error, the resource still exists
		if apiError == nil {
			return fmt.Errorf("resource of type %s with ID '%s' still exists", resourceType, rs.Primary.ID)
		}

		// Any error other than a 404 is unexpected
		if !strings.Contains(apiError.Error(), "404") {
			return fmt.Errorf("unexpected error verifying resource of type %s with ID '%s' was destroyed: %v", resourceType, rs.Primary.ID, apiError)
		}
	}

	return nil
}

func testAccOpensearchMLModelConfig_OSProvidedPretrained_WithOptional() string {
	return `
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_os_pretrained_with_optional"
}

resource "opensearch_ml_model" "os_pretrained_with_optional" {
  name           = "amazon/neural-sparse/opensearch-neural-sparse-encoding-doc-v2-mini"
  description    = "ML Model for OpenSearch-provided pretrained sparse encoding model"
  version        = "1.0.0"
  model_format   = "TORCH_SCRIPT"
  function_name  = "SPARSE_ENCODING"
  model_group_id = opensearch_ml_model_group.dependency.id
  is_enabled     = true

  deploy_after_registering = false
}
`
}

func testAccOpensearchMLModelConfig_Custom_WithOptional() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_custom_with_optional"
}

resource "opensearch_ml_model" "custom_with_optional" {
  name                     = "custom_with_optional"
  description              = "ML Model for custom model with optional attributes"
  model_group_id           = opensearch_ml_model_group.dependency.id
  version                  = "1.0.2"
  model_format             = "TORCH_SCRIPT"
  function_name            = "TEXT_EMBEDDING"
  url                      = "%s"
  model_content_hash_value = "%s"
  model_config {
    model_type          = "bert"
    embedding_dimension = 384
    framework_type      = "sentence_transformers"
    pooling_mode        = "MEAN"
    all_config          = "{\"_name_or_path\": \"sentence-transformers/paraphrase-MiniLM-L3-v2\", \"architectures\": [\"BertModel\"], \"attention_probs_dropout_prob\": 0.1, \"classifier_dropout\": null, \"gradient_checkpointing\": false, \"hidden_act\": \"gelu\", \"hidden_dropout_prob\": 0.1, \"hidden_size\": 384, \"initializer_range\": 0.02, \"intermediate_size\": 1536, \"layer_norm_eps\": 1e-12, \"max_position_embeddings\": 512, \"model_type\": \"bert\", \"num_attention_heads\": 12, \"num_hidden_layers\": 3, \"pad_token_id\": 0, \"position_embedding_type\": \"absolute\", \"torch_dtype\": \"float32\", \"transformers_version\": \"4.49.0\", \"type_vocab_size\": 2, \"use_cache\": true, \"vocab_size\": 30522}"
    additional_config {
      space_type = "cosinesimil"
    }
    normalize_result = false
  }
  rate_limiter {
    limit = 4
    unit  = "SECONDS" 
  }
  interface {
    input = "{\"properties\":{\"parameters\":{\"properties\":{\"messages\":{\"type\":\"string\",\"description\":\"Test description field\"}}}}}"
  }
  is_enabled = false

  deploy_after_registering = false
}
`, testModelURL, testModelContentHashValue)
}

func testAccOpensearchMLModelConfig_Custom_WithOptional_Updated() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_custom_with_optional"
}

resource "opensearch_ml_model" "custom_with_optional" {
  name                     = "custom_with_optional_updated_name"
  description              = "Updated description for ML Model for custom model with optional attributes"
  model_group_id           = opensearch_ml_model_group.dependency.id
  version                  = "1.0.2"
  model_format             = "TORCH_SCRIPT"
  function_name            = "TEXT_EMBEDDING"
  url                      = "%s"
  model_content_hash_value = "%s"
  model_config {
    model_type          = "bert"
    embedding_dimension = 384
    framework_type      = "sentence_transformers"
    pooling_mode        = "MEAN"
    all_config          = "{\"updated\": true}"
    additional_config {
      space_type = "l2"
    }
    normalize_result = true
  }
  rate_limiter {
    limit = 8
    unit  = "MINUTES"
  }
  interface {
    input = "{\"properties\":{\"parameters\":{\"properties\":{\"text\":{\"type\":\"string\",\"description\":\"Updated text field\"}}}}}"
  }
  is_enabled = true

  deploy_after_registering = false
}
`, testModelURL, testModelContentHashValue)
}

func testAccOpensearchMLConnectorConfig_Dependency() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "dependency" {
  name        = "dependency_connector_for_tests"
  description = "Shared ML Connector for tests"
  version     = "1"
  protocol    = "aws_sigv4"
  credential = {
	access_key = "%s"
    secret_key = "%s"
  }
  parameters = {
    region       = "eu-west-3"
    service_name = "bedrock"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    headers     = {
      content-type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
}
`, mlModelTestAWSAccessKey, mlModelTestAWSSecretKey)
}

func testAccOpensearchMLModelConfig_ThirdParty_Minimal() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_thirdparty_minimal"
}

%s

resource "opensearch_ml_model" "thirdparty_minimal" {
  name           = "thirdparty_minimal"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.dependency.id
  connector_id   = opensearch_ml_connector.dependency.id

  deploy_after_registering = false
}
`, testAccOpensearchMLConnectorConfig_Dependency())
}

func testAccOpensearchMLModelConfig_ThirdParty_WithOptional() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_thirdparty_with_optional"
}

%s

resource "opensearch_ml_model" "thirdparty_with_optional" {
  name           = "thirdparty_with_optional"
  description    = "ML Model for third-party model with optional attributes"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.dependency.id
  connector_id   = opensearch_ml_connector.dependency.id
  rate_limiter {
    limit = 4
    unit  = "SECONDS" 
  }
  interface {
    input = "{\"properties\":{\"parameters\":{\"properties\":{\"messages\":{\"type\":\"string\",\"description\":\"Test description field\"}}}}}"
  }
  guardrails {
    type = "local_regex"
    input_guardrail {
      stop_words {
        index_name    = "stop_words_inputs"
        source_fields = ["message"]
      }
      regex = [".*kill.*"]
    }
    output_guardrail {
      regex = [".*forbidden.*"]
    }
  }
  is_enabled = false

  deploy_after_registering = false
}
`, testAccOpensearchMLConnectorConfig_Dependency())
}

func testAccOpensearchMLModelConfig_ThirdParty_WithOptional_Updated() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_thirdparty_with_optional"
}

%s

resource "opensearch_ml_model" "thirdparty_with_optional" {
  name           = "thirdparty_with_optional_updated"
  description    = "Updated description of ML Model for third-party model with optional attributes"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.dependency.id
  connector_id   = opensearch_ml_connector.dependency.id
  rate_limiter {
    limit = 10
    unit  = "MINUTES" 
  }
  interface {
    input = "{\"properties\":{\"parameters\":{\"properties\":{\"prompt\":{\"type\":\"string\",\"description\":\"Updated input field\"}}}}}"
  }
  guardrails {
    type = "local_regex"
    input_guardrail {
      stop_words {
        index_name    = "stop_words_inputs_updated"
        source_fields = ["message_updated"]
      }
      regex = [".*kill_updated.*"]
    }
    output_guardrail {
      stop_words {
        index_name    = "stop_words_outputs"
        source_fields = ["response"]
      }
      regex = [".*sensitive.*"]
    }
  }
  is_enabled = true

  deploy_after_registering = false
}
`, testAccOpensearchMLConnectorConfig_Dependency())
}

func testAccOpensearchMLModelConfig_ThirdParty_WithGuardrailsTypeModel() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_guardrails_type_model"
}

%s

resource "opensearch_ml_model" "guardrails_type_model" {
  name           = "guardrails_type_model"
  description    = "ML Model with model-based guardrails"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.dependency.id
  connector_id   = opensearch_ml_connector.dependency.id
  guardrails {
    type                       = "model"
    model_id                   = "test_guardrail_model_id"
    response_validation_regex  = "^[regex]$"
  }
  is_enabled = false

  deploy_after_registering = false
}
`, testAccOpensearchMLConnectorConfig_Dependency())
}

func testAccOpensearchMLModelConfig_ThirdParty_WithGuardrailsTypeModel_Updated() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "dependency_group_for_guardrails_type_model"
}

%s

resource "opensearch_ml_model" "guardrails_type_model" {
  name           = "guardrails_type_model_updated"
  description    = "Updated ML Model with model-based guardrails"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.dependency.id
  connector_id   = opensearch_ml_connector.dependency.id
  guardrails {
    type                       = "model"
    model_id                   = "updated_guardrail_model_id"
    response_validation_regex  = "^[updated regex]$"
  }
  is_enabled = true

  deploy_after_registering = false
}
`, testAccOpensearchMLConnectorConfig_Dependency())
}

func testAccOpensearchMLModelConfig_CustomModel_MissingURL() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "test" {
  name = "test_group_missing_url"
}

resource "opensearch_ml_model" "invalid" {
  name                     = "test_missing_url"
  function_name            = "TEXT_EMBEDDING"
  model_group_id           = opensearch_ml_model_group.test.id
  version                  = "1.0.2"
  model_format             = "TORCH_SCRIPT"
  model_content_hash_value = "%s"
  model_config {
    model_type          = "bert"
    embedding_dimension = 384
    framework_type      = "sentence_transformers"
  }
}
`, testModelContentHashValue)
}

func testAccOpensearchMLModelConfig_CustomModel_MissingModelConfig() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "test" {
  name = "test_group_missing_model_config"
}

resource "opensearch_ml_model" "invalid" {
  name                     = "test_missing_model_config"
  function_name            = "TEXT_EMBEDDING"
  model_group_id           = opensearch_ml_model_group.test.id
  version                  = "1.0.2"
  model_format             = "TORCH_SCRIPT"
  url                      = "%s"
  model_content_hash_value = "%s"
}
`, testModelURL, testModelContentHashValue)
}

func testAccOpensearchMLModelConfig_ThirdPartyModel_MissingConnectorID() string {
	return `
resource "opensearch_ml_model_group" "dependency" {
  name = "test_group_missing_connector"
}

resource "opensearch_ml_model" "invalid" {
  name           = "test_missing_connector_id"
  function_name  = "REMOTE"
  model_group_id = opensearch_ml_model_group.dependency.id
}
`
}

func testAccOpensearchMLModelConfig_WithDeploy() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "test_group_deploy"
}

%s

resource "opensearch_ml_model" "with_deploy" {
  name                     = "test_model_with_deploy"
  function_name            = "REMOTE"
  model_group_id           = opensearch_ml_model_group.dependency.id
  connector_id             = opensearch_ml_connector.dependency.id
  deploy_after_registering = true
  wait_for_predict         = false
}
`, testAccOpensearchMLConnectorConfig_Dependency())
}

func testAccOpensearchMLModelConfig_WithoutDeploy() string {
	return fmt.Sprintf(`
resource "opensearch_ml_model_group" "dependency" {
  name = "test_group_no_deploy"
}

%s

resource "opensearch_ml_model" "without_deploy" {
  name                     = "test_model_without_deploy"
  function_name            = "REMOTE"
  model_group_id           = opensearch_ml_model_group.dependency.id
  connector_id             = opensearch_ml_connector.dependency.id
  deploy_after_registering = false
}
`, testAccOpensearchMLConnectorConfig_Dependency())
}
