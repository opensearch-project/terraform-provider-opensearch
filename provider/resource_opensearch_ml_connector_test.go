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
	mlConnectorTestAWSAccessKey = "AKIAIOSFODNN7EXAMPLE"
	mlConnectorTestAWSSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

// Map of resource IDS to verify resources are not recreated on updates.
// Note: This global map is shared across tests.
var savedMLConnectorIDs = make(map[string]string)

func testSaveOpensearchMLConnectorID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		savedMLConnectorIDs[name] = resourceID
		return nil
	}
}

func testCheckOpensearchMLConnectorIDEqualsSavedID(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		currentID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}

		savedID, ok := savedMLConnectorIDs[name]
		if !ok {
			return fmt.Errorf("resource with name '%s' not found in savedMLConnectorIDs", name)
		}

		if savedID != currentID {
			return fmt.Errorf("ID of ML Connector with name %s does not match original resource. ID of original resource: %s. ID of resource after update: %s", name, savedID, currentID)
		}

		return nil
	}
}

func TestAccOpensearchMLConnector_Minimal_AWSSigV4(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLConnectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLConnectorConfig_Minimal_AWSSigV4(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLConnectorExists("opensearch_ml_connector.minimal_aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "name", "minimal_aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "description", "Minimal ML Connector with AWS SigV4 authentication"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "version", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "protocol", "aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "credential.access_key", mlConnectorTestAWSAccessKey),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "credential.secret_key", mlConnectorTestAWSSecretKey),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "parameters.region", "eu-west-3"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "parameters.service_name", "bedrock"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "actions.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "actions.0.action_type", "predict"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "actions.0.method", "POST"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "actions.0.url", "https://bedrock-runtime.${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "actions.0.headers.Content-Type", "application/json"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "actions.0.headers.x-amz-content-sha256", "required"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_aws_sigv4", "actions.0.request_body", "{ \"inputText\": \"${parameters.inputText}\" }"),
				),
			},
			{
				ResourceName:            "opensearch_ml_connector.minimal_aws_sigv4",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential"},
			},
		},
	})
}

func TestAccOpensearchMLConnector_Minimal_HTTP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLConnectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLConnectorConfig_Minimal_HTTP(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLConnectorExists("opensearch_ml_connector.minimal_http"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "name", "minimal_http"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "description", "Minimal ML Connector with HTTP authentication"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "version", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "protocol", "http"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "credential.openAIKey", "sk-test123456789"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "parameters.endpoint", "api.openai.com"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "parameters.max_tokens", "7"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "parameters.temperature", "0"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "parameters.model", "gpt-4.1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "actions.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "actions.0.action_type", "predict"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "actions.0.method", "POST"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "actions.0.url", "https://bedrock-runtime.${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "actions.0.headers.Content-Type", "application/json"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "actions.0.headers.x-amz-content-sha256", "required"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.minimal_http", "actions.0.request_body", "{ \"inputText\": \"${parameters.inputText}\" }"),
				),
			},
			{
				ResourceName:            "opensearch_ml_connector.minimal_http",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential"},
			},
		},
	})
}

func TestAccOpensearchMLConnector_WithPrePostProcessing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLConnectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLConnectorConfig_WithPrePostProcessing(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLConnectorExists("opensearch_ml_connector.with_pre_post_processing"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "name", "with_pre_post_processing"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "description", "ML Connector with pre and post processing functions"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "version", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "protocol", "aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.action_type", "predict"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.method", "POST"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.url", "https://bedrock-runtime.${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.headers.Content-Type", "application/json"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.headers.x-amz-content-sha256", "required"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.request_body", "{ \"inputText\": \"${parameters.inputText}\" }"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.pre_process_function", "connector.pre_process.bedrock.embedding"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_pre_post_processing", "actions.0.post_process_function", "connector.post_process.bedrock.embedding"),
				),
			},
			{
				ResourceName:            "opensearch_ml_connector.with_pre_post_processing",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential"},
			},
		},
	})
}

func TestAccOpensearchMLConnector_WithAccessModeAndBackendRoles(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLConnectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLConnectorConfig_WithAccessModeAndBackendRoles(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLConnectorExists("opensearch_ml_connector.with_access_mode_and_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_backend_roles", "name", "with_access_mode_and_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_backend_roles", "description", "ML Connector with 'access_mode' and 'backend_roles'"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_backend_roles", "version", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_backend_roles", "protocol", "aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_backend_roles", "access_mode", "restricted"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_backend_roles", "backend_roles.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_backend_roles", "backend_roles.0", "ml_full_access"),
				),
			},
			{
				ResourceName:            "opensearch_ml_connector.with_access_mode_and_backend_roles",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential"},
			},
		},
	})
}

