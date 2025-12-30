# Backend Configuration Schemas

**Feature Branch**: `001-tfstate-provider`  
**Date**: 2025-12-30

## Overview

This document defines the configuration schemas for all supported backend types. These schemas are passed to the provider via the Init RPC's `config` parameter.

---

## Common Fields

All backend configurations MUST include:

```json
{
  "type": "<backend-type>"
}
```

**Field**: `type`  
**Required**: Yes  
**Type**: String  
**Valid Values**: `"local"`, `"azurerm"`  
**Purpose**: Identifies which backend implementation to use

---

## 1. Local Backend Schema

### Purpose
Read Terraform state from local filesystem.

### Configuration Schema

```json
{
  "type": "local",
  "path": "<path-to-state-file>",
  "workspace": "<workspace-name>"
}
```

### Field Definitions

#### `type`
- **Required**: Yes
- **Type**: String
- **Value**: MUST be `"local"`
- **Validation**: Reject if not exactly `"local"`

#### `path`
- **Required**: Yes
- **Type**: String
- **Description**: Path to Terraform state file
- **Examples**:
  - Absolute: `/var/terraform/terraform.tfstate`
  - Relative: `./terraform.tfstate`
  - Relative with subdirs: `../infra/terraform.tfstate`
- **Validation Rules**:
  - MUST be non-empty string
  - MUST resolve to existing file (checked during Init)
  - MUST be readable (checked during Init)
  - Path resolution:
    - Absolute paths used as-is
    - Relative paths resolved from current working directory
- **Error Handling**:
  - Empty path → `InvalidArgument`
  - File not found → `FailedPrecondition` (during Init)
  - Not readable → `PermissionDenied` (during Init)

#### `workspace`
- **Required**: No (optional)
- **Type**: String
- **Default**: `"default"`
- **Description**: Terraform workspace name
- **Validation Rules**:
  - If omitted, defaults to `"default"`
  - If provided, MUST be non-empty string
  - Valid workspace names: alphanumeric + hyphens/underscores
- **Path Resolution Logic**:
  - Default workspace (`"default"`): Use `path` directly
  - Named workspace: Construct path as:
    ```
    <dir>/terraform.tfstate.d/<workspace>/<basename>
    ```
    Where:
    - `<dir>` = directory containing `path`
    - `<workspace>` = workspace name
    - `<basename>` = filename from `path`
  - Example:
    - Config path: `./terraform.tfstate`
    - Workspace: `dev`
    - Resolved path: `./terraform.tfstate.d/dev/terraform.tfstate`

### Examples

#### Minimal Configuration (Default Workspace)
```json
{
  "type": "local",
  "path": "./terraform.tfstate"
}
```

#### With Named Workspace
```json
{
  "type": "local",
  "path": "./terraform.tfstate",
  "workspace": "dev"
}
```

#### Absolute Path
```json
{
  "type": "local",
  "path": "/var/terraform/prod/terraform.tfstate"
}
```

### Validation Example (Go)

```go
type LocalBackendConfig struct {
    Type      string `json:"type"`
    Path      string `json:"path"`
    Workspace string `json:"workspace,omitempty"`
}

func ValidateLocalConfig(config map[string]interface{}) (*LocalBackendConfig, error) {
    // Parse type
    typeVal, ok := config["type"].(string)
    if !ok || typeVal != "local" {
        return nil, status.Error(codes.InvalidArgument, "type must be 'local'")
    }
    
    // Parse path (required)
    path, ok := config["path"].(string)
    if !ok || path == "" {
        return nil, status.Error(codes.InvalidArgument, "path is required and must be non-empty")
    }
    
    // Parse workspace (optional)
    workspace := "default"
    if ws, ok := config["workspace"].(string); ok && ws != "" {
        workspace = ws
    }
    
    return &LocalBackendConfig{
        Type:      typeVal,
        Path:      path,
        Workspace: workspace,
    }, nil
}
```

---

## 2. Azure Blob Backend Schema

### Purpose
Read Terraform state from Azure Blob Storage.

### Configuration Schema

