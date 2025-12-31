# Data Model: Backend Type Configuration

**Feature**: Separate Backend Type from Provider Type  
**Branch**: `002-separate-backend-type`  
**Date**: 2025-12-31

## Overview

This document defines the data structures and parsing logic for the refactored backend configuration that separates CLI provider source identification (`type`) from runtime backend selection (`backend_type`).

---

## 1. Configuration Structure

### Input Configuration (from Nomos .csl file)

The configuration is passed to the provider via the Init RPC as a `google.protobuf.Struct` (free-form JSON-like map).

#### With Explicit Backend Type

```json
{
  "backend_type": "local",
  "path": "./terraform.tfstate",
  "workspace": "default"
}
```

```json
{
  "backend_type": "azurerm",
  "storage_account_name": "mystorageacct",
  "container_name": "tfstate",
  "key": "prod.terraform.tfstate",
  "workspace": "default"
}
```

#### With Auto-detection (backend_type omitted)

```json
{
  "path": "./terraform.tfstate"
}
```
*Auto-detects: backend_type = "local"*

```json
{
  "storage_account_name": "mystorageacct",
  "container_name": "tfstate",
  "key": "prod.terraform.tfstate"
}
```
*Auto-detects: backend_type = "azurerm"*

### Fields NOT Used by Provider

The `type` field (if present) is reserved for CLI provider source identification and MUST be silently ignored by the provider. No validation or error checking is performed on this field:

```json
{
  "type": "autonomous-bits/nomos-provider-terraform-remote-state",  // ← Silently ignored by provider (CLI-only)
  "backend_type": "local",
  "path": "./terraform.tfstate"
}
```

---

## 2. Data Structures

### BackendConfig Interface

*No changes to existing interface:*

```go
type BackendConfig interface {
    Type() string                      // Returns backend type ("local" or "azurerm")
    Workspace() string                 // Returns workspace name (default: "default")
    Raw() map[string]interface{}       // Returns raw config for backend-specific parsing
}
```

### Config Implementation

*Updated to use `backend_type` field:*

```go
type Config struct {
    backendType string                 // Extracted from "backend_type" or auto-detected
    workspace   string                 // Extracted from "workspace" (default: "default")
    raw         map[string]interface{} // Complete config map
}

func (c *Config) Type() string {
    return c.backendType
}

func (c *Config) Workspace() string {
    return c.workspace
}

func (c *Config) Raw() map[string]interface{} {
    return c.raw
}
```

---

## 3. Auto-detection Logic

### Detection Rules

```go
// detectBackendType determines backend type from configuration keys
// Returns: backend type string, or error if cannot be determined
func detectBackendType(configMap map[string]interface{}) (string, error) {
    // Rule 1: Explicit backend_type takes precedence
    if bt, ok := configMap["backend_type"].(string); ok && bt != "" {
        return sanitizeString(bt), nil
    }

    // Rule 2: Auto-detect from configuration keys
    hasPath := configMap["path"] != nil
    hasStorageAccount := configMap["storage_account_name"] != nil
    hasContainer := configMap["container_name"] != nil

    // Local backend: has "path", no Azure keys
    if hasPath && !hasStorageAccount && !hasContainer {
        return "local", nil
    }

    // Azure backend: has "storage_account_name" AND "container_name"
    if hasStorageAccount && hasContainer {
        return "azurerm", nil
    }

    // Ambiguous: has both local and Azure keys
    if hasPath && (hasStorageAccount || hasContainer) {
        return "", ErrAmbiguousBackendConfig
    }

    // Cannot determine: no recognizable keys
    return "", ErrCannotDetectBackend
}
```

### Detection Decision Tree

```
Has "backend_type"?
├─ YES → Use explicit value (validate against allowlist)
└─ NO  → Auto-detect
       │
       Has "path"?
       ├─ YES → Has Azure keys (storage_account_name OR container_name)?
       │        ├─ YES → ERROR: Ambiguous (both local and Azure keys)
       │        └─ NO  → backend_type = "local"
       │
       └─ NO  → Has "storage_account_name" AND "container_name"?
                ├─ YES → backend_type = "azurerm"
                └─ NO  → ERROR: Cannot detect (insufficient keys)
```

