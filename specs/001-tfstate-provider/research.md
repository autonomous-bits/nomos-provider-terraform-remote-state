# Technical Research: Terraform Remote State Provider

**Feature Branch**: `001-tfstate-provider`  
**Date**: 2025-12-30  
**Status**: Complete

## Overview

This document consolidates all technical research findings required to implement the Terraform Remote State Provider for Nomos. The provider will read Terraform/OpenTofu state files from various backends and expose outputs via the Nomos provider gRPC contract.

---

## 1. Nomos Provider gRPC Contract

### Contract Source
- **Repository**: `github.com/autonomous-bits/nomos/libs/provider-proto`
- **Package**: `nomos.provider.v1`
- **Service**: `ProviderService`

### RPC Methods

#### Init
**Purpose**: Initialize provider with configuration (called once per provider instance)

**Request** (`InitRequest`):
- `alias` (string): Provider instance name from .csl source declaration
- `config` (google.protobuf.Struct): Provider-specific configuration as free-form map
- `source_file_path` (string): Absolute path to the declaring .csl file

**Response** (`InitResponse`): Empty (success indicated by lack of error)

**Error Codes**:
- `InvalidArgument`: Invalid configuration format or missing required fields
- `FailedPrecondition`: Provider cannot be initialized (e.g., missing dependencies, unreachable backends)
- `Unavailable`: External resource unreachable (applies to remote backends)
- `PermissionDenied`: Insufficient permissions to access backend

**Implementation Requirements**:
- MUST validate all required config parameters
- MUST read credentials from environment variables (NEVER from config)
- MUST establish backend connection and verify accessibility
- SHOULD validate state file format version if possible without full read

#### Fetch
**Purpose**: Retrieve configuration data at specified path (called multiple times)

**Request** (`FetchRequest`):
- `path` (repeated string): Path segments identifying data to fetch
  - Example: `["vpc_id"]` → root output `vpc_id`
  - Example: `["app", "database_url"]` → module.app.output.database_url

**Response** (`FetchResponse`):
- `value` (google.protobuf.Struct): Structured data compatible with Nomos value types

**Error Codes**:
- `NotFound`: Path/output does not exist in state
- `InvalidArgument`: Invalid path format (e.g., empty path)
- `FailedPrecondition`: Provider not initialized (Init not called)
- `PermissionDenied`: Access denied to state resource
- `DeadlineExceeded`: Fetch operation timed out
- `Unavailable`: Backend temporarily unreachable

**Implementation Requirements**:
- MUST check initialization state before processing
- MUST fetch state on EVERY call (no caching)
- MUST support dot notation for nested module outputs
- MUST convert Terraform output types to google.protobuf.Struct compatible types
- MUST return NotFound with clear error message identifying missing output name

#### Info
**Purpose**: Return provider metadata (can be called anytime, before or after Init)

**Request** (`InfoRequest`): Empty

**Response** (`InfoResponse`):
- `alias` (string): Provider instance name (from Init, or empty if not initialized)
- `version` (string): Provider implementation version (e.g., "1.0.0")
- `type` (string): Provider type identifier (MUST be "terraform-remote-state")

**Implementation Requirements**:
- MUST be callable before Init
- SHOULD return provider version from build metadata

#### Health
**Purpose**: Check provider operational status (lightweight, fast check)

**Request** (`HealthRequest`): Empty

**Response** (`HealthResponse`):
- `status` (enum): STATUS_UNKNOWN, STATUS_OK, STATUS_DEGRADED, STATUS_STARTING
- `message` (string): Additional diagnostic context

**Error Codes**: Internal (if health check itself fails)

**Implementation Requirements**:
- MUST respond within 100ms
- MUST return STATUS_OK for normal operation
- MAY return STATUS_DEGRADED if backend is slow but accessible
- SHOULD check backend connectivity without full state read
- SHOULD include helpful diagnostic messages

#### Shutdown
**Purpose**: Gracefully shut down provider and clean up resources

**Request** (`ShutdownRequest`): Empty

**Response** (`ShutdownResponse`): Empty

**Implementation Requirements**:
- MUST close all connections and release resources
- MUST complete within 5 seconds
- Best-effort; compiler may force terminate

### Process Model & Discovery

**Subprocess Architecture**:
1. Provider starts as independent subprocess
2. Listens on random TCP port: `net.Listen("tcp", ":0")`
3. Prints discovery line to stdout: `PROVIDER_PORT=<port>` (IMMEDIATELY after binding)
4. Responds to gRPC requests on that port
5. Exits cleanly on SIGTERM/SIGINT

