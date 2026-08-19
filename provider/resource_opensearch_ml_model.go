package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/opensearch-project/opensearch-go/v2"
)

const (
	maxModelRegistrationWait               = 120 * time.Second
	modelRegistrationPollInterval          = 2 * time.Second
	maxModelDeploymentStateChangeWait      = 120 * time.Second
	modelDeploymentStateChangePollInterval = 2 * time.Second
)

// Declared as vars so unit tests can override them.
var (
	maxPredictProbeWait      = 60 * time.Second
	predictProbePollInterval = 5 * time.Second
	defaultPredictProbeBody  = `{"parameters": {"inputText": "healthcheck"}}`
)

func resourceOpensearchMLModel() *schema.Resource {
	return &schema.Resource{
		Description:   "OpenSearch ML Model resource",
		CreateContext: resourceOpensearchMLModelCreate,
		ReadContext:   resourceOpensearchMLModelRead,
		UpdateContext: resourceOpensearchMLModelUpdate,
		DeleteContext: resourceOpensearchMLModelDelete,
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
				Description: "Name of the ML Model",
			},
			"model_group_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ML Model Group ID to which this ML Model belongs",
			},

			// ============================================
			// ===         Optional attributes          ===
			// ============================================
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the ML Model (not returned by OpenSearch API for OS-provided models, but returned for custom and third-party models)",
			},
			"function_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "For text embedding models, should be set to `\"TEXT_EMBEDDING\"`; for sparse encoding models, to `\"SPARSE_ENCODING\"` or `\"SPARSE_TOKENIZE\"`; for cross-encoder models, to `\"TEXT_SIMILARITY\"`; for question-answering models, to `\"QUESTION_ANSWERING\"`. Required for custom and third-party models",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"is_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the model is enabled. Disabled ML Models are unavailable to the Predict API requests regardless of the ML Model deployment status. Defaults to `true`.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// When 'is_enabled' is set to false, the API doesn't return it.
					// Suppress the diff when the resource is being updated and both old and new values are 'false'.
					return suppressBooleanFalseWhenNotReturned(k, old, new, d)
				},
			},
			"deploy_after_registering": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Description: "Whether to deploy the model after registration. Defaults to `true`.",
			},
			"wait_for_predict": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When true (default), polls `POST /_plugins/_ml/models/<id>/_predict` after deployment to confirm the model is actually usable in memory — not just `DEPLOYED` in the index. Only takes effect when `deploy_after_registering = true`. Set to false to rely only on `model_state`.",
			},
			"predict_probe_body": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     defaultPredictProbeBody,
				Description: "JSON body sent to `POST /_plugins/_ml/models/<id>/_predict` when probing readiness. Defaults to `{\"parameters\": {\"inputText\": \"healthcheck\"}}`, which works for remote connector models. Override this for models whose predict API expects a different input format (e.g. `{\"text_docs\": [\"healthcheck\"]}`). Only takes effect when `wait_for_predict = true`.",
			},

			// ========================================================
			// === Optional attributes - OpenSearch-provided models ===
			// ========================================================
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Version of the ML Model. For OS-provided models, this identifies which version to download.",
			},
			"model_format": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Portable format of the model file. Valid values are `\"TORCH_SCRIPT\"` and `\"ONNX\"`. Required for OS-provided pretrained and custom models)",
			},

			// ============================================
			// === Optional attributes - custom models  ===
			// ============================================
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "URL from which to download the model. Required for custom models.",
			},
			"model_content_hash_value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "SHA-256 hash of the downloaded model. Required for custom models.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diff if the new value is empty (not set in config) but old value exists (from API)
					return new == "" && old != ""
				},
			},
			"model_config": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Model’s configuration, including the `model_type`, `embedding_dimension`, and `framework_type`. The optional `all_config` JSON string contains all model configurations. The `additional_config` object contains the corresponding `space_type` for pretrained models or the specified `space_type` for custom models. Required for custom models.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"model_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Model type (e.g., `\"bert\"`)",
						},
						"embedding_dimension": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Dimension of the model-generated dense vector",
						},
						"framework_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Framework the model is using. Currently, OpenSearch supports `\"sentence_transformers\"` and `\"huggingface_transformers\"` frameworks.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return strings.EqualFold(old, new)
							},
						},
						"all_config": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Used for reference purposes. Can be used to specify all model configurations.",
						},
						"additional_config": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Contains the `space_type`, which specifies the distance metric for k-NN search. For OpenSearch-provided pretrained models, this value is automatically set to the corresponding metric.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								// Only suppress diff if resource exists (has ID) and API doesn't return this field
								// During creation (no ID), we want the field to be sent
								if d.Id() == "" {
									return false
								}
								// After creation, suppress diff when count changes from 0 to non-zero (API doesn't return it)
								return k == "model_config.0.additional_config.#" && old == "0"
							},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"space_type": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Space type",
									},
								},
							},
						},
						"pooling_mode": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The post-process model output, either `\"mean\"`, `\"mean_sqrt_len\"`, `\"max\"`, `\"weightedmean\"`, or `\"cls\"`.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return strings.EqualFold(old, new)
							},
						},
						"normalize_result": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "When set to `true`, normalizes the model output in order to scale to a standard range for the model.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								// When 'normalize_result' is set to false, the API doesn't return it.
								// Suppress the diff when the resource is being updated and both old and new values are 'false'.
								return suppressBooleanFalseWhenNotReturned(k, old, new, d)
							},
						},
					},
				},
			},

			// ========================================================
			// === Optional attributes - third-party-hosted models  ===
			// ========================================================
			"connector_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ML Connector ID this ML Model uses. Required for third-party models.",
			},

			// ===================================================================
			// === Optional attributes - custom and third-party-hosted models  ===
			// ===================================================================
			"rate_limiter": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Limits the number of times that any user can call the Predict API on the ML Model.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"limit": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Maximum number of requests",
						},
						"unit": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Time frame (e.g., `\"SECONDS\"`)",
						},
					},
				},
			},
			"interface": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Interface for the model",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"input": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "JSON schema for the model input.",
						},
						"output": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "JSON schema for the model output.",
						},
					},
				},
			},
			"guardrails": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Guardrails for the model input.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Valid values are `\"local_regex\"` and `\"model\"`. Using `\"local_regex\"`, the attributes `stop_words` and `regex` are available. Using `\"model\"`, the attributes `model_id` and `response_validation_regex` are available.",
						},
						"model_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Guardrail model used to validate user input and LLM output. Required when type is `\"model\"`.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								if d.Id() == "" {
									return false
								}
								return k == "guardrails.0.model_id" && old == ""
							},
						},
						"response_validation_regex": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Regular expression used to validate the guardrail model response. Required when type is `\"model\"`.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								if d.Id() == "" {
									return false
								}
								return k == "guardrails.0.response_validation_regex" && old == ""
							},
						},
						"input_guardrail": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Guardrail for the model input.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								if d.Id() == "" {
									return false
								}
								return k == "guardrails.0.input_guardrail.#" && old == "0"
							},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"stop_words": {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "List of indexes containing stopwords used for model input validation",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"index_name": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of index storing the stopwords",
												},
												"source_fields": {
													Type:        schema.TypeList,
													Required:    true,
													Description: "Name of field(s) in the index storing the stopwords",
													Elem: &schema.Schema{
														Type: schema.TypeString,
													},
												},
											},
										},
									},
									"regex": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Regular expression used for input validation",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
						"output_guardrail": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Guardrail for the model output.",
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								if d.Id() == "" {
									return false
								}
								return k == "guardrails.0.output_guardrail.#" && old == "0"
							},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"stop_words": {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "List of indexes containing stopwords used for model output validation",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"index_name": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of index storing the stopwords",
												},
												"source_fields": {
													Type:        schema.TypeList,
													Required:    true,
													Description: "Name of field(s) in the index storing the stopwords",
													Elem: &schema.Schema{
														Type: schema.TypeString,
													},
												},
											},
										},
									},
									"regex": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Regular expression used for output validation",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceOpensearchMLModelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	payload := map[string]interface{}{
		"name":           d.Get("name").(string),
		"model_group_id": d.Get("model_group_id").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		payload["description"] = v.(string)
	}
	if v, ok := d.GetOk("function_name"); ok {
		payload["function_name"] = v.(string)
	}
	// Note: OpenSearch seems to ignore this value in the register payload (it only persists the field
	// via the Update Model API). We keep it in the registration payload in case a future version of the
	// API uses it. The actual disabling (the default of the API is to enable) is done by an explicit update
	// after registration (see below).
	payload["is_enabled"] = d.Get("is_enabled").(bool)

	// ============================================
	// === OpenSearch-provided models
	// ============================================
	if v, ok := d.GetOk("version"); ok {
		payload["version"] = v.(string)
	}
	if v, ok := d.GetOk("model_format"); ok {
		payload["model_format"] = v.(string)
	}

	// ============================================
	// === Custom models
	// ============================================
	if v, ok := d.GetOk("url"); ok {
		payload["url"] = v.(string)
	}
	if v, ok := d.GetOk("model_content_hash_value"); ok {
		payload["model_content_hash_value"] = v.(string)
	}
	if v, ok := d.GetOk("model_config"); ok {
		modelConfigList := v.([]interface{})
		if len(modelConfigList) > 0 {
			if modelConfigMap, ok := modelConfigList[0].(map[string]interface{}); ok {
				cleanedModelConfig := buildModelConfigPayload(modelConfigMap)
				if len(cleanedModelConfig) > 0 {
					payload["model_config"] = cleanedModelConfig
				}
			}
		}
	}

	// ============================================
	// === Third-party models
	// ============================================
	if v, ok := d.GetOk("connector_id"); ok {
		payload["connector_id"] = v.(string)
	}

	// ============================================
	// === Custom and third-party models
	// ============================================
	if v, ok := d.GetOk("rate_limiter"); ok {
		rateLimiterList := v.([]interface{})
		if len(rateLimiterList) > 0 {
			if rateLimiterMap, ok := rateLimiterList[0].(map[string]interface{}); ok {
				payload["rate_limiter"] = rateLimiterMap
			}
		}
	}
	if v, ok := d.GetOk("interface"); ok {
		interfaceList := v.([]interface{})
		if len(interfaceList) > 0 {
			if interfaceMap, ok := interfaceList[0].(map[string]interface{}); ok {
				cleanedInterface := buildInterfacePayload(interfaceMap)
				if len(cleanedInterface) > 0 {
					payload["interface"] = cleanedInterface
				}
			}
		}
	}
	if v, ok := d.GetOk("guardrails"); ok {
		guardrailsList := v.([]interface{})
		if len(guardrailsList) > 0 {
			if guardrailsMap, ok := guardrailsList[0].(map[string]interface{}); ok {
				guardrailsPayload := buildGuardrailsPayload(guardrailsMap)
				if len(guardrailsPayload) > 0 {
					payload["guardrails"] = guardrailsPayload
				}
			}
		}
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return diag.Errorf("failed to marshal ML Model payload: %s", err)
	}

	result, err := performRequestAndParse(ctx, conf.osClient, "POST", conf.rawUrl+"/_plugins/_ml/models/_register", strings.NewReader(string(jsonPayload)), "register ML Model")
	if err != nil {
		return diag.FromErr(err)
	}

	var modelID string

	// Check if response contains 'model_id' (synchronous registration for third-party models)
	if id, ok := result["model_id"].(string); ok && id != "" {
		modelID = id
	} else if taskID, ok := result["task_id"].(string); ok && taskID != "" {
		// Async registration (OS-provided pretrained and custom models) - poll for task completion
		modelID, err = waitForModelRegistrationTask(ctx, conf.osClient, conf.rawUrl, taskID)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		return diag.Errorf("neither 'model_id' nor 'task_id' found in response: %v", result)
	}

	d.SetId(modelID)

	// OpenSearch ignores `is_enabled` at registration, so a model registered with
	// `is_enabled = false` would otherwise stay enabled.
	if err := setMLModelEnabled(ctx, conf, modelID, d.Get("name").(string), d.Get("is_enabled").(bool)); err != nil {
		return diag.Errorf("error disabling ML Model: %s", err)
	}

	// Deploy the model if 'deploy_after_registering' is true or not set (defaults to true)
	deployAfterRegistering := d.Get("deploy_after_registering").(bool)
	if deployAfterRegistering {
		if err := deployMLModel(ctx, conf, modelID); err != nil {
			return diag.Errorf("error deploying ML Model: %s", err)
		}
		// Confirm the model is usable if 'wait_for_predict' is true or not set (defaults to true)
		if d.Get("wait_for_predict").(bool) {
			if err := waitForModelPredictReady(ctx, conf, modelID, d.Get("predict_probe_body").(string)); err != nil {
				return diag.Errorf("error waiting for ML Model predict readiness: %s", err)
			}
		}
	}

	return resourceOpensearchMLModelRead(ctx, d, m)
}

func resourceOpensearchMLModelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	model, err := getMLModelFromAPI(ctx, conf, d.Id())
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
	if name, ok := model["name"].(string); ok {
		if err := d.Set("name", name); err != nil {
			return diag.Errorf("error setting name: %s", err)
		}
	}
	if modelGroupID, ok := model["model_group_id"].(string); ok {
		if err := d.Set("model_group_id", modelGroupID); err != nil {
			return diag.Errorf("error setting model_group_id: %s", err)
		}
	}

	// ============================================
	// === Optional attributes
	// ============================================
	if desc, ok := model["description"].(string); ok && desc != "" {
		if err := d.Set("description", desc); err != nil {
			return diag.Errorf("error setting description: %s", err)
		}
	}
	if algorithm, ok := model["algorithm"].(string); ok && algorithm != "" {
		if err := d.Set("function_name", algorithm); err != nil {
			return diag.Errorf("error setting function_name: %s", err)
		}
	}
	// The OpenSearch model GET response never echos is_enabled, so the value in `d` is persisted
	// (the planned value from Create / Update, which respects the default of `true`). The
	// overwrite below is in case a future OpenSearch version starts returning the `is_enabled`
	// value (or if it's already returned in special cases not caught during testing).
	//
	// Limitation: on Import there is no prior value and the schema default is not applied, so
	// `d.Get` returns false; so an enabled model is imported with `is_enabled = false` until the
	// next apply reconciles it (hence `is_enabled` is in ImportStateVerifyIgnore for the case
	// where the config omits it).
	if err := d.Set("is_enabled", d.Get("is_enabled").(bool)); err != nil {
		return diag.Errorf("error setting is_enabled: %s", err)
	}
	if isEnabled, ok := model["is_enabled"].(bool); ok {
		if err := d.Set("is_enabled", isEnabled); err != nil {
			return diag.Errorf("error setting is_enabled: %s", err)
		}
	}

	// ============================================
	// === OpenSearch-provided models
	// ============================================
	if modelFormat, ok := model["model_format"].(string); ok && modelFormat != "" {
		if err := d.Set("model_format", modelFormat); err != nil {
			return diag.Errorf("error setting model_format: %s", err)
		}
	}

	// ============================================
	// === Custom models
	// ============================================
	if url, ok := model["url"].(string); ok && url != "" {
		if err := d.Set("url", url); err != nil {
			return diag.Errorf("error setting url: %s", err)
		}
	}
	if modelContentHashValue, ok := model["model_content_hash_value"].(string); ok && modelContentHashValue != "" {
		if err := d.Set("model_content_hash_value", modelContentHashValue); err != nil {
			return diag.Errorf("error setting model_content_hash_value: %s", err)
		}
	}
	if modelConfig, ok := model["model_config"].(map[string]interface{}); ok {
		if additionalConfigMap, ok := modelConfig["additional_config"].(map[string]interface{}); ok {
			modelConfig["additional_config"] = []interface{}{additionalConfigMap}
		}
		if err := d.Set("model_config", []interface{}{modelConfig}); err != nil {
			return diag.Errorf("error setting model_config: %s", err)
		}
	}

	// ============================================
	// === Third-party models
	// ============================================
	if connectorID, ok := model["connector_id"].(string); ok && connectorID != "" {
		if err := d.Set("connector_id", connectorID); err != nil {
			return diag.Errorf("error setting connector_id: %s", err)
		}
	}

	// ============================================
	// === Custom and third-party models
	// ============================================
	if rateLimiter, ok := model["rate_limiter"].(map[string]interface{}); ok {
		if limitStr, ok := rateLimiter["limit"].(string); ok {
			if limitInt, err := strconv.Atoi(limitStr); err == nil {
				rateLimiter["limit"] = limitInt
			}
		}
		if err := d.Set("rate_limiter", []interface{}{rateLimiter}); err != nil {
			return diag.Errorf("error setting rate_limiter: %s", err)
		}
	}
	if interfaceData, ok := model["interface"].(map[string]interface{}); ok {
		if err := d.Set("interface", []interface{}{interfaceData}); err != nil {
			return diag.Errorf("error setting interface: %s", err)
		}
	}
	if guardrailsData, ok := model["guardrails"].(map[string]interface{}); ok {
		guardrails := make(map[string]interface{})
		if gType, ok := guardrailsData["type"].(string); ok {
			guardrails["type"] = gType
		}
		if modelID, ok := guardrailsData["model_id"].(string); ok {
			guardrails["model_id"] = modelID
		}
		if responseValidationRegex, ok := guardrailsData["response_validation_regex"].(string); ok {
			guardrails["response_validation_regex"] = responseValidationRegex
		}
		if len(guardrails) > 0 {
			if err := d.Set("guardrails", []interface{}{guardrails}); err != nil {
				return diag.Errorf("error setting guardrails: %s", err)
			}
		}
	}

	return nil
}

func resourceOpensearchMLModelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	payload := make(map[string]interface{})

	// ============================================
	// === Required attributes
	// ============================================
	if d.HasChange("name") {
		payload["name"] = d.Get("name").(string)
	}
	if d.HasChange("model_group_id") {
		payload["model_group_id"] = d.Get("model_group_id").(string)
	}

	// ============================================
	// === Optional attributes
	// ============================================
	if d.HasChange("description") {
		payload["description"] = d.Get("description").(string)
	}
	if d.HasChange("is_enabled") {
		payload["is_enabled"] = d.Get("is_enabled").(bool)
	}

	// ============================================
	// === Custom models
	// ============================================
	if d.HasChange("model_config") {
		if v, ok := d.GetOk("model_config"); ok {
			modelConfigList := v.([]interface{})
			if len(modelConfigList) > 0 {
				if modelConfigMap, ok := modelConfigList[0].(map[string]interface{}); ok {
					cleanedModelConfig := buildModelConfigPayload(modelConfigMap)
					if len(cleanedModelConfig) > 0 {
						payload["model_config"] = cleanedModelConfig
					}
				}
			}
		}
	}

	// ============================================
	// === Third-party models
	// ============================================
	if d.HasChange("connector_id") {
		payload["connector_id"] = d.Get("connector_id").(string)
	}

	// ============================================
	// === Custom and third-party models
	// ============================================
	if d.HasChange("rate_limiter") {
		if v, ok := d.GetOk("rate_limiter"); ok {
			rateLimiterList := v.([]interface{})
			if len(rateLimiterList) > 0 {
				if rateLimiterMap, ok := rateLimiterList[0].(map[string]interface{}); ok {
					payload["rate_limiter"] = rateLimiterMap
				}
			}
		}
	}
	if d.HasChange("interface") {
		if v, ok := d.GetOk("interface"); ok {
			interfaceList := v.([]interface{})
			if len(interfaceList) > 0 {
				if interfaceMap, ok := interfaceList[0].(map[string]interface{}); ok {
					cleanedInterface := buildInterfacePayload(interfaceMap)
					if len(cleanedInterface) > 0 {
						payload["interface"] = cleanedInterface
					}
				}
			}
		}
	}
	if d.HasChange("guardrails") {
		if v, ok := d.GetOk("guardrails"); ok {
			guardrailsList := v.([]interface{})
			if len(guardrailsList) > 0 {
				if guardrailsMap, ok := guardrailsList[0].(map[string]interface{}); ok {
					guardrailsPayload := buildGuardrailsPayload(guardrailsMap)
					if len(guardrailsPayload) > 0 {
						payload["guardrails"] = guardrailsPayload
					}
				}
			}
		}
	}

	if len(payload) == 0 {
		return resourceOpensearchMLModelRead(ctx, d, m)
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return diag.Errorf("failed to marshal ML Model update payload: %s", err)
	}

	if _, err := performRequestAndParse(ctx, conf.osClient, "PUT", conf.rawUrl+fmt.Sprintf("/_plugins/_ml/models/%s", d.Id()), strings.NewReader(string(jsonPayload)), "update ML Model"); err != nil {
		return diag.FromErr(err)
	}

	return resourceOpensearchMLModelRead(ctx, d, m)
}

func resourceOpensearchMLModelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conf := m.(*ProviderConf)

	if err := undeployMLModel(ctx, conf, d.Id()); err != nil {
		return diag.Errorf("error undeploying ML Model: %s", err)
	}

	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/models/%s", d.Id())
	_, err := performRequestAndParse(ctx, conf.osClient, "DELETE", url, nil, "delete ML Model")
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

func getMLModelFromAPI(ctx context.Context, conf *ProviderConf, modelID string) (map[string]interface{}, error) {
	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/models/%s", modelID)
	return performRequestAndParse(ctx, conf.osClient, "GET", url, nil, "get ML Model")
}

func buildGuardrailsPayload(guardrailsMap map[string]interface{}) map[string]interface{} {
	guardrailsPayload := make(map[string]interface{})

	if gType, ok := guardrailsMap["type"].(string); ok {
		guardrailsPayload["type"] = gType
	}

	if modelID, ok := guardrailsMap["model_id"].(string); ok && modelID != "" {
		guardrailsPayload["model_id"] = modelID
	}

	if responseValidationRegex, ok := guardrailsMap["response_validation_regex"].(string); ok && responseValidationRegex != "" {
		guardrailsPayload["response_validation_regex"] = responseValidationRegex
	}

	if inputGuardrails, ok := guardrailsMap["input_guardrail"].([]interface{}); ok && len(inputGuardrails) > 0 {
		if inputGuardrailsMap, ok := inputGuardrails[0].(map[string]interface{}); ok {
			inputGuardrailsPayload := make(map[string]interface{})

			if stopWords, ok := inputGuardrailsMap["stop_words"].([]interface{}); ok && len(stopWords) > 0 {
				if stopWordsMap, ok := stopWords[0].(map[string]interface{}); ok {
					inputGuardrailsPayload["stop_words"] = stopWordsMap
				}
			}
			if regex, ok := inputGuardrailsMap["regex"].([]interface{}); ok {
				inputGuardrailsPayload["regex"] = regex
			}

			guardrailsPayload["input_guardrails"] = []interface{}{inputGuardrailsPayload}
		}
	}

	if outputGuardrails, ok := guardrailsMap["output_guardrail"].([]interface{}); ok && len(outputGuardrails) > 0 {
		if outputGuardrailsMap, ok := outputGuardrails[0].(map[string]interface{}); ok {
			outputGuardrailsPayload := make(map[string]interface{})

			if stopWords, ok := outputGuardrailsMap["stop_words"].([]interface{}); ok && len(stopWords) > 0 {
				if stopWordsMap, ok := stopWords[0].(map[string]interface{}); ok {
					outputGuardrailsPayload["stop_words"] = stopWordsMap
				}
			}
			if regex, ok := outputGuardrailsMap["regex"].([]interface{}); ok {
				outputGuardrailsPayload["regex"] = regex
			}

			guardrailsPayload["output_guardrails"] = []interface{}{outputGuardrailsPayload}
		}
	}

	return guardrailsPayload
}

