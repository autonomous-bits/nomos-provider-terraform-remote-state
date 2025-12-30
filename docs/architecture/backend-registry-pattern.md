# Backend Registry Pattern Architecture

## Overview

The backend registry pattern provides a flexible, extensible mechanism for registering and creating backend instances. This architecture replaces the hardcoded switch statement in the provider Init RPC with a dynamic registry that allows backends to self-register during package initialization.

## Architecture Goals

1. **Extensibility**: Add new backend types without modifying provider code
2. **Type Safety**: Enforce constructor signatures at compile time
3. **Thread Safety**: Support concurrent registration and retrieval
4. **Testability**: Enable backend mocking and replacement
5. **Clear Separation**: Backend logic stays in backend package

## Component Design

### 1. BackendConstructor Type

```go
type BackendConstructor func(ctx context.Context, config map[string]interface{}) (Backend, error)
```

**Purpose**: Factory function signature for creating backend instances.

**Parameters**:
- `ctx context.Context`: Cancellation and timeout control for initialization
- `config map[string]interface{}`: Raw configuration from gRPC InitRequest

**Returns**:
- `Backend`: Initialized backend instance
- `error`: Configuration validation or initialization error

**Responsibilities**:
- Extract and type-assert configuration fields
- Validate configuration values
- Return descriptive errors for invalid config
- Create and initialize backend instance

### 2. Registry Variables

```go
var (
    registry   = make(map[string]BackendConstructor)
    registryMu sync.RWMutex
)
```

**registry**: Map of backend type names to constructor functions
- Key: Backend type string (e.g., "local", "azurerm", "s3")
- Value: Constructor function implementing BackendConstructor signature

**registryMu**: Read-write mutex for thread-safe access
- Write lock (Lock/Unlock): Used by Register for adding constructors
- Read lock (RLock/RUnlock): Used by Get and List for retrieving constructors

### 3. Registry Functions

#### Register(backendType string, constructor BackendConstructor)

**Purpose**: Add a backend constructor to the registry.

**Called by**: Backend implementations in their init() functions.

**Parameters**:
- `backendType`: Unique identifier for the backend (e.g., "local", "azurerm")
- `constructor`: Factory function that creates backend instances

**Behavior**:
- Panics if backendType is empty
- Panics if constructor is nil
- Panics if backendType already registered (prevents duplicate registration)
- Uses write lock (registryMu.Lock/Unlock) for thread safety

**Error Handling**: Panics on invalid input (appropriate for init-time errors)

#### Get(backendType string) BackendConstructor

**Purpose**: Retrieve a backend constructor from the registry.

**Called by**: Provider Init RPC to create backend instances.

**Parameters**:
- `backendType`: Backend type identifier

**Returns**:
- `BackendConstructor`: Registered constructor, or nil if not found

**Behavior**:
- Uses read lock (registryMu.RLock/RUnlock) for thread safety
- Returns nil if backendType not registered
- Allows concurrent reads from multiple goroutines

**Error Handling**: Returns nil for unknown backend types (caller handles error)

#### List() []string

**Purpose**: Return all registered backend type names.

**Called by**: Error messages, diagnostics, testing.

**Returns**:
- `[]string`: Slice of registered backend type names

**Behavior**:
- Uses read lock (registryMu.RLock/RUnlock) for thread safety
- Returns a copy of keys (safe to modify)
- Useful for generating "available backends: ..." error messages

## Implementation Pattern

### Backend Registration (local.go, azurerm.go)

Each backend implementation registers itself during package initialization:

```go
// In internal/backend/local.go
func init() {
    Register("local", func(ctx context.Context, config map[string]interface{}) (Backend, error) {
        // Extract path
        path, ok := config["path"].(string)
        if !ok {
            return nil, fmt.Errorf("missing required field: path")
        }
        if path == "" {
            return nil, fmt.Errorf("path cannot be empty")
        }

        // Extract optional workspace
        workspace := "default"
        if ws, ok := config["workspace"].(string); ok && ws != "" {
            workspace = ws
        }

        // Create backend
        return NewLocalBackend(LocalBackendConfig{
            Path:      path,
            Workspace: workspace,
        })
    })
}
```

**Key Points**:
- Registration happens in init(), executed automatically at startup
- Constructor validates all required fields
- Constructor provides clear error messages for missing/invalid config
- Constructor calls the existing New*Backend() function

### Provider Init RPC Usage

The provider Init RPC uses the registry to create backends:

```go
// In internal/provider/provider.go Init()

// Get constructor from registry
constructor := backend.Get(cfg.Type())
if constructor == nil {
    return nil, status.Errorf(codes.InvalidArgument, 
        "unsupported backend type: %s (available: %v)", 
        cfg.Type(), backend.List())
}

// Create backend instance
b, err := constructor(ctx, cfg.Raw())
if err != nil {
    return nil, status.Errorf(codes.InvalidArgument, 
        "failed to create backend: %v", err)
}

// Store instance
s.instances[req.Alias] = &instance{
    alias:   req.Alias,
    backend: b,
}
```

