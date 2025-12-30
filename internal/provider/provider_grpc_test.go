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

// TestInit_LocalBackend tests Init RPC with local backend configuration.
func TestInit_LocalBackend(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Create a temporary state file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateData := `{
		"version": 4,
		"terraform_version": "1.5.0",
		"serial": 1,
		"lineage": "abc-123-def-456",
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

	// Create config struct
	config, err := structpb.NewStruct(map[string]interface{}{
		"type":      "local",
		"path":      stateFile,
		"workspace": "default",
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	req := &pb.InitRequest{
		Alias:  "local-test",
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
	inst, exists := service.instances["local-test"]
	if !exists {
		t.Fatal("instance was not created")
	}

	if inst.alias != "local-test" {
		t.Errorf("instance alias = %s, want local-test", inst.alias)
	}

	if inst.backend == nil {
		t.Error("backend is nil")
	}
}

// TestInit_AzureBackend tests Init RPC with Azure backend configuration.
func TestInit_AzureBackend(t *testing.T) {
	t.Skip("Skipping Azure backend test - requires authentication")

	service := NewService()
	ctx := context.Background()

	config, err := structpb.NewStruct(map[string]interface{}{
		"type":                 "azurerm",
		"storage_account_name": "testaccount",
		"container_name":       "tfstate",
		"key":                  "terraform.tfstate",
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	req := &pb.InitRequest{
		Alias:  "azure-test",
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
	inst, exists := service.instances["azure-test"]
	if !exists {
		t.Fatal("instance was not created")
	}

	if inst.backend == nil {
		t.Error("backend is nil")
	}
}

// TestInit_InvalidConfig tests Init RPC with invalid configurations.
func TestInit_InvalidConfig(t *testing.T) {
	tests := []struct {
		name       string
		alias      string
		config     map[string]interface{}
		wantCode   codes.Code
		wantSubstr string
	}{
		{
			name:     "nil config",
			alias:    "test",
			config:   nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "missing type",
			alias: "test",
			config: map[string]interface{}{
				"path": "/some/path",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "unsupported backend type",
			alias: "test",
			config: map[string]interface{}{
				"type": "s3",
			},
			wantCode:   codes.InvalidArgument,
			wantSubstr: "unsupported",
		},
		{
			name:  "empty alias",
			alias: "",
			config: map[string]interface{}{
				"type": "local",
				"path": "/some/path",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "invalid local config - empty path",
			alias: "test",
			config: map[string]interface{}{
				"type": "local",
				"path": "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "invalid azure config - invalid storage account",
			alias: "test",
			config: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "Invalid",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService()
			ctx := context.Background()

			var config *structpb.Struct
			if tt.config != nil {
				var err error
				config, err = structpb.NewStruct(tt.config)
				if err != nil {
					t.Fatalf("failed to create config struct: %v", err)
				}
			}

			req := &pb.InitRequest{
				Alias:  tt.alias,
				Config: config,
			}

			resp, err := service.Init(ctx, req)

			if err == nil {
				t.Errorf("Init() expected error, got nil")
				return
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Errorf("expected gRPC status error, got %v", err)
				return
			}

			if st.Code() != tt.wantCode {
				t.Errorf("Init() code = %v, want %v", st.Code(), tt.wantCode)
			}

			if tt.wantSubstr != "" && !contains(st.Message(), tt.wantSubstr) {
				t.Errorf("Init() message = %q, want substring %q", st.Message(), tt.wantSubstr)
			}

			if resp != nil {
				t.Errorf("Init() expected nil response, got %v", resp)
			}
		})
	}
}

// TestInit_DuplicateAlias tests that initializing twice fails.
func TestInit_DuplicateAlias(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateData := `{
		"version": 4,
		"terraform_version": "1.5.0",
		"serial": 1,
		"lineage": "abc-123-def-456",
		"outputs": {}
	}`
	if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
		t.Fatalf("failed to create state file: %v", err)
	}

	config, err := structpb.NewStruct(map[string]interface{}{
		"type": "local",
		"path": stateFile,
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	req := &pb.InitRequest{
		Alias:  "duplicate",
		Config: config,
	}

	// First Init should succeed
	_, err = service.Init(ctx, req)
	if err != nil {
		t.Fatalf("first Init() error = %v", err)
	}

	// Second Init should fail with FailedPrecondition
	_, err = service.Init(ctx, req)
	if err == nil {
		t.Errorf("second Init() expected error, got nil")
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("expected gRPC status error, got %v", err)
		return
	}

	if st.Code() != codes.FailedPrecondition {
		t.Errorf("Init() code = %v, want %v", st.Code(), codes.FailedPrecondition)
	}
}

// TestFetch tests Fetch RPC with local backend.
func TestFetch(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Create a temporary state file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateData := `{
		"version": 4,
		"terraform_version": "1.5.0",
		"serial": 1,
		"lineage": "abc-123-def-456",
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
			},
			"instance_count": {
				"value": 3,
				"type": "number",
				"sensitive": false
			}
		}
	}`
	if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
		t.Fatalf("failed to create state file: %v", err)
	}

	// Initialize provider
	config, err := structpb.NewStruct(map[string]interface{}{
		"type": "local",
		"path": stateFile,
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

	tests := []struct {
		name      string
		alias     string
		path      []string
		wantValue interface{}
		wantCode  codes.Code
	}{
		{
			name:      "fetch string output",
			alias:     "test",
			path:      []string{"vpc_id"},
			wantValue: "vpc-12345",
		},
		{
			name:      "fetch number output",
			alias:     "test",
			path:      []string{"instance_count"},
			wantValue: float64(3),
		},
		{
			name:     "output not found",
			alias:    "test",
			path:     []string{"nonexistent"},
			wantCode: codes.NotFound,
		},
		{
			name:     "empty path",
			alias:    "test",
			path:     []string{},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.FetchRequest{
				Path: tt.path,
			}

			resp, err := service.Fetch(ctx, req)

			if tt.wantCode != codes.OK {
				if err == nil {
					t.Errorf("Fetch() expected error, got nil")
					return
				}

				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("expected gRPC status error, got %v", err)
					return
				}

				if st.Code() != tt.wantCode {
					t.Errorf("Fetch() code = %v, want %v", st.Code(), tt.wantCode)
				}

				return
			}

			if err != nil {
				t.Errorf("Fetch() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Fetch() returned nil response")
			}

			if resp.Value == nil {
				t.Fatal("Fetch() returned nil value")
			}

			// Extract the value field
			fields := resp.Value.AsMap()
			if fields == nil {
				t.Fatal("Fetch() value AsMap returned nil")
			}

			value, ok := fields["value"]
			if !ok {
				t.Error("Fetch() response missing 'value' field")
				return
			}

			if value != tt.wantValue {
				t.Errorf("Fetch() value = %v (%T), want %v (%T)", value, value, tt.wantValue, tt.wantValue)
			}
		})
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