func buildInterfacePayload(interfaceMap map[string]interface{}) map[string]interface{} {
	cleanedInterface := make(map[string]interface{})

	if input, ok := interfaceMap["input"].(string); ok && input != "" {
		cleanedInterface["input"] = input
	}
	if output, ok := interfaceMap["output"].(string); ok && output != "" {
		cleanedInterface["output"] = output
	}

	return cleanedInterface
}

func buildModelConfigPayload(modelConfigMap map[string]interface{}) map[string]interface{} {
	cleanedModelConfig := make(map[string]interface{})

	// Only include non-empty fields
	if modelType, ok := modelConfigMap["model_type"].(string); ok && modelType != "" {
		cleanedModelConfig["model_type"] = modelType
	}
	if embeddingDim, ok := modelConfigMap["embedding_dimension"].(int); ok && embeddingDim > 0 {
		cleanedModelConfig["embedding_dimension"] = embeddingDim
	}
	if frameworkType, ok := modelConfigMap["framework_type"].(string); ok && frameworkType != "" {
		cleanedModelConfig["framework_type"] = frameworkType
	}
	if poolingMode, ok := modelConfigMap["pooling_mode"].(string); ok && poolingMode != "" {
		cleanedModelConfig["pooling_mode"] = poolingMode
	}
	// Only include all_config if it's not empty
	if allConfig, ok := modelConfigMap["all_config"].(string); ok && allConfig != "" {
		cleanedModelConfig["all_config"] = allConfig
	}
	// Only include normalize_result if explicitly set
	if normalizeResult, ok := modelConfigMap["normalize_result"].(bool); ok {
		cleanedModelConfig["normalize_result"] = normalizeResult
	}
	// Handle additional_config - only include if it has actual content
	if additionalConfigList, ok := modelConfigMap["additional_config"].([]interface{}); ok && len(additionalConfigList) > 0 {
		if additionalConfigMap, ok := additionalConfigList[0].(map[string]interface{}); ok {
			if spaceType, ok := additionalConfigMap["space_type"].(string); ok && spaceType != "" {
				cleanedModelConfig["additional_config"] = additionalConfigMap
			}
		}
	}

	return cleanedModelConfig
}

