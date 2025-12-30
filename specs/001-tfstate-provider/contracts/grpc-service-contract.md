# gRPC Service Contract: Terraform Remote State Provider

**Feature Branch**: `001-tfstate-provider`  
**Date**: 2025-12-30  
**Proto Package**: `nomos.provider.v1`  
**Service**: `ProviderService`

## Overview

This document details the gRPC service contract implementation requirements for the Terraform Remote State Provider. The provider MUST implement all methods defined in the `nomos.provider.v1.ProviderService` interface.

---

## Service Definition

```protobuf
service ProviderService {
  rpc Init(InitRequest) returns (InitResponse);
  rpc Fetch(FetchRequest) returns (FetchResponse);
  rpc Info(InfoRequest) returns (InfoResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
  rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
}
```

---

## 1. Init RPC

### Purpose
Initialize the provider with backend configuration. MUST be called before Fetch operations.

### Request Message

```protobuf
message InitRequest {
  string alias = 1;
  google.protobuf.Struct config = 2;
  string source_file_path = 3;
}
```

**Fields**:
- `alias`: Provider instance name from .csl source declaration (e.g., "tfstate-prod")
- `config`: Backend-specific configuration (see Backend Schemas section)
- `source_file_path`: Absolute path to the .csl file declaring this provider

### Response Message

```protobuf
message InitResponse {}
```

**Empty response** - success indicated by lack of error.

### Implementation Requirements

1. **State Validation**:
   - MUST ensure Init is only called once
   - MUST return `FailedPrecondition` if already initialized

2. **Config Parsing**:
   - MUST parse `config` as map[string]interface{}
   - MUST validate `type` field is present and valid ("local" or "azurerm")
   - MUST validate all required backend-specific fields

3. **Backend Initialization**:
   - MUST create appropriate backend instance (LocalBackend or AzureBackend)
   - MUST read credentials from environment variables (NEVER from config)
   - For local backend: MUST validate file path exists and is readable
   - For Azure backend: MUST validate connection parameters and test connectivity

4. **State Storage**:
   - MUST store `alias` for use in Info RPC
   - MUST store backend instance for Fetch operations
   - MUST set initialized flag to prevent re-initialization

### Error Responses

| Condition | gRPC Code | Message Example |
|-----------|-----------|-----------------|
| Already initialized | `FailedPrecondition` | `"provider already initialized"` |
| Missing type field | `InvalidArgument` | `"config missing required field 'type'"` |
| Unsupported backend type | `InvalidArgument` | `"unsupported backend type: 's3'"` |
| Local file not found | `FailedPrecondition` | `"state file not found: ./terraform.tfstate"` |
| Local file not readable | `PermissionDenied` | `"permission denied reading state file"` |
| Azure connection failed | `Unavailable` | `"failed to connect to azure storage: connection timeout"` |
| Azure auth failed | `PermissionDenied` | `"azure authentication failed: ensure AZURE_CLIENT_ID is set"` |
| Missing required config field | `InvalidArgument` | `"config missing required field 'storage_account_name'"` |

### Example Implementations

**Success Case**:
```go
func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.initialized {
        return nil, status.Error(codes.FailedPrecondition, "provider already initialized")
    }
    
    // Parse config
    config := req.Config.AsMap()
    backendType, ok := config["type"].(string)
    if !ok {
        return nil, status.Error(codes.InvalidArgument, "config missing required field 'type'")
    }
    
    // Create backend
    var backend Backend
    var err error
    switch backendType {
    case "local":
        backend, err = NewLocalBackend(ctx, config)
    case "azurerm":
        backend, err = NewAzureBackend(ctx, config)
    default:
        return nil, status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", backendType)
    }
    
    if err != nil {
        return nil, err // Backend constructors return gRPC status errors
    }
    
    // Store state
    p.alias = req.Alias
    p.backend = backend
    p.initialized = true
    
    return &providerv1.InitResponse{}, nil
}
```

---

## 2. Fetch RPC

### Purpose
Retrieve output value at the specified path from Terraform state.

### Request Message

