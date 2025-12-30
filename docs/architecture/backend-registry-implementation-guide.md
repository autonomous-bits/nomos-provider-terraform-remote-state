# Backend Registry Implementation Guide

**For**: go-provider-implementer  
**Task**: Complete implementation of registry pattern functions  
**Status**: Architecture defined, implementation needed

## Quick Reference

### What's Already Done ✅

1. ✅ `BackendConstructor` type defined in [internal/backend/backend.go](../../internal/backend/backend.go)
2. ✅ `registry` and `registryMu` variables declared
3. ✅ Function signatures defined: `Register()`, `Get()`, `List()`
4. ✅ Comprehensive godoc comments and examples
5. ✅ Architecture document created

### What Needs Implementation 🔨

Tasks for go-provider-implementer:

#### Task A5: Implement Registry Functions

**File**: [internal/backend/backend.go](../../internal/backend/backend.go)

**Functions to Implement**:

1. **Register(backendType string, constructor BackendConstructor)**
   ```go
   func Register(backendType string, constructor BackendConstructor) {
       // Replace panic with implementation:
       // 1. Validate backendType is not empty (panic if empty)
       // 2. Validate constructor is not nil (panic if nil)
       // 3. Lock registryMu for writing
       // 4. Check if already registered (panic if duplicate)
       // 5. Add to registry map
       // 6. Unlock registryMu
   }
   ```

2. **Get(backendType string) BackendConstructor**
   ```go
   func Get(backendType string) BackendConstructor {
       // Replace panic with implementation:
       // 1. RLock registryMu for reading
       // 2. Look up backendType in registry
       // 3. RUnlock registryMu
       // 4. Return constructor (or nil if not found)
   }
   ```

3. **List() []string**
   ```go
   func List() []string {
       // Replace panic with implementation:
       // 1. RLock registryMu for reading
       // 2. Create slice with capacity len(registry)
       // 3. Iterate registry and append keys to slice
       // 4. RUnlock registryMu
       // 5. Return slice (copy, safe to modify)
   }
   ```

**Testing Requirements**:
- Create [internal/backend/backend_test.go](../../internal/backend/backend_test.go)
- Test Register: success, empty type (panic), nil constructor (panic), duplicate (panic)
- Test Get: found, not found, concurrent reads
- Test List: empty registry, multiple types, returned slice is copy
- Coverage target: 100% for registry functions

#### Task A6: Add Local Backend Registration

**File**: [internal/backend/local.go](../../internal/backend/local.go)

**Add init() function**:
```go
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

        // Extract optional workspace with default
        workspace := "default"
        if ws, ok := config["workspace"].(string); ok && ws != "" {
            workspace = ws
        }

        // Create backend using existing constructor
        return NewLocalBackend(LocalBackendConfig{
            Path:      path,
            Workspace: workspace,
        })
    })
}
```

**Testing**:
- Add test in [internal/backend/local_test.go](../../internal/backend/local_test.go)
- Verify "local" type is registered
- Test constructor with valid config
- Test constructor with missing path (returns error)
- Test constructor with empty path (returns error)

#### Task A7: Add Azure Backend Registration

**File**: [internal/backend/azurerm.go](../../internal/backend/azurerm.go)

**Add init() function**:
```go
func init() {
    Register("azurerm", func(ctx context.Context, config map[string]interface{}) (Backend, error) {
        // Extract storage_account_name
        storageAccountName, ok := config["storage_account_name"].(string)
        if !ok {
            return nil, fmt.Errorf("missing required field: storage_account_name")
        }

        // Extract container_name
        containerName, ok := config["container_name"].(string)
        if !ok {
            return nil, fmt.Errorf("missing required field: container_name")
        }

        // Extract key
        key, ok := config["key"].(string)
        if !ok {
            return nil, fmt.Errorf("missing required field: key")
        }

        // Create backend using existing constructor
        return NewAzureBackend(ctx, AzureBackendConfig{
            StorageAccountName: storageAccountName,
            ContainerName:      containerName,
            Key:                key,
        })
    })
}
```

**Testing**:
- Add test in [internal/backend/azurerm_test.go](../../internal/backend/azurerm_test.go)
- Verify "azurerm" type is registered
- Test constructor with valid config
- Test constructor with missing required fields (returns error)