func setMLModelEnabled(ctx context.Context, conf *ProviderConf, modelID string, modelName string, enabled bool) error {
	jsonPayload, err := json.Marshal(map[string]interface{}{"is_enabled": enabled})
	if err != nil {
		return fmt.Errorf("failed to marshal 'is_enabled' update payload: %w", err)
	}
	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/models/%s", modelID)
	_, err = performRequestAndParse(ctx, conf.osClient, "PUT", url, strings.NewReader(string(jsonPayload)), "update ML Model 'is_enabled'")
	if err != nil {
		// Some model types (e.g. OpenSearch-provided pretrained SPARSE_ENCODING models) reject the update
		// of the 'is_enabled' parameter with a 403 "... is not supported at this time". For those, 'is_enabled'
		// cannot be changed, so we leave it as the OpenSearch default instead of failing the apply.
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden && strings.Contains(httpErr.Message, "is not supported at this time") {
			log.Printf("[WARN] ML Model %s (ID %s) does not support updating 'is_enabled'; leaving it at the OpenSearch default. Error from OpenSearch: %s", modelName, modelID, httpErr.Message)
			return nil
		}
		return err
	}
	return nil
}

func deployMLModel(ctx context.Context, conf *ProviderConf, modelID string) error {
	state, err := deployAndWaitForModel(ctx, conf, modelID)
	if err != nil {
		return err
	}
	if state == "DEPLOYED" {
		return nil
	}

	// OpenSearch sometimes settles into PARTIALLY_DEPLOYED (deployed to some nodes
	// but not all) or UNDEPLOYED (drifted out of DEPLOYED) instead of DEPLOYED.
	// Retry once: for PARTIALLY_DEPLOYED the model is undeployed first; for UNDEPLOYED a
	// plain redeploy is enough.
	log.Printf("[INFO] ML Model %s reached state %s instead of DEPLOYED; retrying deployment once", modelID, state)

	if state == "PARTIALLY_DEPLOYED" {
		log.Printf("[INFO] Undeploying ML Model %s before redeploy retry", modelID)
		if err := undeployMLModel(ctx, conf, modelID); err != nil {
			return fmt.Errorf("failed to undeploy ML Model %s before redeploy retry: %w", modelID, err)
		}
	}

	log.Printf("[INFO] Redeploying ML Model %s", modelID)
	state, err = deployAndWaitForModel(ctx, conf, modelID)
	if err != nil {
		return err
	}
	if state != "DEPLOYED" {
		return fmt.Errorf("ML Model %s did not reach DEPLOYED state after retry; final state: %s", modelID, state)
	}
	return nil
}

