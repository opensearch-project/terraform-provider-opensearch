# OpenSearch Provider SDK Migration Plan

## Executive Summary

Migrate the Terraform OpenSearch Provider from the deprecated **olivere/elastic v7** library to the official **opensearch-go/v4** SDK, while also upgrading from AWS SDK v1 to v2.

**Business Drivers:**
- olivere/elastic is deprecated and no longer maintained
- AWS SDK v1 reaches end-of-support on July 31, 2025
- Official SDK provides better OpenSearch feature support
- Native AWS SigV4 signing support

## Migration Status

| Phase | Step | Status | Notes |
|-------|------|--------|-------|
| Phase 1 | 1.1 Update Dependencies | ✅ Complete | Added `opensearch-go/v4 v4.6.0` to go.mod |
| Phase 1 | 1.2 Create New Client Factory | ✅ Complete | Created `provider/client.go` (170 lines) with `NewOpenSearchClient()` |
| Phase 1 | 1.3 AWS Signer Migration | ✅ Complete | Created `aws_signer_v2.go` with AWS SDK v2 implementation |
| Phase 2 | 2.1 Update ProviderConf | ⏳ Pending | Not started |
| Phase 2 | 2.2 Update getClient Function | ⏳ Pending | Not started |
| Phase 2 | 2.3 HTTP Transport Wrappers | ⏳ Pending | Not started |
| Phase 3 | Resource Migration | ⏳ Pending | Not started |
| Phase 4 | Error Handling | ⏳ Pending | Not started |
| Phase 5 | Testing & Validation | ⏳ Pending | Not started |

**Last Updated:** February 15, 2026

## Current State Analysis

### Dependencies
```go
// Current (go.mod)
github.com/olivere/elastic/v7 v7.0.32
github.com/aws/aws-sdk-go v1.52.2
github.com/deoxxa/aws_signing_client v0.0.0-20161109131055-c20ee106809e
```

### Architecture Overview
- **Client Initialization:** `provider/provider.go:296-427` (getClient function)
- **HTTP Transport:** Custom wrappers in `provider/http.go`
- **AWS Signing:** Custom implementation in `provider/awsv4.go`
- **Resource Count:** 25 resources across 26 files
- **SDK Usage Pattern:** 
  - High-level APIs for standard operations (indices, templates)
  - `PerformRequest` for OpenSearch plugin APIs (security, ISM, alerting)

### Files Requiring Changes
```
provider/
├── provider.go              # Client initialization
├── http.go                  # HTTP transport wrappers
├── awsv4.go                 # AWS signing (replace)
├── client.go                # NEW: Client factory
├── resource_opensearch_*.go # 25 resource files
└── *_test.go                # Test files
```

## Target State Architecture

### New Dependencies
```go
// Target (go.mod)
github.com/opensearch-project/opensearch-go/v4 v4.6.0
github.com/aws/aws-sdk-go-v2 v1.32.0
github.com/aws/aws-sdk-go-v2/config v1.28.0
github.com/aws/aws-sdk-go-v2/credentials v1.17.0
```

### Client Architecture
```
┌─────────────────────────────────────────┐
│         Terraform Provider              │
│         (ProviderConf)                  │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│     OpenSearchClient Wrapper            │
│  (Encapsulates opensearchapi.Client)    │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│    opensearchapi.Client                 │
│  (Official OpenSearch Go SDK)           │
└──────────────┬──────────────────────────┘
               │
    ┌──────────┴──────────┐
    ▼                     ▼
┌──────────┐        ┌──────────┐
│ HTTP     │        │ AWS SigV4│
│ Transport│        │ Signer   │
│ (Custom) │        │ (SDK v2) │
└──────────┘        └──────────┘
```

## Migration Phases

### Phase 1: Foundation (Weeks 1-2)

#### 1.1 Update Dependencies ✅ COMPLETE

**Status:** Completed on February 15, 2026

