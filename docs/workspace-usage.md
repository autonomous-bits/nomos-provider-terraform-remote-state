# Workspace Usage Guide

This guide explains how to use Terraform workspaces with the Nomos Terraform Remote State Provider.

## Overview

Terraform workspaces allow you to manage multiple environments (dev, staging, production) with a single configuration. Each workspace has its own state file, enabling environment-specific infrastructure without duplicating Terraform code.

The provider supports workspace selection and handles workspace path resolution automatically based on the backend type.

## Workspace Concepts

### What Are Workspaces?

Workspaces are named environments within a single Terraform configuration. Each workspace maintains separate state, allowing you to deploy the same infrastructure definition to multiple environments.

**Common Use Cases**:
- Environment separation (dev, staging, production)
- Feature branch deployments
- Multi-region deployments
- Multi-tenant applications

### Default Workspace

Every Terraform configuration has a **default** workspace. When you initialize Terraform without explicitly creating workspaces, all state is stored in the default workspace.

```bash
# Initialize Terraform (creates default workspace)
terraform init

# Check current workspace
terraform workspace show
# Output: default
```

### Named Workspaces

Create named workspaces for different environments:

```bash
# Create and switch to development workspace
terraform workspace new development

# Create production workspace
terraform workspace new production

# List all workspaces
terraform workspace list

# Switch between workspaces
terraform workspace select development
terraform workspace select production
```

Each workspace stores state independently:
- Same Terraform code
- Different state files
- Environment-specific values via variables

---

## Local Backend Workspaces

### Directory Structure

The local backend uses the `terraform.tfstate.d/` directory structure for workspace state files:

```
project/
├── terraform.tfstate              # Default workspace
└── terraform.tfstate.d/
    ├── development/
    │   └── terraform.tfstate      # Development workspace
    ├── staging/
    │   └── terraform.tfstate      # Staging workspace
    └── production/
        └── terraform.tfstate      # Production workspace
```

### Path Resolution

The provider automatically resolves workspace paths:

#### Default Workspace

```csl
source tfstate = terraform-remote-state {
  type = "local"
  path = "./infra/terraform.tfstate"
  workspace = "default"  // or omit workspace parameter
}
// Reads: ./infra/terraform.tfstate
```

#### Named Workspace

```csl
source tfstate = terraform-remote-state {
  type = "local"
  path = "./infra/terraform.tfstate"
  workspace = "production"
}
// Reads: ./infra/terraform.tfstate.d/production/terraform.tfstate
```

**Path Resolution Formula**:
```
default:     <directory>/<filename>
named:       <directory>/terraform.tfstate.d/<workspace>/<filename>
```

### Examples

#### Single Environment (Default Workspace)

```csl
source tfstate_infra = terraform-remote-state {
  type = "local"
  path = "./infrastructure/terraform.tfstate"
  // No workspace parameter = default workspace
}

config App {
  vpc_id = tfstate_infra.vpc_id.value
}
```

#### Multiple Environments (Named Workspaces)

**Development Configuration** (`app-dev.csl`):
```csl
source tfstate_infra = terraform-remote-state {
  type = "local"
  path = "./infrastructure/terraform.tfstate"
  workspace = "development"
}

config App {
  environment = "development"
  vpc_id = tfstate_infra.vpc_id.value
  instance_type = "t3.micro"  // Small instance for dev
}
```

**Production Configuration** (`app-prod.csl`):
```csl
source tfstate_infra = terraform-remote-state {
  type = "local"
  path = "./infrastructure/terraform.tfstate"
  workspace = "production"
}

config App {
  environment = "production"
  vpc_id = tfstate_infra.vpc_id.value
  instance_type = "t3.large"  // Larger instance for prod
}
```

#### Dynamic Workspace Selection with Variables

```csl
// Pass workspace via Nomos variable
source tfstate_infra = terraform-remote-state {
  type = "local"
  path = "./infrastructure/terraform.tfstate"
  workspace = var.environment  // Set via --var environment=staging
}

config App {
  environment = var.environment
  vpc_id = tfstate_infra.vpc_id.value
  
  // Environment-specific configuration
  instance_count = var.environment == "production" ? 5 : 2
}
```

