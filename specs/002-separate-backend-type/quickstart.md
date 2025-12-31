# Quick Start: Backend Type Configuration

**Feature**: Separate Backend Type from Provider Type  
**Branch**: `002-separate-backend-type`  
**Date**: 2025-12-31

## Overview

This guide demonstrates the updated provider configuration that separates CLI provider discovery (`type`) from runtime backend selection (`backend_type`).

---

## Key Concept

The provider configuration now uses two distinct fields for two distinct purposes:

| Field | Purpose | Used By | Values |
|-------|---------|---------|--------|
| `type` | Provider source identification | Nomos CLI | `autonomous-bits/nomos-provider-terraform-remote-state` |
| `backend_type` | Runtime backend selection | Provider | `local`, `azurerm`, or auto-detected |

---

## Basic Examples

### Local Backend (Explicit)

```yaml
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  backend_type: 'local'
  path: './terraform.tfstate'
```

### Local Backend (Auto-detected)

```yaml
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  path: './terraform.tfstate'  # backend_type auto-detected as "local"
```

### Azure Backend (Explicit)

```yaml
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  backend_type: 'azurerm'
  storage_account_name: 'mystorageacct'
  container_name: 'tfstate'
  key: 'prod.terraform.tfstate'
```

### Azure Backend (Auto-detected)

```yaml
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  storage_account_name: 'mystorageacct'  # backend_type auto-detected as "azurerm"
  container_name: 'tfstate'
  key: 'prod.terraform.tfstate'
```

---

## Auto-detection Rules

The provider automatically detects the backend type when `backend_type` is omitted:

### Local Backend Detection

✅ **Detected when**:
- `path` field is present
- No Azure keys (`storage_account_name`, `container_name`) are present

```yaml
path: './terraform.tfstate'  # ✅ Detects: backend_type = "local"
```

### Azure Backend Detection

✅ **Detected when**:
- `storage_account_name` AND `container_name` fields are present

```yaml
storage_account_name: 'myaccount'
container_name: 'tfstate'  # ✅ Detects: backend_type = "azurerm"
key: 'terraform.tfstate'
```

### Ambiguous Configuration (Error)

❌ **Error when**:
- Both local and Azure keys are present

```yaml
path: './terraform.tfstate'
storage_account_name: 'myaccount'  # ❌ ERROR: Ambiguous configuration
```

**Error Message**:
```
ambiguous backend configuration: both local 'path' and Azure keys present. 
Specify 'backend_type' explicitly.
```

### Cannot Detect (Error)

❌ **Error when**:
- No recognizable backend keys are present

```yaml
some_field: 'value'  # ❌ ERROR: Cannot detect backend type
```

**Error Message**:
```
backend_type not specified and cannot be auto-detected. 
Supported backends: local (requires 'path'), azurerm (requires 'storage_account_name' and 'container_name')
```

---

## Complete Working Examples

### Example 1: Local Development

**Scenario**: Reading Terraform state from local filesystem during development

**Configuration**:
```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  backend_type: 'local'
  path: './infrastructure/terraform.tfstate'

environments:
  dev:
    vpc_id: reference:infra:vpc_id
    subnet_ids: reference:infra:subnet_ids
```

**Usage**:
```bash
nomos init
nomos build -p config.csl
```

---

### Example 2: Azure Production

**Scenario**: Reading Terraform state from Azure Blob Storage in production

**Prerequisites**:
```bash
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"
```

**Configuration**:
```yaml
source:
  alias: 'prod-infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  backend_type: 'azurerm'
  storage_account_name: 'prodstorageacct'
  container_name: 'tfstate'
  key: 'production/terraform.tfstate'

environments:
  production:
    database_host: reference:prod-infra:rds_endpoint
    database_port: reference:prod-infra:rds_port
```

**Usage**:
```bash
nomos init
nomos build -p config.csl
```

---

### Example 3: Multi-environment with Auto-detection

**Scenario**: Multiple environments using auto-detected backends

**Configuration**:
```yaml
# Development - local state
source:
  alias: 'dev-infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  path: './dev/terraform.tfstate'  # Auto-detects "local"

# Production - Azure state
source:
  alias: 'prod-infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  storage_account_name: 'prodstorageacct'  # Auto-detects "azurerm"
  container_name: 'tfstate'
  key: 'prod/terraform.tfstate'

environments:
  dev:
    vpc_id: reference:dev-infra:vpc_id
  
  prod:
    vpc_id: reference:prod-infra:vpc_id
```

---

## Explicit vs Auto-detection

