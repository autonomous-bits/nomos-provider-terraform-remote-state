//go:build !integration

package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

// mockBackend is a simple mock implementation of the Backend interface for testing.
type mockBackend struct {
	ctx    context.Context
	config map[string]interface{}
	data   *state.StateFile
	err    error
}

func (m *mockBackend) FetchState(_ context.Context) (*state.StateFile, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

// mockConstructor creates a mock backend from config for testing.
func mockConstructor(ctx context.Context, config map[string]interface{}) (Backend, error) {
	// Validate required field for testing
	if _, ok := config["test_field"]; !ok {
		return nil, errors.New("missing required field: test_field")
	}

	return &mockBackend{
		ctx:    ctx,
		config: config,
		data: &state.StateFile{
			Version:          4,
			TerraformVersion: "1.5.0",
			Serial:           1,
			Lineage:          "test-lineage",
			Outputs:          make(map[string]*state.OutputValue),
		},
	}, nil
}

// errorConstructor always returns an error for testing error propagation.
func errorConstructor(_ context.Context, _ map[string]interface{}) (Backend, error) {
	return nil, errors.New("constructor error")
}

// contextAwareConstructor checks if context is cancelled for testing context handling.
func contextAwareConstructor(ctx context.Context, config map[string]interface{}) (Backend, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return &mockBackend{
			ctx:    ctx,
			config: config,
		}, nil
	}
}

// TestRegister tests the Register function.
func TestRegister(t *testing.T) {
	tests := []struct {
		name         string
		backendType  string
		constructor  Constructor
		setupFunc    func()
		wantPanic    bool
		wantPanicMsg string
	}{
		{
			name:        "successful registration",
			backendType: "test-backend",
			constructor: mockConstructor,
			setupFunc:   func() {},
			wantPanic:   false,
		},
		{
			name:         "empty backend type",
			backendType:  "",
			constructor:  mockConstructor,
			setupFunc:    func() {},
			wantPanic:    true,
			wantPanicMsg: "backend: cannot register backend with empty type",
		},
		{
			name:         "nil constructor",
			backendType:  "test-backend-nil",
			constructor:  nil,
			setupFunc:    func() {},
			wantPanic:    true,
			wantPanicMsg: "backend: cannot register nil constructor",
		},
		{
			name:        "duplicate registration",
			backendType: "test-backend-dup",
			constructor: mockConstructor,
			setupFunc: func() {
				// Register the backend first
				Register("test-backend-dup", mockConstructor)
			},
			wantPanic:    true,
			wantPanicMsg: `backend: backend type "test-backend-dup" already registered`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setupFunc()

			// Clean up registry after test
			defer func() {
				registryMu.Lock()
				delete(registry, tt.backendType)
				delete(registry, "test-backend-dup")
				registryMu.Unlock()
			}()

			if tt.wantPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Error("Register() did not panic, expected panic")
						return
					}
					if msg, ok := r.(string); ok {
						if msg != tt.wantPanicMsg {
							t.Errorf("Register() panic message = %v, want %v", msg, tt.wantPanicMsg)
						}
					}
				}()
			}

			Register(tt.backendType, tt.constructor)

			if !tt.wantPanic {
				// Verify registration succeeded
				registryMu.RLock()
				_, exists := registry[tt.backendType]
				registryMu.RUnlock()
				if !exists {
					t.Errorf("Register() did not add backend type %q to registry", tt.backendType)
				}
			}
		})
	}
}

