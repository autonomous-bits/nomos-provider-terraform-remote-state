# Error Handling and Troubleshooting

This guide provides comprehensive troubleshooting information for common errors and issues with the Nomos Terraform Remote State Provider.

## Overview

The provider returns standard gRPC error codes with descriptive messages to help diagnose issues. This guide explains each error type, common causes, and solutions.

## gRPC Error Codes

The provider uses these gRPC error codes:

| Code | Description | Common Causes |
|------|-------------|---------------|
| `InvalidArgument` | Invalid configuration or path | Bad config, empty fields, invalid types |
| `NotFound` | Resource not found | Missing state file, missing output, missing blob |
| `PermissionDenied` | Authentication/authorization failed | Invalid Azure credentials, insufficient permissions |
| `FailedPrecondition` | Operation not allowed in current state | Not initialized, unsupported state version |
| `Unavailable` | Service temporarily unavailable | Network issues, Azure Storage down |
| `Internal` | Internal provider error | JSON parsing errors, unexpected errors |
| `Canceled` | Operation canceled | Context canceled or timeout |
| `DeadlineExceeded` | Operation timed out | Long-running operation exceeded timeout |

---

## Common Errors by Category

### Configuration Errors

#### Error: `missing required field: path`

**Full Message**: `code = InvalidArgument, desc = missing required field: path`

**Cause**: Local backend configuration missing the `path` parameter

**Solution**:
```csl
// Bad
source tfstate = terraform-remote-state {
  type = "local"
  // Missing path!
}

// Good
source tfstate = terraform-remote-state {
  type = "local"
  path = "./terraform.tfstate"
}
```

---

#### Error: `missing required field: storage_account_name`

**Full Message**: `code = InvalidArgument, desc = missing required field: storage_account_name`

**Cause**: Azure backend configuration missing required parameters

**Solution**: Ensure all required Azure parameters are present:
```csl
source tfstate = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"    // Required
  container_name = "tfstate"            // Required
  key = "terraform.tfstate"             // Required
}
```

---

#### Error: `unsupported backend type`

**Full Message**: `code = InvalidArgument, desc = unsupported backend type "s3", available types: [local, azurerm]`

**Cause**: Specified backend type is not supported in current version

**Supported Backends** (MVP):
- `local`
- `azurerm`

**Future Backends**:
- `s3` (planned)
- `gcs` (planned)
- `http` (planned)

**Solution**: Use a supported backend type or wait for future releases

---

#### Error: `invalid configuration`

**Full Message**: `code = InvalidArgument, desc = invalid configuration: <specific-error>`

**Common Causes**:
- Type mismatch (e.g., path is not a string)
- Invalid values (empty strings, negative numbers)
- Failed validation (invalid format)

**Solution**: Check the specific error message for details and validate configuration:

```csl
// Bad: path is not a string
source tfstate = terraform-remote-state {
  type = "local"
  path = 12345  // Wrong type!
}

// Good
source tfstate = terraform-remote-state {
  type = "local"
  path = "./terraform.tfstate"
}
```

---

### Path and Validation Errors

#### Error: `path traversal not allowed`

**Full Message**: `code = InvalidArgument, desc = path traversal detected: path traversal not allowed`

**Cause**: Path contains `..` (path traversal attempt)

**Solution**: Use absolute or relative paths without `..`:
```csl
// Bad
path = "../../etc/passwd"
path = "./infra/../../../secrets/state.tfstate"

// Good
path = "/absolute/path/terraform.tfstate"
path = "./infra/terraform.tfstate"
```

---

#### Error: `invalid path: contains invalid characters`

**Full Message**: `code = InvalidArgument, desc = invalid path: contains invalid characters (allowed: a-z A-Z 0-9 . _ - /)`

**Cause**: Path contains disallowed characters

**Allowed Characters**:
- Letters: `a-z`, `A-Z`
- Numbers: `0-9`
- Symbols: `.`, `_`, `-`, `/`

**Not Allowed**:
- Backslashes: `\`
- Special characters: `@`, `#`, `!`, etc.
- Control characters
- Null bytes

**Solution**:
```csl
// Bad
path = "C:\terraform\state.tfstate"  // Backslashes
path = "state@2024.tfstate"          // Special char

// Good
path = "C:/terraform/state.tfstate"  // Forward slashes
path = "state-2024.tfstate"          // Hyphen
```

