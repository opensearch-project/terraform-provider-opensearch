.PHONY: docs up down test dev-up dev-down dev-build dev-config dev-plan dev-apply dev-destroy dev dev-teardown

# OpenSearch version to test against/use
OS_VERSION ?= 2

OSS_IMAGE ?= opensearchproject/opensearch:${OS_VERSION}
OSS_DASHBOARDS_IMAGE ?=opensearchproject/opensearch-dashboards:${OS_VERSIONN}
OPENSEARCH_INITIAL_ADMIN_PASSWORD ?= myStrongPassword123@456
OPENSEARCH_URL ?= http://admin:myStrongPassword123%40456@localhost:9200
DEV_TF_ENV = TF_CLI_CONFIG_FILE=$(CURDIR)/.dev.terraformrc

# Terraform settings
TF_LOG ?= INFO
TF_ACC ?= 1

# Test settings
TEST_PARALLEL := 20
TEST_TIMEOUT := 120m

# =============================================================================
# Core targets — generate documentation and run acceptance tests against a
# local OpenSearch cluster.
#
# Usage:
#   make docs          # regenerate provider documentation (go generate)
#   make up            # start OpenSearch cluster
#   make down          # stop OpenSearch cluster
#   make test          # start cluster + run acceptance tests (TF_ACC=1)
#   make test-os2      # Run tests using OpenSearch 2.x
#   make test-os2      # Run tests using OpenSearch 3.x
#   make check-tools   # Checks that the requires tools are installed

# =============================================================================

docs:
	go generate ./...

wait: ## Wait for OpenSearch to be ready (same as CI)
	@echo "Waiting for OpenSearch at $(OPENSEARCH_URL)..."
	./script/wait-for-endpoint --timeout=60 $(OPENSEARCH_URL)

up:
	@echo "Starting OpenSearch ${OS_VERSION}"
	@export OSS_IMAGE=$(OSS_IMAGE) && \
	export OPENSEARCH_INITIAL_ADMIN_PASSWORD=$(OPENSEARCH_INITIAL_ADMIN_PASSWORD) && \
	docker compose up -d
	@echo "Containers started. Run 'make wait' to wait for OpenSearch to be ready."

down:
	@echo "Stopping OpenSearch containers..."
	@docker compose down
	@echo "Containers stopped."

test: up
	@echo "Running tests against OpenSearch $(OS_VERSION)..."
	go clean -testcache
	@export OPENSEARCH_URL=$(OPENSEARCH_URL) && \
	export OPENSEARCH_PREFIX=$(OPENSEARCH_PREFIX) && \
	export TF_LOG=INFO && \
	TF_ACC=$(TF_ACC) go test ./provider -v -parallel $(TEST_PARALLEL) -cover -short -timeout $(TEST_TIMEOUT)

test-os2: check-tools
	$(MAKE) OS_VERSION=2 up
	$(MAKE) OS_VERSION=2 wait
	$(MAKE) OS_VERSION=2 test || (EXIT_CODE=$$?; $(MAKE) OS_VERSION=2 down; exit $$EXIT_CODE)
	$(MAKE) OS_VERSION=2 down

test-os3: check-tools
	$(MAKE) OS_VERSION=3 up
	$(MAKE) OS_VERSION=3 wait
	$(MAKE) OS_VERSION=3 test || (EXIT_CODE=$$?; $(MAKE) OS_VERSION=3 down; exit $$EXIT_CODE)
	$(MAKE) OS_VERSION=3 down

check-tools:
	@which go > /dev/null || (echo "Error: Go is not installed" && exit 1)
	@which terraform > /dev/null || (echo "Error: terraform is not installed" && exit 1)
	@which docker > /dev/null || (echo "Error: docker is not installed" && exit 1)
	@docker compose version > /dev/null 2>&1 || (echo "Error: docker compose is not available" && exit 1)
	@echo "All required tools are installed."
	
check: tidy-check fmt-check
	@echo "All pre-commit checks passed."

tidy:
	go mod tidy

tidy-check:
	./script/test-mod-tidy

fmt-check:
	terraform fmt -check -recursive

fmt: 
	terraform fmt -recursive

validate:
	terraform validate -no-color

ci-test: tidy-check fmt-check validate 
	@echo "=== Starting full CI test for OpenSearch $(OS_VERSOIN) ==="
	$(MAKE) up
	$(MAKE) wait
	$(MAKE) test || (EXIT_CODE=$$?; $(MAKE) down; exit $$EXIT_CODE)
	$(MAKE) down
	@echo "=== Full CI test completed ==="

ci-test-os2:
	$(MAKE) OS_VERSION=2 ci-test

ci-test-os3:
	$(MAKE) OS_VERSION=3 ci-test

# =============================================================================
# Developer sandbox — spin up a real cluster (with OpenSearch Dashboards), build
# the provider from source, and apply a representative Terraform configuration
# (dev/) to verify resources work end-to-end.
#
# Terraform variables are read from dev/terraform.tfvars (gitignored). Copy
# dev/terraform.tfvars.example to get started.
#
# Usage:
#   make dev           # full one-command setup: dev-up + wait + dev-apply
#   make dev-up        # start OpenSearch cluster + Dashboards (compose `dashboards` profile)
#   make dev-down      # stop OpenSearch cluster + Dashboards
#   make dev-build     # build the provider binary in the repo root
#   make dev-config    # write .dev.terraformrc with dev_overrides → local binary
#   make dev-plan      # rebuild + preview changes (no apply) (cluster must already be up)
#   make dev-apply     # rebuild + apply (cluster must already be up)
#   make dev-destroy   # destroy Terraform resources (cluster stays up)
#   make dev-teardown  # dev-destroy + stop OpenSearch cluster + Dashboards
# =============================================================================

dev-up:
	@export OSS_IMAGE=$(OSS_IMAGE) && \
	export OSS_DASHBOARDS_IMAGE=$(OSS_DASHBOARDS_IMAGE) && \
	export OPENSEARCH_INITIAL_ADMIN_PASSWORD=$(OPENSEARCH_INITIAL_ADMIN_PASSWORD) && \
	export COMPOSE_PROFILES=dashboards && \
	docker compose up -d

dev-down:
	@export COMPOSE_PROFILES=dashboards && \
	docker compose down

dev-build:
	@echo "Building provider binary..."
	go build -o terraform-provider-opensearch .
	@echo "Provider binary built."

dev-config:
	@echo "Generating .dev.terraformrc to enable Terraform to use locally-built binary..."
	@printf 'provider_installation {\n  dev_overrides {\n    "opensearch-project/opensearch" = "%s"\n  }\n  direct {}\n}\n' "$(CURDIR)" > .dev.terraformrc
	@echo "Generated .dev.terraformrc."

dev-plan: dev-build dev-config
	@echo "Running terraform plan..."
	$(DEV_TF_ENV) terraform -chdir=dev plan

dev-apply: dev-build dev-config
	@echo "Running terraform apply with auto-approve..."
	$(DEV_TF_ENV) terraform -chdir=dev apply -auto-approve

dev-destroy:
	@echo "Running terraform destroy with auto-approve..."
	$(DEV_TF_ENV) terraform -chdir=dev destroy -auto-approve

# Uses `dev-up` instead of `up` to activate the `dashboards` compose profile so OpenSearch Dashboards container is used.
dev: dev-up
	@echo "Waiting for OpenSearch to be ready (up to 120s)..."
	./script/wait-for-endpoint --timeout=120 $(OPENSEARCH_URL)
	$(MAKE) dev-apply

dev-teardown: dev-destroy down

# Commmands used by CI and local testing