**Discovery Protocol**:
- Nomos CLI spawns provider binary
- Reads stdout for `PROVIDER_PORT=<port>` line
- Establishes gRPC client connection to `localhost:<port>`
- Calls Init RPC to configure provider
- Calls Fetch RPC for each configuration data request
- Calls Shutdown RPC on completion
- Terminates process if Shutdown takes too long

### Context Propagation

**Requirements**:
- Context MUST be first parameter in all functions performing I/O
- MUST propagate context through call chains
- MUST check context cancellation in loops and long operations
- MUST use `context.WithTimeout` for operations with deadlines
- All RPC handlers receive context from gRPC framework

### Error Handling Patterns

From the Nomos codebase examples:
```go
// Use gRPC status codes for errors
return nil, status.Error(codes.NotFound, "path not found: %v")
return nil, status.Errorf(codes.InvalidArgument, "invalid config: %v", err)
return nil, status.Error(codes.FailedPrecondition, "provider not initialized")
```

### Data Type Mapping

Terraform outputs → google.protobuf.Struct:
- Use `structpb.NewStruct()` to convert Go maps to protobuf Struct
- Use `structpb.NewValue()` for individual values
- Support all JSON-compatible types (strings, numbers, booleans, objects, arrays, null)

---

## 2. Terraform State File Format

### State Format Version

**Supported Versions**: v4+ (Terraform 0.12+, OpenTofu 1.x+)

**Version Detection**:
```json
{
  "version": 4,
  "terraform_version": "1.6.5",
  "serial": 123,
  "lineage": "uuid-here",
  "outputs": { ... },
  "resources": [ ... ]
}
```

**Validation Strategy**:
- Check `version` field during Init or first Fetch
- Reject versions < 4 with FailedPrecondition error
- Provide clear error message indicating unsupported version

### Root Module Outputs

**Location**: Top-level `outputs` object in state file

**Structure**:
```json
{
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
    },
    "config_map": {
      "value": {
        "host": "localhost",
        "port": 5432
      },
      "type": ["object", {"host": "string", "port": "number"}],
      "sensitive": false
    }
  }
}
```

**Access Pattern**:
- Path `["vpc_id"]` → returns `"vpc-12345"`
- Path `["subnet_ids"]` → returns `["subnet-1", "subnet-2"]`
- Path `["config_map"]` → returns entire object

### Module Outputs (Nested Modules)

**Location**: In `resources` array with `"mode": "data"` and `"type": "terraform_remote_state"`
OR within module structure if provider has access to full state

**Dot Notation Pattern**:
- Path `["app", "database_url"]` → accesses `module.app.output.database_url`
- This requires parsing the state's resource/module structure to find module outputs

**Alternative Approach** (simpler for MVP):
- Only support root module outputs initially
- Document that nested module outputs require workspace/backend configuration that exposes them at root level
- Phase 2: Add full nested module support

### State File Parsing Strategy

**Go Standard Library**:
```go
import "encoding/json"

type StateFile struct {
    Version         int                    `json:"version"`
    TerraformVersion string                `json:"terraform_version"`
    Outputs         map[string]OutputValue `json:"outputs"`
    Resources       []Resource             `json:"resources,omitempty"`
}

type OutputValue struct {
    Value     interface{} `json:"value"`
    Type      interface{} `json:"type"`
    Sensitive bool        `json:"sensitive"`
}
```

**Libraries** (optional, evaluate if needed):
- Consider `github.com/hashicorp/terraform-json` for official type definitions
- For MVP: Direct JSON parsing with Go stdlib is sufficient

---

## 3. Azure Blob Storage SDK for Go

### Package Information

**Primary Package**: `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`  
**Auth Package**: `github.com/Azure/azure-sdk-for-go/sdk/azidentity`

### Installation

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/storage/azblob
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
```

### Authentication via Environment Variables

**Recommended**: `DefaultAzureCredential` - automatically tries multiple authentication methods in order:

1. Environment variables (highest priority)
2. Managed Identity (when running in Azure)
3. Azure CLI credentials (for local development)
4. Azure PowerShell credentials

**Environment Variables** (follows Terraform conventions):
- `AZURE_TENANT_ID`: Azure Active Directory tenant ID
- `AZURE_CLIENT_ID`: Service principal client ID
- `AZURE_CLIENT_SECRET`: Service principal client secret
- `AZURE_SUBSCRIPTION_ID`: (optional) Azure subscription ID

**Alternative**: `EnvironmentCredential` - ONLY uses environment variables (more explicit)

```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// Use DefaultAzureCredential (recommended for flexibility)
cred, err := azidentity.NewDefaultAzureCredential(nil)
if err != nil {
    return fmt.Errorf("failed to create credential: %w", err)
}