---

## 4. Validation Rules

### Backend Type Validation

```go
var allowedBackendTypes = map[string]bool{
    "local":   true,
    "azurerm": true,
}

func validateBackendType(backendType string) error {
    if !allowedBackendTypes[backendType] {
        return fmt.Errorf("%w: %q (allowed: local, azurerm)", 
            ErrUnsupportedBackendType, backendType)
    }
    return nil
}
```

### Configuration Conflict Detection

When explicit `backend_type` is provided, verify it matches the configuration keys:

```go
func validateBackendConfigMatch(backendType string, configMap map[string]interface{}) error {
    hasPath := configMap["path"] != nil
    hasStorageAccount := configMap["storage_account_name"] != nil
    hasContainer := configMap["container_name"] != nil

    switch backendType {
    case "local":
        if hasStorageAccount || hasContainer {
            return fmt.Errorf("backend_type is 'local' but Azure keys are present")
        }
    case "azurerm":
        if hasPath {
            return fmt.Errorf("backend_type is 'azurerm' but local path is present")
        }
    }

    return nil
}
```

---

## 5. Error Definitions

### New Errors for Auto-detection

```go
var (
    // ErrAmbiguousBackendConfig is returned when configuration contains keys for multiple backends
    ErrAmbiguousBackendConfig = errors.New("ambiguous backend configuration")

    // ErrCannotDetectBackend is returned when backend_type is not specified and cannot be auto-detected
    ErrCannotDetectBackend = errors.New("backend_type not specified and cannot be auto-detected")

    // ErrBackendConfigMismatch is returned when explicit backend_type conflicts with config keys
    ErrBackendConfigMismatch = errors.New("backend_type conflicts with configuration keys")
)
```

### Error Message Examples

| Scenario | Error Code | Error Message |
|----------|-----------|---------------|
| Ambiguous config (both path and Azure keys) | InvalidArgument | `ambiguous backend configuration: both local 'path' and Azure keys present. Specify 'backend_type' explicitly.` |
| Cannot detect (no recognizable keys) | InvalidArgument | `backend_type not specified and cannot be auto-detected. Supported backends: local (requires 'path'), azurerm (requires 'storage_account_name' and 'container_name')` |
| Explicit type conflicts with keys | InvalidArgument | `backend_type is 'local' but Azure keys (storage_account_name, container_name) are present` |
| Unsupported backend_type | InvalidArgument | `unsupported backend type: "s3" (allowed: local, azurerm)` |

---

## 6. Parsing Flow

### Updated ParseConfig() Flow

```
1. Extract "backend_type" field (if present)
   ├─ If present → Sanitize and validate
   └─ If absent → Auto-detect from config keys

2. Validate backend type against allowlist
   └─ If invalid → Return ErrUnsupportedBackendType

3. If explicit backend_type provided:
   └─ Validate it matches configuration keys
      └─ If mismatch → Return ErrBackendConfigMismatch

4. Extract "workspace" field (default: "default")
   └─ Validate workspace name (security check)

5. Perform backend-specific validation
   ├─ Local: Validate "path" field
   └─ Azure: Validate Azure-specific fields

6. Return BackendConfig interface
```

---

## 7. State Transitions

### Configuration Processing States

```
┌─────────────────┐
│  Raw Config Map │
│  (from Init RPC)│
└────────┬────────┘
         │
         ▼
┌─────────────────────┐
│  Extract Fields     │
│  - backend_type     │
│  - workspace        │
│  - backend params   │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Determine Backend  │
│  Type               │
│  (explicit or       │
│   auto-detect)      │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Validate Backend   │
│  Type               │
│  (allowlist check)  │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Check Config/Type  │
│  Consistency        │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Validate Backend   │
│  Specific Config    │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Return             │
│  BackendConfig      │
└─────────────────────┘
```

