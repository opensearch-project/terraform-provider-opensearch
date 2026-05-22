<img src="https://opensearch.org/assets/brand/SVG/Logo/opensearch_logo_default.svg" height="64px"/>

- [Terraform Provider OpenSearch](#terraform-provider-opensearch)
  - [Supported Functionalities](#supported-functionalities)
    - [OpenSearch and OpenSearch Dashboards](#opensearch-and-opensearch-dashboards)
  - [Running tests locally](#running-tests-locally)
    - [To Run Specific Test](#to-run-specific-test)
    - [Fix the go-lint errors](#fix-the-go-lint-errors)
  - [Debugging this provider](#debugging-this-provider)
- [Version and Branching](#version-and-branching)
- [Contributing](#contributing)
- [Getting Help](#getting-help)
- [Code of Conduct](#code-of-conduct)
- [Security](#security)
- [License](#license)
- [Copyright](#copyright)

## Terraform Provider OpenSearch

This is a terraform provider to provision OpenSearch resources.

### Upgrade Guide

#### SDK Migration (v2.0.0)

This version migrates the provider from the deprecated `olivere/elastic v7` library and AWS SDK v1 to the official `opensearch-go/v4` SDK and AWS SDK v2. Several provider configuration fields have been renamed for consistency. The old field names are still accepted but emit deprecation warnings.

##### Renamed Provider Arguments

| Old Argument | New Argument | Notes |
|---|---|---|
| `aws_token` | `aws_session_token` | Renamed for clarity |
| `version_ping_timeout` | `ping_timeout_seconds` | Renamed for consistency; default changed from 5s to 15s |
| `aws_signature_service` | *(auto-detected)* | Now auto-detected from URL (`es` for OpenSearch Service, `aoss` for Serverless). Set `opensearch_version` for Serverless instead. |

##### Renamed Environment Variables

| Old Environment Variable | New Environment Variable | Notes |
|---|---|---|
| `OS_CLIENT_CERTIFICATE_PATH` | `OPENSEARCH_CLIENT_CERT_PATH` | New name preferred; old name still works as fallback |
| `OS_CLIENT_KEY_PATH` | `OPENSEARCH_CLIENT_KEY_PATH` | New name preferred; old name still works as fallback |

New environment variables are also available for previously-hardcoded-default fields:

| Environment Variable | Field | Default |
|---|---|---|
| `OPENSEARCH_INSECURE` | `insecure` | `false` |
| `OPENSEARCH_CACERT` | `cacert_file` | *(empty)* |
| `OPENSEARCH_TOKEN_NAME` | `token_name` | `ApiKey` |
| `OPENSEARCH_SIGN_AWS` | `sign_aws_requests` | `true` |
| `OPENSEARCH_VERSION` | `opensearch_version` | *(empty)* |
| `OPENSEARCH_PING_TIMEOUT` | `ping_timeout_seconds` | `15` |
| `OPENSEARCH_AWS_ASSUME_ROLE_ARN` | `aws_assume_role_arn` | *(empty)* |
| `OPENSEARCH_AWS_ASSUME_ROLE_EXTERNAL_ID` | `aws_assume_role_external_id` | *(empty)* |
| `OPENSEARCH_HOST_OVERRIDE` | `host_override` | *(empty)* |
| `OPENSEARCH_PROXY` | `proxy` | *(empty)* |
| `AWS_REGION` | `aws_region` | *(empty)* |
| `AWS_ACCESS_KEY_ID` | `aws_access_key` | *(empty)* |
| `AWS_SECRET_ACCESS_KEY` | `aws_secret_key` | *(empty)* |
| `AWS_SESSION_TOKEN` | `aws_session_token` | *(empty)* |
| `AWS_PROFILE` | `aws_profile` | *(empty)* |

##### Default Value Changes

- `ping_timeout_seconds`: default changed from **5** to **15** seconds. If you relied on the shorter timeout, set `ping_timeout_seconds = 5` explicitly.

##### Migration Steps

1. **No immediate action required** — old field names are still accepted with deprecation warnings.
2. Update your provider block to use the new field names:

```hcl
# Before (still works, but deprecated)
provider "opensearch" {
  url                   = "https://example.com:9200"
  aws_token             = "my-session-token"
  version_ping_timeout  = 5
  aws_signature_service = "es"
}

# After (recommended)
provider "opensearch" {
  url                  = "https://example.com:9200"
  aws_session_token    = "my-session-token"
  ping_timeout_seconds = 5
  # aws_signature_service is no longer needed - auto-detected from URL
}
```

3. If using environment variables, update `OS_CLIENT_CERTIFICATE_PATH` to `OPENSEARCH_CLIENT_CERT_PATH` and `OS_CLIENT_KEY_PATH` to `OPENSEARCH_CLIENT_KEY_PATH`.

4. For AWS OpenSearch Serverless endpoints (`*.aoss.amazonaws.com`), the service name is now auto-detected. Remove `aws_signature_service = "aoss"` from your provider config and set `opensearch_version` if needed.

### Supported Functionalities 

Examples of resources can be found in the examples directory.

#### OpenSearch and OpenSearch Dashboards

- [x] [Cluster Settings](https://opensearch.org/docs/latest/api-reference/cluster-api/cluster-settings/)
- [x] [Audit Config](https://opensearch.org/docs/latest/security/audit-logs/index/)
- [x] [Component templates](https://opensearch.org/docs/latest/dashboards/im-dashboards/component-templates/)
- [x] [Index and Composable templates](https://opensearch.org/docs/latest/im-plugin/index-templates/)
- [x] [Data Streams](https://opensearch.org/docs/2.9/dashboards/im-dashboards/datastream/)
- [x] [Ingest Pipeline](https://opensearch.org/docs/2.9/api-reference/ingest-apis/create-update-ingest/)
- [x] [Security](https://opensearch.org/docs/latest/security/index/)
- [x] [Snapshot Repository](https://opensearch.org/docs/2.9/tuning-your-cluster/availability-and-recovery/snapshots/snapshot-restore/#register-repository)
- [x] [Anomaly Detection](https://opensearch.org/docs/latest/observing-your-data/ad/index/)
- [x] [Index State Management](https://opensearch.org/docs/latest/im-plugin/ism/index/)
- [x] [Dashboards Visualization](https://opensearch.org/docs/latest/dashboards/visualize/viz-index/)
- [x] [Dashboards Tenant](https://opensearch.org/docs/latest/security/multi-tenancy/tenant-index/)
- [x] [Alerting Monitors](https://opensearch.org/docs/latest/observing-your-data/alerting/monitors/)
- [x] [Notification Channels](https://opensearch.org/docs/latest/observing-your-data/notifications/index/)

### Running tests locally

The provider supports both OpenSearch 2.x and 3.x. You can test against either version by setting the `OSS_IMAGE` environment variable.

#### Testing against OpenSearch 2.x

```sh
export OSS_IMAGE="opensearchproject/opensearch:2"
docker compose up -d
docker compose ps -a  # Checks that the process is running
# Before OS 2.12.0
# export OPENSEARCH_URL=http://admin:admin@localhost:9200
export OPENSEARCH_URL=http://admin:myStrongPassword123%40456@localhost:9200
export TF_LOG=INFO
TF_ACC=1 go test ./... -v -parallel 20 -cover -short
```

#### Testing against OpenSearch 3.x

```sh
export OSS_IMAGE="opensearchproject/opensearch:3"
docker compose up -d
docker compose ps -a  # Checks that the process is running
export OPENSEARCH_URL=http://admin:myStrongPassword123%40456@localhost:9200
export TF_LOG=INFO
TF_ACC=1 go test ./... -v -parallel 20 -cover -short
```

Note:  Starting from version `2.12.0`, the `admin` user password is determined by the `OPENSEARCH_INITIAL_ADMIN_PASSWORD` environment variable. If testing against a cluster with version `2.12.0` or later and have set `OPENSEARCH_INITIAL_ADMIN_PASSWORD=myStrongPassword123@456`, please update the URL as follows: `export OPENSEARCH_URL=http://admin:myStrongPassword123%40456@localhost:9200`

#### To Run Specific Test

```sh
cd provider/
TF_ACC=2 go test -run TestAccOpensearchOpenDistroDashboardTenant  -v -cover -short
```

#### Fix the go-lint errors

```sh
golangci-lint run --out-format=github-actions 
```

### Debugging this provider

Build the executable, and start in debug mode:

```console
$ go build
$ ./terraform-provider-opensearch -debuggable # or start in debug mode in your IDE
{"@level":"debug","@message":"plugin address","@timestamp":"2022-05-17T10:10:04.331668+01:00","address":"/var/folders/32/3mbbgs9x0r5bf991ltrl3p280010fs/T/plugin1346340234","network":"unix"}
Provider started, to attach Terraform set the TF_REATTACH_PROVIDERS env var:

        TF_REATTACH_PROVIDERS='{"registry.terraform.io/opensearch-project/opensearch":{"Protocol":"grpc","ProtocolVersion":5,"Pid":79075,"Test":true,"Addr":{"Network":"unix","String":"/var/folders/32/3mbbgs9x0r5bf991ltrl3p280010fs/T/plugin1346340234"}}}'
```

In another terminal, you can test your terraform code:

```console
$ cd <my-project/terraform>
$ export TF_REATTACH_PROVIDERS=<env var above>
$ terraform apply
```

The local provider will be used instead, and you should see debug information printed to the terminal.

## Version and Branching

As of now, this terraform-provider-opensearch repository maintains 2 branches:

- _main_ (2.x.x OpenSearch development)
- _1.x_ (1.x.x OpenSearch development)

Contributors should choose the corresponding branch(es) when commiting their change(s):

- If you have a change for a specific version, only open PR to specific branch
- If you have a change for all available versions, first open a PR on `main`, then open a backport PR with `[x]` in the title, with label `backport 1.x`, etc.

## Contributing

See [developer guide](DEVELOPER_GUIDE.md) and [how to contribute to this project](CONTRIBUTING.md). 

## Getting Help

If you find a bug, or have a feature request, please don't hesitate to open an issue in this repository.

For more information, see [project website](https://opensearch.org/) and [documentation](https://opensearch.org/docs/latest/). If you need help and are unsure where to open an issue, try [forums](https://discuss.opendistrocommunity.dev/).

## Code of Conduct

This project has adopted the [Amazon Open Source Code of Conduct](CODE_OF_CONDUCT.md). For more information see the [Code of Conduct FAQ](https://aws.github.io/code-of-conduct-faq), or contact [opensource-codeofconduct@amazon.com](mailto:opensource-codeofconduct@amazon.com) with any additional questions or comments.

## Security

If you discover a potential security issue in this project we ask that you notify AWS/Amazon Security via our [vulnerability reporting page](http://aws.amazon.com/security/vulnerability-reporting/). Please do **not** create a public GitHub issue.

## License

This project is licensed under the [Apache v2.0 License](LICENSE).

## Copyright

Copyright OpenSearch Contributors. See [NOTICE](NOTICE) for details.
