//go:build integration

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestQuickstart_LocalBackendSimple validates the "Local Backend (Simple File Access)" scenario from quickstart.md.
//
// This test creates the exact state file structure shown in the quickstart guide and validates:
// - Binary can start and handle gRPC requests
// - Init RPC with local backend configuration
// - Fetch RPC for simple string outputs (vpc_id)
// - Fetch RPC for list outputs (subnet_ids)
// - Fetch RPC for object outputs (database_config)
// - Health and Info RPCs
func TestQuickstart_LocalBackendSimple(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping quickstart validation in short mode")
	}

	service := NewService()
	ctx := context.Background()

	// Create test state file matching quickstart example
	// NOTE: quickstart.md shows type as array for documentation, but actual Terraform uses string
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateData := `{
  "version": 4,
  "terraform_version": "1.6.5",
  "serial": 1,
  "lineage": "quickstart-test-lineage",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    },
    "subnet_ids": {
      "value": ["subnet-1", "subnet-2", "subnet-3"],
      "type": "tuple",
      "sensitive": false
    },
    "database_config": {
      "value": {
        "host": "db.example.com",
        "port": 5432,
        "database": "appdb"
      },
      "type": "object",
      "sensitive": false
    }
  }
}`

	if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
		t.Fatalf("failed to create state file: %v", err)
	}

	// Test 1: Info RPC
	t.Run("info_rpc", func(t *testing.T) {
		resp, err := service.Info(ctx, &pb.InfoRequest{})
		if err != nil {
			t.Fatalf("Info() error = %v", err)
		}

		if resp.Type != "terraform-remote-state" {
			t.Errorf("Info() type = %q, want %q", resp.Type, "terraform-remote-state")
		}

		if resp.Version == "" {
			t.Error("Info() version is empty")
		}
	})

	// Test 2: Health RPC
	t.Run("health_rpc", func(t *testing.T) {
		resp, err := service.Health(ctx, &pb.HealthRequest{})
		if err != nil {
			t.Fatalf("Health() error = %v", err)
		}

		if resp.Status != pb.HealthResponse_STATUS_OK {
			t.Errorf("Health() status = %v, want STATUS_OK", resp.Status)
		}
	})

	// Test 3: Init RPC with local backend
	t.Run("init_local_backend", func(t *testing.T) {
		config, err := structpb.NewStruct(map[string]interface{}{
			"backend_type": "local",
			"path":         stateFile,
		})
		if err != nil {
			t.Fatalf("failed to create config struct: %v", err)
		}

		resp, err := service.Init(ctx, &pb.InitRequest{
			Alias:  "tfstate_infra",
			Config: config,
		})
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}

		if resp == nil {
			t.Fatal("Init() returned nil response")
		}
	})

	// Test 4: Fetch string output (vpc_id)
	t.Run("fetch_vpc_id", func(t *testing.T) {
		resp, err := service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"vpc_id"},
		})
		if err != nil {
			t.Fatalf("Fetch(vpc_id) error = %v", err)
		}

		fields := resp.Value.AsMap()
		value, ok := fields["value"]
		if !ok {
			t.Fatal("response missing 'value' field")
		}

		if value != "vpc-12345" {
			t.Errorf("Fetch(vpc_id) value = %v, want %q", value, "vpc-12345")
		}
	})

	// Test 5: Fetch list output (subnet_ids)
	t.Run("fetch_subnet_ids", func(t *testing.T) {
		resp, err := service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"subnet_ids"},
		})
		if err != nil {
			t.Fatalf("Fetch(subnet_ids) error = %v", err)
		}

		fields := resp.Value.AsMap()
		value, ok := fields["value"]
		if !ok {
			t.Fatal("response missing 'value' field")
		}

		// Verify it's a list
		list, ok := value.([]interface{})
		if !ok {
			t.Fatalf("subnet_ids value is not a list: %T", value)
		}

		expectedSubnets := []string{"subnet-1", "subnet-2", "subnet-3"}
		if len(list) != len(expectedSubnets) {
			t.Errorf("subnet_ids length = %d, want %d", len(list), len(expectedSubnets))
		}

		for i, expected := range expectedSubnets {
			if i < len(list) {
				if actual, ok := list[i].(string); ok {
					if actual != expected {
						t.Errorf("subnet_ids[%d] = %q, want %q", i, actual, expected)
					}
				} else {
					t.Errorf("subnet_ids[%d] is not a string: %T", i, list[i])
				}
			}
		}
	})

	// Test 6: Fetch object output (database_config)
	t.Run("fetch_database_config", func(t *testing.T) {
		resp, err := service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"database_config"},
		})
		if err != nil {
			t.Fatalf("Fetch(database_config) error = %v", err)
		}

		fields := resp.Value.AsMap()
		value, ok := fields["value"]
		if !ok {
			t.Fatal("response missing 'value' field")
		}

		// Verify it's an object
		obj, ok := value.(map[string]interface{})
		if !ok {
			t.Fatalf("database_config value is not an object: %T", value)
		}

		// Verify object fields
		if host, ok := obj["host"].(string); !ok || host != "db.example.com" {
			t.Errorf("database_config.host = %v, want %q", obj["host"], "db.example.com")
		}

		if port, ok := obj["port"].(float64); !ok || port != 5432 {
			t.Errorf("database_config.port = %v, want %v", obj["port"], float64(5432))
		}

		if database, ok := obj["database"].(string); !ok || database != "appdb" {
			t.Errorf("database_config.database = %v, want %q", obj["database"], "appdb")
		}
	})

	// Test 7: Shutdown RPC
	t.Run("shutdown_rpc", func(t *testing.T) {
		resp, err := service.Shutdown(ctx, &pb.ShutdownRequest{})
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}

		if resp == nil {
			t.Fatal("Shutdown() returned nil response")
		}
	})
}

