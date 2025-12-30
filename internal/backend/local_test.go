package backend

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

// TestLocalBackend_FetchState tests the LocalBackend.FetchState method.
func TestLocalBackend_FetchState(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a valid state file for testing
	validStateData := `{
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

	// Create invalid state data (version < 4)
	invalidVersionData := `{
		"version": 3,
		"terraform_version": "1.5.0",
		"serial": 1,
		"lineage": "abc-123-def-456",
		"outputs": {}
	}`

	// Create corrupted state data
	corruptedData := `{invalid json`

	tests := []struct {
		name      string
		setupFunc func() string
		workspace string
		wantErr   bool
		errType   error
	}{
		{
			name: "valid state file - default workspace",
			setupFunc: func() string {
				path := filepath.Join(tmpDir, "terraform.tfstate")
				if err := os.WriteFile(path, []byte(validStateData), 0600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			workspace: "default",
			wantErr:   false,
		},
		{
			name: "valid state file - named workspace",
			setupFunc: func() string {
				workspaceDir := filepath.Join(tmpDir, "workspace-test", "terraform.tfstate.d", "production")
				// #nosec G301 -- test directory, relaxed permissions acceptable
				if err := os.MkdirAll(workspaceDir, 0755); err != nil {
					t.Fatalf("failed to create workspace directory: %v", err)
				}
				path := filepath.Join(workspaceDir, "terraform.tfstate")
				if err := os.WriteFile(path, []byte(validStateData), 0600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filepath.Join(tmpDir, "workspace-test", "terraform.tfstate")
			},
			workspace: "production",
			wantErr:   false,
		},
		{
			name: "file not found",
			setupFunc: func() string {
				return filepath.Join(tmpDir, "nonexistent.tfstate")
			},
			workspace: "default",
			wantErr:   true,
			errType:   ErrStateFileNotFound,
		},
		{
			name: "invalid state version",
			setupFunc: func() string {
				path := filepath.Join(tmpDir, "invalid-version.tfstate")
				if err := os.WriteFile(path, []byte(invalidVersionData), 0600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			workspace: "default",
			wantErr:   true,
			errType:   state.ErrUnsupportedVersion,
		},
		{
			name: "corrupted state file",
			setupFunc: func() string {
				path := filepath.Join(tmpDir, "corrupted.tfstate")
				if err := os.WriteFile(path, []byte(corruptedData), 0600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			workspace: "default",
			wantErr:   true,
		},
		{
			name: "context cancellation",
			setupFunc: func() string {
				path := filepath.Join(tmpDir, "cancel.tfstate")
				if err := os.WriteFile(path, []byte(validStateData), 0600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			workspace: "default",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc()

			backend, err := NewLocalBackend(LocalBackendConfig{
				Path:      path,
				Workspace: tt.workspace,
			})
			if err != nil {
				t.Fatalf("NewLocalBackend() error = %v", err)
			}

			ctx := context.Background()
			if tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			got, err := backend.FetchState(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("FetchState() error = nil, wantErr = true")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Logf("FetchState() error = %v, want error type %v", err, tt.errType)
				}
				return
			}

			if err != nil {
				t.Errorf("FetchState() unexpected error = %v", err)
				return
			}

			if got == nil {
				t.Error("FetchState() returned nil state")
				return
			}

			if got.Version != 4 {
				t.Errorf("FetchState() version = %d, want 4", got.Version)
			}
			if got.Outputs == nil {
				t.Error("FetchState() outputs is nil")
			}
		})
	}
}

// TestNewLocalBackend tests the LocalBackend constructor.
func TestNewLocalBackend(t *testing.T) {
	tests := []struct {
		name    string
		config  LocalBackendConfig
		wantErr bool
	}{
		{
			name: "valid config with default workspace",
			config: LocalBackendConfig{
				Path:      "/path/to/terraform.tfstate",
				Workspace: "default",
			},
			wantErr: false,
		},
		{
			name: "valid config with named workspace",
			config: LocalBackendConfig{
				Path:      "/path/to/terraform.tfstate",
				Workspace: "production",
			},
			wantErr: false,
		},
		{
			name: "empty workspace defaults to default",
			config: LocalBackendConfig{
				Path:      "/path/to/terraform.tfstate",
				Workspace: "",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			config: LocalBackendConfig{
				Path:      "",
				Workspace: "default",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLocalBackend(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLocalBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewLocalBackend() returned nil backend")
			}
		})
	}
}

// TestLocalBackend_ResolveWorkspacePath tests workspace path resolution.
func TestLocalBackend_ResolveWorkspacePath(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string
		workspace string
		want      string
	}{
		{
			name:      "default workspace",
			basePath:  "/path/to/terraform.tfstate",
			workspace: "default",
			want:      "/path/to/terraform.tfstate",
		},
		{
			name:      "named workspace",
			basePath:  "/path/to/terraform.tfstate",
			workspace: "production",
			want:      "/path/to/terraform.tfstate.d/production/terraform.tfstate",
		},
		{
			name:      "empty workspace defaults to default",
			basePath:  "/path/to/terraform.tfstate",
			workspace: "",
			want:      "/path/to/terraform.tfstate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &LocalBackend{
				config: LocalBackendConfig{
					Path:      tt.basePath,
					Workspace: tt.workspace,
				},
			}

			got := backend.resolveWorkspacePath()
			if got != tt.want {
				t.Errorf("resolveWorkspacePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