// Create blob client
accountURL := "https://<storage-account>.blob.core.windows.net/"
client, err := azblob.NewClient(accountURL, cred, nil)
if err != nil {
    return fmt.Errorf("failed to create client: %w", err)
}
```

### Downloading Blobs

**Method**: `DownloadStream` - recommended for state files (efficient for small-to-medium files)

```go
import (
    "context"
    "bytes"
    "io"
)

func downloadStateFile(ctx context.Context, client *azblob.Client, containerName, blobName string) ([]byte, error) {
    // Download the blob
    resp, err := client.DownloadStream(ctx, containerName, blobName, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to download blob: %w", err)
    }

    // Read the response body
    downloadedData := bytes.Buffer{}
    retryReader := resp.NewRetryReader(ctx, &azblob.RetryReaderOptions{})
    defer retryReader.Close()

    _, err = downloadedData.ReadFrom(retryReader)
    if err != nil {
        return nil, fmt.Errorf("failed to read blob data: %w", err)
    }

    return downloadedData.Bytes(), nil
}
```

**Alternative**: `DownloadFile` - downloads directly to file (use if caching needed, but NOT for this provider per requirements)

### Configuration Parameters for Azure Backend

Based on Terraform azurerm backend conventions:

```go
type AzureBackendConfig struct {
    StorageAccountName string `json:"storage_account_name"` // Required
    ContainerName      string `json:"container_name"`       // Required
    Key                string `json:"key"`                   // Required (blob name/path)
    ResourceGroupName  string `json:"resource_group_name"`  // Optional
    // Credentials from environment variables ONLY (never in config)
}
```

**URL Construction**:
```go
accountURL := fmt.Sprintf("https://%s.blob.core.windows.net/", config.StorageAccountName)
```

### Error Handling

**Common Errors**:
- `404 Not Found`: Blob doesn't exist → return gRPC NotFound
- `403 Forbidden`: Authentication/authorization failure → return gRPC PermissionDenied
- `Timeout`: Network issues → return gRPC DeadlineExceeded or Unavailable
- Connection errors → return gRPC Unavailable

**Error Detection**:
```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/azcore"
    "net/http"
)

var responseError *azcore.ResponseError
if errors.As(err, &responseError) {
    switch responseError.StatusCode {
    case http.StatusNotFound:
        return status.Error(codes.NotFound, "state file not found")
    case http.StatusForbidden:
        return status.Error(codes.PermissionDenied, "access denied to state file")
    default:
        return status.Errorf(codes.Internal, "azure blob error: %v", err)
    }
}
```

---

## 4. Local Filesystem Backend

### Configuration Parameters

```go
type LocalBackendConfig struct {
    Path string `json:"path"` // Required: absolute or relative path to state file
}
```

**Path Resolution**:
- If relative path provided, resolve relative to current working directory
- Validate path exists and is readable during Init
- Use `filepath.Abs()` to normalize paths

### File Reading

```go
import (
    "os"
    "path/filepath"
)

func readLocalStateFile(ctx context.Context, path string) ([]byte, error) {
    // Resolve absolute path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve path: %w", err)
    }

    // Check context cancellation before I/O
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }

    // Read file
    data, err := os.ReadFile(absPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, status.Error(codes.NotFound, "state file not found")
        }
        if os.IsPermission(err) {
            return nil, status.Error(codes.PermissionDenied, "permission denied")
        }
        return nil, fmt.Errorf("failed to read state file: %w", err)
    }

    return data, nil
}
```

### Error Handling

- `os.IsNotExist(err)` → gRPC NotFound
- `os.IsPermission(err)` → gRPC PermissionDenied
- Other errors → gRPC Internal or FailedPrecondition

---

## 5. gRPC Service Implementation Patterns

### Server Setup (from Nomos examples)

```go
import (
    "fmt"
    "net"
    "os"

    providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
    "google.golang.org/grpc"
)

