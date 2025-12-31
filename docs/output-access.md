# Output Access Guide

This guide explains how to access and use Terraform output values through the Nomos Terraform Remote State Provider.

## Overview

Terraform outputs are values explicitly exported from a Terraform configuration using the `output` block. The provider retrieves these outputs from state files and makes them available to Nomos configurations.

## Output Types

Terraform supports various output types, all of which are accessible through the provider:

- **Primitives**: string, number, boolean
- **Collections**: list, map, set
- **Structural**: object, tuple

## MVP Scope: Root Module Outputs Only

The current MVP version supports **root module outputs only**. Outputs from child modules (e.g., `module.app.output.database_url`) are not directly accessible.

**Workaround**: Pass module outputs through to the root level in your Terraform configuration.

---

## Basic Output Access

### Path Format

Output paths consist of a single element: the output name.

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

// Access output by name
config App {
  // Path: ["vpc_id"]
  vpc_id = tfstate.vpc_id.value
}
```

### Output Structure

Each output is returned as a structured object with three fields:

```json
{
  "value": <actual-value>,
  "type": "<terraform-type>",
  "sensitive": <boolean>
}
```

Always access the `.value` property to get the actual output value:

```csl
// Correct
vpc_id = tfstate.vpc_id.value

// Wrong (returns the entire output object)
vpc_id = tfstate.vpc_id
```

---

## Output Type Examples

### String Outputs

**Terraform Configuration**:
```hcl
output "vpc_id" {
  value = aws_vpc.main.id
  description = "The VPC ID"
}

output "region" {
  value = "us-west-2"
}
```

**Terraform State** (`terraform.tfstate`):
```json
{
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    },
    "region": {
      "value": "us-west-2",
      "type": "string",
      "sensitive": false
    }
  }
}
```

**Nomos Usage**:
```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  vpc_id = tfstate.vpc_id.value     // "vpc-12345"
  region = tfstate.region.value     // "us-west-2"
}
```

### Number Outputs

**Terraform Configuration**:
```hcl
output "instance_count" {
  value = 3
}

output "database_port" {
  value = 5432
}
```

**Nomos Usage**:
```csl
config App {
  instance_count = tfstate.instance_count.value  // 3
  db_port = tfstate.database_port.value          // 5432
}
```

### Boolean Outputs

**Terraform Configuration**:
```hcl
output "monitoring_enabled" {
  value = true
}

output "multi_az" {
  value = false
}
```

**Nomos Usage**:
```csl
config App {
  enable_monitoring = tfstate.monitoring_enabled.value  // true
  multi_az = tfstate.multi_az.value                     // false
}
```

### List Outputs

**Terraform Configuration**:
```hcl
output "availability_zones" {
  value = ["us-west-2a", "us-west-2b", "us-west-2c"]
}