### When to Use Explicit `backend_type`

✅ **Use explicit `backend_type` when**:
- Configuration clarity is important (e.g., team onboarding)
- Preventing accidental auto-detection
- Configuration will be template-generated

```yaml
backend_type: 'local'  # Explicit for clarity
path: './terraform.tfstate'
```

### When to Rely on Auto-detection

✅ **Rely on auto-detection when**:
- Reducing configuration verbosity
- Backend type is obvious from keys
- Fast prototyping or development

```yaml
path: './terraform.tfstate'  # backend_type implicitly "local"
```

---

## Common Patterns

### Pattern 1: Workspace-aware Local Development

```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  path: './terraform.tfstate'
  workspace: 'dev'  # Use "dev" workspace
```

### Pattern 2: Azure with Workspace

```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  storage_account_name: 'mystorageacct'
  container_name: 'tfstate'
  key: 'env:prod/terraform.tfstate'
  workspace: 'default'
```

### Pattern 3: Multiple State Files

```yaml
# VPC state
source:
  alias: 'vpc'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  path: './terraform/vpc/terraform.tfstate'

# Database state
source:
  alias: 'database'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  path: './terraform/database/terraform.tfstate'

environments:
  app:
    vpc_id: reference:vpc:vpc_id
    db_endpoint: reference:database:rds_endpoint
```

---

## Error Handling

### Error 1: Missing backend_type and cannot auto-detect

**Configuration**:
```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  some_field: 'value'  # No recognizable backend keys
```

**Error**:
```
Error: backend_type not specified and cannot be auto-detected. 
Supported backends: local (requires 'path'), azurerm (requires 'storage_account_name' and 'container_name')
```

**Fix**: Add required fields for desired backend:
```yaml
path: './terraform.tfstate'  # For local
# OR
storage_account_name: 'myaccount'
container_name: 'tfstate'  # For azurerm
```

---

### Error 2: Ambiguous configuration

**Configuration**:
```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  path: './terraform.tfstate'
  storage_account_name: 'myaccount'  # ❌ Both local and Azure keys
```

**Error**:
```
Error: ambiguous backend configuration: both local 'path' and Azure keys present. 
Specify 'backend_type' explicitly.
```

**Fix**: Choose one backend and remove conflicting keys:
```yaml
# Option 1: Local only
path: './terraform.tfstate'

# Option 2: Azure only
storage_account_name: 'myaccount'
container_name: 'tfstate'
key: 'terraform.tfstate'
```

---

### Error 3: Unsupported backend type

**Configuration**:
```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  backend_type: 's3'  # ❌ Not supported yet
  bucket: 'my-bucket'
```

**Error**:
```
Error: unsupported backend type: "s3" (allowed: local, azurerm)
```

**Fix**: Use a supported backend type:
```yaml
backend_type: 'local'  # or 'azurerm'
```

---

## Migrating from Old Configuration

**Old Configuration** (Incorrect - `type` used for backend):
```yaml
source:
  alias: 'infra'
  type: 'local'  # ❌ Wrong - conflicts with CLI provider discovery
  path: './terraform.tfstate'
```

**New Configuration** (Correct):
```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'  # ✅ CLI provider source
  version: '0.1.0'
  backend_type: 'local'  # ✅ Runtime backend selection
  path: './terraform.tfstate'
```

**Or with auto-detection** (Even simpler):
```yaml
source:
  alias: 'infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  path: './terraform.tfstate'  # backend_type auto-detected
```

---

## Testing Your Configuration

### Step 1: Validate Configuration

```bash
nomos init
```

If configuration is correct, you'll see:
```
✓ Provider 'tfstate' initialized (backend: local)
```

Or with explicit type:
```
✓ Provider 'tfstate' initialized (backend: azurerm)
```

### Step 2: Test Fetching Outputs

```bash
nomos build -p config.csl
```

Successful output:
```
✓ Fetched vpc_id from tfstate
✓ Fetched subnet_ids from tfstate
✓ Build complete
```

---

## Summary

✅ **Do**:
- Use `type` for CLI provider source identification
- Use `backend_type` for runtime backend selection (or rely on auto-detection)
- Provide all required fields for your chosen backend

❌ **Don't**:
- Use `type` field for backend selection (it's for CLI only)
- Mix local and Azure backend keys without explicit `backend_type`
- Omit required backend-specific fields

**Quick Reference**:
- Local backend: Needs `path`
- Azure backend: Needs `storage_account_name` + `container_name` + `key`
- Auto-detection works when keys are unambiguous
- Explicit `backend_type` always takes precedence
