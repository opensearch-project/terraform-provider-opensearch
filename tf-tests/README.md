# OpenSearch Terraform Provider Test Suite

> Test suite for the official OpenSearch Terraform provider using native HCL and `terraform test`.

## Overview

This repository provides a comprehensive test suite covering 21 Terraform resources across the OpenSearch provider. Tests run against ephemeral OpenSearch 2.x and 3.x containers via `terraform test`.

For detailed instructions, see [`AGENTS.md`](AGENTS.md). For coverage metrics, see [`TEST_COVERAGE.md`](TEST_COVERAGE.md).

## Quick Start

```bash
# Run tests against OpenSearch 2.x with the local provider
make tf-test-local

# Run against OpenSearch 3.x
make OS_VERSION=3 tf-test-local

# Filter by test file
make tf-test-local TF_TEST_ARGS="-filter=tests/opensearch_index.tftest.hcl"

# Run with verbose output
make tf-test-local TF_TEST_ARGS="-verbose"
```

## Project Structure

```
├── docker-compose.yml          # Local OpenSearch (security enabled)
├── providers.tf                # Declares input variables
├── modules/                    # Reusable resource modules
├── tests/                      # Test files (*.tftest.hcl)
├── fixtures/                   # Sample policies, mappings, data
└── ../script/                  # shared test helpers
```

## Prerequisites

- Docker Compose
- Terraform >= 1.6.0
- curl

## Docker Compose

- **Image**: `opensearchproject/opensearch:$(OS_VERSION)` selected by `OS_VERSION`
- **Security**: Enabled (demo certificates)
- **Credentials**: `admin` / `myStrongPassword123@456`

## Local Provider Development

To build the provider and run the tests against it:

```bash
# OpenSearch 2.x
make tf-test-local

# OpenSearch 3.x
make OS_VERSION=3 tf-test-local
```

The Makefile builds the provider, generates the Terraform CLI configuration,
starts the root Compose service, waits for OpenSearch, loads integration data,
runs `terraform test`, and cleans up the service.

## Test Coverage

| Category | Count |
|----------|-------|
| Resources | 21 |
| Unit Tests | 58 run blocks |
| Integration Tests | 33 run blocks |
| OpenSearch Versions | 2.x, 3.x |

All resources include Minimal, Full, and Update tests. See [`TEST_COVERAGE.md`](TEST_COVERAGE.md) for the full matrix.

## References

- Provider: https://github.com/opensearch-project/terraform-provider-opensearch
- Provider Docs: https://registry.terraform.io/providers/opensearch-project/opensearch/latest/docs
- Terraform Tests: https://developer.hashicorp.com/terraform/language/tests
