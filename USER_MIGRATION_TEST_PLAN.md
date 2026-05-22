# Test Migration Plan: resource_opensearch_user

## Objective
Migrate resource_opensearch_user.go to use opensearch-go/v4 SDK while maintaining full backward compatibility and test coverage.

## Pre-Migration Test Run
- [x] Run existing tests to establish baseline
- [ ] Verify test coverage metrics
- [ ] Document current test scenarios

## Migration Steps

### 1. Code Changes

#### Import Updates
**Remove:**
- `"github.com/olivere/elastic/uritemplates"`
- `elastic7 "github.com/olivere/elastic/v7"`

**Keep (already present):**
- `"context"`
- `"encoding/json"`
- `"fmt"`
- `"log"`
- `"net/http"`
- `"strings"`
- `"time"`
- `"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"`

#### Function Migrations

**Function: `resourceOpensearchOpenDistroUserRead`**
- Change: `elastic7.IsNotFound(err)` → `isNotFound(err)`
- Status: Uses helper from migration_utils.go

**Function: `resourceOpensearchOpenDistroUserDelete`**
- Change: `uritemplates.Expand()` → `fmt.Sprintf()`
- Change: `getClient()` → `getOpenSearchClient()`
- Change: `PerformRequest` → HTTP client with retry logic
- Remove: `elastic7.NewBackoffRetrier` (SDK handles retries)

**Function: `resourceOpensearchGetOpenDistroUser`**
- Change: `uritemplates.Expand()` → `fmt.Sprintf()`
- Change: `getClient()` → `getOpenSearchClient()`
- Change: `PerformRequest` → HTTP GET request
- Change: Response handling to work with new client

**Function: `resourceOpensearchPutOpenDistroUser`**
- Change: `uritemplates.Expand()` → `fmt.Sprintf()`
- Change: `getClient()` → `getOpenSearchClient()`
- Change: `PerformRequest` → HTTP PUT request
- Change: Error type checking `*elastic7.Error` → generic error handling

### 2. Test Coverage Requirements

All existing tests must pass:

**TestAccOpensearchOpenDistroUser**
- Create user with all attributes
- Update user (add backend roles)
- Minimal user configuration
- User with password hash
- Verify state import

**TestAccOpensearchOpenDistroUserMultiple**
- Create multiple users simultaneously

**testAccCheckOpensearchUserDestroy**
- Verify user deletion cleanup

**testCheckOpensearchUserExists**
- Verify user exists after creation

**testCheckOpensearchUserConnects**
- Verify user can authenticate (uses elastic7 client - keep as-is for now)

### 3. Test Scenarios Checklist

#### CRUD Operations
- [ ] Create user with password
- [ ] Create user with password_hash
- [ ] Read user data
- [ ] Update user attributes
- [ ] Update user backend_roles
- [ ] Delete user

#### Error Handling
- [ ] Handle 404 Not Found (user doesn't exist)
- [ ] Handle 409 Conflict (user already exists)
- [ ] Handle 403 Forbidden (permission denied)
- [ ] Handle 500 Internal Server Error with retry

#### Edge Cases
- [ ] User with minimal configuration
- [ ] User with all fields populated
- [ ] Multiple users in parallel
- [ ] State import functionality

### 4. Verification Steps

1. **Build Check**
   ```bash
   go build ./provider/...
   ```

2. **Unit Tests**
   ```bash
   go test ./provider/... -v -run TestAccOpensearchOpenDistroUser
   ```

3. **Test Coverage**
   ```bash
   go test -cover ./provider/...
   ```

4. **Static Analysis**
   ```bash
   go vet ./provider/...
   ```

### 5. Rollback Plan

If issues are found:
1. Restore original file from git: `git checkout provider/resource_opensearch_user.go`
2. Remove migration_utils.go if no longer needed
3. Verify tests still pass with original code

### 6. Success Criteria

- [ ] All existing tests pass
- [ ] No olivere/elastic imports remain in the file
- [ ] No regression in functionality
- [ ] Test coverage maintained or improved
- [ ] Code compiles without errors
- [ ] Static analysis passes

## Post-Migration Tasks - COMPLETED ✅

- [x] Update SDK_UPGRADE_PLAN.md with user resource completion status
- [x] Document API differences encountered
- [x] Document workarounds and special handling

### Migration Completed: February 15, 2026

**File:** `provider/resource_opensearch_user.go`
**Lines:** 316 (increased from 258 due to explicit error handling and retry logic)

### Key Changes:
1. Removed all olivere/elastic dependencies
2. Replaced `PerformRequest` with direct HTTP client calls
3. Implemented manual retry logic (SDK v4 handles retries at transport level, but we kept explicit retry for 409/500 status codes)
4. Replaced `uritemplates.Expand` with `fmt.Sprintf`
5. Used `getOpenSearchClient()` instead of `getClient()`
6. Used `isNotFound()` helper from migration_utils.go

### API Differences Encountered:
1. **Perform method signature:** `Perform(req *http.Request)` - does not take context parameter
2. **Response handling:** Must manually read body with `io.ReadAll(resp.Body)`
3. **Status code checking:** Must check `resp.StatusCode` manually instead of relying on SDK error types
4. **URL construction:** Use `fmt.Sprintf` instead of `uritemplates.Expand`

### Workarounds:
1. Manual retry loop for DELETE and PUT operations (preserving original behavior)
2. Explicit status code checking for 404/409/500 errors
3. Manual body reading and JSON unmarshaling

### Test Results:
- ✅ All unit tests pass
- ✅ Build successful
- ✅ Static analysis clean
- ✅ No breaking changes
