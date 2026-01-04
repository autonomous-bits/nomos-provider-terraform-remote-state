# Nomos Terraform Remote State Provider

A Nomos provider that reads Terraform/OpenTofu remote state files and exposes outputs via the Nomos provider interface.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](/LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue)](https://go.dev/)

## Overview

The Nomos Terraform Remote State Provider enables Nomos configurations to consume Terraform/OpenTofu state outputs, allowing seamless integration with existing infrastructure-as-code workflows. The provider implements the `nomos.provider.v1.ProviderService` gRPC contract and supports multiple backend types for retrieving remote state.

### Key Features

- **Multiple Backends**: Local filesystem and Azure Blob Storage (MVP)
- **Workspace Support**: Access state from specific Terraform workspaces
- **Root Module Outputs**: Read and access root module output values
- **Type-Safe Parsing**: Strict validation of Terraform state format v4+ (Terraform 0.12+, OpenTofu 1.x+)
- **Authentication**: Secure credential handling
- **No State Caching**: Fresh state retrieval on every request for guaranteed consistency
- **gRPC Interface**: Standard Nomos provider contract with Init, Fetch, Info, Health, and Shutdown RPCs
- **Subprocess Discovery**: TCP port discovery pattern for Nomos tooling integration
- **Structured Logging**: JSON-formatted logs for debugging and monitoring
- **Graceful Shutdown**: Proper resource cleanup and signal handling

## Prerequisites

- **Go**: 1.25+ (for building from source)
- **Make**: For build automation
- **Platform**: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)

## Usage

### With Nomos CLI

Declare the provider in your `.csl` file:

#### Local Backend Example

Create a Terraform state file:

```json
// terraform.tfstate
{
  "version": 4,
  "terraform_version": "1.6.5",
  "serial": 1,
  "lineage": "abc123-def456-...",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    }
  }
}
```

Use in Nomos configuration:

```csl
// config.csl

source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.2'
  backend_type: 'local'
  path: "/storage/terraform.tfstate"

app:
  name: 'example-app'
  url: reference:tfstate:blob_url
```

OR

#### Azure Backend Example

Use in Nomos configuration:

```csl
// config.csl
source:
  alias: 'tfstate'
  type:  'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.2'
  backend_type: 'azurerm'
  storage_account_name: "storageaccountname"
  container_name: "state"
  key: "storage.tfstate"

config MyApp {
  vpc_id = reference:tfstate:blob_url
```

Run `nomos init` to install the provider:

```bash
nomos init
```

Then build your configuration:

```bash
nomos build -p ./config.csl
```

> [!NOTE]
> The system uses default azure credentials. 

## Configuration

### Local Backend

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `backend_type` | string | No* | Must be `"local"` if specified |
| `path` | string | Yes | Path to terraform.tfstate file |
| `workspace` | string | No | Terraform workspace name (default: `"default"`) |

\* The `backend_type` field is optional. If omitted, the backend type is automatically detected from configuration keys (presence of `path` → local backend).

### Azure Blob Storage Backend

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `backend_type` | string | No* | Must be `"azurerm"` if specified |
| `storage_account_name` | string | Yes | Azure storage account name (3-24 chars, lowercase alphanumeric) |
| `container_name` | string | Yes | Blob container name (3-63 chars) |
| `key` | string | Yes | Blob path/key (include workspace path if applicable) |

\* The `backend_type` field is optional. If omitted, the backend type is automatically detected from configuration keys (presence of `storage_account_name` + `container_name` → azurerm backend).

## Configuration Clarification

### Understanding `type` vs `backend_type`

The Nomos provider ecosystem uses two distinct configuration fields that serve different purposes:

#### CLI-Level: `type` (Provider Source)

The `type` field in Nomos source declarations identifies **which provider binary to load**. This is used by the Nomos tooling to locate and start the correct provider subprocess. It uses the fully-qualified provider name including the organization.

