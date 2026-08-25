# Changelog
## Unreleased
### Changed

### Added
* Support for `AssumeRoleWithWebIdentity` via the new `aws_web_identity_role_arn` and `aws_web_identity_token_file` arguments, which default to the standard `AWS_ROLE_ARN` and `AWS_WEB_IDENTITY_TOKEN_FILE` environment variables so EKS IRSA works with no configuration ([#89](https://github.com/opensearch-project/terraform-provider-opensearch/issues/89))
* Support the `search.cancel_after_time_interval` persistent cluster setting via `search_cancel_after_time_interval` on `opensearch_cluster_settings`

### Fixed

## [1.0.0] - 2023-04-15
### Added
* Release 1.0 based on phillbaker/terraform-provider-elasticsearch
