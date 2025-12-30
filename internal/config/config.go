// Package config provides configuration parsing and validation for backend types.
//
// The config package handles parsing of backend configuration from the gRPC Init request.
// It validates common fields across all backend types and prepares configuration
// for backend-specific initialization.
package config

import (
	"errors"
	"fmt"
)

var (
	// ErrNilConfig is returned when the provided configuration map is nil.
	ErrNilConfig = errors.New("configuration map is nil")

	// ErrMissingType is returned when the "type" field is missing from configuration.
	ErrMissingType = errors.New("missing required field: type")

	// ErrInvalidType is returned when the "type" field is not a string or is empty.
	ErrInvalidType = errors.New("invalid type field: must be a non-empty string")
)

// BackendConfig represents configuration for a Terraform backend.
// Implementations provide access to the backend type and raw configuration data.
type BackendConfig interface {
	// Type returns the backend type identifier (e.g., "local", "azurerm").
	Type() string

	// Raw returns the complete raw configuration map for backend-specific parsing.
	Raw() map[string]interface{}
}

// Config holds parsed backend configuration.
// It implements the BackendConfig interface and provides access to both
// the backend type and the raw configuration map for backend constructors.
type Config struct {
	backendType string
	workspace   string
	raw         map[string]interface{}
}

// Type returns the backend type identifier.
func (c *Config) Type() string {
	return c.backendType
}

// Workspace returns the workspace name, defaulting to "default" if not specified.
func (c *Config) Workspace() string {
	return c.workspace
}

// Raw returns the complete configuration map.
// Backend constructors use this to extract backend-specific fields.
func (c *Config) Raw() map[string]interface{} {
	return c.raw
}

// ParseConfig parses and validates backend configuration from the gRPC Init request.
//
// The function extracts and validates the required "type" field, which identifies
// the backend type (e.g., "local", "azurerm"). The complete configuration map is
// preserved for backend-specific validation during backend construction.
//
// Example configuration for local backend:
//
//	{
//	  "type": "local",
//	  "path": "terraform.tfstate",
//	  "workspace": "default"
//	}
//
// Example configuration for Azure backend:
//
//	{
//	  "type": "azurerm",
//	  "storage_account_name": "mystorageacct",
//	  "container_name": "tfstate",
//	  "key": "terraform.tfstate"
//	}
//
// Returns ErrNilConfig if configMap is nil.
// Returns ErrMissingType if the "type" field is not present.
// Returns ErrInvalidType if the "type" field is not a string or is empty.
func ParseConfig(configMap map[string]interface{}) (BackendConfig, error) {
	if configMap == nil {
		return nil, ErrNilConfig
	}

	// Extract the type field
	typeValue, ok := configMap["type"]
	if !ok {
		return nil, ErrMissingType
	}

	// Validate type is a string
	backendType, ok := typeValue.(string)
	if !ok {
		return nil, fmt.Errorf("%w: got %T", ErrInvalidType, typeValue)
	}

	// Validate type is non-empty
	if backendType == "" {
		return nil, ErrInvalidType
	}

	// Extract workspace field (optional, defaults to "default")
	workspace := "default"
	if workspaceValue, ok := configMap["workspace"]; ok {
		if ws, ok := workspaceValue.(string); ok && ws != "" {
			workspace = ws
		}
	}

	return &Config{
		backendType: backendType,
		workspace:   workspace,
		raw:         configMap,
	}, nil
}