// TestGet tests the Get function for retrieving constructors.
func TestGet(t *testing.T) {
	// Setup: Register a test backend
	testBackendType := "test-get-backend"
	Register(testBackendType, mockConstructor)
	defer func() {
		registryMu.Lock()
		delete(registry, testBackendType)
		registryMu.Unlock()
	}()

	tests := []struct {
		name        string
		backendType string
		wantNil     bool
	}{
		{
			name:        "retrieve registered constructor",
			backendType: testBackendType,
			wantNil:     false,
		},
		{
			name:        "retrieve non-existent constructor",
			backendType: "nonexistent-backend",
			wantNil:     true,
		},
		{
			name:        "retrieve local backend (built-in)",
			backendType: "local",
			wantNil:     false,
		},
		{
			name:        "retrieve azurerm backend (built-in)",
			backendType: "azurerm",
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get(tt.backendType)
			if (got == nil) != tt.wantNil {
				t.Errorf("Get(%q) = %v, wantNil = %v", tt.backendType, got != nil, tt.wantNil)
			}
		})
	}
}

// TestList tests the List function for retrieving all backend types.
func TestList(t *testing.T) {
	t.Run("list returns registered backend types", func(t *testing.T) {
		// Get initial list
		types := List()

		// Should contain built-in backends
		hasLocal := false
		hasAzurerm := false
		for _, backendType := range types {
			if backendType == "local" {
				hasLocal = true
			}
			if backendType == "azurerm" {
				hasAzurerm = true
			}
		}

		if !hasLocal {
			t.Error("List() does not include 'local' backend")
		}
		if !hasAzurerm {
			t.Error("List() does not include 'azurerm' backend")
		}
	})

	t.Run("list returns a copy", func(t *testing.T) {
		// Get the list
		types := List()

		// Store original length
		originalLen := len(types)

		// Get the list again
		newTypes := List()

		// Should have original length
		if len(newTypes) != originalLen {
			t.Errorf("List() slice was not a copy, length changed from %d to %d", originalLen, len(newTypes))
		}

		// Modifying the returned slice should not affect subsequent calls
		_ = append(types, "modified-backend")

		// Should not contain the modification
		for _, backendType := range newTypes {
			if backendType == "modified-backend" {
				t.Error("List() slice was not a copy, modification affected registry")
			}
		}
	})

	t.Run("list includes dynamically registered backends", func(t *testing.T) {
		testBackendType := "test-list-backend"
		Register(testBackendType, mockConstructor)
		defer func() {
			registryMu.Lock()
			delete(registry, testBackendType)
			registryMu.Unlock()
		}()

		types := List()

		found := false
		for _, backendType := range types {
			if backendType == testBackendType {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("List() does not include dynamically registered backend %q", testBackendType)
		}
	})
}

// TestGetBackend tests the GetBackend function for creating backend instances.
func TestGetBackend(t *testing.T) {
	// Setup: Register test backends
	successBackendType := "test-getbackend-success"
	errorBackendType := "test-getbackend-error"
	contextBackendType := "test-getbackend-context"

	Register(successBackendType, mockConstructor)
	Register(errorBackendType, errorConstructor)
	Register(contextBackendType, contextAwareConstructor)

	defer func() {
		registryMu.Lock()
		delete(registry, successBackendType)
		delete(registry, errorBackendType)
		delete(registry, contextBackendType)
		registryMu.Unlock()
	}()

	tests := []struct {
		name        string
		backendType string
		config      map[string]interface{}
		setupCtx    func() context.Context
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful backend creation",
			backendType: successBackendType,
			config: map[string]interface{}{
				"test_field": "test_value",
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name:        "unknown backend type",
			backendType: "nonexistent-backend",
			config:      map[string]interface{}{},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "unsupported backend type",
		},
		{
			name:        "error message includes available types",
			backendType: "invalid-backend",
			config:      map[string]interface{}{},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "available types:",
		},
		{
			name:        "constructor error is propagated",
			backendType: errorBackendType,
			config:      map[string]interface{}{},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "constructor error",
		},
		{
			name:        "constructor config validation error",
			backendType: successBackendType,
			config:      map[string]interface{}{
				// Missing required test_field
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr:     true,
			errContains: "missing required field",
		},
		{
			name:        "context is passed to constructor",
			backendType: contextBackendType,
			config:      map[string]interface{}{},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name:        "context cancellation is handled",
			backendType: contextBackendType,
			config:      map[string]interface{}{},
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErr:     true,
			errContains: "context canceled",
		},
		{
			name:        "create local backend through factory",
			backendType: "local",
			config: map[string]interface{}{
				"path":      "/tmp/terraform.tfstate",
				"workspace": "default",
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name:        "create azurerm backend through factory",
			backendType: "azurerm",
			config: map[string]interface{}{
				"storage_account_name": "testaccount",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			got, err := GetBackend(ctx, tt.backendType, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBackend() error = nil, wantErr = true")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetBackend() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("GetBackend() unexpected error = %v", err)
				return
			}

			if got == nil {
				t.Error("GetBackend() returned nil backend")
			}
		})
	}
}

// TestGetBackend_ErrorListsAvailableTypes verifies that error messages include available backend types.
func TestGetBackend_ErrorListsAvailableTypes(t *testing.T) {
	ctx := context.Background()
	_, err := GetBackend(ctx, "invalid-backend-xyz", map[string]interface{}{})

	if err == nil {
		t.Fatal("GetBackend() error = nil, want error")
	}

	errMsg := err.Error()

	// Should mention the invalid backend type
	if !strings.Contains(errMsg, "invalid-backend-xyz") {
		t.Errorf("GetBackend() error message should contain the invalid backend type, got: %v", errMsg)
	}

	// Should list available types
	if !strings.Contains(errMsg, "available types:") {
		t.Errorf("GetBackend() error message should list available types, got: %v", errMsg)
	}

	// Should include built-in backends in the list
	if !strings.Contains(errMsg, "local") {
		t.Errorf("GetBackend() error message should include 'local' in available types, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "azurerm") {
		t.Errorf("GetBackend() error message should include 'azurerm' in available types, got: %v", errMsg)
	}
}

// TestConcurrentRegistration tests thread safety of registration operations.
func TestConcurrentRegistration(t *testing.T) {
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Each goroutine registers a unique backend type
			backendType := fmt.Sprintf("concurrent-backend-%d", id)
			Register(backendType, mockConstructor)

			// Clean up
			defer func() {
				registryMu.Lock()
				delete(registry, backendType)
				registryMu.Unlock()
			}()

			// Verify registration
			constructor := Get(backendType)
			if constructor == nil {
				t.Errorf("Concurrent registration failed for backend type %q", backendType)
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentGetBackend tests thread safety of backend creation.
func TestConcurrentGetBackend(t *testing.T) {
	// This test is more realistic with parallel subtests
	t.Parallel()

	testBackendType := "test-concurrent-getbackend"
	Register(testBackendType, mockConstructor)
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, testBackendType)
		registryMu.Unlock()
	})

	const numGoroutines = 20

	tests := []struct {
		name        string
		backendType string
		config      map[string]interface{}
	}{
		{
			name:        "concurrent access 1",
			backendType: testBackendType,
			config:      map[string]interface{}{"test_field": "value1"},
		},
		{
			name:        "concurrent access 2",
			backendType: testBackendType,
			config:      map[string]interface{}{"test_field": "value2"},
		},
		{
			name:        "concurrent access 3",
			backendType: "local",
			config:      map[string]interface{}{"path": "/tmp/test.tfstate"},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer wg.Done()

					ctx := context.Background()
					backend, err := GetBackend(ctx, tt.backendType, tt.config)
					if err != nil {
						t.Errorf("GetBackend() error = %v", err)
						return
					}
					if backend == nil {
						t.Error("GetBackend() returned nil backend")
					}
				}()
			}

			wg.Wait()
		})
	}
}