func TestAccOpensearchMLConnector_WithAccessModeAndAddAllBackendRoles(t *testing.T) {
	t.Skip("Skipping test: OpenSearch API restriction - Admin users cannot add all backend roles to a connector")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLConnectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLConnectorConfig_WithAccessModeAndAddAllBackendRoles(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLConnectorExists("opensearch_ml_connector.with_access_mode_and_add_all_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_add_all_backend_roles", "name", "with_access_mode_and_add_all_backend_roles"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_add_all_backend_roles", "description", "ML Connector with 'access_mode' and 'add_all_backend_roles'"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_add_all_backend_roles", "version", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_add_all_backend_roles", "protocol", "aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_add_all_backend_roles", "access_mode", "restricted"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_access_mode_and_add_all_backend_roles", "add_all_backend_roles", "true"),
				),
			},
			{
				ResourceName:            "opensearch_ml_connector.with_access_mode_and_add_all_backend_roles",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential"},
			},
		},
	})
}

func TestAccOpensearchMLConnector_WithConflictingAccessModeAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccOpensearchMLConnectorConfig_WithConflictingAccessModeAttributes(),
				ExpectError: regexp.MustCompile("conflicts with"),
			},
		},
	})
}

func TestAccOpensearchMLConnector_WithClientConfig(t *testing.T) {
	defer func() {
		delete(savedMLConnectorIDs, "opensearch_ml_connector.with_client_config")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchMLConnectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchMLConnectorConfig_WithClientConfig(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLConnectorExists("opensearch_ml_connector.with_client_config"),
					testSaveOpensearchMLConnectorID("opensearch_ml_connector.with_client_config"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "name", "with_client_config"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "description", "ML Connector with all 'client_config' attributes"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "version", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "protocol", "aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.max_connection", "10"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.connection_timeout", "10"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.read_timeout", "30"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.max_retry_times", "2"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.retry_backoff_policy", "constant"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.retry_backoff_millis", "500"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.retry_timeout_seconds", "30"),
				),
			},
			{
				Config: testAccOpensearchMLConnectorConfig_WithClientConfig_Updated(),
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchMLConnectorExists("opensearch_ml_connector.with_client_config"),
					testCheckOpensearchMLConnectorIDEqualsSavedID("opensearch_ml_connector.with_client_config"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "name", "with_client_config_updated"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "description", "Updated ML Connector with all 'client_config' attributes"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "version", "2"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "protocol", "aws_sigv4"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.#", "1"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.max_connection", "20"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.connection_timeout", "20"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.read_timeout", "60000"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.max_retry_times", "3"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.retry_backoff_policy", "exponential_equal_jitter"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.retry_backoff_millis", "600"),
					resource.TestCheckResourceAttr("opensearch_ml_connector.with_client_config", "client_config.0.retry_timeout_seconds", "60"),
				),
			},
			{
				ResourceName:            "opensearch_ml_connector.with_client_config",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credential"},
			},
		},
	})
}

func testCheckOpensearchMLConnectorExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID, stateError := getResourceIDFromState(s, name)
		if stateError != nil {
			return stateError
		}
		conf := testAccOpendistroProvider.Meta().(*ProviderConf)

		_, apiError := getMLConnectorFromAPI(context.Background(), conf, resourceID)
		return apiError
	}
}

func testCheckOpensearchMLConnectorDestroy(s *terraform.State) error {
	resourceType := "opensearch_ml_connector"
	for _, rs := range s.RootModule().Resources {
		if rs.Type != resourceType {
			continue
		}

		conf := testAccOpendistroProvider.Meta().(*ProviderConf)
		_, apiError := getMLConnectorFromAPI(context.Background(), conf, rs.Primary.ID)

		if apiError == nil {
			return fmt.Errorf("resource of type %s with ID '%s' still exists", resourceType, rs.Primary.ID)
		}
		if !strings.Contains(apiError.Error(), "404") {
			return fmt.Errorf("unexpected error verifying resource of type %s with ID '%s' was destroyed: %v", resourceType, rs.Primary.ID, apiError)
		}
	}
	return nil
}

