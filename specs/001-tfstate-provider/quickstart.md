# Quickstart Guide: Terraform Remote State Provider

**Feature Branch**: `001-tfstate-provider`  
**Date**: 2025-12-30  
**Provider Type**: `terraform-remote-state`

## Overview

This guide shows how to use the Terraform Remote State Provider with Nomos CLI to access Terraform/OpenTofu state outputs during compilation.

---

## Installation

### Download Provider Binary

Download the appropriate binary for your platform:

```bash
# Linux AMD64
curl -L https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/latest/download/nomos-provider-terraform-remote-state-linux-amd64 \
  -o ~/.nomos/providers/nomos-provider-terraform-remote-state

# macOS AMD64 (Intel)
curl -L https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/latest/download/nomos-provider-terraform-remote-state-darwin-amd64 \
  -o ~/.nomos/providers/nomos-provider-terraform-remote-state

# macOS ARM64 (Apple Silicon)
curl -L https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/latest/download/nomos-provider-terraform-remote-state-darwin-arm64 \
  -o ~/.nomos/providers/nomos-provider-terraform-remote-state

# Make executable
chmod +x ~/.nomos/providers/nomos-provider-terraform-remote-state
```

### Verify Installation

```bash
# Provider should print discovery info
~/.nomos/providers/nomos-provider-terraform-remote-state
# Expected output: PROVIDER_PORT=<port>
```

---

## Usage Patterns

### 1. Local Backend (Simple File Access)

**Use Case**: Read Terraform state from local filesystem.

#### Example Terraform State

File: `./infra/terraform.tfstate`
```json
{
  "version": 4,
  "terraform_version": "1.6.5",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    },
    "subnet_ids": {
      "value": ["subnet-1", "subnet-2", "subnet-3"],
      "type": ["list", "string"],
      "sensitive": false
    },
    "database_config": {
      "value": {
        "host": "db.example.com",
        "port": 5432,
        "database": "appdb"
      },
      "type": ["object", {"host": "string", "port": "number", "database": "string"}],
      "sensitive": false
    }
  }
}
```

#### Nomos Configuration

File: `app.csl`
```csl
// Declare Terraform state as external source
source tfstate_infra = terraform-remote-state {
  backend_type = "local"
  path = "./infra/terraform.tfstate"
}

// Use outputs in configuration
config MyApp {
  vpc_id = tfstate_infra.vpc_id
  subnets = tfstate_infra.subnet_ids
  
  database {
    host = tfstate_infra.database_config.host
    port = tfstate_infra.database_config.port
    name = tfstate_infra.database_config.database
  }
}
```

#### Compile

```bash
nomos compile app.csl
```

**Result**: Provider reads state file, extracts outputs, makes them available during compilation.

---

### 2. Local Backend with Workspace

**Use Case**: Read state from a specific Terraform workspace (dev, staging, prod).

#### Terraform Workspace Structure

```
infra/
├── terraform.tfstate              # Default workspace
└── terraform.tfstate.d/
    ├── dev/
    │   └── terraform.tfstate
    ├── staging/
    │   └── terraform.tfstate
    └── prod/
        └── terraform.tfstate
```

#### Nomos Configuration (Dev Workspace)

File: `app-dev.csl`
```csl
source tfstate_infra = terraform-remote-state {
  backend_type = "local"
  path = "./infra/terraform.tfstate"
  workspace = "dev"
}

config MyApp {
  environment = "dev"
  vpc_id = tfstate_infra.vpc_id
}
```

#### Nomos Configuration (Prod Workspace)

File: `app-prod.csl`
```csl
source tfstate_infra = terraform-remote-state {
  backend_type = "local"
  path = "./infra/terraform.tfstate"
  workspace = "prod"
}

config MyApp {
  environment = "prod"
  vpc_id = tfstate_infra.vpc_id
}
```

**Path Resolution**:
- Dev: `./infra/terraform.tfstate.d/dev/terraform.tfstate`
- Prod: `./infra/terraform.tfstate.d/prod/terraform.tfstate`

---

### 3. Azure Blob Storage Backend

**Use Case**: Read Terraform state from Azure Blob Storage (common in enterprise setups).

#### Prerequisites

Set Azure authentication environment variables:

```bash
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_ID="11111111-1111-1111-1111-111111111111"
export AZURE_CLIENT_SECRET="your-client-secret"
```

**Note**: These credentials should have "Storage Blob Data Reader" role on the storage account/container.

#### Nomos Configuration

File: `app.csl`
```csl
source tfstate_infra = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "prod/app/terraform.tfstate"
}

config MyApp {
  vpc_id = tfstate_infra.vpc_id
  region = tfstate_infra.region
}
```

