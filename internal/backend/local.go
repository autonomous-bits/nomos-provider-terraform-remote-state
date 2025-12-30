package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

// Sentinel errors for local backend operations
var (
	// ErrStateFileNotFound indicates the state file does not exist at the specified path
	ErrStateFileNotFound = errors.New("state file not found")

	// ErrInvalidPath indicates the path is invalid or empty
	ErrInvalidPath = errors.New("path cannot be empty")
)

// LocalBackendConfig holds configuration for the local file backend.
//
// The local backend reads Terraform state files from the local filesystem.
// It supports both default workspaces (state file at the exact path) and
// named workspaces (state file in terraform.tfstate.d/<workspace>/ subdirectory).
type LocalBackendConfig struct {
	// Path is the base path to the Terraform state file.
	// For default workspace, this is the exact path to terraform.tfstate.
	// For named workspaces, the actual path is derived by inserting
	// terraform.tfstate.d/<workspace>/ into the path.
	Path string

	// Workspace is the Terraform workspace name.
	// "default" or empty string uses the path as-is.
	// Any other value uses terraform.tfstate.d/<workspace>/terraform.tfstate pattern.
	Workspace string
}

// LocalBackend implements the Backend interface for local filesystem state files.
//
// LocalBackend reads Terraform state files from the local filesystem and
// handles workspace path resolution. It supports both default and named workspaces
// following Terraform's workspace directory structure conventions.
//
// The backend performs validation to ensure:
//   - The state file exists and is readable
//   - The state file format is valid (version 4+)
//   - Context cancellation is respected
type LocalBackend struct {
	config LocalBackendConfig
}

// NewLocalBackend creates a new local filesystem backend.
//
// Returns ErrInvalidPath if the path is empty.
// If workspace is empty, it defaults to "default".
func NewLocalBackend(cfg LocalBackendConfig) (*LocalBackend, error) {
	if cfg.Path == "" {
		return nil, ErrInvalidPath
	}

	// Default to "default" workspace if empty
	if cfg.Workspace == "" {
		cfg.Workspace = "default"
	}

	return &LocalBackend{
		config: cfg,
	}, nil
}

// FetchState retrieves the Terraform state file from the local filesystem.
//
// For default workspace, reads from the configured path directly.
// For named workspaces, reads from terraform.tfstate.d/<workspace>/terraform.tfstate.
//
// Returns ErrStateFileNotFound if the file doesn't exist.
// Returns state.ErrUnsupportedVersion if the state version is < 4.
// Returns context.Canceled if the context is cancelled before reading completes.
func (b *LocalBackend) FetchState(ctx context.Context) (*state.StateFile, error) {
	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Resolve the actual file path based on workspace
	filePath := b.resolveWorkspacePath()

	// Read the state file
	// #nosec G304 -- file path is validated from configuration
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrStateFileNotFound, filePath)
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Check context cancellation before parsing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Parse the state file
	stateFile, err := state.ParseStateFile(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return stateFile, nil
}

// resolveWorkspacePath resolves the actual file path based on the workspace.
//
// For "default" workspace: returns the path as-is.
// For named workspaces: inserts terraform.tfstate.d/<workspace>/ into the path.
//
// Examples:
//   - default workspace: "/path/to/terraform.tfstate" -> "/path/to/terraform.tfstate"
//   - production workspace: "/path/to/terraform.tfstate" -> "/path/to/terraform.tfstate.d/production/terraform.tfstate"
func (b *LocalBackend) resolveWorkspacePath() string {
	workspace := b.config.Workspace
	if workspace == "" || workspace == "default" {
		return b.config.Path
	}

	// For named workspaces, insert terraform.tfstate.d/<workspace>/ into the path
	dir := filepath.Dir(b.config.Path)
	filename := filepath.Base(b.config.Path)

	return filepath.Join(dir, "terraform.tfstate.d", workspace, filename)
}
