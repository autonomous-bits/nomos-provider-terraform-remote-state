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

// TestLocalBackend_WorkspaceErrorMessages tests workspace-specific error messages.
func TestLocalBackend_WorkspaceErrorMessages(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name            string
		workspace       string
		wantErrContains string
	}{
		{
			name:            "default workspace not found",
			workspace:       "default",
			wantErrContains: "(workspace: default)",
		},
		{
			name:            "named workspace not found",
			workspace:       "production",
			wantErrContains: "(workspace: production)",
		},
		{
			name:            "empty workspace treated as default",
			workspace:       "",
			wantErrContains: "(workspace: default)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewLocalBackend(LocalBackendConfig{
				Path:      filepath.Join(tmpDir, "nonexistent.tfstate"),
				Workspace: tt.workspace,
			})
			if err != nil {
				t.Fatalf("NewLocalBackend() error = %v", err)
			}

			_, err = backend.FetchState(context.Background())
			if err == nil {
				t.Error("FetchState() expected error, got nil")
				return
			}

			errMsg := err.Error()
			if !errors.Is(err, ErrStateFileNotFound) {
				t.Errorf("FetchState() error not ErrStateFileNotFound: %v", err)
			}

			if tt.wantErrContains != "" && !contains(errMsg, tt.wantErrContains) {
				t.Errorf("FetchState() error = %v, want error containing %q", errMsg, tt.wantErrContains)
			}
		})
	}
}

