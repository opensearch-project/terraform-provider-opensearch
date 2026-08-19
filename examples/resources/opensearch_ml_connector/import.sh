# Import an ML Connector by its ID
terraform import opensearch_ml_connector.example <connector_id>

# Note: The following field is not returned by the OpenSearch API and will need to be
# manually added to the Terraform configuration after import:
#
#   - credential (write-only, used during creation)