```yaml
# Nomos source configuration
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'  # CLI: Which provider binary to load
  version: '0.0.1'
  backend_type: 'local'           # Runtime: Which backend to use
  path: "./terraform.tfstate"      # Runtime: Backend configuration
```

The `type` field is consumed by the Nomos CLI to locate and start the provider subprocess.

#### Runtime: `backend_type` (Backend Selection)

The `backend_type` field specifies **which backend implementation to use at runtime** for retrieving state files. This is sent to the provider during the `Init` RPC call along with other configuration parameters.

**Local Backend Example**:
```yaml
source:
  alias: 'tfstate_infra'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.2'
  backend_type: 'local'           # Runtime: Use local filesystem backend
  path: "./terraform.tfstate"      # Runtime: Path to state file
```

**Azure Backend Example**:
```yaml
source:
  alias: 'tfstate_azure'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.2'
  backend_type: 'azurerm'         # Runtime: Use Azure Blob Storage backend
  storage_account_name: 'mytfstate'
  container_name: 'tfstate'
  key: 'terraform.tfstate'
```

#### Key Differences

| Aspect | `type` (CLI) | `backend_type` (Runtime) |
|--------|--------------|-------------------------|
| **Scope** | Nomos CLI / tooling | Provider runtime |
| **Purpose** | Identify which provider binary to load | Select backend implementation (local, azurerm) |
| **Values** | Fully-qualified provider names (`autonomous-bits/nomos-provider-terraform-remote-state`) | Backend types (`local`, `azurerm`, future: `s3`, `gcs`) |
| **When Used** | Provider subprocess discovery | During provider `Init` RPC |
| **Visibility** | Nomos CLI (loads provider) | Provider runtime (selects backend) |
| **Required** | Yes (CLI must know which provider to start) | No (auto-detected from config keys if omitted) |

#### Auto-Detection Feature

The `backend_type` field is **optional**. If omitted, the provider automatically detects the backend type from configuration keys:

- Configuration with `path` → detects as `local` backend
- Configuration with `storage_account_name` + `container_name` → detects as `azurerm` backend

```yaml
# Explicit backend_type (recommended for clarity)
source:
  alias: 'tfstate_explicit'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.0.1'
  backend_type: 'local'                # Explicit
  path: "./terraform.tfstate"

# Auto-detected backend_type (convenient for simple cases)
source:
  alias: 'tfstate_auto'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.0.1'
  path: "./terraform.tfstate"          # Auto-detects as "local"
```

**Recommendation**: Use explicit `backend_type` in production configurations for clarity, even though auto-detection works reliably.

## Architecture

```
┌──────────────┐          gRPC           ┌─────────────────────┐
│    Nomos     │ ──────────────────────▶ │   Provider          │
│   Compiler   │   Init/Fetch/Info...    │   (subprocess)      │
└──────────────┘                         └─────────────────────┘
                                                  │
                                                  ▼
                                         ┌─────────────────────┐
                                         │   Backend           │
                                         │   (Local/Azure)     │
                                         └─────────────────────┘
                                                  │
                                                  ▼
                                         ┌─────────────────────┐
                                         │  Terraform State    │
                                         │  (JSON v4+)         │
                                         └─────────────────────┘
```

### Package Structure

- `cmd/provider/`: Main executable entry point
- `internal/provider/`: gRPC service implementation
- `internal/backend/`: Backend interface and implementations (local, Azure)
- `internal/state/`: State file parsing and type definitions
- `internal/config/`: Configuration validation and parsing

## Development

### Prerequisites

- Go 1.25+
- Make
- golangci-lint (for linting)
- Git

### Building

```bash
# Install dependencies
make deps

# Build binary
make build

# Build for specific platform
GOOS=linux GOARCH=amd64 make build
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage (80%+ required)
make test-coverage

# Run with race detection
go test -race ./...

# Run integration tests
go test -tags=integration -v ./...

# Run benchmarks
go test -bench=. ./...
```