**Changes Made:**
- Added `github.com/opensearch-project/opensearch-go/v4 v4.6.0` to go.mod
- Ran `go mod tidy` to fetch dependencies
- All builds and tests passing

**Dependencies Added:**
- `github.com/opensearch-project/opensearch-go/v4 v4.6.0`

**Note:** Old dependencies (olivere/elastic, aws-sdk-go v1) temporarily remain in go.mod until all source code imports are migrated in later phases.

**Target Dependencies (to be fully activated in Phase 3):**
- `github.com/opensearch-project/opensearch-go/v4 v4.6.0` ✅
- `github.com/aws/aws-sdk-go-v2 v1.32.0` (pending Phase 1.3)
- `github.com/aws/aws-sdk-go-v2/config v1.28.0` (pending Phase 1.3)
- `github.com/aws/aws-sdk-go-v2/credentials v1.17.0` (pending Phase 1.3)

#### 1.2 Create New Client Factory ✅ COMPLETE

**Status:** Completed on February 15, 2026

**File Created:** `provider/client.go` (170 lines)

**Implementation Details:**
- Created `OpenSearchClient` struct wrapping `*opensearchapi.Client`
- Implemented `NewOpenSearchClient(conf *ProviderConf)` function
- Supports all authentication methods:
  - Basic auth (username/password)
  - URL-based credentials extraction
  - Token-based auth (ApiKey/Bearer)
  - AWS SigV4 (stub for Phase 1.3)
- TLS configuration:
  - Client certificates
  - CA certificate validation
  - Insecure mode support
  - ServerName override for host
- Custom transport wrappers:
  - `tokenTransport` - Adds Authorization header
  - `hostOverrideTransport` - Overrides Host header
- Proxy support

**Code Location:** `/Users/zampettim/git/terraform/terraform-provider-opensearch/provider/client.go`

**Test Results:**
- ✅ Build: Successful
- ✅ Unit Tests: All 13 tests passing
- ✅ Go Vet: No issues
- ✅ Static Analysis: Clean

**Bug Fixes During Implementation:**
Fixed 3 pre-existing issues in `provider/resource_opensearch_user.go` (lines 143, 158, 159):
- Changed non-constant format strings in `log.Printf` calls to proper format specifiers
- **Before:** `log.Printf("path is " + string(path))`
- **After:** `log.Printf("path is %s", path)`

**API Usage:**
```go
client, err := NewOpenSearchClient(conf)
if err != nil {
    return err
}
// Use client.Indices.Create(), client.Cluster.GetSettings(), etc.
```

**Note:** The new client is ready but not yet integrated into the provider. Existing code continues to use olivere/elastic client for backward compatibility.

#### 1.3 AWS Signer Migration ✅ COMPLETE

**Status:** Completed on February 15, 2026

**File Created:** `provider/aws_signer_v2.go` (93 lines)

**Implementation Details:**
- Created `newAWSSigner(conf *ProviderConf)` function using AWS SDK v2
- Supports both OpenSearch Service (`es`) and Serverless (`aoss`)
- Handles all credential scenarios in priority order:
  1. **Static credentials** - Access key ID, secret key, session token
  2. **Assume Role** - With optional external ID support
  3. **Named Profile** - AWS profile from config
  4. **Default chain** - Environment variables, EC2 metadata, etc.

**Dependencies Added:**
- `github.com/aws/aws-sdk-go-v2 v1.41.0`
- `github.com/aws/aws-sdk-go-v2/config v1.32.6`
- `github.com/aws/aws-sdk-go-v2/credentials v1.19.6`
- `github.com/aws/aws-sdk-go-v2/service/sts v1.41.5`
- `github.com/aws/aws-sdk-go-v2/credentials/stscreds` (for assume role)

**Code Location:** `/Users/zampettim/git/terraform/terraform-provider-opensearch/provider/aws_signer_v2.go`