// TestLocalBackend_WorkspacePathResolution tests comprehensive workspace path resolution scenarios.
//
// This test verifies that LocalBackend correctly resolves workspace paths following
// Terraform's workspace directory structure conventions:
//   - Default workspace: Uses the configured path as-is
//   - Named workspaces: Uses terraform.tfstate.d/<workspace>/ pattern
func TestLocalBackend_WorkspacePathResolution(t *testing.T) {
	t.Helper()

	tests := []struct {
		name           string
		workspace      string
		setupFunc      func(*testing.T) (string, string) // Returns tmpDir, basePath
		wantPathSuffix string                            // Path suffix relative to tmpDir
		wantErr        bool
		errCheck       func(error) bool
		expectedValue  string
	}{
		{
			name:      "default workspace - path used as-is",
			workspace: "default",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				validStateData := `{
					"version": 4,
					"terraform_version": "1.5.0",
					"serial": 1,
					"lineage": "test-lineage",
					"outputs": {
						"test_output": {
							"value": "test_value",
							"type": "string",
							"sensitive": false
						}
					}
				}`
				if err := os.WriteFile(basePath, []byte(validStateData), 0600); err != nil {
					t.Fatalf("failed to create default workspace state: %v", err)
				}
				return tmpDir, basePath
			},
			wantPathSuffix: "terraform.tfstate",
			wantErr:        false,
			expectedValue:  "test_value",
		},
		{
			name:      "empty workspace - defaults to default workspace",
			workspace: "",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				validStateData := `{
					"version": 4,
					"terraform_version": "1.5.0",
					"serial": 1,
					"lineage": "test-lineage",
					"outputs": {
						"test_output": {
							"value": "test_value",
							"type": "string",
							"sensitive": false
						}
					}
				}`
				if err := os.WriteFile(basePath, []byte(validStateData), 0600); err != nil {
					t.Fatalf("failed to create default workspace state: %v", err)
				}
				return tmpDir, basePath
			},
			wantPathSuffix: "terraform.tfstate",
			wantErr:        false,
			expectedValue:  "test_value",
		},
		{
			name:      "named workspace - production",
			workspace: "production",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", "production")
				// #nosec G301 -- test directory, relaxed permissions acceptable
				if err := os.MkdirAll(workspaceDir, 0755); err != nil {
					t.Fatalf("failed to create production workspace dir: %v", err)
				}
				workspacePath := filepath.Join(workspaceDir, "terraform.tfstate")
				productionState := `{
					"version": 4,
					"terraform_version": "1.5.0",
					"serial": 2,
					"lineage": "production-lineage",
					"outputs": {
						"test_output": {
							"value": "production_value",
							"type": "string",
							"sensitive": false
						}
					}
				}`
				if err := os.WriteFile(workspacePath, []byte(productionState), 0600); err != nil {
					t.Fatalf("failed to create production workspace state: %v", err)
				}
				return tmpDir, basePath
			},
			wantPathSuffix: "terraform.tfstate.d/production/terraform.tfstate",
			wantErr:        false,
			expectedValue:  "production_value",
		},
		{
			name:      "named workspace - dev",
			workspace: "dev",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", "dev")
				// #nosec G301 -- test directory, relaxed permissions acceptable
				if err := os.MkdirAll(workspaceDir, 0755); err != nil {
					t.Fatalf("failed to create dev workspace dir: %v", err)
				}
				workspacePath := filepath.Join(workspaceDir, "terraform.tfstate")
				devState := `{
					"version": 4,
					"terraform_version": "1.5.0",
					"serial": 3,
					"lineage": "dev-lineage",
					"outputs": {
						"test_output": {
							"value": "dev_value",
							"type": "string",
							"sensitive": false
						}
					}
				}`
				if err := os.WriteFile(workspacePath, []byte(devState), 0600); err != nil {
					t.Fatalf("failed to create dev workspace state: %v", err)
				}
				return tmpDir, basePath
			},
			wantPathSuffix: "terraform.tfstate.d/dev/terraform.tfstate",
			wantErr:        false,
			expectedValue:  "dev_value",
		},
		{
			name:      "non-existent workspace - staging",
			workspace: "staging",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				return tmpDir, basePath
			},
			wantPathSuffix: "terraform.tfstate.d/staging/terraform.tfstate",
			wantErr:        true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrStateFileNotFound) && contains(err.Error(), "(workspace: staging)")
			},
		},
		{
			name:      "workspace with special characters",
			workspace: "feature-branch-123",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", "feature-branch-123")
				// #nosec G301 -- test directory, relaxed permissions acceptable
				if err := os.MkdirAll(workspaceDir, 0755); err != nil {
					t.Fatalf("failed to create feature-branch-123 workspace dir: %v", err)
				}
				workspacePath := filepath.Join(workspaceDir, "terraform.tfstate")
				featureState := `{
					"version": 4,
					"terraform_version": "1.5.0",
					"serial": 4,
					"lineage": "feature-lineage",
					"outputs": {
						"test_output": {
							"value": "feature_value",
							"type": "string",
							"sensitive": false
						}
					}
				}`
				if err := os.WriteFile(workspacePath, []byte(featureState), 0600); err != nil {
					t.Fatalf("failed to create feature workspace state: %v", err)
				}
				return tmpDir, basePath
			},
			wantPathSuffix: "terraform.tfstate.d/feature-branch-123/terraform.tfstate",
			wantErr:        false,
			expectedValue:  "feature_value",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable for parallel execution
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir, basePath := tt.setupFunc(t)
			wantPath := filepath.Join(tmpDir, tt.wantPathSuffix)

			backend, err := NewLocalBackend(LocalBackendConfig{
				Path:      basePath,
				Workspace: tt.workspace,
			})
			if err != nil {
				t.Fatalf("NewLocalBackend() error = %v", err)
			}

			// Verify path resolution
			resolvedPath := backend.resolveWorkspacePath()
			if resolvedPath != wantPath {
				t.Errorf("resolveWorkspacePath() = %q, want %q", resolvedPath, wantPath)
			}

			// Verify state fetching
			stateFile, err := backend.FetchState(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("FetchState() expected error, got nil")
					return
				}
				if tt.errCheck != nil && !tt.errCheck(err) {
					t.Errorf("FetchState() error validation failed: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("FetchState() unexpected error = %v", err)
				return
			}

			if stateFile == nil {
				t.Error("FetchState() returned nil state")
				return
			}

			// Verify the correct state was fetched
			if output, ok := stateFile.Outputs["test_output"]; ok {
				if value, ok := output.Value.(string); ok {
					if value != tt.expectedValue {
						t.Errorf("FetchState() output value = %q, want %q", value, tt.expectedValue)
					}
				} else {
					t.Errorf("FetchState() output value is not a string: %T", output.Value)
				}
			} else {
				t.Error("FetchState() missing test_output in outputs")
			}
		})
	}
}

