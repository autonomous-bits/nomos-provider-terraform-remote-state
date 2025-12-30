// Package backend provides abstractions for retrieving Terraform state from various backend types.
//
// The backend package defines the core interface that all Terraform backend implementations
// must satisfy. This abstraction allows the provider to support multiple backend types
// (local filesystem, Azure Storage, S3, etc.) through a consistent interface.
//
// Backend implementations are responsible for:
//   - Connecting to the underlying storage system
//   - Retrieving raw state data
//   - Parsing state data into structured StateFile objects
//   - Handling backend-specific errors and converting them to standard errors
//
// The Backend interface uses context.Context for cancellation and timeout support,
// ensuring that long-running fetch operations can be properly controlled.
//
// # Registry Pattern
//
// The package uses a registry pattern to allow dynamic backend registration and creation.
// Backend types (local, azurerm, s3, etc.) register themselves using Register() during
// package initialization. The Init RPC uses Get() to retrieve the appropriate constructor
// and create backend instances.
//
// This architecture provides:
//   - Extensibility: New backend types can be added without modifying the provider
//   - Testability: Backends can be mocked or replaced for testing
//   - Type safety: Constructor signatures are enforced at compile time
//   - Thread safety: Registry access is protected by sync.RWMutex
//
// Example registration (in backend/local.go init()):
//
//	func init() {
//	    Register("local", func(ctx context.Context, config map[string]interface{}) (Backend, error) {
//	        // Extract and validate config fields
//	        path, ok := config["path"].(string)
//	        if !ok {
//	            return nil, fmt.Errorf("path must be a string")
//	        }
//	        workspace, _ := config["workspace"].(string)
//	        return NewLocalBackend(LocalBackendConfig{
//	            Path: path,
//	            Workspace: workspace,
//	        })
//	    })
//	}
//
// Example usage (in provider Init RPC):
//
//	constructor := backend.Get(cfg.Type())
//	if constructor == nil {
//	    return status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", cfg.Type())
//	}
//	backend, err := constructor(ctx, cfg.Raw())
//	if err != nil {
//	    return status.Errorf(codes.InvalidArgument, "failed to create backend: %v", err)
//	}
package backend