**Key Implementation:**
```go
func newAWSSigner(conf *ProviderConf) (signer.Signer, error) {
    ctx := context.Background()
    
    var opts []func(*config.LoadOptions) error
    
    // Handle credentials in priority order
    if conf.awsAccessKeyId != "" {
        // Static credentials
        creds := credentials.NewStaticCredentialsProvider(...)
        opts = append(opts, config.WithCredentialsProvider(creds))
    } else if conf.awsAssumeRoleArn != "" {
        // Assume role with STS
        creds := stscreds.NewAssumeRoleProvider(stsClient, conf.awsAssumeRoleArn, ...)
        opts = append(opts, config.WithCredentialsProvider(creds))
    } else if conf.awsProfile != "" {
        // Named profile
        opts = append(opts, config.WithSharedConfigProfile(conf.awsProfile))
    }
    
    cfg, err := config.LoadDefaultConfig(ctx, opts...)
    if err != nil {
        return nil, err
    }
    
    // Create signer for appropriate service
    if service == "aoss" {
        return awsv2.NewSignerWithService(cfg, service)
    }
    return awsv2.NewSigner(cfg)
}
```

**Test Results:**
- ✅ Build: Successful
- ✅ Unit Tests: All passing
- ✅ AWS credential tests: PASS (TestAWSCredsManualKey, TestAWSCredsNamedProfile, TestAWSCredsEnv, TestAWSCredsAssumeRole, etc.)

**Notes:**
- The old `awsv4.go` wrapper is deprecated but kept for backward compatibility
- All existing AWS credential configuration options are preserved
- AWS SDK v1 and `aws_signing_client` remain in codebase temporarily until Phase 3

### Phase 2: Provider Configuration (Weeks 2-3)

#### 2.1 Update ProviderConf
Modify `ProviderConf` to hold the new client type:

```go
type ProviderConf struct {
    // ... existing fields ...
    client *OpenSearchClient  // Replace *elastic7.Client
}
```

#### 2.2 Update getClient Function
Refactor `provider.go:296-427` to use new client factory while maintaining backward compatibility for provider schema.

#### 2.3 HTTP Transport Wrappers
Refactor `http.go` to work with opensearch-go transport:

```go
// Custom transport for header injection
type customTransport struct {
    base         http.RoundTripper
    hostOverride string
    headers      map[string]string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    if t.hostOverride != "" {
        req.Host = t.hostOverride
    }
    for k, v := range t.headers {
        req.Header.Set(k, v)
    }
    return t.base.RoundTrip(req)
}
```

### Phase 3: Resource Migration (Weeks 3-6)

#### 3.1 Index Management (resource_opensearch_index.go)

**Current:**
```go
osClient.CreateIndex(name).BodyJson(body).Do(ctx)
```

**New:**
```go
client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
    Index: name,
    Body:  strings.NewReader(body),
})
```

#### 3.2 Security APIs (resource_opensearch_role.go)

**Current:**
```go
path, _ := uritemplates.Expand("/_plugins/_security/api/roles/{name}", ...)
res, err := osClient.PerformRequest(ctx, elastic7.PerformRequestOptions{
    Method: "GET",
    Path:   path,
})
```

**New:**
```go
path := fmt.Sprintf("/_plugins/_security/api/roles/%s", name)
res, err := client.Client.Perform(ctx, opensearchapi.Request{
    Method: "GET",
    Path:   path,
})
```

#### 3.3 ISM Policies (resource_opensearch_ism_policy.go)

**Current:**
```go
res, err := osClient.PerformRequest(ctx, elastic7.PerformRequestOptions{
    Method:           "PUT",
    Path:             path,
    Body:             string(policyJSON),
    RetryStatusCodes: []int{http.StatusConflict},
    Retrier: elastic7.NewBackoffRetrier(...),
})
```

**New:**
```go
// opensearch-go/v4 has built-in retry support via transport config
res, err := client.Client.Perform(ctx, opensearchapi.Request{
    Method: "PUT",
    Path:   path,
    Body:   strings.NewReader(policyJSON),
})
```