**Key Points**:
- Replace switch statement with registry lookup
- Use backend.Get() to retrieve constructor
- Check for nil (unknown backend type)
- Call constructor with context and raw config
- Handle constructor errors appropriately

### Removal of createLocalBackend and createAzureBackend

The existing helper functions in provider.go will be removed:
- `createLocalBackend(cfg config.BackendConfig) (backend.Backend, error)`
- `createAzureBackend(ctx context.Context, cfg config.BackendConfig) (backend.Backend, error)`

These are replaced by constructor functions registered in the backend implementations.

## Thread Safety Model

### Registration Phase (init time)
- Occurs during package initialization before main() runs
- Sequential execution (Go guarantees init() functions run in dependency order)
- Write lock (registryMu.Lock) protects against theoretical concurrent init

### Runtime Phase (after server starts)
- Read-only access to registry via Get() and List()
- Read lock (registryMu.RLock) allows concurrent reads
- No modifications to registry after initialization

### Why sync.RWMutex?
- Supports multiple concurrent readers (Get, List)
- Single writer during registration (Register)
- More efficient than sync.Mutex for read-heavy workloads
- Standard Go pattern for registry implementations

## Error Handling Strategy

### Registration Errors (init time)
- **Panic**: Empty backendType, nil constructor, duplicate registration
- **Rationale**: Registration happens at init time; panics prevent invalid server startup
- **Recovery**: Not attempted; server should fail to start with clear error

### Construction Errors (runtime)
- **Return error**: Missing/invalid config fields, validation failures
- **Rationale**: Configuration errors are user errors, not programming errors
- **Handling**: Provider maps to codes.InvalidArgument gRPC status

### Retrieval Errors (runtime)
- **Return nil**: Unknown backend type
- **Rationale**: Allows provider to generate helpful error with available types
- **Handling**: Provider checks for nil and returns codes.InvalidArgument

## Testing Strategy

### Unit Tests for Registry Functions

Test file: `internal/backend/backend_test.go`

**Register() Tests**:
- Successful registration of valid constructor
- Panic on empty backendType
- Panic on nil constructor
- Panic on duplicate registration

**Get() Tests**:
- Returns correct constructor for registered type
- Returns nil for unregistered type
- Thread-safe concurrent reads

**List() Tests**:
- Returns empty slice for empty registry
- Returns all registered types
- Returns copy (modifications don't affect registry)

### Integration Tests with Backends

Test that backend implementations register correctly:
- local backend registers "local" type
- azurerm backend registers "azurerm" type
- Constructors create valid backend instances

### Provider Init Tests

Test provider Init RPC with registry:
- Unknown backend type returns InvalidArgument
- Constructor errors mapped to InvalidArgument
- Successful creation stores backend instance

## Migration Path

### Phase 1: Add Registry Architecture (This Task)
- ✅ Define BackendConstructor type
- ✅ Add registry variables (registry, registryMu)
- ✅ Add Register, Get, List function signatures with panic stubs

### Phase 2: Implement Registry Functions (A5)
- Implement Register with validation and locking
- Implement Get with read locking
- Implement List with read locking
- Add unit tests

### Phase 3: Update Backend Implementations (A6, A7)
- Add init() to local.go with Register call
- Add init() to azurerm.go with Register call
- Test registration via integration tests

### Phase 4: Refactor Provider Init (A8)
- Replace switch statement with backend.Get()
- Remove createLocalBackend and createAzureBackend helpers
- Update error messages to use backend.List()
- Update tests

### Phase 5: Validation (A9)
- Run full test suite
- Verify coverage ≥80%
- Integration tests pass
- Manual testing with nomos-provider-file pattern

## Benefits Summary

### Before (Switch Statement)
```go
switch cfg.Type() {
case "local":
    b, err = createLocalBackend(cfg)
case "azurerm":
    b, err = createAzureBackend(ctx, cfg)
default:
    return status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", cfg.Type())
}
```

**Problems**:
- Adding backend requires modifying provider code
- Backend logic leaks into provider package
- Hard to test in isolation
- Tight coupling between provider and backends

### After (Registry Pattern)
```go
constructor := backend.Get(cfg.Type())
if constructor == nil {
    return status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", cfg.Type())
}
b, err := constructor(ctx, cfg.Raw())
```

**Benefits**:
- Add backends without modifying provider
- Backend logic stays in backend package
- Easy to mock for testing
- Loose coupling via interface
- Self-documenting (backend.List() shows available types)

## Future Extensibility

This architecture enables future enhancements:

1. **Plugin System**: Load backends from external packages
2. **Backend Metadata**: Register description, schema, capabilities
3. **Dynamic Registration**: Register backends at runtime (not just init)
4. **Backend Versioning**: Support multiple versions of same backend type
5. **Backend Discovery**: API to query available backends and their schemas

## Reference Implementation

This pattern follows standard Go registry patterns used in:
- `database/sql` driver registration
- `image` format registration
- `encoding/json` unmarshaler registration
- `net/http` handler registration

It adheres to Go proverbs:
- "Clear is better than clever"
- "Accept interfaces, return structs"
- "Make the zero value useful"
- "Design with composition in mind"