output "subnet_ids" {
  value = aws_subnet.private[*].id
}
```

**Terraform State**:
```json
{
  "outputs": {
    "availability_zones": {
      "value": ["us-west-2a", "us-west-2b", "us-west-2c"],
      "type": ["list", "string"],
      "sensitive": false
    }
  }
}
```

**Nomos Usage**:
```csl
config App {
  // Access entire list
  zones = tfstate.availability_zones.value
  // ["us-west-2a", "us-west-2b", "us-west-2c"]
  
  // Access list elements
  primary_zone = tfstate.availability_zones.value[0]  // "us-west-2a"
  
  // Iterate over list
  for zone in tfstate.availability_zones.value {
    availability_zone = zone
  }
}
```

### Map Outputs

**Terraform Configuration**:
```hcl
output "tags" {
  value = {
    Environment = "production"
    Project     = "myapp"
    ManagedBy   = "terraform"
  }
}
```

**Terraform State**:
```json
{
  "outputs": {
    "tags": {
      "value": {
        "Environment": "production",
        "Project": "myapp",
        "ManagedBy": "terraform"
      },
      "type": ["map", "string"],
      "sensitive": false
    }
  }
}
```

**Nomos Usage**:
```csl
config App {
  // Access entire map
  all_tags = tfstate.tags.value
  
  // Access specific map keys
  environment = tfstate.tags.value["Environment"]  // "production"
  project = tfstate.tags.value["Project"]          // "myapp"
}
```

### Object Outputs

**Terraform Configuration**:
```hcl
output "database_config" {
  value = {
    host     = aws_db_instance.main.endpoint
    port     = aws_db_instance.main.port
    database = aws_db_instance.main.name
    ssl_mode = "require"
  }
}
```

**Terraform State**:
```json
{
  "outputs": {
    "database_config": {
      "value": {
        "host": "db.example.com",
        "port": 5432,
        "database": "appdb",
        "ssl_mode": "require"
      },
      "type": ["object", {
        "host": "string",
        "port": "number",
        "database": "string",
        "ssl_mode": "string"
      }],
      "sensitive": false
    }
  }
}
```

**Nomos Usage**:
```csl
config App {
  // Access entire object
  db_config = tfstate.database_config.value
  
  // Access object properties
  db_host = tfstate.database_config.value["host"]      // "db.example.com"
  db_port = tfstate.database_config.value["port"]      // 5432
  db_name = tfstate.database_config.value["database"]  // "appdb"
  
  // Use in nested config
  database {
    host = tfstate.database_config.value["host"]
    port = tfstate.database_config.value["port"]
    name = tfstate.database_config.value["database"]
  }
}
```

---

## Sensitive Outputs

### Checking Sensitivity

**Terraform Configuration**:
```hcl
output "database_password" {
  value     = random_password.db.result
  sensitive = true
}
```

**Terraform State**:
```json
{
  "outputs": {
    "database_password": {
      "value": "super-secret-password",
      "type": "string",
      "sensitive": true
    }
  }
}
```

**Nomos Usage**:
```csl
config App {
  // Check if output is sensitive
  if tfstate.database_password.sensitive {
    // Handle sensitive value appropriately
    db_password = tfstate.database_password.value
    // Warning: Value is not redacted, handle with care!
  }
}
```

**Important**: The provider returns sensitive values as-is. It's the responsibility of the Nomos configuration and tooling to handle sensitive values appropriately (avoid logging, redact in output, etc.).

---

## Handling Missing Outputs

### Using Default Values

**Problem**: Output might not exist in older state versions

**Solution**: Use null coalescing operator (`??`) for fallback values:

```csl
config App {
  // If output doesn't exist, use default
  instance_type = tfstate.instance_type.value ?? "t3.micro"
  
  // Conditional with existence check
  enable_feature = tfstate.feature_flag.value ?? false
}
```

### Error Handling

If an output doesn't exist and no default is provided, the provider returns a gRPC `NotFound` error:

```
gRPC error: code = NotFound, message = "output 'vpc_id' not found in state"
```

### Checking Output Existence

**Terraform Verification**:
```bash
# List all outputs
terraform output

# Check specific output
terraform output vpc_id
```

**State File Inspection**:
```bash
# Pretty-print outputs section
jq '.outputs' terraform.tfstate
```

---

## Advanced Patterns

### Multiple State Sources

Access outputs from multiple Terraform states:

```csl
// Network infrastructure
source tfstate_network = terraform-remote-state {
  backend_type = "local"
  path = "./network/terraform.tfstate"
}

// Database infrastructure
source tfstate_database = terraform-remote-state {
  backend_type = "local"
  path = "./database/terraform.tfstate"
}

// Application infrastructure
source tfstate_app = terraform-remote-state {
  backend_type = "azurerm"
  storage_account_name = "mytfstate"
  container_name = "tfstate"
  key = "app/terraform.tfstate"
}

config MyApp {
  // Compose configuration from multiple states
  vpc_id = tfstate_network.vpc_id.value
  subnet_ids = tfstate_network.private_subnet_ids.value
  
  db_endpoint = tfstate_database.primary_endpoint.value
  db_port = tfstate_database.port.value
  
  load_balancer_dns = tfstate_app.lb_dns_name.value
  security_group = tfstate_app.app_sg_id.value
}
```

### Conditional Configuration

Use output values for conditional logic:

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  environment = tfstate.environment.value
  
  // Enable monitoring only in production
  monitoring_enabled = tfstate.environment.value == "production"
  
  // Scale based on environment
  instance_count = tfstate.environment.value == "production" ? 5 : 2
  
  // Conditional resource configuration
  if tfstate.multi_az.value {
    availability_zones = tfstate.availability_zones.value
  }
}
```

