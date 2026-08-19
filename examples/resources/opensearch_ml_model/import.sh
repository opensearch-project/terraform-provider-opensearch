# Import an ML Model by its ID
terraform import opensearch_ml_model.example <model_id>

# Note: The following fields are not returned by the OpenSearch API and will need to be
# manually added to the Terraform configuration after import:
#
# For OpenSearch-provided pretrained models:
#   - description (not persisted by OpenSearch for OS-provided models)
#   - version (write-only, used during registration)
#   - deploy_after_registering (provider-only attribute)
#
# For custom models:
#   - url (write-only, used during registration)
#   - version (write-only, used during registration)
#   - model_config.additional_config (write-only, used during registration)
#   - deploy_after_registering (provider-only attribute)
#
# For third-party models:
#   - guardrails.input_guardrail (write-only, used during registration for type "local_regex")
#   - guardrails.output_guardrail (write-only, used during registration for type "local_regex")
#   - guardrails.model_id (write-only, used during registration for type "model")
#   - guardrails.response_validation_regex (write-only, used during registration type "model")
#   - deploy_after_registering (provider-only attribute)
