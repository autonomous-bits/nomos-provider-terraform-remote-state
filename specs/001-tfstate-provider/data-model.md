# Data Model: Terraform Remote State Provider

**Feature Branch**: `001-tfstate-provider`  
**Date**: 2025-12-30  
**Status**: Complete

## Overview

This document defines all entities, data structures, and type mappings for the Terraform Remote State Provider. These entities represent the core domain concepts that the provider manipulates.

---

## 1. StateFile Entity

Represents a parsed Terraform/OpenTofu state file.

### JSON Structure (from Terraform)

```json
{
  "version": 4,
  "terraform_version": "1.6.5",
  "serial": 123,
  "lineage": "550e8400-e29b-41d4-a716-446655440000",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    },
    "subnet_ids": {
      "value": ["subnet-1", "subnet-2"],
      "type": ["list", "string"],
      "sensitive": false
    }
  },
  "resources": []
}
```

### Go Type Definition

```go
// StateFile represents a Terraform/OpenTofu state file
type StateFile struct {
    Version          int                    `json:"version"`           // State format version (must be >= 4)
    TerraformVersion string                 `json:"terraform_version"` // Terraform/OpenTofu version that wrote state
    Serial           int                    `json:"serial"`            // State serial number (increments on changes)
    Lineage          string                 `json:"lineage"`           // UUID identifying state lineage
    Outputs          map[string]OutputValue `json:"outputs"`           // Root module outputs
    Resources        []Resource             `json:"resources,omitempty"` // State resources (for module output resolution)
}
```

### Validation Rules

- `Version` MUST be >= 4 (Terraform 0.12+, OpenTofu 1.x+)
- `Outputs` MAY be empty (valid state with no outputs)
- `Lineage` SHOULD be a valid UUID (not enforced)
- All fields are required except `Resources` (optional for MVP)

### Access Patterns

1. **Read outputs**: `state.Outputs[outputName]`
2. **Check version**: `if state.Version < 4 { return error }`
3. **List output names**: `for name := range state.Outputs { ... }`

---

## 2. OutputValue Entity

Represents a single Terraform output value.

### JSON Structure

```json
{
  "value": "vpc-12345",
  "type": "string",
  "sensitive": false
}
```

### Go Type Definition

```go
// OutputValue represents a Terraform output value
type OutputValue struct {
    Value     interface{} `json:"value"`     // Actual output value (any JSON type)
    Type      interface{} `json:"type"`      // Terraform type information (string or complex type)
    Sensitive bool        `json:"sensitive"` // Whether output is marked sensitive
}
```

### Supported Value Types

Based on Terraform type system and JSON compatibility:

| Terraform Type | JSON Type | Go Type | Example |
|---------------|-----------|---------|---------|
| string | string | `string` | `"vpc-12345"` |
| number | number | `float64` | `42`, `3.14` |
| bool | boolean | `bool` | `true`, `false` |
| list | array | `[]interface{}` | `["a", "b"]` |
| set | array | `[]interface{}` | `["x", "y"]` (unordered) |
| map | object | `map[string]interface{}` | `{"key": "value"}` |
| object | object | `map[string]interface{}` | `{"host": "localhost", "port": 5432}` |
| tuple | array | `[]interface{}` | `["string", 42, true]` |
| null | null | `nil` | `null` |

### Type Information Structure

The `Type` field can be:
- Simple string: `"string"`, `"number"`, `"bool"`
- List type: `["list", "string"]` or `["set", "number"]`
- Map type: `["map", "string"]`
- Object type: `["object", {"key1": "string", "key2": "number"}]`
- Tuple type: `["tuple", ["string", "number", "bool"]]`

**For MVP**: Type information is informational only; we pass values as-is without type coercion.

### Conversion to google.protobuf.Struct

```go
import "google.golang.org/protobuf/types/known/structpb"

// Convert output value to protobuf Struct
func (o *OutputValue) ToProtoStruct() (*structpb.Struct, error) {
    // Convert the value (interface{}) to protobuf Value
    value, err := structpb.NewValue(o.Value)
    if err != nil {
        return nil, fmt.Errorf("failed to convert value: %w", err)
    }
    
    // If value is a map/object, return its Struct representation
    if structValue := value.GetStructValue(); structValue != nil {
        return structValue, nil
    }
    
    // If value is a scalar/list, wrap in a Struct with "value" key
    return &structpb.Struct{
        Fields: map[string]*structpb.Value{
            "value": value,
        },
    }, nil
}
```