// TestConcurrentList tests thread safety of listing backend types.
func TestConcurrentList(t *testing.T) {
	// Don't run in parallel to avoid interference from other tests
	// that may be registering backends concurrently

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Use a single channel with buffering to collect results quickly
	results := make(chan []string, numGoroutines)

	// Start all goroutines at roughly the same time
	start := make(chan struct{})
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // Wait for signal
			types := List()
			results <- types
		}()
	}

	// Signal all goroutines to start
	close(start)
	wg.Wait()
	close(results)

	// Collect all results
	var allResults [][]string
	for types := range results {
		allResults = append(allResults, types)
	}

	// All calls should return the same set (but potentially different order)
	if len(allResults) == 0 {
		t.Fatal("No results collected")
	}

	// Create map of first result
	firstMap := make(map[string]bool)
	for _, t := range allResults[0] {
		firstMap[t] = true
	}

	// Verify all results have the same set of backends
	for i, types := range allResults {
		typeMap := make(map[string]bool)
		for _, t := range types {
			typeMap[t] = true
		}

		if len(typeMap) != len(firstMap) {
			t.Errorf("Result %d: List() returned different number of types: got %d, want %d", i, len(typeMap), len(firstMap))
			t.Logf("First result: %v", allResults[0])
			t.Logf("Result %d: %v", i, types)
			continue
		}

		for backendType := range firstMap {
			if !typeMap[backendType] {
				t.Errorf("Result %d: List() missing backend type %q", i, backendType)
			}
		}
	}
}