#### Compile

```bash
nomos compile app.csl
```

**Result**: Provider authenticates with Azure using environment variables, downloads state blob, extracts outputs.

---

### 4. Azure Backend with Workspace

**Use Case**: Multiple environments in Azure Blob Storage.

#### Azure Blob Structure

```
mytfstate (storage account)
└── tfstate (container)
    ├── default/terraform.tfstate
    ├── env:/dev/terraform.tfstate       # Terraform convention
    ├── env:/staging/terraform.tfstate
    └── env:/prod/terraform.tfstate
```

#### Nomos Configuration (Dev)

```csl
source tfstate_infra = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "env:/dev/terraform.tfstate"
}

config MyApp {
  environment = "dev"
  // ... use outputs
}
```

**Note**: For Azure backend, workspace is embedded in the `key` path (not a separate parameter).

---

### 5. Multiple State Sources

**Use Case**: Compose configuration from multiple Terraform states (e.g., network, database, app).

#### Nomos Configuration

File: `app.csl`
```csl
// Network infrastructure state
source tfstate_network = terraform-remote-state {
  backend_type = "local"
  path = "./terraform/network/terraform.tfstate"
}

// Database infrastructure state
source tfstate_database = terraform-remote-state {
  backend_type = "local"
  path = "./terraform/database/terraform.tfstate"
}

// App infrastructure state (Azure)
source tfstate_app = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "app/terraform.tfstate"
}

config MyApp {
  // Network outputs
  vpc_id = tfstate_network.vpc_id
  subnet_ids = tfstate_network.private_subnet_ids
  
  // Database outputs
  database_endpoint = tfstate_database.primary_endpoint
  database_port = tfstate_database.port
  
  // App infrastructure outputs
  load_balancer_dns = tfstate_app.lb_dns_name
  security_group_id = tfstate_app.app_sg_id
}
```

---

## Common Usage Patterns

### Accessing Simple Outputs

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  // String output
  region = tfstate.aws_region
  
  // Number output
  instance_count = tfstate.instance_count
  
  // Boolean output
  enable_monitoring = tfstate.monitoring_enabled
}
```

### Accessing List Outputs

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  // List of strings
  availability_zones = tfstate.availability_zones
  subnet_ids = tfstate.subnet_ids
  
  // Iterate over list
  for subnet in tfstate.subnet_ids {
    // Use subnet
  }
}
```

### Accessing Map/Object Outputs

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  // Access object properties
  database_host = tfstate.database_config.host
  database_port = tfstate.database_config.port
  
  // Use entire object
  database_config = tfstate.database_config
}
```

### Conditional Configuration

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  // Conditional based on Terraform output
  monitoring_enabled = tfstate.environment == "prod"
  
  // Default value if output missing
  instance_type = tfstate.instance_type ?? "t3.micro"
}
```

---

## Troubleshooting

### Error: "state file not found"

**Problem**: Local state file doesn't exist at specified path.

**Solution**:
1. Verify path is correct: `ls -la ./terraform.tfstate`
2. Use absolute path if relative path doesn't work
3. Check if using workspace (state might be in `terraform.tfstate.d/`)

```csl
// Try absolute path
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "/absolute/path/to/terraform.tfstate"
}
```

### Error: "permission denied reading state file"

**Problem**: Insufficient permissions to read state file.

**Solution**:
```bash
# Check file permissions
ls -l ./terraform.tfstate

# Make readable
chmod 644 ./terraform.tfstate
```

### Error: "output 'vpc_id' not found in state"

**Problem**: Requested output doesn't exist in Terraform state.

**Solution**:
1. Check output exists in Terraform configuration:
   ```bash
   terraform output
   ```
