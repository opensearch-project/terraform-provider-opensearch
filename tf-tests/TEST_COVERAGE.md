# Test Coverage Matrix

> This document tracks the current test coverage for each OpenSearch Terraform provider resource in this suite.

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Covered / Verified |
| ⚠️ | Partially covered |
| ❌ | Not covered |
| N/A | Not applicable (e.g., immutable resources, data sources) |

### Version Compatibility (v2, v3 columns)

| Symbol | Meaning |
|--------|---------|
| ✅ | Tested and passing against this version |
| ⚠️ | Infrastructure ready but not yet empirically verified |

---

## Resource Coverage

| Resource | Test File | Minimal | Full | Update | Invalid | v2 | v3 |
|----------|-----------|:-------:|:----:|:------:|:-------:|:---:|:---:|
| `opensearch_cluster_settings` | `tests/opensearch_cluster_settings.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_composable_index_template` | `tests/opensearch_index_template.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_component_template` | `tests/opensearch_component_template.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_data_stream` | `tests/opensearch_data_stream.tftest.hcl` | ✅ | ✅ | N/A | ❌ | ✅ | ✅ |
| `opensearch_index` | `tests/opensearch_index.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_ingest_pipeline` | `tests/opensearch_ingest_pipeline.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_script` | `tests/opensearch_script.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_snapshot_repository` | `tests/opensearch_snapshot_repository.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_ism_policy` | `tests/opensearch_ism_policy.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_ism_policy_mapping` | `tests/opensearch_ism_policy_mapping.tftest.hcl` | ✅ | ✅ | N/A | ❌ | ✅ | ✅ |
| `opensearch_role` | `tests/opensearch_role.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_user` | `tests/opensearch_user.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_roles_mapping` | `tests/opensearch_roles_mapping.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_audit_config` | `tests/opensearch_audit_config.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_dashboard_tenant` | `tests/opensearch_dashboard_tenant.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_dashboard_object` | `tests/opensearch_dashboard_object.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_monitor` | `tests/opensearch_monitor.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_channel_configuration` | `tests/opensearch_channel_configuration.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_anomaly_detection` | `tests/opensearch_anomaly_detection.tftest.hcl` | ✅ | ✅ | N/A | ❌ | ✅ | ✅ |
| `opensearch_sm_policy` | `tests/opensearch_sm_policy.tftest.hcl` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| `opensearch_host` (data source) | `tests/opensearch_host.tftest.hcl` | ✅ | N/A | N/A | N/A | ✅ | ✅ |

**Resources with typed modules:** `opensearch_index`, `opensearch_role`, `opensearch_script`, `opensearch_audit_config`, `opensearch_dashboard_tenant`, `opensearch_roles_mapping`, `opensearch_cluster_settings`, `opensearch_snapshot_repository`, `opensearch_user`, `opensearch_ism_policy_mapping`.

**Resources with JSON-body modules:** `opensearch_composable_index_template`, `opensearch_component_template`, `opensearch_ingest_pipeline`, `opensearch_ism_policy`, `opensearch_dashboard_object`, `opensearch_monitor`, `opensearch_channel_configuration`, `opensearch_anomaly_detection`, `opensearch_sm_policy`.

---

## Test Run Summary