### Linting and Formatting

```bash
# Run all verification (fmt + vet + lint + test)
make verify

# Format code
make fmt

# Run linter
make lint
```

## Protocol

This provider implements the `nomos.provider.v1.ProviderService` gRPC contract:

- **Init**: Initialize provider with backend configuration
- **Fetch**: Retrieve output values from state by path
- **Info**: Return provider metadata (type, version)
- **Health**: Check provider health status
- **Shutdown**: Graceful shutdown and cleanup

### Path Format

Output paths consist of a single element containing the output name:

```
path: ["vpc_id"]        → Fetches root module output "vpc_id"
path: ["subnet_ids"]    → Fetches root module output "subnet_ids"
```

**MVP Scope**: Root module outputs only. Nested module outputs require root-level passthroughs in Terraform configuration.

### Response Format

Fetch responses return output values as structured data:

```json
{
  "value": {
    "value": "vpc-12345",
    "type": "string",
    "sensitive": false
  }
}
```

## Troubleshooting

### Common Issues

**Issue**: `state file not found`

**Solution**: Verify the path is correct and the file exists. For workspaces, ensure the workspace-specific path exists.

```bash
# Default workspace
ls -l ./terraform.tfstate

# Named workspace
ls -l ./terraform.tfstate.d/dev/terraform.tfstate
```

---

**Issue**: `output "vpc_id" not found in state`

**Solution**: Verify the output exists in Terraform configuration:

```bash
terraform output
```

**Issue**: `unsupported state version`

**Solution**: Upgrade Terraform/OpenTofu to version 0.12+ or 1.x+ and reapply:

```bash
terraform version
terraform apply
```

See [docs/error-handling.md](docs/error-handling.md) for complete troubleshooting guide.

## Documentation

- [Backend Configuration Guide](docs/backend-configuration.md): Detailed backend setup instructions
- [Output Access Guide](docs/output-access.md): How to access and use Terraform outputs
- [Workspace Usage Guide](docs/workspace-usage.md): Managing workspaces
- [Error Handling](docs/error-handling.md): Common errors and solutions
- [Architecture Decisions](docs/architecture/): Design documentation
- [Quick Start](specs/001-tfstate-provider/quickstart.md): Comprehensive usage examples

## Best Practices

1. **Use Absolute Paths**: For production, use absolute paths in local backend configuration
2. **Environment Variables Only**: Never put credentials in configuration files
3. **Workspace Naming**: Use consistent workspace naming across environments
4. **State Validation**: Ensure Terraform state version is 4+ (Terraform 0.12+)
5. **Error Handling**: Handle missing outputs gracefully with fallback values

## Limitations (MVP)

- **Root Outputs Only**: Nested module outputs require root-level passthrough in Terraform
- **Backend Types**: Local and Azure only (S3, GCS coming in future releases)
- **No State Caching**: Fresh fetch on every request (by design for consistency)
- **No State Locking**: Read-only operations, no locking implemented

## Versioning

This provider follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes to behavior or contract
- **MINOR**: New features, backward compatible (e.g., new backend support)
- **PATCH**: Bug fixes, backward compatible

Current version: **0.1.0** (MVP)

## Contributing

See [CONTRIBUTING.md](/CONTRIBUTING.md) for development guidelines, including:

- Development workflow
- Code standards
- Testing requirements
- Commit message format
- Pull request process

## Changelog

See [CHANGELOG.md](/CHANGELOG.md) for version history and release notes.

## License

See [LICENSE](/LICENSE)) for licensing information.

## Support

- **Issues**: [GitHub Issues](https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/issues)
- **Discussions**: [GitHub Discussions](https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/discussions)
- **Documentation**: [docs/](docs/)

## Acknowledgments

Built following the [autonomous-bits development standards](https://github.com/autonomous-bits/development-standards) and Go community best practices.