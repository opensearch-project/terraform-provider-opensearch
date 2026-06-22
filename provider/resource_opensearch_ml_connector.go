package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpensearchMLConnector() *schema.Resource {
	return &schema.Resource{
		Description:   "OpenSearch ML Connector resource",
		CreateContext: resourceOpensearchMLConnectorCreate,
		ReadContext:   resourceOpensearchMLConnectorRead,
		UpdateContext: resourceOpensearchMLConnectorUpdate,
		DeleteContext: resourceOpensearchMLConnectorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// ============================================
			// ===         Required attributes          ===
			// ============================================
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the ML Connector",
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Description of the ML Connector",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Version of the ML Connector",
			},
			"protocol": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The protocol for the connection. For AWS services, such as Amazon SageMaker and Amazon Bedrock, should be set to `\"aws_sigv4\"`; otherwise, to `\"http\"`.",
			},
			"credential": {
				Type:        schema.TypeMap,
				Required:    true,
				Sensitive:   true,
				Description: "Defines any credential variables required for connecting to the endpoint",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"parameters": {
				Type:        schema.TypeMap,
				Required:    true,
				Description: "The default ML Connector parameters, including `endpoint`, `model`, and `skip_validating_missing_parameters`. Any parameters indicated in this field can be overridden by parameters specified in a PREDICT request.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"actions": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Defines the actions that can run within the connector",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "predict",
							Description: "Sets the ML Commons API operation to use upon connection. As of OpenSearch 2.9, only `\"predict\"` is supported. Defaults to `\"predict\"`.",
						},
						"method": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "POST",
							Description: "Defines the HTTP method for the API call. Supports `POST` and `GET`. Defaults to `\"POST\"`.",
						},
						"url": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Specifies the connection endpoint at which the action occurs",
						},
						"request_body": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Sets the parameters contained in the request body of the action. The parameters must include `\"inputText\"`, which specifies how users of the connector should construct the request payload for the `action_type`.",
						},
						"pre_process_function": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Built-in or custom Painless script used to preprocess the input data",
						},
						"post_process_function": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Built-in or custom Painless script used to post-process the model output data",
						},
						"headers": {
							Type:        schema.TypeMap,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Default:     map[string]interface{}{"Content-Type": "application/json"},
							Description: "Headers used in the request or response body. Default is `\"Content-Type\": \"application/json\"`. If the third-party ML tool requires access control, the required credential parameters should be defined here.",
						},
					},
				},
			},

			// ============================================
			// ===         Optional attributes          ===
			// ============================================
			"access_mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "private",
				Description: "Sets the access mode for the ML Connector, either `\"public\"`, `\"restricted\"`, or `\"private\"`. Default is `\"private\"`.",
			},
			"backend_roles": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "List of the ML Connector owner’s OpenSearch backend roles to add to the ML Connector. Conflicts with `add_all_backend_roles`.",
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"add_all_backend_roles"},
			},
			"add_all_backend_roles": {
				Type:          schema.TypeBool,
				Optional:      true,
				Description:   "If `true`, all OpenSearch backend roles of the ML Connector owner are added to the ML Connector. Can be specified only if `access_mode` is `\"restricted\"`. Conflicts with `backend_roles`. Admin users cannot set this to `true`.",
				ConflictsWith: []string{"backend_roles"},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// When 'add_all_backend_roles' is set to false, the API doesn't return it.
					// Suppress the diff when the resource is being updated and both old and new values are 'false'.
					return suppressBooleanFalseWhenNotReturned(k, old, new, d)
				},
			},
			"client_config": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "The client configuration object, which provides settings that control the behavior of the client connections used by the ML Connector. Allow to manage connection limits and timeouts.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_connection": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Maximum number of concurrent connections that the client can establish to the server",
						},
						"connection_timeout": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Maximum amount of time (in seconds) that the client will wait while trying to establish a connection to the server",
						},
						"read_timeout": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Maximum amount of time (in seconds) that the client will wait for a response from the server after sending a request",
						},
						"max_retry_times": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "Maximum number of times that a single remote inference request will be retried. Default is `0`.",
						},
						"retry_backoff_policy": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "constant",
							Description: "Backoff policy for retries to the remote connector. Supported policies are `\"constant\"`, `\"exponential_equal_jitter\"`, and `\"exponential_full_jitter\"`. Default is `\"constant\"`.",
						},
						"retry_backoff_millis": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     200,
							Description: "Base backoff time in milliseconds for retry policy. The suspend time during two retries is determined by this parameter and `retry_backoff_policy`. Default is `200`",
						},
						"retry_timeout_seconds": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     30,
							Description: "Timeout, in seconds, for the retry. If the retry can not succeed within the specified amount of time, the connector will stop retrying and throw an exception. Default is `30`.",
						},
					},
				},
			},
		},
	}
}

func resourceOpensearchMLConnectorCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	payload, err := buildConnectorPayload(d)
	if err != nil {
		return diag.Errorf("failed to build ML Connector payload: %s", err)
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return diag.Errorf("failed to marshal ML Connector payload: %s", err)
	}

	url := conf.rawUrl + "/_plugins/_ml/connectors/_create"
	result, err := performRequestAndParse(ctx, conf.osClient, "POST", url, strings.NewReader(string(jsonPayload)), "create ML Connector")
	if err != nil {
		return diag.FromErr(err)
	}

	connectorID, ok := result["connector_id"].(string)
	if !ok {
		return diag.Errorf("connector_id not found in response: %v", result)
	}

	d.SetId(connectorID)
	return resourceOpensearchMLConnectorRead(ctx, d, m)
}

func resourceOpensearchMLConnectorRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/connectors/%s", d.Id())
	connector, err := performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "read ML Connector")
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// ============================================
	// === Required attributes
	// ============================================
	if name, ok := connector["name"].(string); ok {
		if err := d.Set("name", name); err != nil {
			return diag.Errorf("error setting name: %s", err)
		}
	}
	if description, ok := connector["description"].(string); ok {
		if err := d.Set("description", description); err != nil {
			return diag.Errorf("error setting description: %s", err)
		}
	}
	if version, ok := connector["version"].(string); ok {
		if err := d.Set("version", version); err != nil {
			return diag.Errorf("error setting version: %s", err)
		}
	}
	if protocol, ok := connector["protocol"].(string); ok {
		if err := d.Set("protocol", protocol); err != nil {
			return diag.Errorf("error setting protocol: %s", err)
		}
	}
	if params, ok := connector["parameters"].(map[string]interface{}); ok {
		if err := d.Set("parameters", params); err != nil {
			return diag.Errorf("error setting parameters: %s", err)
		}
	}
	if actions, ok := connector["actions"].([]interface{}); ok {
		tfActions := make([]map[string]interface{}, 0, len(actions))
		for _, a := range actions {
			action := a.(map[string]interface{})
			tfAction := map[string]interface{}{}

			// Normalize action_type to lowercase to match user input
			if actionType, ok := action["action_type"].(string); ok {
				tfAction["action_type"] = strings.ToLower(actionType)
			}
			if method, ok := action["method"]; ok {
				tfAction["method"] = method
			}
			if url, ok := action["url"]; ok {
				tfAction["url"] = url
			}
			if requestBody, ok := action["request_body"]; ok {
				tfAction["request_body"] = requestBody
			}
			if preProcess, ok := action["pre_process_function"]; ok {
				tfAction["pre_process_function"] = preProcess
			}
			if postProcess, ok := action["post_process_function"]; ok {
				tfAction["post_process_function"] = postProcess
			}
			if headers, ok := action["headers"].(map[string]interface{}); ok && len(headers) > 0 {
				tfAction["headers"] = headers
			}
			tfActions = append(tfActions, tfAction)
		}
		if err := d.Set("actions", tfActions); err != nil {
			return diag.Errorf("error setting actions: %s", err)
		}
	}

	// ============================================
	// === Optional attributes
	// ============================================
	// API returns 'access' field, map it to 'access_mode' in state
	if access, ok := connector["access"].(string); ok {
		if err := d.Set("access_mode", access); err != nil {
			return diag.Errorf("error setting access_mode: %s", err)
		}
	}
	if backendRoles, ok := connector["backend_roles"].([]interface{}); ok {
		roles := make([]string, len(backendRoles))
		for i, role := range backendRoles {
			if roleStr, ok := role.(string); ok {
				roles[i] = roleStr
			}
		}
		if err := d.Set("backend_roles", roles); err != nil {
			return diag.Errorf("error setting backend_roles: %s", err)
		}
	}
	if clientConfig, ok := connector["client_config"].(map[string]interface{}); ok {
		tfClientConfig := make([]map[string]interface{}, 0, 1)
		tfConfig := map[string]interface{}{}
		if v, ok := clientConfig["max_connection"]; ok {
			tfConfig["max_connection"] = v
		}
		if v, ok := clientConfig["connection_timeout"]; ok {
			tfConfig["connection_timeout"] = v
		}
		if v, ok := clientConfig["read_timeout"]; ok {
			tfConfig["read_timeout"] = v
		}
		if v, ok := clientConfig["max_retry_times"]; ok {
			tfConfig["max_retry_times"] = v
		}
		if v, ok := clientConfig["retry_backoff_policy"]; ok {
			tfConfig["retry_backoff_policy"] = v
		}
		if v, ok := clientConfig["retry_backoff_millis"]; ok {
			tfConfig["retry_backoff_millis"] = v
		}
		if v, ok := clientConfig["retry_timeout_seconds"]; ok {
			tfConfig["retry_timeout_seconds"] = v
		}
		tfClientConfig = append(tfClientConfig, tfConfig)
		if err := d.Set("client_config", tfClientConfig); err != nil {
			return diag.Errorf("error setting client_config: %s", err)
		}
	}

	return nil
}

func resourceOpensearchMLConnectorUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	payload, err := buildConnectorPayload(d)
	if err != nil {
		return diag.Errorf("failed to build ML Connector update payload: %s", err)
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return diag.Errorf("failed to marshal ML Connector update payload: %s", err)
	}

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/connectors/%s", d.Id())
	if _, err := performRequestAndParse(ctx, conf.osClient, "PUT", url, strings.NewReader(string(jsonPayload)), "update ML Connector"); err != nil {
		return diag.FromErr(err)
	}

	return resourceOpensearchMLConnectorRead(ctx, d, m)
}

func resourceOpensearchMLConnectorDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/connectors/%s", d.Id())
	_, err := performRequestAndParse(ctx, conf.osClient, "DELETE", url, nil, "delete ML Connector")
	if err != nil {
		var httpErr *HTTPError
		// Ignore 404 errors - resource is already deleted
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

// ============================================
// ===         Helper functions             ===
// ============================================

func getMLConnectorFromAPI(ctx context.Context, conf *ProviderConf, connectorID string) (map[string]interface{}, error) {
	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/connectors/%s", connectorID)
	return performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "get ML Connector")
}

func buildConnectorPayload(d *schema.ResourceData) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"name":        d.Get("name").(string),
		"description": d.Get("description").(string),
		"version":     d.Get("version").(string),
		"protocol":    d.Get("protocol").(string),
	}

	if v, ok := d.GetOk("parameters"); ok {
		payload["parameters"] = v.(map[string]interface{})
	}

	if v, ok := d.GetOk("credential"); ok {
		credMap := v.(map[string]interface{})
		creds := make(map[string]interface{})
		for k, val := range credMap {
			creds[k] = val
		}
		if len(creds) > 0 {
			payload["credential"] = creds
		}
	}

	if v, ok := d.GetOk("actions"); ok && len(v.([]interface{})) > 0 {
		actions := v.([]interface{})
		payloadActions := make([]map[string]interface{}, 0, len(actions))
		for _, a := range actions {
			action := a.(map[string]interface{})
			payloadAction := map[string]interface{}{
				"action_type": action["action_type"].(string),
			}
			if v, ok := action["method"]; ok && v.(string) != "" {
				payloadAction["method"] = v.(string)
			}
			if v, ok := action["url"]; ok && v.(string) != "" {
				payloadAction["url"] = v.(string)
			}
			if v, ok := action["headers"]; ok {
				payloadAction["headers"] = v.(map[string]interface{})
			}
			if v, ok := action["request_body"]; ok && v.(string) != "" {
				payloadAction["request_body"] = v.(string)
			}
			if v, ok := action["pre_process_function"]; ok && v.(string) != "" {
				payloadAction["pre_process_function"] = v.(string)
			}
			if v, ok := action["post_process_function"]; ok && v.(string) != "" {
				payloadAction["post_process_function"] = v.(string)
			}
			payloadActions = append(payloadActions, payloadAction)
		}
		payload["actions"] = payloadActions
	}

	if v, ok := d.GetOk("access_mode"); ok {
		payload["access"] = v.(string)
	}

	if v, ok := d.GetOk("backend_roles"); ok {
		backendRoles := v.([]interface{})
		roles := make([]string, len(backendRoles))
		for i, role := range backendRoles {
			roles[i] = role.(string)
		}
		payload["backend_roles"] = roles
	}

	if v, ok := d.GetOk("add_all_backend_roles"); ok {
		payload["add_all_backend_roles"] = v.(bool)
	}

	if v, ok := d.GetOk("client_config"); ok && len(v.([]interface{})) > 0 {
		config := v.([]interface{})[0].(map[string]interface{})
		clientConfig := make(map[string]interface{})
		if val, ok := config["max_connection"]; ok && val.(int) != 0 {
			clientConfig["max_connection"] = val
		}
		if val, ok := config["connection_timeout"]; ok && val.(int) != 0 {
			clientConfig["connection_timeout"] = val
		}
		if val, ok := config["read_timeout"]; ok && val.(int) != 0 {
			clientConfig["read_timeout"] = val
		}
		if val, ok := config["max_retry_times"]; ok && val.(int) != 0 {
			clientConfig["max_retry_times"] = val
		}
		if val, ok := config["retry_backoff_policy"]; ok && val.(string) != "" {
			clientConfig["retry_backoff_policy"] = val
		}
		if val, ok := config["retry_backoff_millis"]; ok && val.(int) != 0 {
			clientConfig["retry_backoff_millis"] = val
		}
		if val, ok := config["retry_timeout_seconds"]; ok && val.(int) != 0 {
			clientConfig["retry_timeout_seconds"] = val
		}
		if len(clientConfig) > 0 {
			payload["client_config"] = clientConfig
		}
	}

	return payload, nil
}