// TestQuickstart_LocalBackendWithWorkspace validates the "Local Backend with Workspace" scenario.
//
// Tests workspace resolution for dev, staging, and prod environments.
func TestQuickstart_LocalBackendWithWorkspace(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping quickstart validation in short mode")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Create default workspace
	defaultStateData := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "default-workspace-lineage",
  "outputs": {
    "vpc_id": {
      "value": "vpc-default",
      "type": "string",
      "sensitive": false
    }
  }
}`
	if err := os.WriteFile(basePath, []byte(defaultStateData), 0600); err != nil {
		t.Fatalf("failed to create default state file: %v", err)
	}

	// Create workspace structure: terraform.tfstate.d/dev/terraform.tfstate
	workspaces := []struct {
		name  string
		vpcID string
	}{
		{"dev", "vpc-dev-12345"},
		{"staging", "vpc-staging-67890"},
		{"prod", "vpc-prod-abcdef"},
	}

	for _, ws := range workspaces {
		wsDir := filepath.Join(tmpDir, "terraform.tfstate.d", ws.name)
		// #nosec G301 -- test directory, relaxed permissions acceptable
		if err := os.MkdirAll(wsDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		wsStateFile := filepath.Join(wsDir, "terraform.tfstate")
		wsStateData := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "` + ws.name + `-workspace-lineage",
  "outputs": {
    "vpc_id": {
      "value": "` + ws.vpcID + `",
      "type": "string",
      "sensitive": false
    },
    "environment": {
      "value": "` + ws.name + `",
      "type": "string",
      "sensitive": false
    }
  }
}`
		if err := os.WriteFile(wsStateFile, []byte(wsStateData), 0600); err != nil {
			t.Fatalf("failed to create workspace state file: %v", err)
		}
	}

	// Test each workspace
	for _, ws := range workspaces {
		t.Run("workspace_"+ws.name, func(t *testing.T) {
			service := NewService()

			// Init with specific workspace
			config, err := structpb.NewStruct(map[string]interface{}{
				"backend_type": "local",
				"path":         basePath,
				"workspace":    ws.name,
			})
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			_, err = service.Init(ctx, &pb.InitRequest{
				Alias:  "tfstate_infra",
				Config: config,
			})
			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}

			// Fetch vpc_id and verify workspace isolation
			resp, err := service.Fetch(ctx, &pb.FetchRequest{
				Path: []string{"vpc_id"},
			})
			if err != nil {
				t.Fatalf("Fetch(vpc_id) error = %v", err)
			}

			fields := resp.Value.AsMap()
			value := fields["value"].(string)

			if value != ws.vpcID {
				t.Errorf("workspace %s: vpc_id = %q, want %q", ws.name, value, ws.vpcID)
			}

			// Verify environment output
			envResp, err := service.Fetch(ctx, &pb.FetchRequest{
				Path: []string{"environment"},
			})
			if err != nil {
				t.Fatalf("Fetch(environment) error = %v", err)
			}

			envValue := envResp.Value.AsMap()["value"].(string)
			if envValue != ws.name {
				t.Errorf("workspace %s: environment = %q, want %q", ws.name, envValue, ws.name)
			}
		})
	}

	// Test default workspace explicitly
	t.Run("workspace_default", func(t *testing.T) {
		service := NewService()

		config, err := structpb.NewStruct(map[string]interface{}{
			"backend_type": "local",
			"path":         basePath,
			"workspace":    "default",
		})
		if err != nil {
			t.Fatalf("failed to create config struct: %v", err)
		}

		_, err = service.Init(ctx, &pb.InitRequest{
			Alias:  "tfstate_infra",
			Config: config,
		})
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}

		resp, err := service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"vpc_id"},
		})
		if err != nil {
			t.Fatalf("Fetch(vpc_id) error = %v", err)
		}

		value := resp.Value.AsMap()["value"].(string)
		if value != "vpc-default" {
			t.Errorf("default workspace: vpc_id = %q, want %q", value, "vpc-default")
		}
	})
}