#### 3.4 Cluster Settings (resource_opensearch_cluster_settings.go)

**Current:**
```go
osClient.PerformRequest(ctx, elastic7.PerformRequestOptions{
    Method: "PUT",
    Path:   "/_cluster/settings",
    Body:   string(body),
})
```

**New:**
```go
client.Cluster.PutSettings(ctx, opensearchapi.ClusterPutSettingsReq{
    Body: strings.NewReader(body),
})
```

### Phase 4: Error Handling (Weeks 5-6)

#### 4.1 Error Type Mapping

| olivere/elastic | opensearch-go/v4 |
|-----------------|------------------|
| `elastic7.IsNotFound(err)` | Check `res.StatusCode == 404` |
| `elastic7.IsConflict(err)` | Check `res.StatusCode == 409` |
| `elastic7.IsUnauthorized(err)` | Check `res.StatusCode == 401` |

Create helper functions in `util.go`:

```go
func IsNotFound(err error) bool {
    if err == nil {
        return false
    }
    // Check if it's a 404 response error
    var respErr *opensearch.ResponseError
    if errors.As(err, &respErr) {
        return respErr.StatusCode == http.StatusNotFound
    }
    return false
}
```

### Phase 5: Testing & Validation (Weeks 6-8)

#### 5.1 Unit Tests
- Mock opensearchapi.Client for unit tests
- Update all test files to use new client interface

#### 5.2 Integration Tests
Test against:
- OpenSearch 1.x
- OpenSearch 2.x
- AWS OpenSearch Service
- AWS OpenSearch Serverless

#### 5.3 Acceptance Tests
Run full acceptance test suite:
```bash
make testacc
```

## Detailed API Migration Mapping

### Index Management

| Operation | olivere/elastic v7 | opensearch-go/v4 |
|-----------|-------------------|------------------|
| Create Index | `client.CreateIndex(name).BodyJson(body).Do(ctx)` | `client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{Index: name, Body: strings.NewReader(body)})` |
| Delete Index | `client.DeleteIndex(name).Do(ctx)` | `client.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{Indices: []string{name}})` |
| Get Settings | `client.IndexGetSettings(index).FlatSettings(true).Do(ctx)` | `client.Indices.GetSettings(ctx, &opensearchapi.IndicesGetSettingsReq{Indices: []string{index}})` |
| Put Settings | `client.IndexPutSettings(name).BodyJson(body).Do(ctx)` | `client.Indices.PutSettings(ctx, opensearchapi.IndicesPutSettingsReq{Indices: []string{name}, Body: strings.NewReader(body)})` |
| Get Template | `client.IndexGetTemplate(name).Do(ctx)` | `client.Indices.GetIndexTemplate(ctx, &opensearchapi.IndicesGetIndexTemplateReq{Name: []string{name}})` |
| Put Template | `client.IndexPutTemplate(name).BodyJson(body).Do(ctx)` | `client.Indices.PutIndexTemplate(ctx, opensearchapi.IndicesPutIndexTemplateReq{Name: name, Body: strings.NewReader(body)})` |

### Document Operations

| Operation | olivere/elastic v7 | opensearch-go/v4 |
|-----------|-------------------|------------------|
| Count | `client.Count(index).Do(ctx)` | `client.Count(ctx, &opensearchapi.CountReq{Indices: []string{index}})` |

### Ingest & Cluster