```json
{
  "type": "azurerm",
  "storage_account_name": "<storage-account>",
  "container_name": "<container>",
  "key": "<blob-path>",
  "resource_group_name": "<resource-group>"
}
```

### Field Definitions

#### `type`
- **Required**: Yes
- **Type**: String
- **Value**: MUST be `"azurerm"`
- **Validation**: Reject if not exactly `"azurerm"`

#### `storage_account_name`
- **Required**: Yes
- **Type**: String
- **Description**: Azure storage account name
- **Examples**: `"mytfstate"`, `"prodstorageaccount"`
- **Validation Rules**:
  - MUST be non-empty string
  - MUST be lowercase
  - MUST be 3-24 characters
  - MUST contain only alphanumeric characters
  - Validation regex: `^[a-z0-9]{3,24}$`
- **Error Handling**:
  - Empty → `InvalidArgument`
  - Invalid format → `InvalidArgument`
  - Account not accessible → `Unavailable` or `PermissionDenied` (during Init)

#### `container_name`
- **Required**: Yes
- **Type**: String
- **Description**: Blob container name
- **Examples**: `"tfstate"`, `"terraform-state"`
- **Validation Rules**:
  - MUST be non-empty string
  - MUST be lowercase
  - MUST be 3-63 characters
  - MUST start with letter or number
  - MUST contain only lowercase alphanumeric and hyphens
  - MUST NOT have consecutive hyphens
  - Validation regex: `^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`
- **Error Handling**:
  - Empty → `InvalidArgument`
  - Invalid format → `InvalidArgument`
  - Container not found → `NotFound` (during Fetch)

#### `key`
- **Required**: Yes
- **Type**: String
- **Description**: Blob name/path within container (state file key)
- **Examples**:
  - Simple: `"terraform.tfstate"`
  - With path: `"prod/app/terraform.tfstate"`
  - With workspace: `"workspaces/dev/terraform.tfstate"`
- **Validation Rules**:
  - MUST be non-empty string
  - Can contain forward slashes for path hierarchy
  - No format restrictions (any valid blob name)
- **Workspace Handling**:
  - Workspace is embedded in the key path (user responsibility)
  - Provider does NOT manipulate key based on workspace parameter
  - Example workspace keys:
    - Default: `"terraform.tfstate"`
    - Dev workspace: `"env:/dev/terraform.tfstate"` (Terraform convention)
    - Custom: `"workspaces/dev/terraform.tfstate"`
- **Error Handling**:
  - Empty → `InvalidArgument`
  - Blob not found → `NotFound` (during Fetch)

#### `resource_group_name`
- **Required**: No (optional)
- **Type**: String
- **Description**: Azure resource group name (informational only)
- **Usage**: NOT used by provider; included for compatibility with Terraform backend config
- **Validation**: No validation (ignored by provider)

### Authentication

**CRITICAL**: Credentials MUST come from environment variables, NEVER from config.

**Required Environment Variables**:
- `AZURE_TENANT_ID`: Azure Active Directory tenant ID
- `AZURE_CLIENT_ID`: Service principal client ID
- `AZURE_CLIENT_SECRET`: Service principal client secret

**Optional Environment Variables**:
- `AZURE_SUBSCRIPTION_ID`: Azure subscription ID

**Implementation**:
```go
import "github.com/Azure/azure-sdk-for-go/sdk/azidentity"

// DefaultAzureCredential automatically reads environment variables
credential, err := azidentity.NewDefaultAzureCredential(nil)
if err != nil {
    return status.Errorf(codes.PermissionDenied, 
        "azure authentication failed: ensure AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, and AZURE_TENANT_ID are set")
}
```

### Examples

#### Minimal Configuration
```json
{
  "type": "azurerm",
  "storage_account_name": "mytfstate",
  "container_name": "tfstate",
  "key": "terraform.tfstate"
}
```

#### With Path Hierarchy
```json
{
  "type": "azurerm",
  "storage_account_name": "prodtfstate",
  "container_name": "tfstate",
  "key": "prod/app/terraform.tfstate"
}
```

#### With Workspace in Key (Terraform Convention)
```json
{
  "type": "azurerm",
  "storage_account_name": "mytfstate",
  "container_name": "tfstate",
  "key": "env:/dev/terraform.tfstate"
}
```