import (
	"context"
	"fmt"
	"sync"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

// Backend represents a Terraform backend that can retrieve state files.
//
// Backend implementations handle the storage-specific logic for fetching
// Terraform state data. Each implementation (local, azurerm, s3, etc.) is
// responsible for authenticating, connecting to storage, and retrieving
// the raw state file data.
//
// All Backend implementations must:
//   - Accept context for cancellation and timeout control
//   - Return properly parsed StateFile objects
//   - Convert backend-specific errors to appropriate gRPC status codes
//   - Handle concurrent requests safely
type Backend interface {
	// FetchState retrieves the Terraform state file from the backend.
	//
	// The context parameter enables cancellation and timeout control for
	// long-running fetch operations. Implementations should check ctx.Done()
	// before and during expensive I/O operations.
	//
	// Returns the parsed StateFile on success, or an error if:
	//   - The state file cannot be found (NotFound error)
	//   - Authentication fails (PermissionDenied error)
	//   - The network is unavailable (Unavailable error)
	//   - The state file is invalid or corrupted (InvalidArgument error)
	//   - The context is cancelled (Canceled error)
	//   - An unexpected error occurs (Internal error)
	FetchState(ctx context.Context) (*state.StateFile, error)
}

// Constructor is a factory function that creates a Backend instance from configuration.
//
// Constructor functions are registered with the backend registry during package initialization
// (in init() functions of backend implementation files like local.go, azurerm.go).
// The Init RPC retrieves the appropriate constructor using Get() and calls it to create
// backend instances.
//
// The constructor is responsible for:
//   - Extracting and type-asserting required fields from the config map
//   - Validating configuration values (format, ranges, etc.)
//   - Returning descriptive errors for invalid configuration
//   - Creating and initializing the backend instance
//
// Parameters:
//   - ctx: Context for cancellation and timeout control during initialization
//   - config: Raw configuration map from the gRPC InitRequest
//
// Returns:
//   - Backend: The initialized backend instance
//   - error: Configuration validation or initialization error
//
// The config map contains backend-specific fields as defined in the Nomos configuration.
// Backend implementations must validate all required fields and provide clear error messages
// for missing or invalid configuration.
//
// Example constructor implementation:
//
//	func localConstructor(ctx context.Context, config map[string]interface{}) (backend.Backend, error) {
//	    // Extract required field
//	    path, ok := config["path"].(string)
//	    if !ok {
//	        return nil, fmt.Errorf("missing required field: path")
//	    }
//	    if path == "" {
//	        return nil, fmt.Errorf("path cannot be empty")
//	    }
//
//	    // Extract optional field with default
//	    workspace := "default"
//	    if ws, ok := config["workspace"].(string); ok && ws != "" {
//	        workspace = ws
//	    }
//
//	    // Create backend
//	    return backend.NewLocalBackend(backend.LocalBackendConfig{
//	        Path:      path,
//	        Workspace: workspace,
//	    })
//	}
type Constructor func(ctx context.Context, config map[string]interface{}) (Backend, error)

// registry stores registered backend constructors by type name.
// Access is protected by registryMu to ensure thread-safe registration and retrieval.
//
// The registry is populated during package initialization when backend implementations
// call Register() in their init() functions. The map key is the backend type string
// (e.g., "local", "azurerm", "s3") and the value is the constructor function.
var registry = make(map[string]Constructor)

// registryMu protects concurrent access to the registry map.
//
// Write operations (Register) use Lock/Unlock.
// Read operations (Get, List) use RLock/RUnlock for concurrent read access.
//
// This mutex ensures thread-safe registry operations without requiring registration
// to complete before the server starts accepting requests.
var registryMu sync.RWMutex

// Register adds a backend constructor to the registry.
//
// This function should be called during package initialization (in init() functions)
// by backend implementations to register their constructor functions.
//
// Parameters:
//   - backendType: The backend type identifier (e.g., "local", "azurerm", "s3")
//   - constructor: The factory function that creates backend instances
//
// Panics if:
//   - backendType is empty
//   - constructor is nil
//   - backendType is already registered (duplicate registration)
//
// Example usage (in backend/local.go):
//
//	func init() {
//	    backend.Register("local", func(ctx context.Context, config map[string]interface{}) (backend.Backend, error) {
//	        // Extract and validate config
//	        path, ok := config["path"].(string)
//	        if !ok || path == "" {
//	            return nil, fmt.Errorf("missing required field: path")
//	        }
//	        workspace, _ := config["workspace"].(string)
//	        return backend.NewLocalBackend(backend.LocalBackendConfig{
//	            Path:      path,
//	            Workspace: workspace,
//	        })
//	    })
//	}
func Register(backendType string, constructor Constructor) {
	if backendType == "" {
		panic("backend: cannot register backend with empty type")
	}
	if constructor == nil {
		panic("backend: cannot register nil constructor")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[backendType]; exists {
		panic(fmt.Sprintf("backend: backend type %q already registered", backendType))
	}

	registry[backendType] = constructor
}

// Get retrieves a backend constructor from the registry.
//
// This function is called by the provider Init RPC to retrieve the appropriate
// constructor for creating backend instances based on the configured backend type.
//
// Parameters:
//   - backendType: The backend type identifier (e.g., "local", "azurerm", "s3")
//
// Returns:
//   - Constructor: The registered constructor function, or nil if not found
//
// Thread-safe: Uses RLock for concurrent read access.
//
// Example usage (in provider Init RPC):
//
//	constructor := backend.Get(cfg.Type())
//	if constructor == nil {
//	    return nil, status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", cfg.Type())
//	}
//	b, err := constructor(ctx, cfg.Raw())
//	if err != nil {
//	    return nil, status.Errorf(codes.InvalidArgument, "failed to create backend: %v", err)
//	}
func Get(backendType string) Constructor {
	registryMu.RLock()
	defer registryMu.RUnlock()

	return registry[backendType]
}

// GetBackend retrieves a backend constructor and creates a backend instance.
//
// This is a convenience function that combines Get() with constructor invocation
// and provides a gRPC-friendly error response when the backend type is not found.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control during backend creation
//   - backendType: The backend type identifier (e.g., "local", "azurerm")
//   - config: Raw configuration map for the backend
//
// Returns:
//   - Backend: The initialized backend instance
//   - error: Returns InvalidArgument gRPC error if type not found, or constructor error
//
// Thread-safe: Uses RLock for concurrent read access to the registry.
//
// Example usage (in provider Init RPC):
//
//	backend, err := backend.GetBackend(ctx, cfg.Type(), cfg.Raw())
//	if err != nil {
//	    return nil, err  // Already formatted as gRPC status error
//	}
func GetBackend(ctx context.Context, backendType string, config map[string]interface{}) (Backend, error) {
	registryMu.RLock()
	constructor, ok := registry[backendType]
	registryMu.RUnlock()

	if !ok {
		available := List()
		return nil, fmt.Errorf("unsupported backend type %q, available types: %v", backendType, available)
	}

	return constructor(ctx, config)
}

// List returns all registered backend type names.
//
// This function is useful for:
//   - Generating error messages with available backend types
//   - Diagnostic/debug output
//   - Testing registry state
//
// Returns:
//   - []string: Slice of registered backend type names (e.g., ["local", "azurerm"])
//
// Thread-safe: Uses RLock for concurrent read access.
// The returned slice is a copy and safe to modify.
//
// Example usage:
//
//	available := backend.List()
//	return fmt.Errorf("unsupported backend type %q, available: %v", backendType, available)
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	types := make([]string, 0, len(registry))
	for backendType := range registry {
		types = append(types, backendType)
	}
	return types
}