2. Verify spelling matches exactly (case-sensitive)
3. Ensure Terraform has been applied (outputs won't exist in plan)

### Error: "azure authentication failed"

**Problem**: Missing or invalid Azure credentials.

**Solution**:
1. Verify environment variables are set:
   ```bash
   echo $AZURE_TENANT_ID
   echo $AZURE_CLIENT_ID
   echo $AZURE_CLIENT_SECRET
   ```
2. Ensure service principal has correct permissions:
   - Role: "Storage Blob Data Reader" or higher
   - Scope: Storage account or container
3. Test credentials with Azure CLI:
   ```bash
   az login --service-principal \
     --username $AZURE_CLIENT_ID \
     --password $AZURE_CLIENT_SECRET \
     --tenant $AZURE_TENANT_ID
   ```

### Error: "unsupported state version"

**Problem**: Terraform state file is too old (version < 4).

**Solution**:
1. Upgrade Terraform/OpenTofu to version 0.12+ or 1.x+
2. Run `terraform apply` to upgrade state format
3. For Terraform < 0.12, manually migrate state:
   ```bash
   terraform 0.12upgrade
   terraform apply
   ```

### Error: "provider not initialized"

**Problem**: Nomos is calling Fetch before Init (internal error).

**Solution**: Report bug to Nomos maintainers (should not happen in normal usage).

---

## Best Practices

### 1. Use Absolute Paths for Production

**Bad**:
```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"  // Relative to CWD
}
```

**Good**:
```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "/var/terraform/prod/terraform.tfstate"
}
```

### 2. Keep Credentials in Environment Variables

**Bad**:
```csl
source tfstate = terraform-remote-state {
  backend_type = "azurerm"
  // NEVER put credentials in config!
  azure_client_secret = "secret-here"  // ❌ WRONG
}
```

**Good**:
```bash
# Set in environment before running nomos
export AZURE_CLIENT_SECRET="your-secret"
```

### 3. Use Workspaces for Environments

**Bad** (Separate configs for each environment):
```
app-dev.csl
app-staging.csl
app-prod.csl
```

**Good** (Single config with workspace parameter):
```csl
// Pass workspace as Nomos variable
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
  workspace = var.environment  // dev, staging, prod
}
```

```bash
nomos compile app.csl --var environment=dev
nomos compile app.csl --var environment=prod
```

### 4. Handle Missing Outputs Gracefully

**Bad**:
```csl
config App {
  subnet = tfstate.subnet_id  // Fails if output doesn't exist
}
```

**Good**:
```csl
config App {
  subnet = tfstate.subnet_id ?? "subnet-default"  // Fallback value
}
```

### 5. Document Required Outputs

Add comments to Nomos config documenting expected Terraform outputs:

```csl
// Required Terraform outputs:
// - vpc_id (string): VPC identifier
// - subnet_ids (list(string)): List of subnet IDs
// - database_config (object): Database connection details

source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./infra/terraform.tfstate"
}
```

---

## Performance Considerations

### No Caching (By Design)

The provider fetches state on **every Fetch RPC call** to ensure data freshness. This means:

- **Benefit**: Always get latest state
- **Trade-off**: Slower for many Fetch calls
- **Recommendation**: Group related outputs in single object if possible

### Local Backend Performance

- **Fast**: File I/O is quick for typical state files (< 10MB)
- **Expected**: < 100ms per Fetch for local files
- **Tip**: Use SSDs for faster I/O

### Azure Backend Performance

- **Slower**: Network latency + download time
- **Expected**: 200-500ms per Fetch (depending on region, size)
- **Tip**: Use Azure regions close to your location
- **Tip**: Keep state files small (< 1MB) for faster downloads

---

## Limitations (MVP)

### 1. Module Outputs Not Supported (P2 Feature)

**Current**: Only root module outputs accessible  
**Future**: Support for nested module outputs with dot notation

```csl
// ❌ Not supported in MVP
database_url = tfstate.app.database_url

// ✅ Workaround: Expose nested module outputs at root level in Terraform
output "app_database_url" {
  value = module.app.database_url
}
```

### 2. Limited Backend Types

**Supported**: `local`, `azurerm`  
**Future**: `s3`, `gcs`, `http`, Terraform Cloud

### 3. No State Locking

Provider reads state but doesn't implement locking (read-only by design).

---

## Examples Repository

Find more examples at: `https://github.com/autonomous-bits/nomos-examples/tree/main/providers/terraform-remote-state`

- Basic local backend
- Multi-workspace setup
- Azure backend with service principal
- Multiple state sources
- Error handling patterns

---

## Support

- **Issues**: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/issues
- **Discussions**: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/discussions
- **Nomos Docs**: https://nomos.dev/docs/providers

---

## Version Compatibility

| Provider Version | Nomos Version | Terraform Version | OpenTofu Version |
|-----------------|---------------|-------------------|------------------|
| 1.x | 1.x+ | 0.12+ | 1.x+ |

**State Format**: Requires Terraform state format version 4 or higher.

---

## Quick Reference

### Local Backend

```csl
source <name> = terraform-remote-state {
  backend_type = "local"
  path = "<path-to-tfstate>"
  workspace = "<workspace-name>"  // optional, default: "default"
}
```

### Azure Backend

```csl
source <name> = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "<storage-account>"
  container_name = "<container>"
  key = "<blob-path>"
}
```

**Required Environment Variables**:
- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`

---

## Next Steps

1. ✅ Install provider binary
2. ✅ Try local backend example
3. ✅ Try Azure backend example (if using Azure)
4. ✅ Integrate into your Nomos project
5. ✅ Share feedback on GitHub!