**Usage**:
```bash
# Compile for development
nomos compile app.csl --var environment=development

# Compile for staging
nomos compile app.csl --var environment=staging

# Compile for production
nomos compile app.csl --var environment=production
```

---

## Azure Backend Workspaces

### Key-Based Workspace Handling

Unlike the local backend, the Azure backend does **not** use a separate workspace parameter. Workspace information is embedded directly in the blob `key` (path).

The provider treats the key as an **opaque string** and does NOT manipulate it based on workspace. You must specify the complete blob path including any workspace-specific segments.

### Workspace Patterns

#### Pattern 1: `env:` Prefix (Terraform Default)

Terraform's azurerm backend uses the `env:/<workspace>/` prefix for named workspaces by default:

```bash
# Terraform backend configuration
terraform {
  backend "azurerm" {
    storage_account_name = "mytfstate"
    container_name       = "tfstate"
    key                  = "terraform.tfstate"  # Base key
  }
}

# When you select workspace "production"
# Terraform uses key: "env:/production/terraform.tfstate"
```

**Nomos Configuration**:

```csl
// Default workspace
source tfstate_default = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "terraform.tfstate"  // Default workspace
}

// Production workspace
source tfstate_prod = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "env:/production/terraform.tfstate"  // Workspace embedded in key
}

// Development workspace
source tfstate_dev = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "env:/development/terraform.tfstate"
}
```

#### Pattern 2: Workspaces Directory

Custom pattern using a `workspaces/` directory:

```csl
source tfstate_dev = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "workspaces/development/terraform.tfstate"
}

source tfstate_prod = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "workspaces/production/terraform.tfstate"
}
```

#### Pattern 3: Application-Specific Paths

Organize by application and environment:

```csl
source tfstate_frontend_prod = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "apps/frontend/production.tfstate"
}

source tfstate_backend_prod = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "apps/backend/production.tfstate"
}
```

### Dynamic Workspace Selection (Azure)

**Using Variables**:
```csl
// Pass environment via Nomos variable
source tfstate = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "env:/${var.environment}/terraform.tfstate"
}

config App {
  environment = var.environment
  vpc_id = tfstate.vpc_id.value
}
```

**Usage**:
```bash
nomos compile app.csl --var environment=production
nomos compile app.csl --var environment=staging
```

---

## Multi-Workspace Patterns

### Pattern 1: Separate Configuration Files

Create separate Nomos configuration files for each environment:

```
configs/
├── app-dev.csl       # Development environment
├── app-staging.csl   # Staging environment
└── app-prod.csl      # Production environment
```

**app-dev.csl**:
```csl
source tfstate = terraform-remote-state {
  type = "local"
  path = "./terraform.tfstate"
  workspace = "development"
}

config App {
  environment = "development"
  vpc_id = tfstate.vpc_id.value
}
```

### Pattern 2: Single Configuration with Variables

Use a single configuration file with environment passed as variable:

**app.csl**:
```csl
source tfstate = terraform-remote-state {
  type = "local"
  path = "./terraform.tfstate"
  workspace = var.environment
}

config App {
  environment = var.environment
  vpc_id = tfstate.vpc_id.value
  
  // Environment-specific settings
  debug_mode = var.environment == "development"
  instance_count = var.environment == "production" ? 5 : 2
}
```

### Pattern 3: Multi-Layer State

Different layers of infrastructure in different workspaces:

```csl
// Network layer (production workspace)
source tfstate_network = terraform-remote-state {
  type = "local"
  path = "./network/terraform.tfstate"
  workspace = "production"
}

// Application layer (production workspace)
source tfstate_app = terraform-remote-state {
  type = "local"
  path = "./application/terraform.tfstate"
  workspace = "production"
}

// Database layer (production workspace)
source tfstate_database = terraform-remote-state {
  type = "local"
  path = "./database/terraform.tfstate"
  workspace = "production"
}

config App {
  vpc_id = tfstate_network.vpc_id.value
  subnet_ids = tfstate_network.subnet_ids.value
  
  app_server_id = tfstate_app.server_id.value
  
  db_endpoint = tfstate_database.endpoint.value
}
```

---

## Workspace Validation

### Security Considerations

Workspace names are validated to prevent security issues:

**Allowed Characters**:
- Alphanumeric: `a-z`, `A-Z`, `0-9`
- Hyphens: `-`
- Underscores: `_`

