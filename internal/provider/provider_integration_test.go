//go:build integration

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestIntegration_MultiWorkspaceProvider tests end-to-end provider workflow with multiple workspaces.
//
// This integration test verifies:
//   - Creating multiple workspace state files (default, dev, prod)
//   - Initializing provider with specific workspace configuration
//   - Fetching correct state from correct workspace
//   - Workspace isolation (each instance sees only its workspace)
func TestIntegration_MultiWorkspaceProvider(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create temporary directory structure for multiple workspaces
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Define workspace configurations
	workspaceConfigs := []struct {
		name        string
		outputKey   string
		outputVal   string
		stateSerial int
	}{
		{
			name:        "default",
			outputKey:   "environment",
			outputVal:   "default",
			stateSerial: 1,
		},
		{
			name:        "dev",
			outputKey:   "environment",
			outputVal:   "development",
			stateSerial: 2,
		},
		{
			name:        "prod",
			outputKey:   "environment",
			outputVal:   "production",
			stateSerial: 3,
		},
	}

	// Step 1: Create state files for each workspace
	for _, ws := range workspaceConfigs {
		createWorkspaceStateFile(t, tmpDir, basePath, ws.name, ws.outputKey, ws.outputVal, ws.stateSerial)
	}

	// Step 2: Initialize provider service
	service := NewService()
	ctx := context.Background()

	// Step 3: Initialize provider instances for each workspace
	for _, ws := range workspaceConfigs {
		t.Run("init_"+ws.name, func(t *testing.T) {
			config, err := structpb.NewStruct(map[string]interface{}{
				"type":      "local",
				"path":      basePath,
				"workspace": ws.name,
			})
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			req := &pb.InitRequest{
				Alias:  ws.name,
				Config: config,
			}

			resp, err := service.Init(ctx, req)
			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}

			if resp == nil {
				t.Fatal("Init() returned nil response")
			}

			// Verify instance was created
			service.mu.RLock()
			inst, exists := service.instances[ws.name]
			service.mu.RUnlock()

			if !exists {
				t.Fatalf("instance %q was not created", ws.name)
			}

			if inst.alias != ws.name {
				t.Errorf("instance alias = %s, want %s", inst.alias, ws.name)
			}

			if inst.backend == nil {
				t.Error("backend is nil")
			}
		})
	}

	// Step 4: Fetch data from each workspace and verify isolation
	for _, ws := range workspaceConfigs {
		t.Run("fetch_"+ws.name, func(t *testing.T) {
			// Temporarily set the instance to fetch from
			// In MVP, we fetch from any instance, but we can validate by re-initializing
			service2 := NewService()

			config, err := structpb.NewStruct(map[string]interface{}{
				"type":      "local",
				"path":      basePath,
				"workspace": ws.name,
			})
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			_, err = service2.Init(ctx, &pb.InitRequest{
				Alias:  "test",
				Config: config,
			})
			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}

			// Fetch the environment output
			fetchReq := &pb.FetchRequest{
				Path: []string{ws.outputKey},
			}

			resp, err := service2.Fetch(ctx, fetchReq)
			if err != nil {
				t.Fatalf("Fetch() error = %v", err)
			}

			if resp == nil {
				t.Fatal("Fetch() returned nil response")
			}

			if resp.Value == nil {
				t.Fatal("Fetch() returned nil value")
			}

			// Verify the correct value was fetched
			fields := resp.Value.AsMap()
			value, ok := fields["value"]
			if !ok {
				t.Fatal("response missing 'value' field")
			}

			valueStr, ok := value.(string)
			if !ok {
				t.Fatalf("value is not a string: %T", value)
			}

			if valueStr != ws.outputVal {
				t.Errorf("Fetch() value = %q, want %q for workspace %q", valueStr, ws.outputVal, ws.name)
			}
		})
	}

	// Step 5: Test workspace switching
	t.Run("workspace_switching", func(t *testing.T) {
		service3 := NewService()

		// Initialize with default workspace
		config1, err := structpb.NewStruct(map[string]interface{}{
			"type":      "local",
			"path":      basePath,
			"workspace": "default",
		})
		if err != nil {
			t.Fatalf("failed to create config struct: %v", err)
		}

		_, err = service3.Init(ctx, &pb.InitRequest{
			Alias:  "switch-test",
			Config: config1,
		})
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}

		// Fetch from default workspace
		resp1, err := service3.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"environment"},
		})
		if err != nil {
			t.Fatalf("Fetch() from default workspace error = %v", err)
		}

		value1 := resp1.Value.AsMap()["value"].(string)
		if value1 != "default" {
			t.Errorf("Fetch() from default = %q, want %q", value1, "default")
		}

		// Shutdown and reinitialize with prod workspace
		_, err = service3.Shutdown(ctx, &pb.ShutdownRequest{})
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}

		config2, err := structpb.NewStruct(map[string]interface{}{
			"type":      "local",
			"path":      basePath,
			"workspace": "prod",
		})
		if err != nil {
			t.Fatalf("failed to create config struct: %v", err)
		}

		_, err = service3.Init(ctx, &pb.InitRequest{
			Alias:  "switch-test",
			Config: config2,
		})
		if err != nil {
			t.Fatalf("Init() after workspace switch error = %v", err)
		}

		// Fetch from prod workspace
		resp2, err := service3.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"environment"},
		})
		if err != nil {
			t.Fatalf("Fetch() from prod workspace error = %v", err)
		}

		value2 := resp2.Value.AsMap()["value"].(string)
		if value2 != "production" {
			t.Errorf("Fetch() from prod = %q, want %q", value2, "production")
		}
	})
}

