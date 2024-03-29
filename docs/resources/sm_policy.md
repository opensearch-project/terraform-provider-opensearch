---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "opensearch_sm_policy Resource - terraform-provider-opensearch"
subcategory: ""
description: |-
  Provides an OpenSearch Snapshot Management (SM) policy. Please refer to the OpenSearch SM documentation for details.
---

# opensearch_sm_policy (Resource)

Provides an OpenSearch Snapshot Management (SM) policy. Please refer to the OpenSearch SM documentation for details.

## Example Usage

```terraform
# Create a snapshot repository. Make sure you also have created the bucket (eg. 
# via `terraform-aws-modules/s3-bucket/aws`) and matching IAM role.
resource "opensearch_snapshot_repository" "repo" {
  name = "os-index-backups"
  type = "s3"

  settings = {
    bucket                 = module.s3_snapshot.s3_bucket_id
    region                 = module.s3_snapshot.s3_bucket_region
    role_arn               = aws_iam_role.snapshot_create.arn
    server_side_encryption = true
  }
}

# Create the SM policy
resource "opensearch_sm_policy" "snapshot_to_s3" {
  policy_name = "snapshot_to_s3"

  body = jsonencode({
    "enabled"     = true
    "description" = "My snapshot policy"

    "creation" = {
      "schedule" = {
        "cron" = {
          "expression" = "0 0 * * *"
          "timezone"   = "UTC"
        }
      }

      "time_limit" = "1h"
    }

    "deletion" = {
      "schedule" = {
        "cron" = {
          "expression" = "0 0 * * *"
          "timezone"   = "UTC"
        }
      }

      "condition" = {
        "max_age"   = "14d"
        "max_count" = 400
        "min_count" = 1
      }

      "time_limit" = "1h"
    }

    "snapshot_config" = {
      "timezone"   = "UTC"
      "indices"    = "*"
      "repository" = opensearch_snapshot_repository.repo.name
    }
  })
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `body` (String) The policy document.
- `policy_name` (String) The name of the SM policy.

### Optional

- `primary_term` (Number) The primary term of the SM policy version.
- `seq_no` (Number) The sequence number of the SM policy version.

### Read-Only

- `id` (String) The ID of this resource.

## Import

Import is supported using the following syntax:

```shell
terraform import opensearch_sm_policy.cleanup snapshot_to_s3
```