---

## 8. Backward Compatibility

**Status**: Not Applicable

Per user confirmation, the provider is not yet in production use. No backward compatibility handling required.

**Implications**:
- Clean implementation without legacy code paths
- No need to support `type` field for backend selection
- Simpler testing (no legacy scenarios)

---

## 9. Integration with Existing Backend Implementations

### Backend Constructor Pattern

*No changes required to backend constructors:*

```go
func NewLocalBackend(config BackendConfig) (Backend, error) {
    // Uses config.Type() to verify it's a local backend
    // Uses config.Raw() to extract backend-specific fields
    // ...existing implementation unchanged...
}

func NewAzureBackend(config BackendConfig) (Backend, error) {
    // Uses config.Type() to verify it's an Azure backend
    // Uses config.Raw() to extract backend-specific fields
    // ...existing implementation unchanged...
}
```

### Provider Service Integration

*Minimal changes in provider.go:*

```go
func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
    // Parse configuration (updated to use backend_type)
    config, err := ParseConfig(req.Config.AsMap())
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid configuration: %v", err)
    }

    // Create backend based on type (unchanged logic)
    var backend Backend
    switch config.Type() {
    case "local":
        backend, err = NewLocalBackend(config)
    case "azurerm":
        backend, err = NewAzureBackend(config)
    default:
        return nil, status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", config.Type())
    }

    // ...rest of Init unchanged...
}
```

---

## 10. Test Data Structures

### Test Configuration Examples

```go
// Test cases for config_test.go
var configTestCases = []struct {
    name      string
    input     map[string]interface{}
    wantType  string
    wantError error
}{
    {
        name: "explicit local backend",
        input: map[string]interface{}{
            "backend_type": "local",
            "path": "./terraform.tfstate",
        },
        wantType: "local",
    },
    {
        name: "auto-detect local backend",
        input: map[string]interface{}{
            "path": "./terraform.tfstate",
        },
        wantType: "local",
    },
    {
        name: "explicit azurerm backend",
        input: map[string]interface{}{
            "backend_type": "azurerm",
            "storage_account_name": "myaccount",
            "container_name": "tfstate",
            "key": "terraform.tfstate",
        },
        wantType: "azurerm",
    },
    {
        name: "auto-detect azurerm backend",
        input: map[string]interface{}{
            "storage_account_name": "myaccount",
            "container_name": "tfstate",
            "key": "terraform.tfstate",
        },
        wantType: "azurerm",
    },
    {
        name: "ambiguous config (path + Azure keys)",
        input: map[string]interface{}{
            "path": "./terraform.tfstate",
            "storage_account_name": "myaccount",
            "container_name": "tfstate",
        },
        wantError: ErrAmbiguousBackendConfig,
    },
    {
        name: "cannot detect (no recognizable keys)",
        input: map[string]interface{}{
            "some_field": "value",
        },
        wantError: ErrCannotDetectBackend,
    },
    {
        name: "unsupported backend type",
        input: map[string]interface{}{
            "backend_type": "s3",
            "bucket": "my-bucket",
        },
        wantError: ErrUnsupportedBackendType,
    },
    {
        name: "backend_type conflicts with config (local with Azure keys)",
        input: map[string]interface{}{
            "backend_type": "local",
            "storage_account_name": "myaccount",
            "container_name": "tfstate",
        },
        wantError: ErrBackendConfigMismatch,
    },
}
```

---

## Summary

This data model defines:

1. **Configuration Structure**: Explicit `backend_type` field OR auto-detection from config keys
2. **Auto-detection Logic**: Rule-based detection using key presence
3. **Validation Rules**: Backend type allowlist, config consistency checks
4. **Error Handling**: Clear error types and messages for all failure scenarios
5. **Integration**: Maintains existing BackendConfig interface and backend constructors
6. **Testing**: Comprehensive test cases covering all scenarios

The data model maintains backward compatibility with existing backend implementations while cleanly separating CLI provider discovery from runtime backend selection.