**Note**: For simple values (string, number, bool), the Fetch response wraps them in a struct with a "value" field for consistency with protobuf.Struct requirements.

---

## 3. BackendConfig Entity

Configuration for accessing Terraform state backends.

### Base Configuration

```go
// BackendConfig represents backend-agnostic configuration
type BackendConfig struct {
    Type      string `json:"type"`                // Backend type: "local", "azurerm"
    Workspace string `json:"workspace,omitempty"` // Workspace name (default: "default")
}
```

### Local Backend Configuration

Used for reading state files from local filesystem.

#### JSON Structure (from .csl config)

```json
{
  "type": "local",
  "path": "/path/to/terraform.tfstate",
  "workspace": "default"
}
```

#### Go Type Definition

```go
// LocalBackendConfig represents local filesystem backend configuration
type LocalBackendConfig struct {
    Type      string `json:"type"`                // Must be "local"
    Path      string `json:"path"`                // Required: path to state file
    Workspace string `json:"workspace,omitempty"` // Optional: workspace name (default: "default")
}
```

#### Validation Rules

- `Type` MUST be `"local"`
- `Path` MUST be non-empty string
- `Path` MUST point to readable file (checked during Init)
- `Workspace` defaults to `"default"` if omitted

#### Path Resolution

- **Absolute paths**: Used as-is (e.g., `/var/terraform/terraform.tfstate`)
- **Relative paths**: Resolved relative to CWD (e.g., `./terraform.tfstate`)
- **Workspace handling**:
  - Default workspace: Use `Path` directly
  - Named workspace: Construct path as `filepath.Join(filepath.Dir(Path), "terraform.tfstate.d", Workspace, filepath.Base(Path))`

**Example**:
- Config path: `./terraform.tfstate`
- Workspace: `dev`
- Resolved path: `./terraform.tfstate.d/dev/terraform.tfstate`

---

### Azure Blob Backend Configuration

Used for reading state files from Azure Blob Storage.

#### JSON Structure (from .csl config)

```json
{
  "type": "azurerm",
  "storage_account_name": "mystorageaccount",
  "container_name": "tfstate",
  "key": "prod/terraform.tfstate",
  "resource_group_name": "my-resource-group"
}
```

#### Go Type Definition

```go
// AzureBackendConfig represents Azure Blob Storage backend configuration
type AzureBackendConfig struct {
    Type               string `json:"type"`                 // Must be "azurerm"
    StorageAccountName string `json:"storage_account_name"` // Required: Azure storage account name
    ContainerName      string `json:"container_name"`       // Required: Blob container name
    Key                string `json:"key"`                  // Required: Blob name/path (includes workspace)
    ResourceGroupName  string `json:"resource_group_name,omitempty"` // Optional: Resource group name
    // Credentials from environment variables ONLY:
    // AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET
}
```

#### Validation Rules

- `Type` MUST be `"azurerm"`
- `StorageAccountName` MUST be non-empty, valid Azure storage account name (lowercase alphanumeric, 3-24 chars)
- `ContainerName` MUST be non-empty, valid container name (lowercase alphanumeric + hyphens, 3-63 chars)
- `Key` MUST be non-empty (blob path within container)
- `ResourceGroupName` is optional (not used by provider directly, informational only)

#### Authentication

**NEVER in config**. Always from environment variables:
- `AZURE_TENANT_ID`: Azure Active Directory tenant ID
- `AZURE_CLIENT_ID`: Service principal client ID  
- `AZURE_CLIENT_SECRET`: Service principal client secret
- (Optional) `AZURE_SUBSCRIPTION_ID`: Azure subscription

**SDK**: Use `azidentity.NewDefaultAzureCredential(nil)` which automatically reads these environment variables.

#### URL Construction

```go
accountURL := fmt.Sprintf("https://%s.blob.core.windows.net/", config.StorageAccountName)
```

#### Workspace Handling

For Azure backend, workspace is embedded in the `Key` path:
- Default workspace: `key = "terraform.tfstate"`
- Named workspace: `key = "env:/dev/terraform.tfstate"` or `key = "workspaces/dev/terraform.tfstate"`

