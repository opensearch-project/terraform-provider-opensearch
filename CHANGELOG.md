# Changelog
## Unreleased
### Changed

### Added
* Support for `AssumeRoleWithWebIdentity` via the new `aws_web_identity_role_arn` and `aws_web_identity_token_file` arguments, which default to the standard `AWS_ROLE_ARN` and `AWS_WEB_IDENTITY_TOKEN_FILE` environment variables so EKS IRSA works with no configuration ([#89](https://github.com/opensearch-project/terraform-provider-opensearch/issues/89))

### Fixed
* `opensearch_dashboard_object` no longer reports a diff when the only changes are bookkeeping Dashboards maintains itself: `updated_at`, and an index pattern's per-field popularity counters (`fields[].count`). Using Dashboards is enough to change both — the saved objects API restamps `updated_at` on every write, and Discover increments a counter whenever someone uses a field — which made managed objects drift continuously.

## [1.0.0] - 2023-04-15
### Added
* Release 1.0 based on phillbaker/terraform-provider-elasticsearch