// TestIntegration_WorkspaceNotFound tests error handling for non-existent workspaces.
func TestIntegration_WorkspaceNotFound(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Create only default workspace state
	createWorkspaceStateFile(t, tmpDir, basePath, "default", "test", "value", 1)

	service := NewService()
	ctx := context.Background()

	// Try to initialize with non-existent workspace
	config, err := structpb.NewStruct(map[string]interface{}{
		"type":      "local",
		"path":      basePath,
		"workspace": "nonexistent",
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	_, err = service.Init(ctx, &pb.InitRequest{
		Alias:  "test",
		Config: config,
	})
	if err != nil {
		t.Fatalf("Init() error = %v (init should succeed, error occurs on fetch)", err)
	}

	// Fetch should fail with NotFound
	_, err = service.Fetch(ctx, &pb.FetchRequest{
		Path: []string{"test"},
	})

	if err == nil {
		t.Fatal("Fetch() expected error for non-existent workspace, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	if st.Code() != codes.NotFound {
		t.Errorf("Fetch() code = %v, want %v", st.Code(), codes.NotFound)
	}

	// Verify error message mentions the workspace
	if !contains(st.Message(), "state file not found") {
		t.Errorf("Fetch() error message = %q, want message containing 'state file not found'", st.Message())
	}
}

// TestIntegration_ConcurrentWorkspaceOperations tests concurrent operations across workspaces.
func TestIntegration_ConcurrentWorkspaceOperations(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Create state files for multiple workspaces
	workspaces := []string{"default", "dev", "staging", "production"}
	for i, ws := range workspaces {
		createWorkspaceStateFile(t, tmpDir, basePath, ws, "workspace_id", ws, i+1)
	}

	// Create multiple service instances
	services := make([]*Service, len(workspaces))
	for i := range services {
		services[i] = NewService()
	}

	ctx := context.Background()

	// Concurrently initialize and fetch from all workspaces
	done := make(chan error, len(workspaces))

	for i, ws := range workspaces {
		i, ws := i, ws // Capture for goroutine
		go func() {
			service := services[i]

			// Initialize
			config, err := structpb.NewStruct(map[string]interface{}{
				"type":      "local",
				"path":      basePath,
				"workspace": ws,
			})
			if err != nil {
				done <- err
				return
			}

			_, err = service.Init(ctx, &pb.InitRequest{
				Alias:  "test",
				Config: config,
			})
			if err != nil {
				done <- err
				return
			}

			// Fetch
			resp, err := service.Fetch(ctx, &pb.FetchRequest{
				Path: []string{"workspace_id"},
			})
			if err != nil {
				done <- err
				return
			}

			// Verify correct workspace data
			value := resp.Value.AsMap()["value"].(string)
			if value != ws {
				done <- status.Errorf(codes.Internal, "workspace mismatch: got %q, want %q", value, ws)
				return
			}

			done <- nil
		}()
	}

	// Wait for all operations to complete
	for i := 0; i < len(workspaces); i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}
}

// TestIntegration_WorkspaceWithComplexOutputs tests workspaces with various output types.
func TestIntegration_WorkspaceWithComplexOutputs(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Create state file with complex outputs
	workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", "complex")
	// #nosec G301 -- test directory, relaxed permissions acceptable
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace directory: %v", err)
	}

	statePath := filepath.Join(workspaceDir, "terraform.tfstate")
	stateData := `{
		"version": 4,
		"terraform_version": "1.5.0",
		"serial": 1,
		"lineage": "complex-test",
		"outputs": {
			"string_output": {
				"value": "test_string",
				"type": "string",
				"sensitive": false
			},
			"number_output": {
				"value": 42,
				"type": "number",
				"sensitive": false
			},
			"bool_output": {
				"value": true,
				"type": "bool",
				"sensitive": false
			},
			"sensitive_output": {
				"value": "secret_value",
				"type": "string",
				"sensitive": true
			}
		}
	}`

	if err := os.WriteFile(statePath, []byte(stateData), 0600); err != nil {
		t.Fatalf("failed to create state file: %v", err)
	}

	service := NewService()
	ctx := context.Background()

	// Initialize provider with complex workspace
	config, err := structpb.NewStruct(map[string]interface{}{
		"type":      "local",
		"path":      basePath,
		"workspace": "complex",
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	_, err = service.Init(ctx, &pb.InitRequest{
		Alias:  "test",
		Config: config,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test fetching different output types
	tests := []struct {
		name          string
		outputName    string
		expectedValue interface{}
		expectedType  string
	}{
		{
			name:          "string output",
			outputName:    "string_output",
			expectedValue: "test_string",
			expectedType:  "string",
		},
		{
			name:          "number output",
			outputName:    "number_output",
			expectedValue: float64(42), // JSON numbers are float64
			expectedType:  "number",
		},
		{
			name:          "bool output",
			outputName:    "bool_output",
			expectedValue: true,
			expectedType:  "bool",
		},
		{
			name:          "sensitive output",
			outputName:    "sensitive_output",
			expectedValue: "secret_value",
			expectedType:  "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.Fetch(ctx, &pb.FetchRequest{
				Path: []string{tt.outputName},
			})
			if err != nil {
				t.Fatalf("Fetch() error = %v", err)
			}

			fields := resp.Value.AsMap()
			value, ok := fields["value"]
			if !ok {
				t.Fatal("response missing 'value' field")
			}

			// Compare values based on type
			switch expected := tt.expectedValue.(type) {
			case string:
				if str, ok := value.(string); ok {
					if str != expected {
						t.Errorf("value = %q, want %q", str, expected)
					}
				} else {
					t.Errorf("value type = %T, want string", value)
				}
			case float64:
				if num, ok := value.(float64); ok {
					if num != expected {
						t.Errorf("value = %v, want %v", num, expected)
					}
				} else {
					t.Errorf("value type = %T, want float64", value)
				}
			case bool:
				if b, ok := value.(bool); ok {
					if b != expected {
						t.Errorf("value = %v, want %v", b, expected)
					}
				} else {
					t.Errorf("value type = %T, want bool", value)
				}
			}

			// Verify type field
			if typeVal, ok := fields["type"]; ok {
				if typeStr, ok := typeVal.(string); ok {
					if typeStr != tt.expectedType {
						t.Errorf("type = %q, want %q", typeStr, tt.expectedType)
					}
				}
			}

			// Verify sensitive field for sensitive output
			if tt.outputName == "sensitive_output" {
				if sensitive, ok := fields["sensitive"]; ok {
					if sensitiveBool, ok := sensitive.(bool); ok {
						if !sensitiveBool {
							t.Error("sensitive output not marked as sensitive")
						}
					}
				}
			}
		})
	}
}

// createWorkspaceStateFile is a helper function to create state files for testing.
//
// It creates the appropriate directory structure for the workspace and writes
// a valid Terraform state file with the specified output.
func createWorkspaceStateFile(t *testing.T, tmpDir, basePath, workspace, outputKey, outputValue string, serial int) {
	t.Helper()

	var statePath string
	if workspace == "default" {
		statePath = basePath
	} else {
		workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", workspace)
		// #nosec G301 -- test directory, relaxed permissions acceptable
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace directory for %s: %v", workspace, err)
		}
		statePath = filepath.Join(workspaceDir, "terraform.tfstate")
	}

	stateData := `{
		"version": 4,
		"terraform_version": "1.5.0",
		"serial": ` + string(rune(serial+48)) + `,
		"lineage": "` + workspace + `-lineage",
		"outputs": {
			"` + outputKey + `": {
				"value": "` + outputValue + `",
				"type": "string",
				"sensitive": false
			}
		}
	}`

	if err := os.WriteFile(statePath, []byte(stateData), 0600); err != nil {
		t.Fatalf("failed to create state file for workspace %s: %v", workspace, err)
	}
}