// TestBuiltinBackends verifies that built-in backends are registered.
func TestBuiltinBackends(t *testing.T) {
	tests := []struct {
		name        string
		backendType string
	}{
		{
			name:        "local backend is registered",
			backendType: "local",
		},
		{
			name:        "azurerm backend is registered",
			backendType: "azurerm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constructor := Get(tt.backendType)
			if constructor == nil {
				t.Errorf("Get(%q) = nil, expected built-in backend to be registered", tt.backendType)
			}

			// Verify it's in the List
			types := List()
			found := false
			for _, backendType := range types {
				if backendType == tt.backendType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("List() does not include built-in backend %q", tt.backendType)
			}
		})
	}
}

// TestBackendCreationThroughFactory tests creating actual backends through the factory.
func TestBackendCreationThroughFactory(t *testing.T) {
	tests := []struct {
		name        string
		backendType string
		config      map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "create local backend with valid config",
			backendType: "local",
			config: map[string]interface{}{
				"path":      "/tmp/terraform.tfstate",
				"workspace": "default",
			},
			wantErr: false,
		},
		{
			name:        "create local backend with default workspace",
			backendType: "local",
			config: map[string]interface{}{
				"path": "/tmp/terraform.tfstate",
			},
			wantErr: false,
		},
		{
			name:        "create local backend with missing path",
			backendType: "local",
			config:      map[string]interface{}{},
			wantErr:     true,
		},
		{
			name:        "create azurerm backend with valid config",
			backendType: "azurerm",
			config: map[string]interface{}{
				"storage_account_name": "validaccount",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantErr: false,
		},
		{
			name:        "create azurerm backend with missing storage_account_name",
			backendType: "azurerm",
			config: map[string]interface{}{
				"container_name": "tfstate",
				"key":            "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name:        "create azurerm backend with missing container_name",
			backendType: "azurerm",
			config: map[string]interface{}{
				"storage_account_name": "validaccount",
				"key":                  "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name:        "create azurerm backend with missing key",
			backendType: "azurerm",
			config: map[string]interface{}{
				"storage_account_name": "validaccount",
				"container_name":       "tfstate",
			},
			wantErr: true,
		},
		{
			name:        "create azurerm backend with invalid storage account name",
			backendType: "azurerm",
			config: map[string]interface{}{
				"storage_account_name": "InvalidAccount",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			backend, err := GetBackend(ctx, tt.backendType, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBackend() error = nil, wantErr = true")
				}
				return
			}

			if err != nil {
				t.Errorf("GetBackend() unexpected error = %v", err)
				return
			}

			if backend == nil {
				t.Error("GetBackend() returned nil backend")
			}
		})
	}
}

// TestMockBackend tests the mock backend implementation used in tests.
func TestMockBackend(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		mock := &mockBackend{
			ctx:    context.Background(),
			config: map[string]interface{}{"test": "value"},
			data: &state.StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Outputs:          make(map[string]*state.OutputValue),
			},
		}

		result, err := mock.FetchState(context.Background())
		if err != nil {
			t.Errorf("FetchState() error = %v", err)
			return
		}
		if result == nil {
			t.Error("FetchState() returned nil state")
			return
		}
		if result.Version != 4 {
			t.Errorf("FetchState() version = %d, want 4", result.Version)
		}
	})

	t.Run("fetch with error", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		mock := &mockBackend{
			ctx:    context.Background(),
			config: map[string]interface{}{"test": "value"},
			err:    expectedErr,
		}

		result, err := mock.FetchState(context.Background())
		if err == nil {
			t.Error("FetchState() error = nil, want error")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("FetchState() error = %v, want %v", err, expectedErr)
		}
		if result != nil {
			t.Errorf("FetchState() returned non-nil state with error")
		}
	})
}