func testAccOpensearchMLConnectorConfig_Minimal_AWSSigV4() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "minimal_aws_sigv4" {
  name        = "minimal_aws_sigv4"
  description = "Minimal ML Connector with AWS SigV4 authentication"
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
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
}
`, mlConnectorTestAWSAccessKey, mlConnectorTestAWSSecretKey)
}

func testAccOpensearchMLConnectorConfig_Minimal_HTTP() string {
	return `
resource "opensearch_ml_connector" "minimal_http" {
  name        = "minimal_http"
  description = "Minimal ML Connector with HTTP authentication"
  version     = "1"
  protocol    = "http"
  credential = {
    openAIKey = "sk-test123456789"
  }
  parameters = {
    endpoint    = "api.openai.com"
    max_tokens  = 7
    temperature = 0
    model       = "gpt-4.1"
  }
  actions {
    action_type = "predict"
    method      = "POST"
    url         = "https://bedrock-runtime.$${parameters.region}.amazonaws.com/model/amazon.titan-embed-text-v2:0/invoke"
    headers = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
}
`
}

func testAccOpensearchMLConnectorConfig_WithPrePostProcessing() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "with_pre_post_processing" {
  name        = "with_pre_post_processing"
  description = "ML Connector with pre and post processing functions"
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
    headers = {
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
    pre_process_function  = "connector.pre_process.bedrock.embedding"
    post_process_function = "connector.post_process.bedrock.embedding"
  }
}
`, mlConnectorTestAWSAccessKey, mlConnectorTestAWSSecretKey)
}

func testAccOpensearchMLConnectorConfig_WithAccessModeAndBackendRoles() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "with_access_mode_and_backend_roles" {
  name        = "with_access_mode_and_backend_roles"
  description = "ML Connector with 'access_mode' and 'backend_roles'"
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
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
  access_mode   = "restricted"
  backend_roles = ["ml_full_access"]
}
`, mlConnectorTestAWSAccessKey, mlConnectorTestAWSSecretKey)
}

func testAccOpensearchMLConnectorConfig_WithAccessModeAndAddAllBackendRoles() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "with_access_mode_and_add_all_backend_roles" {
  name        = "with_access_mode_and_add_all_backend_roles"
  description = "ML Connector with 'access_mode' and 'add_all_backend_roles'"
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
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
  access_mode   = "restricted"
  add_all_backend_roles = true
}
`, mlConnectorTestAWSAccessKey, mlConnectorTestAWSSecretKey)
}

func testAccOpensearchMLConnectorConfig_WithConflictingAccessModeAttributes() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "conflicting" {
  name        = "conflicting"
  description = "ML Connector with conflicting 'backend_roles' and 'add_all_backend_roles' attributes"
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
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
  access_mode           = "restricted"
  backend_roles         = ["ml_full_access"]
  add_all_backend_roles = true
}
`, mlConnectorTestAWSAccessKey, mlConnectorTestAWSSecretKey)
}

func testAccOpensearchMLConnectorConfig_WithClientConfig() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "with_client_config" {
  name        = "with_client_config"
  description = "ML Connector with all 'client_config' attributes"
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
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
  client_config {
    max_connection        = 10
    connection_timeout    = 10
    read_timeout          = 30
    max_retry_times       = 2
    retry_backoff_policy  = "constant"
    retry_backoff_millis  = 500
    retry_timeout_seconds = 30
  }
}
`, mlConnectorTestAWSAccessKey, mlConnectorTestAWSSecretKey)
}

func testAccOpensearchMLConnectorConfig_WithClientConfig_Updated() string {
	return fmt.Sprintf(`
resource "opensearch_ml_connector" "with_client_config" {
  name        = "with_client_config_updated"
  description = "Updated ML Connector with all 'client_config' attributes"
  version     = "2"
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
      Content-Type         = "application/json"
      x-amz-content-sha256 = "required"
    }
    request_body = "{ \"inputText\": \"$${parameters.inputText}\" }"
  }
  client_config {
    max_connection        = 20
    connection_timeout    = 20
    read_timeout          = 60000
    max_retry_times       = 3
    retry_backoff_policy  = "exponential_equal_jitter"
    retry_backoff_millis  = 600
    retry_timeout_seconds = 60
  }
}
`, mlConnectorTestAWSAccessKey, mlConnectorTestAWSSecretKey)
}