```protobuf
message FetchRequest {
  repeated string path = 1;
}
```

**Fields**:
- `path`: Path segments identifying the output to fetch
  - Root output: `["vpc_id"]`
  - Module output (P2): `["app", "database_url"]`

### Response Message

```protobuf
message FetchResponse {
  google.protobuf.Struct value = 1;
}
```

**Fields**:
- `value`: Output value as protobuf Struct (supports maps, lists, scalars)

### Implementation Requirements

1. **Precondition Check**:
   - MUST return `FailedPrecondition` if not initialized

2. **Path Validation**:
   - MUST return `InvalidArgument` if path is empty
   - MUST return `InvalidArgument` if any path segment is empty string

3. **State Fetching**:
   - MUST fetch fresh state on EVERY call (no caching)
   - MUST propagate context for cancellation
   - MUST handle context timeout/cancellation

4. **Output Resolution**:
   - MUST lookup output by path[0] in state.Outputs
   - MUST return `NotFound` if output doesn't exist
   - For MVP: Only support root outputs (path length == 1)
   - P2: Support module outputs (path length > 1)

5. **Value Conversion**:
   - MUST convert output.Value to protobuf Struct
   - MUST handle all JSON types (string, number, bool, array, object, null)
   - MUST preserve type information

### Error Responses

| Condition | gRPC Code | Message Example |
|-----------|-----------|-----------------|
| Not initialized | `FailedPrecondition` | `"provider not initialized: call Init first"` |
| Empty path | `InvalidArgument` | `"path cannot be empty"` |
| Empty path segment | `InvalidArgument` | `"path segment 1 is empty"` |
| Output not found | `NotFound` | `"output 'vpc_id' not found in state"` |
| State file missing | `NotFound` | `"state file not found: ./terraform.tfstate"` |
| State read error | `Unavailable` | `"failed to read state file: connection reset"` |
| Invalid state JSON | `Internal` | `"failed to parse state file: invalid JSON"` |
| Unsupported state version | `FailedPrecondition` | `"unsupported state version 3 (requires >= 4)"` |
| Module not found (P2) | `NotFound` | `"module 'app' not found in state"` |
| Context cancelled | `Canceled` | `"operation cancelled"` |
| Context deadline exceeded | `DeadlineExceeded` | `"operation timed out"` |

### Example Implementation

**Success Case**:
```go
func (p *Provider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
    p.mu.RLock()
    initialized := p.initialized
    backend := p.backend
    p.mu.RUnlock()
    
    if !initialized {
        return nil, status.Error(codes.FailedPrecondition, "provider not initialized: call Init first")
    }
    
    // Validate path
    if len(req.Path) == 0 {
        return nil, status.Error(codes.InvalidArgument, "path cannot be empty")
    }
    for i, segment := range req.Path {
        if segment == "" {
            return nil, status.Errorf(codes.InvalidArgument, "path segment %d is empty", i)
        }
    }
    
    // Fetch state (with context)
    state, err := backend.FetchState(ctx)
    if err != nil {
        return nil, err // Backend returns gRPC status errors
    }
    
    // Resolve output (MVP: root outputs only)
    outputName := req.Path[0]
    output, exists := state.Outputs[outputName]
    if !exists {
        return nil, status.Errorf(codes.NotFound, "output %q not found in state", outputName)
    }
    
    // Convert to protobuf Struct
    value, err := structpb.NewValue(output.Value)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to convert value: %v", err)
    }
    
    return &providerv1.FetchResponse{
        Value: value.GetStructValue(),
    }, nil
}
```

---

## 3. Info RPC

### Purpose
Return provider metadata. Can be called at any time (before or after Init).

### Request Message

```protobuf
message InfoRequest {}
```

**Empty request**.

### Response Message

```protobuf
message InfoResponse {
  string alias = 1;
  string version = 2;
  string type = 3;
}
```

**Fields**:
- `alias`: Provider instance name (from Init, or empty if not initialized)
- `version`: Provider implementation version (e.g., "1.0.0")
- `type`: Provider type identifier (MUST be "terraform-remote-state")

### Implementation Requirements