func deployAndWaitForModel(ctx context.Context, conf *ProviderConf, modelID string) (string, error) {
	if _, err := performRequestAndParse(ctx, conf.osClient, "POST", conf.rawUrl+fmt.Sprintf("/_plugins/_ml/models/%s/_deploy", modelID), nil, "deploy ML Model"); err != nil {
		return "", err
	}

	return waitForModelDeploymentStateChange(ctx, conf.osClient, conf.rawUrl, modelID, "DEPLOYED", "PARTIALLY_DEPLOYED", "UNDEPLOYED")
}

func undeployMLModel(ctx context.Context, conf *ProviderConf, modelID string) error {
	_, err := performRequestAndParse(ctx, conf.osClient, "POST", conf.rawUrl+fmt.Sprintf("/_plugins/_ml/models/%s/_undeploy", modelID), nil, "undeploy ML Model")
	if err != nil {
		// Ignore errors if model is not found or not deployed (already in desired state).
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil
		}
		if strings.Contains(err.Error(), "not deployed") ||
			strings.Contains(err.Error(), "Model not found") {
			return nil
		}
		return err
	}

	_, err = waitForModelDeploymentStateChange(ctx, conf.osClient, conf.rawUrl, modelID, "UNDEPLOYED", "REGISTERED")
	return err
}