**Not Allowed**:
- Path separators: `/`, `\`
- Path traversal: `..`
- Control characters or null bytes
- Special characters: `!`, `@`, `#`, etc.

**Examples**:
```csl
// ✅ Valid workspace names
workspace = "development"
workspace = "prod"
workspace = "staging-01"
workspace = "dev_us_west_2"

// ❌ Invalid workspace names
workspace = "../production"      // Path traversal
workspace = "prod/backup"        // Path separator
workspace = "test\environment"   // Backslash
```

### Length Limits

- **Maximum length**: 100 characters
- **Minimum length**: 1 character

### Error Messages

**Invalid workspace name**:
```
Error: invalid workspace name: must contain only alphanumeric characters, hyphens, and underscores
```

**Path traversal attempt**:
```
Error: path traversal detected: path traversal not allowed
```

---

## Best Practices

### 1. Consistent Naming

Use consistent workspace naming across your organization:

```bash
# Good: Consistent pattern
development
staging
production

# Avoid: Inconsistent patterns
dev
stg
prod-01
```

### 2. Document Workspace Strategy

Document your workspace strategy in your repository:

```markdown
# Workspace Strategy

- `default`: Local development only
- `development`: Shared development environment
- `staging`: Pre-production testing
- `production`: Production workloads

## State Location

- Local: `./terraform.tfstate.d/<workspace>/terraform.tfstate`
- Azure: `env:/<workspace>/terraform.tfstate`
```

### 3. Environment Variables for Flexibility

Use Nomos variables for dynamic workspace selection:

```csl
source tfstate = terraform-remote-state {
  type = "local"
  path = "./terraform.tfstate"
  workspace = var.environment
}
```

### 4. Separate Critical Workspaces

For production, consider:
- Separate storage accounts (Azure)
- Separate directories (Local)
- Stricter access controls
- Separate configuration files

### 5. Workspace-Specific Configurations

Adapt configurations based on workspace:

```csl
config App {
  environment = var.workspace
  
  // Production gets more resources
  instance_count = var.workspace == "production" ? 10 : 3
  instance_type = var.workspace == "production" ? "t3.large" : "t3.micro"
  
  // Enable monitoring in production only
  monitoring_enabled = var.workspace == "production"
  
  // More aggressive autoscaling in production
  autoscale_max = var.workspace == "production" ? 20 : 5
}
```

---

## Common Errors

### Local Backend Errors

**Error**: `state file not found: ./terraform.tfstate.d/production/terraform.tfstate (workspace: production)`

**Cause**: Workspace state file doesn't exist

**Solution**:
```bash
# Verify workspace exists in Terraform
cd terraform-directory
terraform workspace list

# If workspace doesn't exist, create it
terraform workspace new production
terraform apply
```

---

**Error**: `invalid workspace name: directory separators not allowed in workspace name`

**Cause**: Workspace name contains `/` or `\`

**Solution**: Use only alphanumeric, hyphens, underscores:
```csl
// Bad
workspace = "prod/us-west-2"

// Good
workspace = "prod-us-west-2"
workspace = "prod_us_west_2"
```

### Azure Backend Errors

**Error**: `blob not found: env:/production/terraform.tfstate`

**Cause**: Blob doesn't exist at the specified key

**Solution**: Verify the blob path matches your Terraform workspace setup:

```bash
# List blobs in container
az storage blob list \
  --account-name mytfstate \
  --container-name tfstate \
  --auth-mode login \
  | jq '.[].name'

# Check if Terraform is using env: prefix
# Look at your Terraform backend configuration
```

---

## Troubleshooting Checklist

- [ ] Workspace exists in Terraform (`terraform workspace list`)
- [ ] State file exists for the workspace
- [ ] Workspace name is valid (alphanumeric, hyphens, underscores only)
- [ ] Path resolution is correct (local: `terraform.tfstate.d/<workspace>/`)
- [ ] For Azure: blob key includes workspace path
- [ ] Terraform state version is 4+ (`terraform version`)
- [ ] Permissions allow reading state file/blob

## Next Steps

- [Backend Configuration](backend-configuration.md): Configure backends
- [Output Access](output-access.md): Access Terraform outputs
- [Error Handling](error-handling.md): Troubleshooting guide
