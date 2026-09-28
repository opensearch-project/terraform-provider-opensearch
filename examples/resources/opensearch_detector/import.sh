#!/bin/bash

# Import an existing OpenSearch Security Analytics detector by ID

# Basic import
terraform import opensearch_detector.example dc2VB4QBrbtylUb_Hfa3

# Import with specific provider configuration
terraform import -var-file="prod.tfvars" opensearch_detector.production_detector x-dwFIYBT6_n8WeuQjo4

# Import multiple detectors
terraform import opensearch_detector.windows_detector MFRg1IMByX0LvTiGHtcN
terraform import opensearch_detector.linux_detector J1RX1IMByX0LvTiGTddR
terraform import opensearch_detector.network_detector IJAXz4QBrmVplM4JYxx_

echo "Import completed. Remember to update your Terraform configuration to match the imported detector settings."
echo "Run 'terraform plan' to see any configuration drift that needs to be addressed."