# Backend Configuration Guide

This guide provides detailed instructions for configuring backends in the Nomos Terraform Remote State Provider.

## Overview

Backends determine where and how the provider retrieves Terraform state files. The provider supports multiple backend types, each with specific configuration requirements and authentication methods.

## Supported Backends (MVP)

- [Local Filesystem](#local-backend)
- [Azure Blob Storage](#azure-backend)

**Future Backends**: AWS S3, Google Cloud Storage, HTTP, Terraform Cloud (planned for Phase 2+)

---

## Local Backend

The local backend reads Terraform state files from the local filesystem. This is useful for:
- Development and testing
- Single-machine workflows
- CI/CD pipelines with local state

### Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `backend_type` | string | No* | (auto-detected) | Must be `"local"` if specified |
| `path` | string | Yes | - | Path to terraform.tfstate file |
| `workspace` | string | No | `"default"` | Terraform workspace name |

\* The `backend_type` field is optional. If omitted, the backend type is automatically detected from configuration keys (presence of `path` → local backend). Explicit specification is recommended for clarity.

### Path Resolution

#### Default Workspace

For the default workspace (`workspace = "default"` or omitted), the provider reads from the exact path specified:

```csl
source tfstate = terraform-remote-state {
  type = "local"
  path = "./infra/terraform.tfstate"
  // Reads: ./infra/terraform.tfstate
}
```

#### Named Workspaces

For named workspaces, the provider follows Terraform's workspace directory structure and resolves paths using the `terraform.tfstate.d/<workspace>/` pattern:

```csl
source tfstate = terraform-remote-state {
  type = "local"
  path = "./infra/terraform.tfstate"
  workspace = "production"
  // Reads: ./infra/terraform.tfstate.d/production/terraform.tfstate
}
```

**Path Resolution Logic**:
1. Extract directory: `./infra/`
2. Extract filename: `terraform.tfstate`
3. Construct workspace path: `./infra/terraform.tfstate.d/<workspace>/terraform.tfstate`

### Security Considerations

- **Path Validation**: All paths are validated to prevent path traversal attacks (`../` patterns rejected)
- **Character Restrictions**: Only alphanumeric, dots, dashes, underscores, and forward slashes allowed
- **Absolute vs Relative**: Both absolute and relative paths supported
- **Workspace Names**: Must be alphanumeric with dashes/underscores only (no slashes)

### Examples

#### Basic Local Backend (Explicit backend_type)

```csl
source tfstate_infra = terraform-remote-state {
  backend_type = "local"
  path = "/var/terraform/infra/terraform.tfstate"
}

config App {
  vpc_id = tfstate_infra.vpc_id.value
}
```

#### Local Backend with Workspace

```csl
source tfstate_staging = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
  workspace = "staging"
}

config App {
  environment = "staging"
  vpc_id = tfstate_staging.vpc_id.value
}
```

#### Multiple Workspaces with Variables

```csl
// Use Nomos variable for workspace selection
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
  workspace = var.environment  // Pass via --var environment=dev
}

config App {
  environment = var.environment
  vpc_id = tfstate.vpc_id.value
}
```
#### Auto-Detected Local Backend (Recommended for Simple Cases)

When only local backend keys are present, the provider automatically detects `backend_type = "local"`:

```csl
// backend_type omitted - auto-detected as "local" from presence of "path" key
source tfstate_infra = terraform-remote-state {
  path = "/var/terraform/infra/terraform.tfstate"
}

config App {
  vpc_id = tfstate_infra.vpc_id.value
}
```

**Auto-Detection Rules**:
- If configuration contains `path` key (and no Azure keys) → detects as `local` backend
- Explicit `backend_type` always takes precedence over auto-detection
- Recommended: Use explicit `backend_type` for production configurations for clarity

**When to Use Auto-Detection**:
- ✅ Simple, single-backend configurations
- ✅ Development and testing
- ✅ When backend type is obvious from configuration

**When to Use Explicit backend_type**:
- ✅ Production configurations (clarity and explicitness)
- ✅ Complex multi-backend setups
- ✅ Team environments (reduces ambiguity)
### Common Errors

**Error**: `state file not found: /path/terraform.tfstate (workspace: default)`

**Cause**: State file doesn't exist at the specified path

**Solution**:
```bash
# Verify file exists
ls -l /path/terraform.tfstate

# For workspaces, check workspace directory
ls -l /path/terraform.tfstate.d/dev/terraform.tfstate
```

---

**Error**: `invalid path: path traversal not allowed`

**Cause**: Path contains `..` (path traversal attempt)

**Solution**: Use absolute paths or relative paths without `..`:
```csl
// Bad
path = "../../../etc/passwd"

// Good
path = "/absolute/path/terraform.tfstate"
path = "./relative/path/terraform.tfstate"
```

---

## Azure Backend

The Azure backend reads Terraform state files from Azure Blob Storage. This is useful for:
- Team collaboration with shared state
- Production deployments
- Multi-region infrastructure

### Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `backend_type` | string | No* | (auto-detected) | Must be `"azurerm"` if specified |
| `storage_account_name` | string | Yes | - | Azure storage account name |
| `container_name` | string | Yes | - | Blob container name |
| `key` | string | Yes | - | Blob path/key |

\* The `backend_type` field is optional. If omitted, the backend type is automatically detected from configuration keys (presence of `storage_account_name` + `container_name` → azurerm backend). Explicit specification is recommended for clarity.

### Authentication

The Azure backend uses **environment variables only** for authentication. Never put credentials in configuration files.

#### Service Principal Authentication (Recommended)

Set these environment variables:

```bash
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_ID="11111111-1111-1111-1111-111111111111"
export AZURE_CLIENT_SECRET="your-client-secret-here"
```

**Required Permissions**:
- Role: `Storage Blob Data Reader` or `Storage Blob Data Contributor`
- Scope: Storage account or specific container

#### Creating a Service Principal

```bash
# Create service principal
az ad sp create-for-rbac --name "nomos-provider-tfstate-reader"

# Output includes:
# - appId (use as AZURE_CLIENT_ID)
# - password (use as AZURE_CLIENT_SECRET)
# - tenant (use as AZURE_TENANT_ID)

# Grant Storage Blob Data Reader role
az role assignment create \
  --assignee <appId> \
  --role "Storage Blob Data Reader" \
  --scope "/subscriptions/<subscription-id>/resourceGroups/<rg>/providers/Microsoft.Storage/storageAccounts/<storage-account>"
```

#### Other Authentication Methods

The provider uses Azure's `DefaultAzureCredential`, which supports (in order):

1. **Environment Variables** (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)
2. **Managed Identity** (when running in Azure VM/AKS/Function)
3. **Azure CLI** (when using `az login`)
4. **Visual Studio Code** credentials
5. **Azure PowerShell** credentials

### Workspace Handling

**Important**: For Azure backend, workspace information is embedded directly in the `key` parameter.

The provider treats the key as an opaque string and does NOT manipulate it based on workspace parameters. You must specify the complete blob path including any workspace-specific segments.

#### Workspace Patterns

##### Default Workspace

```csl
source tfstate = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "terraform.tfstate"  // Default workspace state
}
```

##### Named Workspace with `env:` Prefix (Terraform Default)

Terraform's azurerm backend uses the `env:/<workspace>/` prefix for named workspaces:

```csl
source tfstate_dev = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "env:/dev/terraform.tfstate"  // Dev workspace
}

source tfstate_prod = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "env:/prod/terraform.tfstate"  // Prod workspace
}
```

##### Custom Workspace Patterns

```csl
// Workspaces directory pattern
source tfstate = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "workspaces/staging/terraform.tfstate"
}

// Application-specific pattern
source tfstate = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "apps/frontend/prod.tfstate"
}
```

### Validation Rules

#### Storage Account Name

- **Length**: 3-24 characters
- **Characters**: Lowercase letters and numbers only
- **Pattern**: `^[a-z0-9]{3,24}$`

Examples:
- ✅ `mytfstate123`
- ✅ `prodtfstate`
- ❌ `MyTfState` (uppercase not allowed)
- ❌ `my-tfstate` (hyphens not allowed)
- ❌ `ab` (too short)

#### Container Name

- **Length**: 3-63 characters
- **Characters**: Lowercase letters, numbers, and hyphens
- **Rules**:
  - Must start and end with letter or number
  - No consecutive hyphens
  - No hyphens at start or end
- **Pattern**: `^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`

Examples:
- ✅ `tfstate`
- ✅ `tf-state-prod`
- ✅ `state123`
- ❌ `TfState` (uppercase not allowed)
- ❌ `-tfstate` (starts with hyphen)
- ❌ `tf--state` (consecutive hyphens)

#### Blob Key

- **Length**: 1-1024 characters
- **Characters**: Letters, numbers, forward slashes, dots, hyphens, underscores, colons
- **Rules**:
  - Cannot start or end with forward slash
  - No backslashes
  - No path traversal (`..`)

Examples:
- ✅ `terraform.tfstate`
- ✅ `env:/prod/terraform.tfstate`
- ✅ `path/to/state.tfstate`
- ❌ `/terraform.tfstate` (starts with slash)
- ❌ `path/../other.tfstate` (path traversal)
- ❌ `path\to\state.tfstate` (backslashes)

### Examples

#### Basic Azure Backend (Explicit backend_type)

```csl
source tfstate_infra = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mycompanytfstate"
  container_name = "infrastructure-state"
  key = "network/terraform.tfstate"
}

config NetworkConfig {
  vpc_id = tfstate_infra.vpc_id.value
  subnets = tfstate_infra.subnet_ids.value
}
```

#### Multi-Environment with Variables

```csl
// Pass environment via --var env=prod
source tfstate = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mycompanytfstate"
  container_name = "tfstate"
  key = "env:/${var.env}/terraform.tfstate"
}

config App {
  environment = var.env
  vpc_id = tfstate.vpc_id.value
}
```

#### Multiple State Sources

```csl
// Network infrastructure (Azure)
source tfstate_network = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "prodtfstate"
  container_name = "tfstate"
  key = "network/terraform.tfstate"
}

// Database infrastructure (Azure, different workspace)
source tfstate_database = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "prodtfstate"
  container_name = "tfstate"
  key = "env:/prod/database.tfstate"
}

config App {
  // Use outputs from both states
  vpc_id = tfstate_network.vpc_id.value
  db_endpoint = tfstate_database.endpoint.value
}
```

#### Auto-Detected Azure Backend (Recommended for Simple Cases)

When only Azure backend keys are present, the provider automatically detects `backend_type = "azurerm"`:

```csl
// backend_type omitted - auto-detected as "azurerm" from presence of
// "storage_account_name" + "container_name" keys
source tfstate_infra = terraform-remote-state {
  storage_account_name = "mycompanytfstate"
  container_name = "infrastructure-state"
  key = "network/terraform.tfstate"
}

config NetworkConfig {
  vpc_id = tfstate_infra.vpc_id.value
  subnets = tfstate_infra.subnet_ids.value
}
```

**Auto-Detection Rules**:
- If configuration contains `storage_account_name` + `container_name` keys (and no local keys) → detects as `azurerm` backend
- Explicit `backend_type` always takes precedence over auto-detection
- Partial Azure configuration (only one key) results in error: `ErrCannotDetectBackend`
- Recommended: Use explicit `backend_type` for production configurations for clarity

**When to Use Auto-Detection**:
- ✅ Simple, single-backend configurations
- ✅ Development and testing
- ✅ When backend type is obvious from configuration

**When to Use Explicit backend_type**:
- ✅ Production configurations (clarity and explicitness)
- ✅ Complex multi-backend setups
- ✅ Team environments (reduces ambiguity)

### Common Errors

**Error**: `azure authentication failed`

**Cause**: Missing or invalid Azure credentials

**Solution**:
```bash
# Verify environment variables are set
echo $AZURE_TENANT_ID
echo $AZURE_CLIENT_ID
echo $AZURE_CLIENT_SECRET

# Test authentication with Azure CLI
az login --service-principal \
  --username $AZURE_CLIENT_ID \
  --password $AZURE_CLIENT_SECRET \
  --tenant $AZURE_TENANT_ID

# Verify access to storage account
az storage blob list \
  --account-name mytfstate \
  --container-name tfstate \
  --auth-mode login
```

---

**Error**: `blob not found: terraform.tfstate`

**Cause**: Blob doesn't exist at the specified key

**Solution**:
```bash
# List blobs in container
az storage blob list \
  --account-name mytfstate \
  --container-name tfstate \
  --auth-mode login

# Check if blob exists with different key
# Common issue: workspace path mismatch
```

---

**Error**: `invalid storage account name: must be 3-24 lowercase alphanumeric characters`

**Cause**: Storage account name doesn't meet Azure naming requirements

**Solution**: Use lowercase alphanumeric only, 3-24 characters:
```csl
// Bad
storage_account_name = "My-TfState"

// Good
storage_account_name = "mytfstate"
```

---

## Backend Comparison

| Feature | Local | Azure |
|---------|-------|-------|
| **Team Collaboration** | ❌ No | ✅ Yes |
| **State Locking** | ❌ No | ❌ No (read-only) |
| **Authentication** | None | Service Principal, Managed Identity, Azure CLI |
| **Workspace Support** | ✅ Directory-based | ✅ Key-based |
| **Network Required** | ❌ No | ✅ Yes |
| **Cost** | Free | Azure Storage costs |
| **Use Cases** | Development, CI/CD | Production, Teams |

## Best Practices

1. **Use Environment Variables**: Never hardcode credentials in configuration
2. **Least Privilege**: Grant only `Storage Blob Data Reader` role, not `Contributor`
3. **Workspace Consistency**: Use consistent workspace naming across teams
4. **Path Validation**: Always use forward slashes, avoid special characters
5. **Test Authentication**: Verify Azure credentials before deployment
6. **Absolute Paths**: Use absolute paths for local backend in production
7. **Key Patterns**: Document your blob key/workspace patterns for the team

## Security Checklist

- [ ] No credentials in configuration files
- [ ] Service principal has minimal required permissions
- [ ] Workspace names validated (no path traversal)
- [ ] Paths validated (no `../` patterns)
- [ ] Storage account access audited regularly
- [ ] Environment variables set securely (not in version control)
- [ ] Production uses Azure backend with proper RBAC

## Troubleshooting

For detailed troubleshooting steps, see [Error Handling Guide](error-handling.md).

For common backend configuration issues:

1. **Path Issues**: Verify file/blob exists with correct name and path
2. **Permission Issues**: Check RBAC roles and authentication
3. **Workspace Issues**: Ensure workspace path resolution is correct
4. **Network Issues**: Verify connectivity to Azure Storage (for Azure backend)

## Next Steps

- [Output Access Guide](output-access.md): Learn how to use Terraform outputs
- [Workspace Usage Guide](workspace-usage.md): Advanced workspace patterns
- [Error Handling](error-handling.md): Troubleshooting guide