| Operation | olivere/elastic v7 | opensearch-go/v4 |
|-----------|-------------------|------------------|
| Get Pipeline | `client.IngestGetPipeline(id).Do(ctx)` | `client.Ingest.GetPipeline(ctx, &opensearchapi.IngestGetPipelineReq{PipelineID: id})` |
| Put Pipeline | `client.IngestPutPipeline(id).BodyJson(body).Do(ctx)` | `client.Ingest.PutPipeline(ctx, opensearchapi.IngestPutPipelineReq{PipelineID: id, Body: strings.NewReader(body)})` |
| Get Cluster Settings | `client.PerformRequest(ctx, elastic7.PerformRequestOptions{Method: "GET", Path: "/_cluster/settings"})` | `client.Cluster.GetSettings(ctx, nil)` |
| Put Cluster Settings | `client.PerformRequest(ctx, elastic7.PerformRequestOptions{Method: "PUT", Path: "/_cluster/settings", Body: body})` | `client.Cluster.PutSettings(ctx, opensearchapi.ClusterPutSettingsReq{Body: strings.NewReader(body)})` |

### OpenSearch Plugin APIs

All plugin APIs use `PerformRequest` pattern:

**Current:**
```go
path, _ := uritemplates.Expand("/_plugins/_security/api/roles/{name}", 
    map[string]string{"name": roleName})
res, err := osClient.PerformRequest(ctx, elastic7.PerformRequestOptions{
    Method: "GET",
    Path:   path,
})
```

**New:**
```go
path := fmt.Sprintf("/_plugins/_security/api/roles/%s", url.PathEscape(roleName))
res, err := client.Client.Perform(ctx, opensearchapi.Request{
    Method: "GET",
    Path:   path,
})
```

## Resource Migration Priority

### Phase 3A: Core Resources (Week 3)
1. `resource_opensearch_index.go` - Most complex, foundational
2. `resource_opensearch_index_template.go`
3. `resource_opensearch_component_template.go`

### Phase 3B: Cluster & Settings (Week 4)
4. `resource_opensearch_cluster_settings.go`
5. `resource_opensearch_script.go`
6. `resource_opensearch_ingest_pipeline.go`

### Phase 3C: Security Resources (Week 5)
7. `resource_opensearch_role.go`
8. `resource_opensearch_roles_mapping.go`
9. `resource_opensearch_user.go`

### Phase 3D: Plugin Resources (Week 6)
10. `resource_opensearch_ism_policy.go`
11. `resource_opensearch_ism_policy_mapping.go`
12. `resource_opensearch_monitor.go`
13. `resource_opensearch_channel_configuration.go`
14. `resource_opensearch_anomaly_detection.go`
15. `resource_opensearch_audit_config.go`
16. `resource_opensearch_sm_policy.go`

### Phase 3E: Remaining Resources (Weeks 6-7)
17. `resource_opensearch_dashboard_object.go`
18. `resource_opensearch_dashboard_tenant.go`
19. `resource_opensearch_data_stream.go`
20. `resource_opensearch_destination.go`
21. `resource_opensearch_snapshot_repository.go`

## Implementation Strategy

**Option A: Big Bang (Not Recommended)**
- Migrate everything at once
- High risk, difficult to review
- All or nothing testing

**Option B: Gradual Migration (Recommended)**
- Migrate provider foundation first
- Migrate resources one at a time
- Use feature flags or dual-client approach temporarily
- Allows incremental testing and rollback

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Breaking behavior changes** | High | High | Comprehensive test suite, gradual rollout with feature flags |
| **AWS SigV4 incompatibilities** | Medium | High | Test against both managed and serverless endpoints |
| **Performance degradation** | Medium | Medium | Benchmark critical paths before/after migration |
| **Terraform state incompatibilities** | Low | Critical | Ensure resource IDs and attributes remain stable |
| **Test coverage gaps** | Medium | Medium | Require 80%+ coverage on migrated resources |
| **AWS SDK v2 breaking changes** | Medium | Medium | Follow AWS migration guide, test credential chains |
| **Missing SDK functionality** | Low | High | Verify all required APIs exist in opensearch-go/v4 |

## Testing Strategy

### Unit Testing
- Mock `opensearchapi.Client` interface for isolated resource tests
- Test error handling paths explicitly
- Validate request/response serialization

### Integration Testing Matrix