1. **Callable Anytime**:
   - MUST be callable before Init (return empty alias)
   - MUST be callable after Init (return stored alias)

2. **Version**:
   - SHOULD return semantic version (e.g., "1.0.0")
   - MAY include build metadata (e.g., "1.0.0+git.abc123")

3. **Type**:
   - MUST return `"terraform-remote-state"` (fixed identifier)

### Error Responses

| Condition | gRPC Code | Message Example |
|-----------|-----------|-----------------|
| Internal error | `Internal` | `"failed to retrieve provider info"` |

(Rarely returns errors - should always succeed)

### Example Implementation

```go
func (p *Provider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
    p.mu.RLock()
    alias := p.alias
    p.mu.RUnlock()
    
    return &providerv1.InfoResponse{
        Alias:   alias,
        Version: "1.0.0", // From build constant
        Type:    "terraform-remote-state",
    }, nil
}
```

---

## 4. Health RPC

### Purpose
Check provider operational status. MUST respond quickly (< 100ms).

### Request Message

```protobuf
message HealthRequest {}
```

**Empty request**.

### Response Message

```protobuf
message HealthResponse {
  enum Status {
    STATUS_UNKNOWN = 0;
    STATUS_OK = 1;
    STATUS_DEGRADED = 2;
    STATUS_STARTING = 3;
  }
  Status status = 1;
  string message = 2;
}
```

**Fields**:
- `status`: Health status enum
- `message`: Human-readable diagnostic message

### Implementation Requirements

1. **Performance**:
   - MUST respond within 100ms
   - MUST NOT perform full state file reads
   - SHOULD perform lightweight checks only

2. **Status Logic**:
   - Return `STATUS_OK` if provider operational
   - Return `STATUS_DEGRADED` if backend slow but accessible
   - Return `STATUS_STARTING` if not yet initialized (acceptable state)
   - Return `STATUS_UNKNOWN` only for unexpected conditions

3. **Health Checks**:
   - For local backend: Check file exists (fast stat call)
   - For Azure backend: Lightweight connectivity check (GetProperties, not full download)
   - If not initialized: Return `STATUS_STARTING` with message "not initialized"

4. **Message**:
   - MUST provide helpful diagnostic information
   - SHOULD indicate reason for degraded status

### Error Responses

| Condition | gRPC Code | Message Example |
|-----------|-----------|-----------------|
| Internal health check failed | `Internal` | `"health check failed: unexpected error"` |

(Health check itself should not fail - return degraded status instead)

### Example Implementation

```go
func (p *Provider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
    p.mu.RLock()
    initialized := p.initialized
    backend := p.backend
    p.mu.RUnlock()
    
    if !initialized {
        return &providerv1.HealthResponse{
            Status:  providerv1.HealthResponse_STATUS_STARTING,
            Message: "provider not yet initialized",
        }, nil
    }
    
    // Lightweight backend health check (with timeout)
    ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
    defer cancel()
    
    if err := backend.HealthCheck(ctx); err != nil {
        return &providerv1.HealthResponse{
            Status:  providerv1.HealthResponse_STATUS_DEGRADED,
            Message: fmt.Sprintf("backend health check failed: %v", err),
        }, nil
    }
    
    return &providerv1.HealthResponse{
        Status:  providerv1.HealthResponse_STATUS_OK,
        Message: "provider healthy",
    }, nil
}
```

---

## 5. Shutdown RPC

### Purpose
Gracefully shut down provider and release resources. Best-effort; compiler may force terminate.

### Request Message

```protobuf
message ShutdownRequest {}
```

**Empty request**.

### Response Message

```protobuf
message ShutdownResponse {}
```

**Empty response**.

### Implementation Requirements

1. **Resource Cleanup**:
   - MUST close backend connections
   - MUST release file handles
   - MUST cancel any ongoing operations
   - SHOULD flush logs if applicable

2. **Timeout**:
   - MUST complete within 5 seconds
   - SHOULD be much faster (typically < 1 second)

3. **Idempotency**:
   - SHOULD be idempotent (safe to call multiple times)
   - Subsequent calls after first shutdown SHOULD succeed silently