---

#### Error: `invalid workspace name`

**Full Message**: `code = InvalidArgument, desc = invalid workspace name: must contain only alphanumeric characters, hyphens, and underscores`

**Cause**: Workspace name contains invalid characters

**Solution**:
```csl
// Bad
workspace = "prod/us-west-2"    // Slash not allowed
workspace = "../production"     // Path traversal
workspace = "test.environment"  // Dot not allowed

// Good
workspace = "production"
workspace = "prod-us-west-2"
workspace = "dev_environment"
```

---

### State File Errors

#### Error: `state file not found`

**Full Message**: 
- Local: `code = NotFound, desc = state file not found: /path/terraform.tfstate (workspace: default)`
- Azure: `code = NotFound, desc = blob not found: terraform.tfstate`

**Cause**: State file doesn't exist at specified location

**Solution**:

**For Local Backend**:
```bash
# Check if file exists
ls -l ./terraform.tfstate

# For named workspaces
ls -l ./terraform.tfstate.d/production/terraform.tfstate

# Initialize Terraform and apply
cd terraform-directory
terraform init
terraform apply
```

**For Azure Backend**:
```bash
# List blobs in container
az storage blob list \
  --account-name mytfstate \
  --container-name tfstate \
  --auth-mode login

# Check if blob exists with correct key
# Verify key matches exactly (case-sensitive)
```

---

#### Error: `state file version must be 4 or greater`

**Full Message**: `code = FailedPrecondition, desc = state file version 3: state file version must be 4 or greater`

**Cause**: Terraform state version is too old (< v4)

**State Versions**:
- Version 3: Terraform 0.11 and earlier ❌
- Version 4+: Terraform 0.12+, OpenTofu 1.x+ ✅

**Solution**: Upgrade Terraform and regenerate state:

```bash
# Check current Terraform version
terraform version

# Upgrade to Terraform 0.12+ or OpenTofu 1.x+
# Then upgrade the state format
terraform init
terraform apply -upgrade
```

**Manual Upgrade** (Terraform 0.11 → 0.12):
```bash
# Terraform 0.12 includes upgrade tool
terraform 0.12upgrade
terraform init
terraform apply
```

---

#### Error: `failed to parse state file JSON`

**Full Message**: `code = Internal, desc = failed to parse state file JSON: <json-error>`

**Cause**: State file is corrupted or contains invalid JSON

**Solution**:

1. **Validate JSON**:
```bash
# Check if file is valid JSON
jq '.' terraform.tfstate

# Or use Python
python -m json.tool terraform.tfstate
```

2. **Restore from Backup**:
```bash
# Terraform keeps backups
ls -la *.backup

# Restore from backup
cp terraform.tfstate.backup terraform.tfstate
```

3. **Re-initialize** (last resort):
```bash
# Remove corrupt state (DANGER: only if backed up)
mv terraform.tfstate terraform.tfstate.corrupt

# Import resources manually or run terraform apply
terraform import <resource-type>.<resource-name> <resource-id>
```

---

### Output Errors

#### Error: `output 'vpc_id' not found in state`

**Full Message**: `code = NotFound, desc = output "vpc_id" not found in state`

**Cause**: Requested output doesn't exist in state file

**Solution**:

1. **Verify Output Exists**:
```bash
# List all outputs
terraform output

# Check specific output
terraform output vpc_id
```

2. **Add Missing Output**:
```hcl
// In Terraform configuration
output "vpc_id" {
  value = aws_vpc.main.id
  description = "The VPC ID"
}
```

3. **Apply Terraform**:
```bash
terraform apply
```

4. **Use Default Value** (Nomos):
```csl
config App {
  // Provide fallback if output might not exist
  vpc_id = tfstate.vpc_id.value ?? "vpc-default"
}
```

---

#### Error: `path must contain exactly one element`

**Full Message**: `code = InvalidArgument, desc = path must contain exactly one element (the output name), got 2`

**Cause**: Path contains multiple segments (MVP limitation)

**MVP Scope**: Only root module outputs with single-segment paths

**Solution**: Expose nested module outputs at root level:

```hcl
// In Terraform root module
module "networking" {
  source = "./modules/networking"
  // ...
}

// Expose module output at root level
output "network_vpc_id" {
  value = module.networking.vpc_id
}
```

**Nomos Configuration**:
```csl
// Correct (single-segment path)
vpc_id = tfstate.network_vpc_id.value

// Incorrect (multi-segment path - not supported in MVP)
// vpc_id = tfstate.networking.vpc_id.value
```

---

### Authentication Errors

#### Error: `azure authentication failed`

**Full Message**: `code = PermissionDenied, desc = azure authentication failed: authentication failed: <azure-error>`

**Cause**: Missing or invalid Azure credentials

**Solution**:

1. **Verify Environment Variables**:
```bash
# Check all three are set
echo $AZURE_TENANT_ID
echo $AZURE_CLIENT_ID
echo $AZURE_CLIENT_SECRET

# They should not be empty
```

2. **Test Azure Authentication**:
```bash
# Login with service principal
az login --service-principal \
  --username $AZURE_CLIENT_ID \
  --password $AZURE_CLIENT_SECRET \
  --tenant $AZURE_TENANT_ID

# Verify access to storage
az storage blob list \
  --account-name mytfstate \
  --container-name tfstate \
  --auth-mode login
```

3. **Verify Permissions**:
```bash
# Check role assignments
az role assignment list \
  --assignee $AZURE_CLIENT_ID \
  --all

# Should have "Storage Blob Data Reader" or "Storage Blob Data Contributor"
```

4. **Grant Required Permissions**:
```bash
az role assignment create \
  --assignee $AZURE_CLIENT_ID \
  --role "Storage Blob Data Reader" \
  --scope "/subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.Storage/storageAccounts/<storage-account>"
```

---

#### Error: `permission denied reading state file`

**Full Message**: Local backend: OS permission error

**Cause**: Insufficient file system permissions

**Solution**:
```bash
# Check file permissions
ls -l terraform.tfstate

# Make readable
chmod 644 terraform.tfstate

# Check directory permissions
ls -ld terraform.tfstate.d/production/
chmod 755 terraform.tfstate.d/production/
```

---

### Initialization Errors

#### Error: `provider not initialized`

**Full Message**: `code = FailedPrecondition, desc = provider not initialized: call Init first`

**Cause**: Attempting to call Fetch before Init

**This is Internal**: Nomos tooling should handle this automatically. If you see this error, it's likely a bug in the Nomos tooling or provider integration.

**Solution**: Report issue to Nomos maintainers

---

#### Error: `provider instance already initialized`

**Full Message**: `code = FailedPrecondition, desc = provider instance "myalias" already initialized`

**Cause**: Attempting to initialize the same provider instance twice

**This is Internal**: Should not occur in normal usage

**Solution**: Report issue to Nomos maintainers if encountered

---

### Network Errors

#### Error: `operation timed out`

**Full Message**: `code = DeadlineExceeded, desc = operation timed out`

**Cause**: Network operation took too long (Azure backend)

**Solution**:

1. **Check Network Connectivity**:
```bash
# Test Azure Storage endpoint
curl -I https://mytfstate.blob.core.windows.net/

# Test DNS resolution
nslookup mytfstate.blob.core.windows.net

# Test with Azure CLI
az storage blob list \
  --account-name mytfstate \
  --container-name tfstate \
  --auth-mode login
```

2. **Check Firewall Rules**:
```bash
# Verify storage account allows access
az storage account show \
  --name mytfstate \
  --query "networkRuleSet.defaultAction"

# Should be "Allow" or have appropriate IP rules
```

3. **Increase Timeout** (if Nomos supports):
```csl
// Configuration-level timeout (if supported)
source tfstate = terraform-remote-state {
  type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "terraform.tfstate"
  timeout = "30s"  // If supported
}
```

---

#### Error: `operation cancelled`

**Full Message**: `code = Canceled, desc = operation cancelled`

**Cause**: Context was canceled (user interrupt or timeout)

**Solution**: This is usually intentional (Ctrl+C). If unintentional, check for:
- Nomos timeout configurations
- CI/CD pipeline timeouts
- Process interruptions

---

## Debugging Tips

### Enable Verbose Logging

The provider logs to stderr in JSON format. Check logs for detailed error information:

```bash
# Run provider with logging visible
./nomos-provider-terraform-remote-state 2> provider.log

# View logs
cat provider.log | jq '.'

# Filter for errors
cat provider.log | jq 'select(.level == "ERROR")'
```

### Verify State File Manually

```bash
# Pretty-print state file
jq '.' terraform.tfstate

# Check version
jq '.version' terraform.tfstate

# List outputs
jq '.outputs | keys' terraform.tfstate

# Check specific output
jq '.outputs.vpc_id' terraform.tfstate
```

### Test Backend Connectivity

**Local Backend**:
```bash
# Test file access
cat terraform.tfstate > /dev/null

# Test workspace access
cat terraform.tfstate.d/production/terraform.tfstate > /dev/null
```

**Azure Backend**:
```bash
# Test Azure connectivity
az storage blob show \
  --account-name mytfstate \
  --container-name tfstate \
  --name terraform.tfstate \
  --auth-mode login

# Download blob manually
az storage blob download \
  --account-name mytfstate \
  --container-name tfstate \
  --name terraform.tfstate \
  --file /tmp/state.json \
  --auth-mode login
```

### Validate Configuration

```bash
# Validate JSON in Nomos config (if applicable)
# Check for syntax errors

# Test with minimal configuration
source tfstate = terraform-remote-state {
  type = "local"
  path = "./terraform.tfstate"
}
```

---

## FAQ

### Q: Why does the provider fetch state on every request?

**A**: By design, the provider does not cache state to ensure data freshness. Every Fetch RPC retrieves the latest state from the backend.

---

### Q: Can I use workspace parameter with Azure backend?

**A**: No. For Azure backend, workspace information is embedded in the `key` parameter. Use patterns like `env:/<workspace>/terraform.tfstate`.

---

### Q: How do I access nested module outputs?

**A**: In MVP, nested module outputs are not directly supported. Expose module outputs at the root level:

```hcl
module "app" {
  source = "./modules/app"
}

output "app_database_url" {
  value = module.app.database_url
}
```

---

### Q: What state versions are supported?

**A**: State format version 4 and higher (Terraform 0.12+, OpenTofu 1.x+). Earlier versions are not supported.

---

### Q: Can I use storage account keys instead of service principal?

**A**: Not in MVP. Azure backend uses `DefaultAzureCredential` which supports environment variables, managed identity, and Azure CLI, but not storage account keys directly.

---

## Error Reference Table

| Error Code | Error Message Pattern | Section |
|-----------|----------------------|---------|
| InvalidArgument | `missing required field` | [Configuration Errors](#configuration-errors) |
| InvalidArgument | `invalid configuration` | [Configuration Errors](#configuration-errors) |
| InvalidArgument | `path traversal` | [Path Errors](#path-and-validation-errors) |
| InvalidArgument | `invalid workspace name` | [Path Errors](#path-and-validation-errors) |
| NotFound | `state file not found` | [State File Errors](#state-file-errors) |
| NotFound | `blob not found` | [State File Errors](#state-file-errors) |
| NotFound | `output '...' not found` | [Output Errors](#output-errors) |
| FailedPrecondition | `state file version` | [State File Errors](#state-file-errors) |
| FailedPrecondition | `provider not initialized` | [Initialization Errors](#initialization-errors) |
| PermissionDenied | `authentication failed` | [Authentication Errors](#authentication-errors) |
| Internal | `failed to parse` | [State File Errors](#state-file-errors) |
| DeadlineExceeded | `operation timed out` | [Network Errors](#network-errors) |
| Canceled | `operation cancelled` | [Network Errors](#network-errors) |

## Getting Help

If you encounter an error not covered in this guide:

1. **Check Logs**: Review provider logs for detailed error messages
2. **Verify Configuration**: Double-check all configuration parameters
3. **Test Connectivity**: Verify network and file system access
4. **Check State File**: Validate state file format and version
5. **Search Issues**: Look for similar issues on GitHub
6. **Report Bug**: Open an issue with full error message and logs

**GitHub Issues**: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/issues

---

## Next Steps

- [Backend Configuration](backend-configuration.md): Configure backends correctly
- [Output Access](output-access.md): Access Terraform outputs
- [Workspace Usage](workspace-usage.md): Manage workspaces
- [Contributing](../CONTRIBUTING.md): Report bugs or contribute fixes
