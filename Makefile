.PHONY: docs up down test dev-up dev-down dev-build dev-config dev-plan dev-apply dev-destroy dev dev-teardown

OSS_IMAGE ?= opensearchproject/opensearch:2
OSS_DASHBOARDS_IMAGE ?=opensearchproject/opensearch-dashboards:2
OPENSEARCH_INITIAL_ADMIN_PASSWORD ?= myStrongPassword123@456
OPENSEARCH_URL ?= http://admin:myStrongPassword123%40456@localhost:9200
DEV_TF_ENV = TF_CLI_CONFIG_FILE=$(CURDIR)/.dev.terraformrc

# =============================================================================
# Core targets — generate documentation and run acceptance tests against a
# local OpenSearch cluster.
#
# Usage:
#   make docs          # regenerate provider documentation (go generate)
#   make up            # start OpenSearch cluster
#   make down          # stop OpenSearch cluster
#   make test          # start cluster + run acceptance tests (TF_ACC=1)
# =============================================================================

docs:
	go generate ./...

up:
	@export OSS_IMAGE=$(OSS_IMAGE) && \
	export OPENSEARCH_INITIAL_ADMIN_PASSWORD=$(OPENSEARCH_INITIAL_ADMIN_PASSWORD) && \
	docker compose up -d

down:
	@docker compose down

test: up
	@export OPENSEARCH_URL=$(OPENSEARCH_URL) && \
	export TF_LOG=INFO && \
	TF_ACC=1 go test ./provider -v -parallel 20 -cover -short

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
