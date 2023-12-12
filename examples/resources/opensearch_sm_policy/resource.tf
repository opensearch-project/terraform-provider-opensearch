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