| OpenSearch Version | Deployment Type | Test Priority |
|-------------------|-----------------|---------------|
| 1.3.x | Self-hosted | High |
| 2.11.x | Self-hosted | Critical |
| 2.15.x | Self-hosted | Critical |
| 2.x | AWS Managed | Critical |
| 2.x | AWS Serverless | High |

### Acceptance Tests
```bash
# Full test suite
make testacc

# Specific resource tests
TF_ACC=1 go test ./provider -v -run TestAccOpensearchIndex

# AWS-specific tests
TF_ACC=1 go test ./provider -v -run TestAccAWS
```

## Breaking Changes Analysis

### Provider Configuration
**No breaking changes** - provider schema remains identical

### Resource Behavior
**Potential changes:**
- Error messages may differ in format
- HTTP retry behavior may vary (opensearch-go has transport-level retry)
- Some API responses may have different field ordering

### State Compatibility
**Maintained** - All resource IDs and attributes remain unchanged

## Rollback Strategy

### Git Branching
- Main development on `feature/opensearch-go-v4` branch
- Keep `main` branch stable with old SDK until migration complete
- Tag releases before major phase transitions

### Rollback Criteria
- Critical bugs affecting production workloads
- Performance regression >20%
- AWS authentication failures
- State corruption issues

### Rollback Procedure
1. Revert to last stable commit
2. Restore old `go.mod` dependencies
3. Redeploy provider
4. Notify users of rollback

## Implementation Checklist

### Phase 1: Foundation
- [ ] Update go.mod with new dependencies
- [ ] Create `provider/client.go` with client factory
- [ ] Implement AWS SDK v2 signer
- [ ] Update HTTP transport wrappers
- [ ] Add error helper functions

### Phase 2: Provider
- [ ] Refactor `getClient()` function
- [ ] Update `ProviderConf` struct
- [ ] Migrate logging configuration
- [ ] Test provider initialization

### Phase 3: Resources
- [ ] Migrate core resources (3)
- [ ] Migrate cluster resources (3)
- [ ] Migrate security resources (3)
- [ ] Migrate plugin resources (6)
- [ ] Migrate remaining resources (4)

### Phase 4: Testing
- [ ] Unit test coverage >80%
- [ ] Integration tests passing
- [ ] Acceptance tests passing
- [ ] Performance benchmarks acceptable

### Phase 5: Release
- [ ] Documentation updates
- [ ] Changelog entry
- [ ] Migration guide for users
- [ ] Beta release
- [ ] GA release

## Documentation Updates Required

1. **README.md** - Update dependency information
2. **CHANGELOG.md** - Document breaking changes
3. **docs/index.md** - Update provider configuration examples
4. **MIGRATION_GUIDE.md** - User-facing migration guide
5. **CONTRIBUTING.md** - Update development setup instructions

## Key API Differences to Watch

1. **Response Handling:** opensearch-go returns response structs vs olivere's raw responses
2. **Body Types:** opensearch-go uses `io.Reader`, olivere accepts strings/maps
3. **Context:** Both support context, but error handling differs
4. **Retry Logic:** opensearch-go has transport-level retry vs olivere's per-request retry

## Estimated Timeline

- **Week 1-2:** Foundation (dependencies, client factory)
- **Week 3-4:** Provider configuration and transport
- **Week 5-7:** Resource migration (3-4 resources per week)
- **Week 8:** Testing, validation, bug fixes

**Total: 8 weeks with 1-2 developers**

---

## Migration Complete Checklist

When all items below are complete, the migration is finished:

- [ ] All olivere/elastic imports removed
- [ ] All AWS SDK v1 imports removed
- [ ] All 25+ resources migrated
- [ ] All tests passing
- [ ] Integration tests passing on all supported OpenSearch versions
- [ ] AWS authentication tested (managed and serverless)
- [ ] Documentation updated
- [ ] Changelog published
- [ ] New version released

---

*Last Updated: February 2026*
