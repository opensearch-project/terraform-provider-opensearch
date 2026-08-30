# Terraform Test Agent Guide

This directory contains the native Terraform tests for the OpenSearch provider.
For user-facing commands and prerequisites, see the repository [README.md](../README.md).
For resource coverage and scenario details, see [TEST_COVERAGE.md](TEST_COVERAGE.md).

## Repository Layout

- `tests/`: Terraform test files. Each file uses the `.tftest.hcl` extension.
- `modules/`: Alternate modules loaded by test `run` blocks.
- `fixtures/`: Static test inputs.
- `providers.tf`: Test input variables used by provider configurations.
- `versions.tf`: Terraform and provider constraints.

The OpenSearch service is defined by the root [docker-compose.yml](../docker-compose.yml).
Shared setup helpers are under [script/](../script/), not in this directory.

## Test Conventions

- Keep one resource or data-source test file per provider capability where practical.
- Use `module` blocks to test the reusable modules under `modules/`.
- Keep module inputs explicit and expose values needed by assertions through outputs.
- Cover minimal, full, update, and negative behavior when the resource supports it.
- Use unique resource names so tests do not collide within a shared OpenSearch cluster.
- Use `command = plan` for validation-only runs and `command = apply` when the provider must create resources.
- Use `state_key` deliberately when create, read, and update runs must share state. Do not remove it merely to silence a parser error.
- Preserve dependencies between runs that consume earlier run outputs.

The suite requires Terraform `>= 1.11.0`; `state_key` is part of the test syntax used by these files.

## Provider Configuration

Each test file defines its own provider configuration. Keep the provider source address and variable names consistent:

```hcl
provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}
```

The root Makefile generates `tests/terraform.auto.tfvars` for the local Compose cluster. This file is generated and ignored; do not commit it or put credentials in test files.

## Running Tests

From the repository root, use the Makefile targets documented in [README.md](../README.md):

```bash
make tf-test-local
make OS_VERSION=3 tf-test-local
make test-os2
make test-os3
```

Use `TF_TEST_ARGS` to narrow a run:

```bash
make tf-test-local TF_TEST_ARGS="-filter=tests/opensearch_index.tftest.hcl"
```

The version-specific targets run Go and Terraform tests against one OpenSearch container and clean it up afterward. Do not start a second Compose cluster manually.

For a direct Terraform test invocation, run it from `tf-tests/` only after the Makefile has generated variables and Terraform has been initialized. Prefer the Makefile target because it owns setup, test data, local provider configuration, and cleanup.

## Local Provider Changes

The `tf-test-local` target runs `dev-build` and `dev-config` before `terraform test`. The generated `.dev.terraformrc` uses a `dev_overrides` directory pointing to the repository root, where the provider binary is built.

When changing provider behavior:

1. Update the relevant module or `.tftest.hcl` file.
2. Run the narrowest filtered test that exercises the change.
3. Run the corresponding full version target when the focused test passes.
4. Update [TEST_COVERAGE.md](TEST_COVERAGE.md) only when coverage status or scenarios change.

## Generated Files

Do not commit these generated artifacts:

- `tf-tests/.terraform/`
- `tf-tests/tests/terraform.auto.tfvars`
- root `.dev.terraformrc`
- root `terraform-provider-opensearch` binary

The Terraform provider lock file `tf-tests/.terraform.lock.hcl` is intentional repository configuration and should remain version-controlled unless project policy changes.

## Validation

Before submitting changes, run the relevant filtered Terraform test and verify formatting from the repository root:

```bash
terraform fmt -check -recursive
make OS_VERSION=2 tf-test-local TF_TEST_ARGS="-filter=tests/<changed-file>.tftest.hcl"
```

Avoid broad refactors of unrelated modules or test files. Keep assertions focused on provider behavior and make cleanup reliable for resources created by the test.
