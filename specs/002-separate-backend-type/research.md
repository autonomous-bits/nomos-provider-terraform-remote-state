# Technical Research: Separate Backend Type from Provider Type

**Feature Branch**: `002-separate-backend-type`  
**Date**: 2025-12-31  
**Status**: Complete

## Overview

This feature is a refactoring of the existing provider configuration to separate CLI provider discovery (`type` field) from runtime backend selection (`backend_type` field). No new technologies or external dependencies are introduced - this leverages existing config parsing patterns and backend implementations from feature 001-tfstate-provider.

---

## 1. Configuration Parsing (Existing Pattern)

### Current Implementation

The existing `internal/config/config.go` already implements configuration parsing using `ParseConfig()`:

```go
func ParseConfig(configMap map[string]interface{}) (BackendConfig, error)
```

**Current Behavior**:
- Extracts `type` field for backend selection (conflicting with CLI usage)
- Validates backend type against allowlist ("local", "azurerm")
- Returns `BackendConfig` interface with `Type()` and `Raw()` methods
- Backend constructors use `Type()` to determine which backend to instantiate

### Required Changes

1. **Rename Field**: Extract `backend_type` instead of `type`
2. **Add Auto-detection**: Implement detection logic when `backend_type` is omitted
3. **Ignore CLI field**: Completely ignore any `type` field in configuration

---

## 2. Auto-detection Strategy

### Detection Rules

Based on spec requirements FR-003:

**Local Backend Detection**:
- Presence of `path` field → backend_type = "local"
- No other backend-specific keys present

**Azure Backend Detection**:
- Presence of `storage_account_name` + `container_name` → backend_type = "azurerm"
- May also have `key` field

**Precedence**:
1. Explicit `backend_type` field (if present)
2. Auto-detection from configuration keys
3. Error if neither present or ambiguous

### Detection Algorithm

```go
func detectBackendType(configMap map[string]interface{}) (string, error) {
    // Check for explicit backend_type first
    if bt, ok := configMap["backend_type"].(string); ok && bt != "" {
        return bt, nil
    }

    // Auto-detect from configuration keys
    hasPath := configMap["path"] != nil
    hasStorageAccount := configMap["storage_account_name"] != nil
    hasContainer := configMap["container_name"] != nil

    if hasPath && !hasStorageAccount && !hasContainer {
        return "local", nil
    }

    if hasStorageAccount && hasContainer {
        return "azurerm", nil
    }

    if hasPath && (hasStorageAccount || hasContainer) {
        return "", errors.New("ambiguous configuration: cannot determine backend type")
    }

    return "", errors.New("backend_type not specified and cannot be auto-detected")
}
```

### Edge Cases

| Scenario | Detection Result | Error Handling |
|----------|-----------------|----------------|
| Both `backend_type` and path | Use explicit `backend_type` | Log info: auto-detection skipped |
| Only `path` | Auto-detect "local" | Success |
| Only Azure keys | Auto-detect "azurerm" | Success |
| `path` + Azure keys | Cannot determine | Return InvalidArgument error |
| Neither `backend_type` nor recognizable keys | Cannot determine | Return InvalidArgument error with hint |
| `backend_type` = "unsupported" | Invalid | Return InvalidArgument with allowlist |

---

## 3. Backward Compatibility Considerations (Not Applicable)

Per user clarification: "This is not being used yet so backwards compatibility must not be considered."

**Implications**:
- No need for deprecation warnings
- No need to support legacy `type` field usage
- Clean implementation without compatibility shims
- Simpler testing (no legacy scenarios)

---

## 4. Testing Strategy

### Unit Tests (config_test.go)

**Test Cases Required**:

1. **Explicit backend_type**:
   - Valid backend types ("local", "azurerm")
   - Invalid backend type (error)
   - Empty backend_type (fall through to auto-detection)

2. **Auto-detection - Local**:
   - Config with only `path` → detects "local"
   - Config with `path` + other non-backend keys → detects "local"

3. **Auto-detection - Azure**:
   - Config with `storage_account_name` + `container_name` → detects "azurerm"
   - Config with all Azure keys (`storage_account_name`, `container_name`, `key`) → detects "azurerm"

4. **Auto-detection - Errors**:
   - Config with both `path` and Azure keys → error (ambiguous)
   - Config with neither `backend_type` nor recognizable keys → error
   - Config with partial Azure keys (only `storage_account_name` without `container_name`) → error

5. **Type field handling**:
   - Config with `type` field containing provider source → ignored
   - Config with `type` field containing backend name → ignored (not treated as backend_type)

6. **Precedence**:
   - Both `backend_type` and auto-detectable keys → uses explicit `backend_type`
   - `backend_type: "local"` with Azure keys present → error (conflict)

### Integration Tests

**Test Files to Update**:
- `internal/backend/local_test.go`: Update configs to use `backend_type: "local"`
- `internal/backend/azurerm_test.go`: Update configs to use `backend_type: "azurerm"`
- `internal/provider/provider_test.go`: Update provider initialization tests