// TestLocalBackend_WorkspaceValidation tests workspace name validation and error handling.
func TestLocalBackend_WorkspaceValidation(t *testing.T) {
	t.Helper()

	tests := []struct {
		name            string
		workspace       string
		setupFunc       func(string) string // Returns basePath
		wantErr         bool
		wantErrContains string
		errType         error
	}{
		{
			name:      "valid default workspace",
			workspace: "default",
			setupFunc: func(tmpDir string) string {
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				validState := `{
					"version": 4,
					"terraform_version": "1.5.0",
					"serial": 1,
					"lineage": "test",
					"outputs": {}
				}`
				if err := os.WriteFile(basePath, []byte(validState), 0600); err != nil {
					t.Fatalf("failed to create state file: %v", err)
				}
				return basePath
			},
			wantErr: false,
		},
		{
			name:      "valid named workspace",
			workspace: "production",
			setupFunc: func(tmpDir string) string {
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", "production")
				// #nosec G301 -- test directory, relaxed permissions acceptable
				if err := os.MkdirAll(workspaceDir, 0755); err != nil {
					t.Fatalf("failed to create workspace dir: %v", err)
				}
				validState := `{
					"version": 4,
					"terraform_version": "1.5.0",
					"serial": 1,
					"lineage": "test",
					"outputs": {}
				}`
				workspacePath := filepath.Join(workspaceDir, "terraform.tfstate")
				if err := os.WriteFile(workspacePath, []byte(validState), 0600); err != nil {
					t.Fatalf("failed to create workspace state: %v", err)
				}
				return basePath
			},
			wantErr: false,
		},
		{
			name:      "non-existent default workspace",
			workspace: "default",
			setupFunc: func(tmpDir string) string {
				return filepath.Join(tmpDir, "terraform.tfstate")
			},
			wantErr:         true,
			wantErrContains: "(workspace: default)",
			errType:         ErrStateFileNotFound,
		},
		{
			name:      "non-existent named workspace",
			workspace: "nonexistent",
			setupFunc: func(tmpDir string) string {
				return filepath.Join(tmpDir, "terraform.tfstate")
			},
			wantErr:         true,
			wantErrContains: "(workspace: nonexistent)",
			errType:         ErrStateFileNotFound,
		},
		{
			name:      "workspace directory exists but state file missing",
			workspace: "incomplete",
			setupFunc: func(tmpDir string) string {
				basePath := filepath.Join(tmpDir, "terraform.tfstate")
				workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", "incomplete")
				// #nosec G301 -- test directory, relaxed permissions acceptable
				if err := os.MkdirAll(workspaceDir, 0755); err != nil {
					t.Fatalf("failed to create workspace dir: %v", err)
				}
				// Don't create state file
				return basePath
			},
			wantErr:         true,
			wantErrContains: "(workspace: incomplete)",
			errType:         ErrStateFileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Each test gets its own tmpDir for isolation
			tmpDir := t.TempDir()
			basePath := tt.setupFunc(tmpDir)

			backend, err := NewLocalBackend(LocalBackendConfig{
				Path:      basePath,
				Workspace: tt.workspace,
			})
			if err != nil {
				t.Fatalf("NewLocalBackend() error = %v", err)
			}

			_, err = backend.FetchState(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("FetchState() expected error, got nil")
					return
				}

				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("FetchState() error = %v, want error type %v", err, tt.errType)
				}

				if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("FetchState() error = %q, want error containing %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("FetchState() unexpected error = %v", err)
			}
		})
	}
}

// TestLocalBackend_ConcurrentWorkspaceAccess tests concurrent access to different workspaces.
func TestLocalBackend_ConcurrentWorkspaceAccess(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "terraform.tfstate")

	// Create state files for multiple workspaces
	workspaces := []string{"default", "dev", "staging", "production"}

	for _, workspace := range workspaces {
		var statePath string
		if workspace == "default" {
			statePath = basePath
		} else {
			workspaceDir := filepath.Join(tmpDir, "terraform.tfstate.d", workspace)
			// #nosec G301 -- test directory, relaxed permissions acceptable
			if err := os.MkdirAll(workspaceDir, 0755); err != nil {
				t.Fatalf("failed to create workspace dir: %v", err)
			}
			statePath = filepath.Join(workspaceDir, "terraform.tfstate")
		}

		stateData := `{
			"version": 4,
			"terraform_version": "1.5.0",
			"serial": 1,
			"lineage": "test",
			"outputs": {
				"workspace_name": {
					"value": "` + workspace + `",
					"type": "string",
					"sensitive": false
				}
			}
		}`

		if err := os.WriteFile(statePath, []byte(stateData), 0600); err != nil {
			t.Fatalf("failed to create state file for %s: %v", workspace, err)
		}
	}

	// Test concurrent access to different workspaces
	done := make(chan error, len(workspaces))

	for _, workspace := range workspaces {
		workspace := workspace // Capture for goroutine
		go func() {
			backend, err := NewLocalBackend(LocalBackendConfig{
				Path:      basePath,
				Workspace: workspace,
			})
			if err != nil {
				done <- err
				return
			}

			stateFile, err := backend.FetchState(context.Background())
			if err != nil {
				done <- err
				return
			}

			// Verify correct workspace state was fetched
			output, ok := stateFile.Outputs["workspace_name"]
			if !ok {
				done <- errors.New("workspace_name output not found")
				return
			}

			value, ok := output.Value.(string)
			if !ok {
				done <- errors.New("workspace_name value is not a string")
				return
			}

			expectedWorkspace := workspace
			if workspace == "" {
				expectedWorkspace = "default"
			}

			if value != expectedWorkspace {
				done <- errors.New("workspace mismatch")
				return
			}

			done <- nil
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < len(workspaces); i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent workspace access failed: %v", err)
		}
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