4. **Process Exit**:
   - After Shutdown returns, process SHOULD exit
   - If Shutdown takes too long, compiler will terminate process

### Error Responses

| Condition | gRPC Code | Message Example |
|-----------|-----------|-----------------|
| Shutdown error | `Internal` | `"failed to close backend: resource busy"` |

(Errors are logged but don't prevent shutdown)

### Example Implementation

```go
func (p *Provider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.backend != nil {
        if err := p.backend.Close(); err != nil {
            log.Printf("Error closing backend: %v", err)
            // Continue with shutdown anyway
        }
        p.backend = nil
    }
    
    p.initialized = false
    
    return &providerv1.ShutdownResponse{}, nil
}
```

---

## Lifecycle Ordering

The gRPC methods MUST be called in the following order:

```
1. Info/Health (optional, can be called anytime)
2. Init (required, called once)
3. Fetch (required, called multiple times)
4. Shutdown (optional, called once at end)
```

**Invalid Order Enforcement**:
- Fetch before Init → return `FailedPrecondition`
- Init twice → return `FailedPrecondition`
- Shutdown twice → succeed silently (idempotent)

---

## Context Handling

All RPC handlers receive `context.Context` as first parameter. Handlers MUST:

1. **Propagate Context**: Pass context to all I/O operations
2. **Check Cancellation**: Check `ctx.Err()` before expensive operations
3. **Respect Timeouts**: Honor context deadlines for long operations
4. **Return Appropriate Errors**:
   - `ctx.Err() == context.Canceled` → return gRPC `Canceled`
   - `ctx.Err() == context.DeadlineExceeded` → return gRPC `DeadlineExceeded`

**Example Context Check**:
```go
if err := ctx.Err(); err != nil {
    if err == context.Canceled {
        return nil, status.Error(codes.Canceled, "operation cancelled")
    }
    return nil, status.Error(codes.DeadlineExceeded, "operation timed out")
}
```

---

## Threading & Concurrency

The provider implementation MUST be thread-safe:

1. **State Protection**: Use `sync.RWMutex` for shared state
2. **Read Locks**: Use RLock for read-only operations (Fetch, Info, Health)
3. **Write Locks**: Use Lock for state-modifying operations (Init, Shutdown)
4. **Backend Thread-Safety**: Backend implementations MUST be thread-safe

**Example**:
```go
type Provider struct {
    providerv1.UnimplementedProviderServiceServer
    
    mu          sync.RWMutex  // Protects all fields below
    initialized bool
    alias       string
    backend     Backend
}
```

---

## Error Code Summary

| Condition Category | gRPC Code |
|-------------------|-----------|
| Configuration errors | `InvalidArgument` |
| Not initialized | `FailedPrecondition` |
| Missing outputs | `NotFound` |
| Permission errors | `PermissionDenied` |
| Network/connection errors | `Unavailable` |
| Timeout errors | `DeadlineExceeded` |
| Cancellation | `Canceled` |
| Internal/unexpected errors | `Internal` |
| Unsupported versions | `FailedPrecondition` |

---

## Testing Requirements

All RPC methods MUST have:

1. **Unit Tests**:
   - Success cases with valid inputs
   - Error cases for each error condition
   - Edge cases (empty inputs, boundary values)

2. **Integration Tests** (tagged `//go:build integration`):
   - Full gRPC client-server communication
   - Real backend interactions (local files, test Azure storage)
   - Context cancellation and timeout handling

3. **Table-Driven Tests**:
   - Use table-driven pattern for multiple test cases
   - Test all error code branches

4. **Coverage**:
   - 100% coverage for all RPC method handlers
   - Test all error paths

---

## Conclusion

This contract provides complete specifications for:
- ✅ All 5 RPC methods with detailed requirements
- ✅ Request/response message formats
- ✅ Error codes and error handling
- ✅ Lifecycle ordering constraints
- ✅ Context and concurrency requirements
- ✅ Testing requirements

Implementation MUST follow these specifications exactly to ensure compatibility with the Nomos CLI.