**Integration Test Scenarios**:
1. Initialize provider with explicit `backend_type: "local"` + `path`
2. Initialize provider with only `path` (auto-detect local)
3. Initialize provider with explicit `backend_type: "azurerm"` + Azure keys
4. Initialize provider with only Azure keys (auto-detect azurerm)
5. Verify error messages for ambiguous/missing configuration

---

## 5. Documentation Updates

### Files Requiring Updates

| File | Change Required | Priority |
|------|----------------|----------|
| `docs/backend-configuration.md` | Replace `type` with `backend_type` in examples | High |
| `specs/001-tfstate-provider/quickstart.md` | Update configuration examples | High |
| `README.md` | Update quick start examples, add clarification about `type` vs `backend_type` | High |
| `internal/config/config.go` | Update godoc comments | Medium |
| `docs/error-handling.md` | Add new error scenarios for auto-detection | Medium |

### Documentation Examples

**Before** (incorrect):
```yaml
source:
  alias: 'tfstate'
  type: 'local'  # ❌ Conflicts with CLI usage
  path: './terraform.tfstate'
```

**After** (correct with explicit backend_type):
```yaml
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'  # CLI provider source
  version: '0.1.0'
  backend_type: 'local'  # Runtime backend selection
  path: './terraform.tfstate'
```

**After** (correct with auto-detection):
```yaml
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  # backend_type omitted - auto-detected from path
  path: './terraform.tfstate'
```

---

## 6. Implementation Checklist

### Phase 1: Config Package Changes

- [ ] Update `ParseConfig()` to extract `backend_type` instead of `type`
- [ ] Implement `detectBackendType()` helper function
- [ ] Update `validateBackendType()` to work with detected type
- [ ] Remove any special handling of `type` field for backend selection
- [ ] Update all error messages to reference `backend_type`

### Phase 2: Backend Updates

- [ ] Review `local.go` for any direct references to `type` field
- [ ] Review `azurerm.go` for any direct references to `type` field
- [ ] Update backend constructors if needed

### Phase 3: Test Updates

- [ ] Update all test configurations to use `backend_type`
- [ ] Add unit tests for auto-detection logic
- [ ] Add unit tests for error scenarios
- [ ] Add integration tests for both explicit and auto-detected configs
- [ ] Verify 80%+ coverage maintained

### Phase 4: Documentation Updates

- [ ] Update `docs/backend-configuration.md`
- [ ] Update `specs/001-tfstate-provider/quickstart.md`
- [ ] Update `README.md` with clear `type` vs `backend_type` explanation
- [ ] Update godoc comments in `config.go`
- [ ] Add auto-detection examples

---

## 7. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking existing configs | Low (provider not in use yet) | High | N/A - confirmed no users |
| Auto-detection ambiguity | Medium | Medium | Clear error messages with guidance |
| Missing test coverage | Low | Medium | TDD approach, require 80%+ coverage |
| Documentation drift | Medium | Low | Update all docs in same PR |
| Confusion about `type` vs `backend_type` | Medium | Low | Clear examples in README, error messages |

---

## 8. Success Metrics Alignment

Mapping implementation approach to spec success criteria:

- **SC-001**: Configuration field separation → Achieved by using `backend_type` for runtime, `type` for CLI
- **SC-002**: Auto-detection success rate → Implemented with clear rules for unambiguous cases
- **SC-003**: Existing tests pass → All test configs updated to use `backend_type`
- **SC-004**: Documentation clarity → Examples show both fields with explanations
- **SC-005**: Clear error messages → Error codes indicate backend type vs validation issues
- **SC-006**: Reduced verbosity → Auto-detection allows omitting `backend_type` when obvious

---

## 9. Alternatives Considered

### Alternative 1: Keep `type` for backend selection, use `provider_type` for CLI

**Rejected Because**: 
- Would require CLI changes
- Breaks convention used by nomos-provider-file
- `type` is semantically correct for provider source identification

### Alternative 2: No auto-detection, always require explicit `backend_type`

**Rejected Because**:
- User story P2 specifically requests auto-detection
- Reduces developer experience
- Increases configuration verbosity

### Alternative 3: Use different field names per backend (e.g., `local_backend`, `azure_backend`)

**Rejected Because**:
- More complex configuration schema
- Harder to extend for new backends
- Doesn't solve the CLI `type` field conflict

---

## Conclusion

All technical decisions have been made for implementing the `backend_type` field separation and auto-detection feature:

1. **Config Parsing**: Modify existing `ParseConfig()` to use `backend_type` with auto-detection fallback
2. **Auto-detection**: Simple rule-based detection using configuration key presence
3. **Testing**: Comprehensive unit and integration tests with 80%+ coverage
4. **Documentation**: Update all examples to show correct `type` vs `backend_type` usage
5. **No Backward Compatibility**: Clean implementation without compatibility shims

The implementation can proceed directly to design phase (data model and contracts).