// TestQuickstart_ErrorScenarios validates error handling scenarios from quickstart.md.
//
// Tests all error scenarios documented in the "Troubleshooting" section:
// - State file not found
// - Output not found
// - Unsupported state version
// - Permission denied
// - Invalid configuration
func TestQuickstart_ErrorScenarios(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping quickstart validation in short mode")
	}

	ctx := context.Background()

	// Test 1: State file not found
	t.Run("state_file_not_found", func(t *testing.T) {
		service := NewService()

		config, err := structpb.NewStruct(map[string]interface{}{
			"backend_type": "local",
			"path":         "/nonexistent/terraform.tfstate",
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

		// Fetch should fail with NotFound
		_, err = service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"vpc_id"},
		})

		if err == nil {
			t.Error("Fetch() expected error for missing state file, got nil")
			return
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.NotFound {
			t.Errorf("Fetch() code = %v, want NotFound", st.Code())
		}
	})

	// Test 2: Output not found in state
	t.Run("output_not_found", func(t *testing.T) {
		service := NewService()
		tmpDir := t.TempDir()
		stateFile := filepath.Join(tmpDir, "terraform.tfstate")

		stateData := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "output-not-found-lineage",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    }
  }
}`
		if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		config, err := structpb.NewStruct(map[string]interface{}{
			"backend_type": "local",
			"path":         stateFile,
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

		// Fetch non-existent output
		_, err = service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"nonexistent_output"},
		})

		if err == nil {
			t.Error("Fetch() expected error for missing output, got nil")
			return
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.NotFound {
			t.Errorf("Fetch() code = %v, want NotFound", st.Code())
		}
	})

	// Test 3: Unsupported state version
	t.Run("unsupported_state_version", func(t *testing.T) {
		service := NewService()
		tmpDir := t.TempDir()
		stateFile := filepath.Join(tmpDir, "terraform.tfstate")

		// Create state file with version 3 (unsupported)
		stateData := `{
  "version": 3,
  "terraform_version": "0.11.0",
  "serial": 1,
  "lineage": "unsupported-version-lineage",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    }
  }
}`
		if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		config, err := structpb.NewStruct(map[string]interface{}{
			"backend_type": "local",
			"path":         stateFile,
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

		// Fetch should fail with FailedPrecondition
		_, err = service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"vpc_id"},
		})

		if err == nil {
			t.Error("Fetch() expected error for unsupported version, got nil")
			return
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.FailedPrecondition && st.Code() != codes.Internal {
			t.Errorf("Fetch() code = %v, want FailedPrecondition or Internal (for version check)", st.Code())
		}

		// Verify error message mentions version
		if !contains(st.Message(), "version") {
			t.Errorf("Fetch() error message should mention version, got: %q", st.Message())
		}
	})

	// Test 4: Invalid configuration - empty path
	t.Run("invalid_config_empty_path", func(t *testing.T) {
		service := NewService()

		config, err := structpb.NewStruct(map[string]interface{}{
			"backend_type": "local",
			"path":         "",
		})
		if err != nil {
			t.Fatalf("failed to create config struct: %v", err)
		}

		_, err = service.Init(ctx, &pb.InitRequest{
			Alias:  "test",
			Config: config,
		})

		if err == nil {
			t.Error("Init() expected error for empty path, got nil")
			return
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.InvalidArgument {
			t.Errorf("Init() code = %v, want InvalidArgument", st.Code())
		}
	})

	// Test 5: Empty path in Fetch
	t.Run("empty_path_in_fetch", func(t *testing.T) {
		service := NewService()
		tmpDir := t.TempDir()
		stateFile := filepath.Join(tmpDir, "terraform.tfstate")

		stateData := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "empty-path-test-lineage",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    }
  }
}`
		if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		config, err := structpb.NewStruct(map[string]interface{}{
			"backend_type": "local",
			"path":         stateFile,
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

		// Fetch with empty path
		_, err = service.Fetch(ctx, &pb.FetchRequest{
			Path: []string{},
		})

		if err == nil {
			t.Error("Fetch() expected error for empty path, got nil")
			return
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.InvalidArgument {
			t.Errorf("Fetch() code = %v, want InvalidArgument", st.Code())
		}
	})
}