#### With Resource Group (Ignored)
```json
{
  "type": "azurerm",
  "storage_account_name": "mytfstate",
  "container_name": "tfstate",
  "key": "terraform.tfstate",
  "resource_group_name": "my-rg"
}
```

### Validation Example (Go)

```go
type AzureBackendConfig struct {
    Type               string `json:"type"`
    StorageAccountName string `json:"storage_account_name"`
    ContainerName      string `json:"container_name"`
    Key                string `json:"key"`
    ResourceGroupName  string `json:"resource_group_name,omitempty"`
}

func ValidateAzureConfig(config map[string]interface{}) (*AzureBackendConfig, error) {
    // Parse type
    typeVal, ok := config["type"].(string)
    if !ok || typeVal != "azurerm" {
        return nil, status.Error(codes.InvalidArgument, "type must be 'azurerm'")
    }
    
    // Parse storage_account_name (required)
    storageAccount, ok := config["storage_account_name"].(string)
    if !ok || storageAccount == "" {
        return nil, status.Error(codes.InvalidArgument, "storage_account_name is required")
    }
    if !isValidStorageAccountName(storageAccount) {
        return nil, status.Error(codes.InvalidArgument, 
            "storage_account_name must be 3-24 lowercase alphanumeric characters")
    }
    
    // Parse container_name (required)
    container, ok := config["container_name"].(string)
    if !ok || container == "" {
        return nil, status.Error(codes.InvalidArgument, "container_name is required")
    }
    if !isValidContainerName(container) {
        return nil, status.Error(codes.InvalidArgument, 
            "container_name must be 3-63 lowercase alphanumeric with hyphens")
    }
    
    // Parse key (required)
    key, ok := config["key"].(string)
    if !ok || key == "" {
        return nil, status.Error(codes.InvalidArgument, "key is required")
    }
    
    // Parse resource_group_name (optional, ignored)
    resourceGroup, _ := config["resource_group_name"].(string)
    
    return &AzureBackendConfig{
        Type:               typeVal,
        StorageAccountName: storageAccount,
        ContainerName:      container,
        Key:                key,
        ResourceGroupName:  resourceGroup,
    }, nil
}

func isValidStorageAccountName(name string) bool {
    if len(name) < 3 || len(name) > 24 {
        return false
    }
    for _, c := range name {
        if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
            return false
        }
    }
    return true
}

func isValidContainerName(name string) bool {
    if len(name) < 3 || len(name) > 63 {
        return false
    }
    // Simplified validation (full regex check omitted for brevity)
    return true // Implement full validation per Azure naming rules
}
```

---

## 3. Future Backend Schemas (Out of Scope for MVP)

### S3 Backend (Future)
```json
{
  "type": "s3",
  "bucket": "<bucket-name>",
  "key": "<object-key>",
  "region": "<aws-region>"
}
```

**Authentication**: Environment variables `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`

### GCS Backend (Future)
```json
{
  "type": "gcs",
  "bucket": "<bucket-name>",
  "prefix": "<prefix>"
}
```

**Authentication**: Environment variable `GOOGLE_APPLICATION_CREDENTIALS`

### HTTP Backend (Future)
```json
{
  "type": "http",
  "address": "<url>",
  "lock_address": "<lock-url>",
  "unlock_address": "<unlock-url>"
}
```

**Authentication**: Headers or query parameters (TBD)

---

## Configuration Validation Checklist

For each backend type, the Init handler MUST:

1. ✅ **Parse Config**: Convert protobuf.Struct to map[string]interface{}
2. ✅ **Validate Type**: Ensure `type` field matches expected backend
3. ✅ **Validate Required Fields**: Ensure all required fields present and non-empty
4. ✅ **Validate Field Formats**: Check field values match naming rules
5. ✅ **Reject Unknown Fields**: Optionally warn about unrecognized fields
6. ✅ **Test Connectivity**: Verify backend is accessible (during Init)
7. ✅ **Check Credentials**: Verify environment variables set (for remote backends)
8. ✅ **Return Clear Errors**: Provide actionable error messages for validation failures