func waitForModelRegistrationTask(ctx context.Context, client *opensearch.Client, baseURL, taskID string) (string, error) {
	url := baseURL + fmt.Sprintf("/_plugins/_ml/tasks/%s", taskID)

	deadline := time.Now().Add(maxModelRegistrationWait)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("context cancelled while waiting for ML Model registration: %w", ctx.Err())
		default:
		}

		result, err := performRequestAndParse(ctx, client, "GET", url, nil, "get task status")
		if err != nil {
			return "", err
		}

		state, _ := result["state"].(string)
		switch state {
		case "COMPLETED":
			if modelID, ok := result["model_id"].(string); ok && modelID != "" {
				return modelID, nil
			}
			return "", fmt.Errorf("ML Model registration task completed but 'model_id' not found in response")
		case "FAILED":
			errorMsg := extractTaskErrorMessage(result)
			return "", fmt.Errorf("ML Model registration task failed (task_id: %s, state: %s): %s", taskID, state, errorMsg)
		}

		time.Sleep(modelRegistrationPollInterval)
	}

	return "", fmt.Errorf("timeout waiting for ML Model registration task to complete after %s", maxModelRegistrationWait)
}

// waitForModelDeploymentStateChange polls 'model_state' until it matches any of targetStates,
// returning the observed state. Always errors on DEPLOY_FAILED or context cancellation/timeout.
func waitForModelDeploymentStateChange(ctx context.Context, client *opensearch.Client, baseURL, modelID string, targetStates ...string) (string, error) {
	url := baseURL + fmt.Sprintf("/_plugins/_ml/models/%s", modelID)

	isTarget := func(state string) bool {
		for _, s := range targetStates {
			if state == s {
				return true
			}
		}
		return false
	}

	deadline := time.Now().Add(maxModelDeploymentStateChangeWait)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("context cancelled while waiting for ML Model state change: %w", ctx.Err())
		default:
		}

		result, err := performRequestAndParse(ctx, client, "GET", url, nil, "get ML Model status")
		if err != nil {
			var httpErr *HTTPError
			if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
				return "UNDEPLOYED", nil
			}
			return "", err
		}

		modelState, _ := result["model_state"].(string)
		if isTarget(modelState) {
			return modelState, nil
		}
		if modelState == "DEPLOY_FAILED" {
			errorMsg := "unknown error"
			if errStr, ok := result["error"].(string); ok && errStr != "" {
				errorMsg = errStr
			} else if errMap, ok := result["error"].(map[string]interface{}); ok {
				if reason, ok := errMap["reason"].(string); ok {
					errorMsg = reason
				}
			}
			return modelState, fmt.Errorf("ML Model deployment failed (model_id: %s, state: %s): %s", modelID, modelState, errorMsg)
		}

		time.Sleep(modelDeploymentStateChangePollInterval)
	}

	return "", fmt.Errorf("timeout waiting for ML Model to reach state %v after %s (model_id: %s)", targetStates, maxModelDeploymentStateChangeWait, modelID)
}

func waitForModelPredictReady(ctx context.Context, conf *ProviderConf, modelID string, probeBody string) error {
	url := conf.rawUrl + fmt.Sprintf("/_plugins/_ml/models/%s/_predict", modelID)
	deadline := time.Now().Add(maxPredictProbeWait)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for ML Model predict readiness: %w", ctx.Err())
		default:
		}

		result, err := performRequestAndParse(ctx, conf.osClient, "POST", url,
			strings.NewReader(probeBody), "probe ML Model predict readiness")
		if err == nil {
			if _, ok := result["inference_results"]; ok {
				return nil
			}
		}

		time.Sleep(predictProbePollInterval)
	}

	return fmt.Errorf("timeout waiting for ML Model predict readiness after %s (model_id: %s): model_state is DEPLOYED but _predict does not return inference_results", maxPredictProbeWait, modelID)
}

func extractTaskErrorMessage(taskResult map[string]interface{}) string {
	if errStr, ok := taskResult["error"].(string); ok && errStr != "" {
		return errStr
	}
	if errMap, ok := taskResult["error"].(map[string]interface{}); ok {
		if reason, ok := errMap["reason"].(string); ok {
			return reason
		}
		if errJSON, err := json.Marshal(errMap); err == nil {
			return string(errJSON)
		}
	}
	return "unknown error"
}