#### Task A8: Refactor Provider Init RPC

**File**: [internal/provider/provider.go](../../internal/provider/provider.go)

**Current Code to Replace**:
```go
// Create backend based on type
var b backend.Backend
switch cfg.Type() {
case "local":
    b, err = createLocalBackend(cfg)
case "azurerm":
    b, err = createAzureBackend(ctx, cfg)
default:
    return nil, status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", cfg.Type())
}
```

**New Code**:
```go
// Get constructor from registry
constructor := backend.Get(cfg.Type())
if constructor == nil {
    return nil, status.Errorf(codes.InvalidArgument, 
        "unsupported backend type: %s (available: %v)", 
        cfg.Type(), backend.List())
}

// Create backend instance
b, err := constructor(ctx, cfg.Raw())
```

**Functions to Remove**:
- Delete `createLocalBackend(cfg config.BackendConfig)` helper function
- Delete `createAzureBackend(ctx, cfg config.BackendConfig)` helper function

**Testing**:
- Update provider tests in [internal/provider/provider_test.go](../../internal/provider/provider_test.go)
- Test unknown backend type returns InvalidArgument with available types
- Test constructor errors mapped to InvalidArgument
- Test successful backend creation

## Implementation Order

Follow this sequence to minimize compilation errors:

1. **Implement Registry Functions** (A5)
   - Add implementations to backend.go
   - Create backend_test.go with comprehensive tests
   - Run: `go test ./internal/backend -v`

2. **Add Local Backend Registration** (A6)
   - Add init() to local.go
   - Add registration test to local_test.go
   - Run: `go test ./internal/backend -v`

3. **Add Azure Backend Registration** (A7)
   - Add init() to azurerm.go
   - Add registration test to azurerm_test.go
   - Run: `go test ./internal/backend -v`

4. **Refactor Provider Init** (A8)
   - Update Init() method in provider.go
   - Remove createLocalBackend and createAzureBackend
   - Update provider_test.go
   - Run: `go test ./internal/provider -v`

5. **Full Validation** (A9)
   - Run: `make verify`
   - Check coverage: `make coverage`
   - Integration tests: `go test -tags=integration ./...`

## Common Pitfalls to Avoid

### ❌ Don't: Lock Without Defer
```go
func Get(backendType string) BackendConstructor {
    registryMu.RLock()
    return registry[backendType] // Missing unlock!
}
```

### ✅ Do: Always Defer Unlock
```go
func Get(backendType string) BackendConstructor {
    registryMu.RLock()
    defer registryMu.RUnlock()
    return registry[backendType]
}
```

### ❌ Don't: Panic on Runtime Errors
```go
func Get(backendType string) BackendConstructor {
    constructor, ok := registry[backendType]
    if !ok {
        panic("backend not found") // DON'T PANIC AT RUNTIME!
    }
    return constructor
}
```

### ✅ Do: Return nil for Not Found
```go
func Get(backendType string) BackendConstructor {
    registryMu.RLock()
    defer registryMu.RUnlock()
    return registry[backendType] // Returns nil if not found
}
```

### ❌ Don't: Return Registry Map Directly
```go
func ListMap() map[string]BackendConstructor {
    return registry // Exposes internal map!
}
```

### ✅ Do: Return Copy of Keys
```go
func List() []string {
    registryMu.RLock()
    defer registryMu.RUnlock()
    
    keys := make([]string, 0, len(registry))
    for k := range registry {
        keys = append(keys, k)
    }
    return keys
}
```

## Testing Checklist

Before marking tasks complete:

- [ ] All tests pass: `make test`
- [ ] Coverage ≥80%: `make coverage`
- [ ] No race conditions: `go test -race ./...`
- [ ] Lint passes: `make lint`
- [ ] Format passes: `make fmt`
- [ ] Full verify: `make verify`

## Architecture Reference

See [backend-registry-pattern.md](backend-registry-pattern.md) for:
- Detailed component design
- Thread safety model
- Error handling strategy
- Complete examples
- Benefits analysis

## Questions?

If you encounter ambiguity:
1. Check architecture document for design rationale
2. Look at godoc comments in backend.go for specifications
3. Reference database/sql driver registration pattern
4. Ask go-provider-architect for clarification