The provider treats `Key` as opaque blob path (doesn't manipulate it based on workspace).

---

## 4. Workspace Entity

Represents a Terraform workspace context.

### Concept

Workspaces allow managing multiple state files for the same configuration:
- **Default workspace**: Named `"default"`, created automatically
- **Named workspaces**: User-defined names (e.g., `"dev"`, `"staging"`, `"prod"`)

### Go Type Definition

```go
// Workspace represents a Terraform workspace context
type Workspace struct {
    Name    string // Workspace name (default: "default")
    Backend string // Backend type ("local", "azurerm")
}
```

### Usage in Provider

```go
// Determine workspace name from config
workspace := config.Workspace
if workspace == "" {
    workspace = "default"
}

// Use workspace to construct state file path (backend-specific)
statePath := backend.ResolveWorkspacePath(workspace)
```

### Workspace Priority (P3)

For MVP:
- Accept `workspace` parameter in config
- Default to `"default"` if not specified
- For local backend: Resolve workspace-specific path
- For Azure backend: User specifies full blob path (workspace implicit)

---

## 5. Resource Entity (Optional - for Module Outputs)

Represents a resource in Terraform state (needed for nested module output resolution).

### JSON Structure

```json
{
  "mode": "data",
  "type": "terraform_remote_state",
  "name": "app",
  "provider": "provider[\"registry.terraform.io/hashicorp/terraform\"]",
  "instances": [
    {
      "attributes": {
        "outputs": {
          "database_url": "postgresql://..."
        }
      }
    }
  ]
}
```

### Go Type Definition (Simplified for MVP)

```go
// Resource represents a Terraform state resource (simplified)
type Resource struct {
    Mode      string                 `json:"mode"`      // "managed", "data"
    Type      string                 `json:"type"`      // Resource type
    Name      string                 `json:"name"`      // Resource name
    Module    string                 `json:"module,omitempty"` // Module path (e.g., "module.app")
    Instances []ResourceInstance     `json:"instances"` // Resource instances
}

// ResourceInstance represents a single resource instance
type ResourceInstance struct {
    Attributes map[string]interface{} `json:"attributes"` // Resource attributes
}
```

### Usage (P2 - Module Outputs)

For accessing nested module outputs:
- Path `["app", "database_url"]` → Search for module named "app", extract outputs
- Implementation deferred to Phase 2 (not required for MVP)

---

## 6. Path Notation

### Root Module Outputs

**Path**: `["output_name"]`  
**Example**: `["vpc_id"]` → returns `state.Outputs["vpc_id"].Value`

### Module Outputs (P2)

**Path**: `["module_name", "output_name"]`  
**Example**: `["app", "database_url"]` → returns `module.app.outputs["database_url"]`

### Path Validation

```go
// ValidatePath checks if a path is valid
func ValidatePath(path []string) error {
    if len(path) == 0 {
        return status.Error(codes.InvalidArgument, "path cannot be empty")
    }
    
    for i, segment := range path {
        if segment == "" {
            return status.Errorf(codes.InvalidArgument, "path segment %d is empty", i)
        }
    }
    
    return nil
}
```

---

## 7. Type Mappings

### Terraform → JSON → Go

| Terraform Type | JSON Representation | Go Type | Notes |
|---------------|---------------------|---------|-------|
| `string` | `"text"` | `string` | Direct mapping |
| `number` | `42` or `3.14` | `float64` | JSON numbers are float64 |
| `bool` | `true`/`false` | `bool` | Direct mapping |
| `list(string)` | `["a", "b"]` | `[]interface{}` | Elements as `string` |
| `set(string)` | `["x", "y"]` | `[]interface{}` | Unordered, treated as list |
| `map(string)` | `{"k": "v"}` | `map[string]interface{}` | String keys |
| `object({...})` | `{"f1": "v1"}` | `map[string]interface{}` | Arbitrary keys |
| `tuple([...])` | `["a", 1, true]` | `[]interface{}` | Mixed types |
| `null` | `null` | `nil` | Represents absent value |

### Go → google.protobuf.Struct

| Go Type | protobuf.Value Type | Notes |
|---------|---------------------|-------|
| `string` | `StringValue` | Direct |
| `float64` | `NumberValue` | All JSON numbers |
| `int` | `NumberValue` | Converted to float64 |
| `bool` | `BoolValue` | Direct |
| `nil` | `NullValue` | Represents null |
| `[]interface{}` | `ListValue` | Recursive conversion |
| `map[string]interface{}` | `StructValue` | Recursive conversion |

**Conversion Function**:
```go
import "google.golang.org/protobuf/types/known/structpb"

// ConvertToProtoValue converts any Go value to protobuf Value
func ConvertToProtoValue(v interface{}) (*structpb.Value, error) {
    return structpb.NewValue(v)
}

// ConvertToProtoStruct converts a map to protobuf Struct
func ConvertToProtoStruct(m map[string]interface{}) (*structpb.Struct, error) {
    return structpb.NewStruct(m)
}
```

---

## 8. Error States

### Backend Errors

| Condition | gRPC Code | Example Message |
|-----------|-----------|-----------------|
| State file not found | `NotFound` | `"state file not found at path: ./terraform.tfstate"` |
| Permission denied | `PermissionDenied` | `"permission denied reading state file"` |
| Backend unreachable | `Unavailable` | `"azure storage unreachable: connection timeout"` |
| Invalid credentials | `PermissionDenied` | `"authentication failed: invalid AZURE_CLIENT_SECRET"` |
| State version too old | `FailedPrecondition` | `"unsupported state version 3 (requires version >= 4)"` |

### Output Errors

| Condition | gRPC Code | Example Message |
|-----------|-----------|-----------------|
| Output not found | `NotFound` | `"output 'vpc_id' not found in state"` |
| Empty path | `InvalidArgument` | `"path cannot be empty"` |
| Module not found | `NotFound` | `"module 'app' not found in state"` |
| Provider not initialized | `FailedPrecondition` | `"provider not initialized: call Init first"` |

---

## 9. Configuration Examples

### Local Backend (.csl config)

```json
{
  "type": "local",
  "path": "./terraform.tfstate"
}
```

### Local Backend with Workspace

```json
{
  "type": "local",
  "path": "./terraform.tfstate",
  "workspace": "dev"
}
```

### Azure Backend (.csl config)

```json
{
  "type": "azurerm",
  "storage_account_name": "mytfstate",
  "container_name": "tfstate",
  "key": "prod/app/terraform.tfstate"
}
```

### Azure Backend with Workspace (implicit in key)

```json
{
  "type": "azurerm",
  "storage_account_name": "mytfstate",
  "container_name": "tfstate",
  "key": "workspaces/dev/terraform.tfstate"
}
```

---

## 10. State Transition Diagram

```
┌─────────────┐
│ Uninitialized│
└──────┬──────┘
       │ Init(config)
       ▼
┌─────────────┐
│ Initialized │◄──┐
└──────┬──────┘   │
       │          │
       │ Fetch()  │ Multiple Fetch calls
       ▼          │
┌─────────────┐   │
│  Fetching   ├───┘
└──────┬──────┘
       │ Shutdown()
       ▼
┌─────────────┐
│  Shutdown   │
└─────────────┘
```

**State Rules**:
- `Fetch()` can only be called in `Initialized` or `Fetching` state
- `Init()` can only be called once (from `Uninitialized` state)
- `Shutdown()` can be called from any state after initialization
- `Info()` and `Health()` can be called in any state (including `Uninitialized`)

---

## 11. Data Flow Diagram

```
┌──────────────┐
│  Nomos CLI   │
└──────┬───────┘
       │ Init(config)
       ▼
┌──────────────────┐
│  Provider Init   │
│  - Parse config  │
│  - Create backend│
└──────┬───────────┘
       │
       ▼
┌──────────────────┐      ┌─────────────────┐
│  Backend         │◄─────┤ Local/Azure     │
│  - Fetch state   │      │ - Read file/blob│
└──────┬───────────┘      └─────────────────┘
       │
       ▼
┌──────────────────┐
│  State Parser    │
│  - Parse JSON    │
│  - Extract outputs│
└──────┬───────────┘
       │
       │ Fetch(path)
       ▼
┌──────────────────┐
│  Output Resolver │
│  - Lookup path   │
│  - Convert type  │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│  Protobuf Struct │
│  - Return value  │
└──────────────────┘
```

---

## Conclusion

This data model provides:
1. ✅ Complete type definitions for all entities
2. ✅ Validation rules for each entity
3. ✅ Type mapping from Terraform → Go → Protobuf
4. ✅ Configuration schemas for all backend types
5. ✅ Error state definitions with gRPC codes
6. ✅ Path notation and resolution logic
7. ✅ State transition and data flow diagrams

All entities are ready for implementation with clear contracts and validation rules.