### Dynamic Resource Naming

Use outputs for dynamic naming:

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  // Construct resource names from outputs
  bucket_name = "app-data-${tfstate.environment.value}-${tfstate.region.value}"
  // Result: "app-data-production-us-west-2"
  
  dns_name = "${tfstate.app_name.value}.${tfstate.domain.value}"
  // Result: "myapp.example.com"
}
```

### Transforming Output Values

Transform outputs as needed:

```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  // Convert list to map
  subnet_map = {
    for idx, subnet_id in tfstate.subnet_ids.value:
      "subnet_${idx}" => subnet_id
  }
  
  // Filter lists
  public_subnets = [
    for subnet in tfstate.subnets.value:
      subnet if subnet.public
  ]
  
  // Transform values
  uppercase_region = upper(tfstate.region.value)
}
```

---

## Nested Module Outputs (Phase 2)

**Current Limitation**: Direct access to nested module outputs is not supported in MVP.

### Terraform Workaround

Pass module outputs to the root level:

**Module Definition** (`modules/database/outputs.tf`):
```hcl
output "endpoint" {
  value = aws_db_instance.main.endpoint
}

output "port" {
  value = aws_db_instance.main.port
}
```

**Root Module** (`outputs.tf`):
```hcl
module "database" {
  source = "./modules/database"
  // ...
}

// WORKAROUND: Expose module outputs at root level
output "database_endpoint" {
  value = module.database.endpoint
}

output "database_port" {
  value = module.database.port
}
```

**Nomos Usage**:
```csl
source tfstate = terraform-remote-state {
  backend_type = "local"
  path = "./terraform.tfstate"
}

config App {
  // Access root-level outputs (exposed from module)
  db_endpoint = tfstate.database_endpoint.value
  db_port = tfstate.database_port.value
}
```

### Future Enhancement (Phase 2)

Planned support for nested module outputs with dot notation:

```csl
// Phase 2: Direct module output access (NOT YET SUPPORTED)
db_endpoint = tfstate.modules.database.endpoint.value
```

---

## Best Practices

1. **Always Use `.value`**: Access the value property, not the output object
2. **Handle Missing Outputs**: Use default values with `??` operator
3. **Check Sensitivity**: Handle sensitive outputs appropriately
4. **Document Required Outputs**: List expected outputs in comments
3. **Use Type-Appropriate Access**: List indexing, map key access, object properties
4. **Expose Module Outputs**: Pass module outputs to root level for MVP
5. **Validate Output Types**: Ensure Terraform output types match Nomos expectations

## Common Errors

**Error**: `output 'vpc_id' not found in state`

**Cause**: Output doesn't exist in the Terraform state

**Solution**:
```bash
# Check if output exists
terraform output

# Add output to Terraform configuration
output "vpc_id" {
  value = aws_vpc.main.id
}

# Apply to update state
terraform apply
```

---

**Error**: Type mismatch when accessing output

**Cause**: Accessing output without `.value` property

**Solution**:
```csl
// Wrong - returns entire output object
vpc_id = tfstate.vpc_id

// Correct - returns the value
vpc_id = tfstate.vpc_id.value
```

---

**Error**: Cannot access nested module output

**Cause**: MVP doesn't support nested module outputs

**Solution**: Add root-level output passthrough:
```hcl
// In root module
output "app_database_url" {
  value = module.app.database_url
}
```

---

## Output Access Checklist

- [ ] Using `.value` property to access output values
- [ ] Handling potentially missing outputs with defaults
- [ ] Checking and respecting `sensitive` flag
- [ ] Documenting required Terraform outputs
- [ ] Using appropriate type accessors (list index, map key, object property)
- [ ] Exposing module outputs at root level (MVP workaround)
- [ ] Validating output types match expectations

## Next Steps

- [Backend Configuration](backend-configuration.md): Configure state backends
- [Workspace Usage](workspace-usage.md): Manage workspaces
- [Error Handling](error-handling.md): Troubleshooting guide