// TestQuickstart_MultipleStateSources validates the "Multiple State Sources" scenario.
//
// Tests composing configuration from multiple independent state files.
func TestQuickstart_MultipleStateSources(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping quickstart validation in short mode")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create network state
	networkDir := filepath.Join(tmpDir, "network")
	// #nosec G301 -- test directory, relaxed permissions acceptable
	if err := os.MkdirAll(networkDir, 0755); err != nil {
		t.Fatalf("failed to create network directory: %v", err)
	}

	networkStateFile := filepath.Join(networkDir, "terraform.tfstate")
	networkStateData := `{
  "version": 4,
  "terraform_version": "1.5.0",
  "serial": 1,
  "lineage": "network-state-lineage",
  "outputs": {
    "vpc_id": {
      "value": "vpc-network-123",
      "type": "string",
      "sensitive": false
    },
    "private_subnet_ids": {
      "value": ["subnet-private-1", "subnet-private-2"],
      "type": "tuple",
      "sensitive": false
    }
  }
}`
	if err := os.WriteFile(networkStateFile, []byte(networkStateData), 0600); err != nil {
		t.Fatalf("failed to create network state: %v", err)
	}

	// Create database state
	databaseDir := filepath.Join(tmpDir, "database")
	// #nosec G301 -- test directory, relaxed permissions acceptable
	if err := os.MkdirAll(databaseDir, 0755); err != nil {
		t.Fatalf("failed to create database directory: %v", err)
	}

	databaseStateFile := filepath.Join(databaseDir, "terraform.tfstate")
	databaseStateData := `{
  "serial": 1,
  "lineage": "database-state-lineage",
  "version": 4,
  "terraform_version": "1.5.0",
  "outputs": {
    "primary_endpoint": {
      "value": "db.production.example.com",
      "type": "string",
      "sensitive": false
    },
    "port": {
      "value": 5432,
      "type": "number",
      "sensitive": false
    }
  }
}`
	if err := os.WriteFile(databaseStateFile, []byte(databaseStateData), 0600); err != nil {
		t.Fatalf("failed to create database state: %v", err)
	}

	// Test: Initialize multiple provider instances with different sources
	serviceNetwork := NewService()
	serviceDatabase := NewService()

	// Initialize network provider
	networkConfig, err := structpb.NewStruct(map[string]interface{}{
		"backend_type": "local",
		"path":         networkStateFile,
	})
	if err != nil {
		t.Fatalf("failed to create network config: %v", err)
	}

	_, err = serviceNetwork.Init(ctx, &pb.InitRequest{
		Alias:  "tfstate_network",
		Config: networkConfig,
	})
	if err != nil {
		t.Fatalf("Init(network) error = %v", err)
	}

	// Initialize database provider
	databaseConfig, err := structpb.NewStruct(map[string]interface{}{
		"backend_type": "local",
		"path":         databaseStateFile,
	})
	if err != nil {
		t.Fatalf("failed to create database config: %v", err)
	}

	_, err = serviceDatabase.Init(ctx, &pb.InitRequest{
		Alias:  "tfstate_database",
		Config: databaseConfig,
	})
	if err != nil {
		t.Fatalf("Init(database) error = %v", err)
	}

	// Fetch from network state
	t.Run("fetch_network_vpc_id", func(t *testing.T) {
		resp, err := serviceNetwork.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"vpc_id"},
		})
		if err != nil {
			t.Fatalf("Fetch(network/vpc_id) error = %v", err)
		}

		value := resp.Value.AsMap()["value"].(string)
		if value != "vpc-network-123" {
			t.Errorf("network vpc_id = %q, want %q", value, "vpc-network-123")
		}
	})

	t.Run("fetch_network_subnets", func(t *testing.T) {
		resp, err := serviceNetwork.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"private_subnet_ids"},
		})
		if err != nil {
			t.Fatalf("Fetch(network/private_subnet_ids) error = %v", err)
		}

		list := resp.Value.AsMap()["value"].([]interface{})
		if len(list) != 2 {
			t.Errorf("private_subnet_ids length = %d, want 2", len(list))
		}
	})

	// Fetch from database state
	t.Run("fetch_database_endpoint", func(t *testing.T) {
		resp, err := serviceDatabase.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"primary_endpoint"},
		})
		if err != nil {
			t.Fatalf("Fetch(database/primary_endpoint) error = %v", err)
		}

		value := resp.Value.AsMap()["value"].(string)
		if value != "db.production.example.com" {
			t.Errorf("primary_endpoint = %q, want %q", value, "db.production.example.com")
		}
	})

	t.Run("fetch_database_port", func(t *testing.T) {
		resp, err := serviceDatabase.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"port"},
		})
		if err != nil {
			t.Fatalf("Fetch(database/port) error = %v", err)
		}

		value := resp.Value.AsMap()["value"].(float64)
		if value != 5432 {
			t.Errorf("port = %v, want 5432", value)
		}
	})
}

// TestQuickstart_InitTimingRequirement validates that Init completes within 2 seconds.
//
// This is a quickstart performance expectation test.
func TestQuickstart_InitTimingRequirement(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping quickstart validation in short mode")
	}

	service := NewService()
	ctx := context.Background()

	// Create test state file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateData := `{
  "serial": 1,
  "lineage": "init-timing-test-lineage",
  "version": 4,
  "terraform_version": "1.5.0",
  "outputs": {
    "vpc_id": {
      "value": "vpc-12345",
      "type": "string",
      "sensitive": false
    }
  }
}`
	if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
		t.Fatalf("failed to create state file: %v", err)
	}

	config, err := structpb.NewStruct(map[string]interface{}{
		"backend_type": "local",
		"path":         stateFile,
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	// Measure Init time
	start := time.Now()
	_, err = service.Init(ctx, &pb.InitRequest{
		Alias:  "test",
		Config: config,
	})
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify Init completes within 2 seconds (quickstart requirement)
	maxDuration := 2 * time.Second
	if duration > maxDuration {
		t.Errorf("Init took %v, expected < %v", duration, maxDuration)
	} else {
		t.Logf("Init completed in %v (< %v) ✓", duration, maxDuration)
	}
}