---

## Error Mapping

| Validation Error | gRPC Code | Example Message |
|-----------------|-----------|-----------------|
| Missing type field | `InvalidArgument` | `"config missing required field 'type'"` |
| Invalid type value | `InvalidArgument` | `"unsupported backend type: 's3'"` |
| Missing required field | `InvalidArgument` | `"config missing required field 'path'"` |
| Empty field value | `InvalidArgument` | `"field 'path' cannot be empty"` |
| Invalid field format | `InvalidArgument` | `"storage_account_name must be lowercase alphanumeric"` |
| File not found (local) | `FailedPrecondition` | `"state file not found: ./terraform.tfstate"` |
| Permission denied (local) | `PermissionDenied` | `"permission denied reading state file"` |
| Missing credentials (Azure) | `PermissionDenied` | `"AZURE_CLIENT_ID environment variable not set"` |
| Authentication failed (Azure) | `PermissionDenied` | `"azure authentication failed: invalid credentials"` |
| Connection failed (Azure) | `Unavailable` | `"failed to connect to azure storage: timeout"` |

---

## Configuration Parsing Example

```go
func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
    // Parse config as map
    config := req.Config.AsMap()
    
    // Determine backend type
    backendType, ok := config["type"].(string)
    if !ok {
        return nil, status.Error(codes.InvalidArgument, "config missing required field 'type'")
    }
    
    // Dispatch to appropriate backend validator
    var backend Backend
    var err error
    
    switch backendType {
    case "local":
        localConfig, err := ValidateLocalConfig(config)
        if err != nil {
            return nil, err
        }
        backend, err = NewLocalBackend(ctx, localConfig)
        
    case "azurerm":
        azureConfig, err := ValidateAzureConfig(config)
        if err != nil {
            return nil, err
        }
        backend, err = NewAzureBackend(ctx, azureConfig)
        
    default:
        return nil, status.Errorf(codes.InvalidArgument, 
            "unsupported backend type: %s (supported: local, azurerm)", backendType)
    }
    
    if err != nil {
        return nil, err // Backend constructors return gRPC errors
    }
    
    // Store backend
    p.backend = backend
    p.initialized = true
    
    return &providerv1.InitResponse{}, nil
}
```

---

## Testing Requirements

Each backend configuration schema MUST have:

1. **Valid Config Tests**: Test parsing of valid configurations
2. **Missing Field Tests**: Test error handling for each required field
3. **Invalid Format Tests**: Test validation of field formats
4. **Backend Construction Tests**: Test successful backend creation with valid config
5. **Environment Variable Tests** (Azure): Test credential reading from env vars

**Example Test Cases**:
```go
func TestLocalBackendConfig(t *testing.T) {
    tests := []struct {
        name    string
        config  map[string]interface{}
        wantErr bool
        errCode codes.Code
    }{
        {
            name: "valid config",
            config: map[string]interface{}{
                "type": "local",
                "path": "./terraform.tfstate",
            },
            wantErr: false,
        },
        {
            name: "missing type",
            config: map[string]interface{}{
                "path": "./terraform.tfstate",
            },
            wantErr: true,
            errCode: codes.InvalidArgument,
        },
        {
            name: "missing path",
            config: map[string]interface{}{
                "type": "local",
            },
            wantErr: true,
            errCode: codes.InvalidArgument,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := ValidateLocalConfig(tt.config)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateLocalConfig() error = %v, wantErr %v", err, tt.wantErr)
            }
            if err != nil && tt.wantErr {
                st, ok := status.FromError(err)
                if !ok || st.Code() != tt.errCode {
                    t.Errorf("Expected error code %v, got %v", tt.errCode, st.Code())
                }
            }
        })
    }
}
```

---

## Conclusion

This document provides:
- ✅ Complete schemas for local and azurerm backends
- ✅ Field-by-field validation rules
- ✅ Error handling specifications
- ✅ Authentication requirements
- ✅ Workspace handling logic
- ✅ Code examples for validation
- ✅ Testing requirements

All backend configurations are ready for implementation with clear validation logic and error handling.