func main() {
    // Listen on random port
    lis, err := net.Listen("tcp", ":0")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    // Print discovery line for Nomos CLI
    addr := lis.Addr().(*net.TCPAddr)
    fmt.Printf("PROVIDER_PORT=%d\n", addr.Port)
    os.Stdout.Sync() // Ensure immediate flush

    // Create gRPC server
    server := grpc.NewServer()
    
    // Register provider service
    provider := NewProvider()
    providerv1.RegisterProviderServiceServer(server, provider)

    // Start serving
    log.Printf("Provider listening on %v", addr)
    if err := server.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

### Provider Structure

```go
type Provider struct {
    providerv1.UnimplementedProviderServiceServer
    
    mu          sync.RWMutex
    initialized bool
    alias       string
    backend     Backend // Interface for different backend types
}

type Backend interface {
    FetchState(ctx context.Context) (*StateFile, error)
    Close() error
}
```

### Graceful Shutdown

```go
import (
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // ... server setup ...

    // Handle signals for graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        log.Printf("Received signal %v, shutting down", sig)
        server.GracefulStop()
    }()

    // Start serving
    if err := server.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

---

## 6. Best Practices from Nomos Codebase

### TDD Approach

From constitution:
- Write tests FIRST (table-driven tests in `*_test.go`)
- Red-Green-Refactor cycle
- Minimum 80% coverage, 100% for critical paths (Init, Fetch, Health, Shutdown)
- Integration tests tagged with `//go:build integration`

### Code Quality Standards

- **Formatting**: MUST pass `gofmt` and `goimports` (zero violations)
- **Linting**: MUST pass `golangci-lint` (zero warnings)
- **Error Handling**: NO panics in library code, all errors returned with context
- **Documentation**: ALL exported symbols MUST have godoc comments
- **Naming**: Follow Go conventions (PascalCase exports, camelCase unexported)

### Package Structure (Domain-Driven)

Recommended structure for this provider:
```
cmd/
  nomos-provider-terraform-remote-state/
    main.go
internal/
  provider/
    provider.go          # gRPC service implementation
    provider_test.go
  backend/
    backend.go           # Backend interface
    local.go             # Local filesystem backend
    local_test.go
    azurerm.go           # Azure Blob backend
    azurerm_test.go
  state/
    parser.go            # State file parsing
    parser_test.go
    types.go             # State file type definitions
```

### Context Usage

- Context MUST be first parameter
- MUST check `ctx.Err()` before long operations
- Use `context.WithTimeout` for backend operations
- Example from codebase:
```go
func (p *Provider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }
    
    // Perform operation with context propagation
    data, err := p.backend.FetchState(ctx)
    if err != nil {
        return nil, err
    }
    
    // ...
}
```

---

## 7. Security Considerations

### Credential Management

**CRITICAL REQUIREMENTS**:
- NEVER accept credentials in config parameter
- ALWAYS read credentials from environment variables
- Use `os.Getenv()` or similar to read environment
- For Azure: `DefaultAzureCredential` automatically reads env vars

### Input Validation

**Config Validation**:
- Validate all required fields present
- Validate field types and formats
- Reject unknown/unexpected fields (fail fast)
- Use structured error messages

**Path Validation**:
- Reject empty paths
- Validate path segments are non-empty strings
- Protect against path traversal (not applicable for output paths, but good practice)

### Error Messages

**DO NOT expose**:
- Internal file paths (use relative paths in errors)
- Credential values
- Detailed internal state

**DO provide**:
- Clear indication of what went wrong
- Actionable guidance (e.g., "ensure AZURE_CLIENT_ID is set")
- Error context for debugging

### Resource Limits

**Protection against DoS**:
- Enforce context timeouts for all I/O operations
- Limit state file size (e.g., reject files > 100MB)
- Use context cancellation for long-running operations

---

## 8. Performance Requirements

From spec Success Criteria:

- **SC-001**: Compilation with local backend within 5 seconds
- **SC-008**: Provider startup and initialization in under 2 seconds for local backends

### Optimization Strategies

1. **Fast Health Checks**: Don't read full state file for health check
   - For local: Check file exists and is readable
   - For Azure: Simple HEAD request or GetProperties (not full download)

2. **No Caching**: Per requirements, fetch state on EVERY Fetch call
   - Accept performance trade-off for data freshness guarantee

3. **Efficient Parsing**: Use streaming JSON decoder if state files are large
   - For MVP: Simple `json.Unmarshal` is sufficient (state files typically < 10MB)

4. **Connection Reuse**: Keep backend clients alive between Fetch calls
   - Initialize client during Init, reuse for all Fetch calls

---

## 9. Workspace Support (P3 Priority)

### Terraform Workspace Concept

Workspaces allow managing multiple state files for same configuration:
- Default workspace: "default"
- Named workspaces: e.g., "dev", "staging", "prod"

### State File Location by Backend

**Local Backend**:
- Default workspace: `terraform.tfstate`
- Named workspace: `terraform.tfstate.d/<workspace>/terraform.tfstate`

**Azure Backend**:
- Workspace determined by blob key/path
- Configuration specifies full path to workspace-specific state

### Configuration Parameter

```go
type BackendConfig struct {
    Workspace string `json:"workspace,omitempty"` // Default: "default"
    // ... other backend-specific fields
}
```

**MVP Approach**:
- Support workspace parameter in config
- Default to "default" workspace if not specified
- For local backend: Construct path based on workspace name
- For Azure backend: User specifies full blob path (includes workspace implicitly)

---

## 10. Implementation Phases (Recommended)

Based on spec priorities and research findings:

### Phase 0: Foundation
- Set up Go project structure
- Add provider-proto dependency
- Create basic gRPC server with port discovery
- Implement empty RPC method stubs

### Phase 1: Local Backend (P1)
- Implement Init with local backend config validation
- Implement local filesystem state reading
- Implement state file parsing (root outputs only)
- Implement Fetch for root module outputs
- Implement Info, Health, Shutdown
- Write comprehensive tests

### Phase 2: Azure Backend (P1)
- Add Azure SDK dependencies
- Implement Azure backend with environment-based auth
- Add Azure backend config validation
- Write integration tests (with test storage account)

### Phase 3: Module Outputs (P2)
- Add support for nested module output resolution
- Implement dot notation path parsing
- Update tests for module scenarios

### Phase 4: Workspace Support (P3)
- Add workspace parameter to config
- Implement workspace-aware path resolution for local backend
- Document workspace usage patterns

---

## 11. Open Questions & Decisions

### Q1: Should we validate state file version during Init or first Fetch?

**Decision**: Validate during first Fetch
- **Rationale**: Init should be fast; full state read may be expensive
- **Trade-off**: Later failure, but better performance

### Q2: Should we support nested module outputs in MVP?

**Decision**: Support root outputs only in MVP, nested modules in Phase 2
- **Rationale**: Simpler implementation, covers most common use cases
- **User workaround**: Users can expose nested module outputs at root level

### Q3: Should we use hashicorp/terraform-json library or manual parsing?

**Decision**: Manual parsing with Go stdlib for MVP
- **Rationale**: Simpler dependencies, full control over parsing
- **Re-evaluate**: If complex type handling becomes an issue

### Q4: How to handle sensitive outputs?

**Decision**: Return sensitive outputs with WARNING log
- **Rationale**: Provider shouldn't make security decisions; trust Terraform's sensitivity marking
- **Enhancement**: Add config option to filter sensitive outputs (future)

---

## 12. Dependencies Summary

### Required Go Modules

```
github.com/autonomous-bits/nomos/libs/provider-proto  # gRPC contract
google.golang.org/grpc                                # gRPC server
google.golang.org/protobuf                            # Protobuf types
github.com/Azure/azure-sdk-for-go/sdk/storage/azblob  # Azure backend
github.com/Azure/azure-sdk-for-go/sdk/azidentity      # Azure auth
```

### Development Tools

```
golangci-lint   # Linting
govulncheck     # Security scanning
go test         # Testing
go mod          # Dependency management
```

---

## 13. Success Metrics Alignment

Mapping research findings to spec success criteria:

- **SC-001** (5s local backend): Manual parsing + no caching achievable
- **SC-002** (local + Azure): Both backends researched and feasible
- **SC-003** (NotFound errors): gRPC status codes provide clear errors
- **SC-004** (exact value match): Direct JSON parsing preserves types
- **SC-005** (Nomos integration): Port discovery pattern from examples
- **SC-006** (contract validation): Following Nomos patterns from codebase
- **SC-007** (90% backend configs): Standard Terraform backend conventions
- **SC-008** (2s initialization): Fast Init without full state read

---

## Conclusion

All technical unknowns have been researched and documented. The provider is feasible with the following approach:

1. **gRPC Implementation**: Follow Nomos patterns from provider-proto examples
2. **State Parsing**: Use Go stdlib JSON parsing for Terraform state v4+
3. **Local Backend**: Standard file I/O with proper error handling
4. **Azure Backend**: Azure SDK with DefaultAzureCredential for env-based auth
5. **Architecture**: Domain-driven package structure, TDD approach, 80%+ coverage
6. **Security**: Environment variables only, input validation, proper error messages

The implementation can proceed to design phase with confidence.