// TestConstructors tests the mock constructors.
func TestConstructors(t *testing.T) {
	t.Run("mockConstructor success", func(t *testing.T) {
		ctx := context.Background()
		config := map[string]interface{}{"test_field": "value"}

		backend, err := mockConstructor(ctx, config)
		if err != nil {
			t.Errorf("mockConstructor() error = %v", err)
		}
		if backend == nil {
			t.Error("mockConstructor() returned nil backend")
		}

		// Verify the backend stores context and config
		if mock, ok := backend.(*mockBackend); ok {
			if mock.ctx != ctx {
				t.Error("mockConstructor() did not store context")
			}
			if mock.config["test_field"] != "value" {
				t.Error("mockConstructor() did not store config")
			}
		} else {
			t.Error("mockConstructor() did not return *mockBackend")
		}
	})

	t.Run("mockConstructor missing field", func(t *testing.T) {
		ctx := context.Background()
		config := map[string]interface{}{}

		backend, err := mockConstructor(ctx, config)
		if err == nil {
			t.Error("mockConstructor() error = nil, want error for missing field")
		}
		if backend != nil {
			t.Error("mockConstructor() returned non-nil backend with error")
		}
		if !strings.Contains(err.Error(), "missing required field") {
			t.Errorf("mockConstructor() error = %v, want error containing 'missing required field'", err)
		}
	})

	t.Run("errorConstructor always errors", func(t *testing.T) {
		ctx := context.Background()
		config := map[string]interface{}{}

		backend, err := errorConstructor(ctx, config)
		if err == nil {
			t.Error("errorConstructor() error = nil, want error")
		}
		if backend != nil {
			t.Error("errorConstructor() returned non-nil backend")
		}
		if !strings.Contains(err.Error(), "constructor error") {
			t.Errorf("errorConstructor() error = %v, want error containing 'constructor error'", err)
		}
	})

	t.Run("contextAwareConstructor respects context", func(t *testing.T) {
		config := map[string]interface{}{}

		// Test with active context
		ctx := context.Background()
		backend, err := contextAwareConstructor(ctx, config)
		if err != nil {
			t.Errorf("contextAwareConstructor() error = %v with active context", err)
		}
		if backend == nil {
			t.Error("contextAwareConstructor() returned nil backend with active context")
		}

		// Test with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		backend, err = contextAwareConstructor(ctx, config)
		if err == nil {
			t.Error("contextAwareConstructor() error = nil with cancelled context, want error")
		}
		if backend != nil {
			t.Error("contextAwareConstructor() returned non-nil backend with cancelled context")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("contextAwareConstructor() error = %v, want context.Canceled", err)
		}
	})
}