| Test File | Run Blocks | Type |
|-----------|------------|------|
| **Unit Tests (21 files)** | | |
| `tests/opensearch_anomaly_detection.tftest.hcl` | 2 | Unit |
| `tests/opensearch_audit_config.tftest.hcl` | 2 | Unit |
| `tests/opensearch_channel_configuration.tftest.hcl` | 4 | Unit |
| `tests/opensearch_cluster_settings.tftest.hcl` | 3 | Unit |
| `tests/opensearch_component_template.tftest.hcl` | 4 | Unit |
| `tests/opensearch_dashboard_object.tftest.hcl` | 3 | Unit |
| `tests/opensearch_dashboard_tenant.tftest.hcl` | 2 | Unit |
| `tests/opensearch_data_stream.tftest.hcl` | 2 | Unit |
| `tests/opensearch_host.tftest.hcl` | 1 | Unit |
| `tests/opensearch_index_template.tftest.hcl` | 4 | Unit |
| `tests/opensearch_index.tftest.hcl` | 4 | Unit |
| `tests/opensearch_ingest_pipeline.tftest.hcl` | 3 | Unit |
| `tests/opensearch_ism_policy_mapping.tftest.hcl` | 3 | Unit |
| `tests/opensearch_ism_policy.tftest.hcl` | 4 | Unit |
| `tests/opensearch_monitor.tftest.hcl` | 4 | Unit |
| `tests/opensearch_role.tftest.hcl` | 3 | Unit |
| `tests/opensearch_roles_mapping.tftest.hcl` | 3 | Unit |
| `tests/opensearch_script.tftest.hcl` | 3 | Unit |
| `tests/opensearch_sm_policy.tftest.hcl` | 5 | Unit |
| `tests/opensearch_snapshot_repository.tftest.hcl` | 2 | Unit |
| `tests/opensearch_user.tftest.hcl` | 2 | Unit |
| **Integration Tests (4 files)** | | |
| `tests/log_ingestion_pipeline.tftest.hcl` | 6 | Integration |
| `tests/multi_tenant_dashboards.tftest.hcl` | 11 | Integration |
| `tests/production_cluster_setup.tftest.hcl` | 10 | Integration |
| `tests/security_monitoring.tftest.hcl` | 6 | Integration |
| **Total** | **91** | |
| **Unit Tests** | **58** | |
| **Integration Tests** | **33** | |

---

## Category Definitions

| Category | Description |
|----------|-------------|
| **Minimal** | Tests resource creation with only required fields. |
| **Full** | Tests resource creation with all or most configurable fields populated. |
| **Update** | Tests in-place modification of an existing resource (e.g., changing replicas, adding permissions). |
| **Invalid** | Negative tests that pass invalid inputs and expect Terraform or the provider to reject them (`expect_failures`). |
| **v2 / v3** | OpenSearch version compatibility: whether tests have been verified against OpenSearch 2.x or 3.x. |

---

## Integration Test Coverage

| Scenario | Status | Resources Tested |
|----------|--------|------------------|
| Log Ingestion Pipeline | ✅ Implemented | 6 (component template, index template, ISM policy, ingest pipeline, bootstrap index, role) |
| Security Monitoring & Alerting | ✅ Implemented | 6 (anomaly detection, channel, monitor, role, user, roles mapping) |
| Multi-Tenant Dashboards | ✅ Implemented | 11 (tenants, index template, data streams, dashboard objects, roles, users, ingest pipeline) |
| Production Cluster Setup | ✅ Implemented | 10 (cluster settings, snapshot repo, SM policy, audit config, component template, index template, ingest pipeline, ISM policy, admin roles) |

**Integration test prerequisites:** The `scripts/setup-integration-data.sh` script pre-populates indices with sample documents using `curl` against the OpenSearch REST API. This provides the data required for anomaly detection detector creation and other data-dependent assertions.

---

## Known Gaps

1. **No negative/invalid tests exist** — The `expect_failures` tests were removed because they used module-level `variable` validations, not provider-level validation. Testing provider error handling requires passing malformed inputs directly through the module to the provider resource and asserting on provider errors. No such tests exist.
2. **JSON-body assertions require `jsondecode`** — Resources using an opaque `body` variable can still assert on individual nested fields by using `jsondecode(opensearch_*.this.body)["field"]` inside `plan`-run blocks. Typed modules remain preferable for naturally exposing individual attributes.
3. **Audit config assertions are weak** — `opensearch_audit_config` tests only assert `enabled == true`, but the resource has many more fields (`compliance`, `audit` block, etc.) that are not verified.
4. **Dashboard tenant lacks typed module** — `opensearch_dashboard_tenant` has `index` as a computed attribute, and `opensearch_dashboard_object` has `index` and `tenant_name`, but these typed attributes are not exposed in modules.
5. **Data stream and anomaly detection are intrinsically limited** — Data streams cannot be updated in place, and anomaly detection detectors require indexed documents, so only minimal/plan tests are possible in unit tests. Integration tests address this by pre-populating data before exercising the provider.
