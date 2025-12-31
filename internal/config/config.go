// Package config provides configuration parsing and validation for backend types.
//
// The config package handles parsing of backend configuration from the gRPC Init request.
// It validates common fields across all backend types and prepares configuration
// for backend-specific initialization.
package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const (
	// BackendTypeLocal identifies the local filesystem backend.
	BackendTypeLocal = "local"

	// BackendTypeAzureRM identifies the Azure Blob Storage backend.
	BackendTypeAzureRM = "azurerm"
)

var (
	// ErrNilConfig is returned when the provided configuration map is nil.
	ErrNilConfig = errors.New("configuration map is nil")

	// ErrMissingType is returned when the "type" field is missing from configuration.
	ErrMissingType = errors.New("missing required field: type")

	// ErrInvalidType is returned when the "type" field is not a string or is empty.
	ErrInvalidType = errors.New("invalid type field: must be a non-empty string")

	// ErrUnsupportedBackendType is returned when the backend type is not in the allowlist.
	ErrUnsupportedBackendType = errors.New("unsupported backend type")

	// ErrAmbiguousBackendConfig is returned when configuration contains keys
	// for multiple backend types without an explicit backend_type specification.
	// This occurs when both local backend keys (path) and Azure backend keys
	// (storage_account_name, container_name) are present in the same configuration.
	//
	// Example of ambiguous configuration:
	//   {
	//     "path": "./terraform.tfstate",                  // Local backend key
	//     "storage_account_name": "mytfstate",           // Azure backend key
	//     "container_name": "tfstate"                     // Azure backend key
	//   }
	//
	// Solution: Add explicit "backend_type" field:
	//   { "backend_type": "local", "path": "./terraform.tfstate" }
	ErrAmbiguousBackendConfig = errors.New("ambiguous backend configuration")

	// ErrCannotDetectBackend is returned when backend_type is not specified
	// and auto-detection fails due to insufficient or unrecognizable configuration keys.
	// This occurs when:
	//   - No recognizable backend keys are present (e.g., only "workspace" field)
	//   - Partial Azure configuration (only one of storage_account_name or container_name)
	//
	// Example of insufficient configuration:
	//   { "workspace": "prod" }  // No backend-specific keys
	//
	// Example of partial Azure configuration:
	//   { "storage_account_name": "mytfstate" }  // Missing container_name
	//
	// Solution: Either add explicit "backend_type" or provide complete backend keys:
	//   { "backend_type": "local", "path": "./terraform.tfstate", "workspace": "prod" }
	ErrCannotDetectBackend = errors.New("cannot auto-detect backend type")

	// ErrBackendConfigMismatch is returned when an explicit backend_type value
	// conflicts with the configuration keys present. This prevents configuration
	// errors where the declared backend type doesn't match the provided parameters.
	//
	// Example of mismatch (local backend_type with Azure keys):
	//   {
	//     "backend_type": "local",
	//     "storage_account_name": "mytfstate",  // Conflicts with "local"
	//     "container_name": "tfstate"
	//   }
	//
	// Example of mismatch (azurerm backend_type with local keys):
	//   {
	//     "backend_type": "azurerm",
	//     "path": "./terraform.tfstate"  // Conflicts with "azurerm"
	//   }
	//
	// Solution: Remove conflicting keys or change backend_type to match:
	//   { "backend_type": "local", "path": "./terraform.tfstate" }
	ErrBackendConfigMismatch = errors.New("backend type conflicts with configuration")

	// ErrPathTraversal is returned when path traversal is detected in input.
	ErrPathTraversal = errors.New("path traversal detected")

	// ErrInvalidWorkspace is returned when the workspace name contains invalid characters.
	ErrInvalidWorkspace = errors.New("invalid workspace name")

	// ErrInvalidPath is returned when a path contains invalid characters or patterns.
	ErrInvalidPath = errors.New("invalid path")

	// ErrInvalidBlobKey is returned when a blob key contains invalid characters.
	ErrInvalidBlobKey = errors.New("invalid blob key")

	// ErrInvalidStorageAccountName is returned when storage account name is invalid.
	ErrInvalidStorageAccountName = errors.New("invalid storage account name")

	// ErrInvalidContainerName is returned when container name is invalid.
	ErrInvalidContainerName = errors.New("invalid container name")
)

// allowedBackendTypes defines the allowlist of supported backend types.
// This is a security control - only explicitly allowed backend types are accepted.
var allowedBackendTypes = map[string]bool{
	"local":   true,
	"azurerm": true,
}

// Validation constants
const (
	// Maximum length for workspace names to prevent DoS attacks
	maxWorkspaceNameLength = 100

	// Maximum length for paths to prevent DoS attacks
	maxPathLength = 1024

	// Maximum length for blob keys
	maxBlobKeyLength = 1024

	// Maximum length for storage account names (Azure limit)
	maxStorageAccountNameLength = 24

	// Maximum length for container names (Azure limit)
	maxContainerNameLength = 63
)

// Path validation regex - allows alphanumeric, dots, dashes, underscores, slashes
// Does NOT allow backslashes, null bytes, or control characters
var validPathRegex = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

// Workspace name validation regex - allows alphanumeric, dashes, underscores
// Does NOT allow dots, slashes, or other special characters
var validWorkspaceRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Blob key validation regex - allows Azure blob naming conventions
// Alphanumeric, forward slashes, dots, hyphens, underscores, colons (for env: prefix)
var validBlobKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9:._/-]+$`)

// Storage account name validation regex (3-24 chars, lowercase alphanumeric)
var validStorageAccountRegex = regexp.MustCompile(`^[a-z0-9]{3,24}$`)

// Container name validation regex (3-63 chars, lowercase alphanumeric and hyphens)
var validContainerNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

// validateBackendType validates that the backend type is in the allowlist.
// This validates both explicit backend_type field values (user-provided)
// and auto-detected backend type values (from detectBackendType).
// This is a critical security control to prevent unsupported or potentially
// malicious backend types from being processed.
func validateBackendType(backendType string) error {
	if !allowedBackendTypes[backendType] {
		return fmt.Errorf("%w: %q (allowed: local, azurerm)", ErrUnsupportedBackendType, backendType)
	}
	return nil
}

// validateWorkspaceName validates that a workspace name is safe and doesn't contain
// path traversal attempts or other malicious characters.
func validateWorkspaceName(workspace string) error {
	if workspace == "" {
		return nil // Empty workspace defaults to "default"
	}

	// Check length to prevent DoS
	if len(workspace) > maxWorkspaceNameLength {
		return fmt.Errorf("%w: exceeds maximum length of %d characters", ErrInvalidWorkspace, maxWorkspaceNameLength)
	}

	// Check for path traversal patterns
	if strings.Contains(workspace, "..") {
		return fmt.Errorf("%w: path traversal not allowed", ErrPathTraversal)
	}

	// Check for directory separators
	if strings.ContainsAny(workspace, "/\\") {
		return fmt.Errorf("%w: directory separators not allowed in workspace name", ErrInvalidWorkspace)
	}

	// Validate against allowed character set
	if !validWorkspaceRegex.MatchString(workspace) {
		return fmt.Errorf("%w: must contain only alphanumeric characters, hyphens, and underscores", ErrInvalidWorkspace)
	}

	// Check for control characters and null bytes
	for _, r := range workspace {
		if unicode.IsControl(r) || r == '\x00' {
			return fmt.Errorf("%w: control characters not allowed", ErrInvalidWorkspace)
		}
	}

	return nil
}

// validatePath validates that a file path is safe and doesn't contain
// path traversal attempts or other malicious patterns.
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Check length to prevent DoS
	if len(path) > maxPathLength {
		return fmt.Errorf("%w: exceeds maximum length of %d characters", ErrInvalidPath, maxPathLength)
	}

	// Check for path traversal patterns
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: path traversal not allowed", ErrPathTraversal)
	}

	// Clean the path and verify it doesn't change significantly
	// This catches attempts to use /./ or // to obfuscate paths
	cleanPath := filepath.Clean(path)
	if cleanPath != path && cleanPath != "./"+path {
		// Allow relative paths (filepath.Clean may add ./)
		if !strings.HasPrefix(path, "./") || filepath.Clean(path) != cleanPath {
			return fmt.Errorf("%w: path normalization detected suspicious pattern", ErrInvalidPath)
		}
	}

	// Reject absolute paths that try to escape filesystem boundaries
	// This is a defense-in-depth measure
	// Absolute paths are allowed, but they must not contain traversal patterns
	// (already validated by the path traversal check above)

	// Validate against allowed character set
	if !validPathRegex.MatchString(path) {
		return fmt.Errorf("%w: contains invalid characters (allowed: a-z A-Z 0-9 . _ - /)", ErrInvalidPath)
	}

	// Check for control characters and null bytes
	for _, r := range path {
		if unicode.IsControl(r) || r == '\x00' {
			return fmt.Errorf("%w: control characters not allowed", ErrInvalidPath)
		}
	}

	// Check for backslashes (Windows path separator) - normalize to forward slash
	if strings.Contains(path, "\\") {
		return fmt.Errorf("%w: backslashes not allowed, use forward slashes", ErrInvalidPath)
	}

	return nil
}

// validateBlobKey validates that an Azure blob key is safe and follows
// Azure blob naming conventions.
func validateBlobKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrInvalidBlobKey)
	}

	// Check length to prevent DoS and comply with Azure limits
	if len(key) > maxBlobKeyLength {
		return fmt.Errorf("%w: exceeds maximum length of %d characters", ErrInvalidBlobKey, maxBlobKeyLength)
	}

	// Check for path traversal patterns
	if strings.Contains(key, "..") {
		return fmt.Errorf("%w: path traversal not allowed in blob key", ErrPathTraversal)
	}

	// Validate against allowed character set for Azure blobs
	if !validBlobKeyRegex.MatchString(key) {
		return fmt.Errorf("%w: contains invalid characters (allowed: a-z A-Z 0-9 : . _ - /)", ErrInvalidBlobKey)
	}

	// Check for control characters and null bytes
	for _, r := range key {
		if unicode.IsControl(r) || r == '\x00' {
			return fmt.Errorf("%w: control characters not allowed", ErrInvalidBlobKey)
		}
	}

	// Check for backslashes (not allowed in Azure blob names)
	if strings.Contains(key, "\\") {
		return fmt.Errorf("%w: backslashes not allowed in blob key, use forward slashes", ErrInvalidBlobKey)
	}

	// Prevent blob keys that start or end with slashes (suspicious)
	if strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return fmt.Errorf("%w: cannot start or end with forward slash", ErrInvalidBlobKey)
	}

	return nil
}

// validateStorageAccountName validates Azure storage account name format.
func validateStorageAccountName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: cannot be empty", ErrInvalidStorageAccountName)
	}

	// Check length
	if len(name) < 3 || len(name) > maxStorageAccountNameLength {
		return fmt.Errorf("%w: must be 3-24 characters", ErrInvalidStorageAccountName)
	}

	// Validate format (lowercase alphanumeric only)
	if !validStorageAccountRegex.MatchString(name) {
		return fmt.Errorf("%w: must contain only lowercase letters and numbers", ErrInvalidStorageAccountName)
	}

	return nil
}

// validateContainerName validates Azure container name format.
func validateContainerName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: cannot be empty", ErrInvalidContainerName)
	}

	// Check length
	if len(name) < 3 || len(name) > maxContainerNameLength {
		return fmt.Errorf("%w: must be 3-63 characters", ErrInvalidContainerName)
	}

	// Validate format
	if !validContainerNameRegex.MatchString(name) {
		return fmt.Errorf("%w: must start and end with lowercase letter or number, contain only lowercase letters, numbers, and hyphens", ErrInvalidContainerName)
	}

	// Check for consecutive hyphens (not allowed by Azure)
	if strings.Contains(name, "--") {
		return fmt.Errorf("%w: consecutive hyphens not allowed", ErrInvalidContainerName)
	}

	return nil
}

// sanitizeString removes potentially dangerous characters from a string.
// This is a defense-in-depth measure - validation should be the primary control.
func sanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Remove other control characters except newline and tab
	var builder strings.Builder
	for _, r := range s {
		if !unicode.IsControl(r) || r == '\n' || r == '\t' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// detectBackendType infers the backend type from configuration keys when
// backend_type is not explicitly specified. Returns the detected backend type
// or an error if detection is ambiguous or impossible.
//
// Detection Rules:
//   - If "path" key present (and NO Azure keys) → "local"
//   - If "storage_account_name" AND "container_name" present (and NO local keys) → "azurerm"
//   - If both local and Azure keys present → ErrAmbiguousBackendConfig
//   - If neither recognizable pattern → ErrCannotDetectBackend
func detectBackendType(configMap map[string]interface{}) (string, error) {
	// 1. Check for local backend indicators
	hasPath := false
	if pathValue, ok := configMap["path"]; ok {
		if pathStr, ok := pathValue.(string); ok && pathStr != "" {
			hasPath = true
		}
	}

	// 2. Check for Azure backend indicators
	hasStorageAccount := false
	if saValue, ok := configMap["storage_account_name"]; ok {
		if saStr, ok := saValue.(string); ok && saStr != "" {
			hasStorageAccount = true
		}
	}

	hasContainer := false
	if cnValue, ok := configMap["container_name"]; ok {
		if cnStr, ok := cnValue.(string); ok && cnStr != "" {
			hasContainer = true
		}
	}

	// Check if ANY Azure key is present (for ambiguity detection)
	hasAnyAzureKey := hasStorageAccount || hasContainer
	// Check if BOTH Azure keys are present (for full Azure detection)
	hasBothAzureKeys := hasStorageAccount && hasContainer

	// 3. Apply detection rules
	// If path is present with ANY Azure key, it's ambiguous
	if hasPath && hasAnyAzureKey {
		return "", fmt.Errorf("%w: both local (path) and Azure (storage_account_name, container_name) keys present. Specify 'backend_type' explicitly (local, azurerm)", ErrAmbiguousBackendConfig)
	}

	if hasPath {
		return BackendTypeLocal, nil
	}

	if hasBothAzureKeys {
		return BackendTypeAzureRM, nil
	}

	// Partial Azure config detection
	if hasAnyAzureKey {
		return "", fmt.Errorf("%w: no 'backend_type' specified and cannot infer from configuration. Azure backend requires both 'storage_account_name' and 'container_name'. Supported backends: local (requires 'path'), azurerm (requires 'storage_account_name' and 'container_name')", ErrCannotDetectBackend)
	}

	return "", fmt.Errorf("%w: no 'backend_type' specified and cannot infer from configuration. Supported backends: local (requires 'path'), azurerm (requires 'storage_account_name' and 'container_name')", ErrCannotDetectBackend)
}

// validateBackendConfigMatch validates that an explicit backend_type value
// is consistent with the configuration keys present. This prevents configuration
// errors where the declared backend type doesn't match the provided parameters.
//
// Validation Rules:
//   - If backend_type="local" but Azure keys present → error
//   - If backend_type="azurerm" but local 'path' key present → error
//   - If backend_type="azurerm" but missing required Azure keys → error
//
// This function is only called when backend_type is EXPLICITLY provided.
func validateBackendConfigMatch(backendType string, configMap map[string]interface{}) error {
	switch backendType {
	case BackendTypeLocal:
		// Check for conflicting Azure keys
		conflictingKeys := []string{}
		if _, ok := configMap["storage_account_name"]; ok {
			conflictingKeys = append(conflictingKeys, "storage_account_name")
		}
		if _, ok := configMap["container_name"]; ok {
			conflictingKeys = append(conflictingKeys, "container_name")
		}

		if len(conflictingKeys) > 0 {
			return fmt.Errorf("%w: backend_type is 'local' but Azure keys present: %v", ErrBackendConfigMismatch, conflictingKeys)
		}

		return nil

	case BackendTypeAzureRM:
		// Check for conflicting local keys
		if _, ok := configMap["path"]; ok {
			return fmt.Errorf("%w: backend_type is 'azurerm' but local 'path' key present", ErrBackendConfigMismatch)
		}

		// Validate required Azure keys are present
		if _, ok := configMap["storage_account_name"]; !ok {
			return fmt.Errorf("%w: backend_type is 'azurerm' but 'storage_account_name' missing", ErrBackendConfigMismatch)
		}

		if _, ok := configMap["container_name"]; !ok {
			return fmt.Errorf("%w: backend_type is 'azurerm' but 'container_name' missing", ErrBackendConfigMismatch)
		}

		return nil

	default:
		// Should never reach here due to validateBackendType() call before this
		return fmt.Errorf("%w: %s", ErrUnsupportedBackendType, backendType)
	}
}

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
// The function extracts and validates the optional "backend_type" field, which identifies
// the backend type (e.g., "local", "azurerm"). If backend_type is not specified, the function
// attempts to auto-detect the backend type from configuration keys using rule-based detection:
//   - Presence of "path" key (and NO Azure keys) → "local"
//   - Presence of "storage_account_name" AND "container_name" keys (and NO local keys) → "azurerm"
//
// The complete configuration map is preserved for backend-specific validation during backend construction.
//
// Security validations performed:
//   - Backend type must be in allowlist (local, azurerm) - prevents unsupported/malicious backend types
//   - Workspace name validated for path traversal and invalid characters
//   - All string inputs sanitized to remove control characters
//   - Backend-specific validations (paths, blob keys, storage account names, etc.)
//   - Explicit backend_type must match configuration keys (no conflicting keys)
//
// Example configuration for local backend (explicit backend_type):
//
//	{
//	  "backend_type": "local",
//	  "path": "terraform.tfstate",
//	  "workspace": "default"
//	}
//
// Example configuration for local backend (auto-detected):
//
//	{
//	  "path": "terraform.tfstate"
//	  // backend_type auto-detected as "local" from presence of "path" key
//	}
//
// Example configuration for Azure backend (explicit backend_type):
//
//	{
//	  "backend_type": "azurerm",
//	  "storage_account_name": "mystorageacct",
//	  "container_name": "tfstate",
//	  "key": "terraform.tfstate"
//	}
//
// Example configuration for Azure backend (auto-detected):
//
//	{
//	  "storage_account_name": "mystorageacct",
//	  "container_name": "tfstate",
//	  "key": "terraform.tfstate"
//	  // backend_type auto-detected as "azurerm" from presence of both Azure keys
//	}
//
// Returns ErrNilConfig if configMap is nil.
// Returns ErrCannotDetectBackend if backend_type is not specified and auto-detection fails.
// Returns ErrAmbiguousBackendConfig if auto-detection finds conflicting keys (both local and Azure keys present).
// Returns ErrBackendConfigMismatch if explicit backend_type conflicts with configuration keys.
// Returns ErrUnsupportedBackendType if the backend type is not in the allowlist.
// Returns ErrInvalidWorkspace if the workspace name contains invalid characters.
// Returns validation errors for backend-specific fields (paths, blob keys, storage account names, etc.).
func ParseConfig(configMap map[string]interface{}) (BackendConfig, error) {
	if configMap == nil {
		return nil, ErrNilConfig
	}

	// Step 1: Extract backend_type field (optional)
	var backendType string
	var explicitBackendType bool

	if backendTypeValue, ok := configMap["backend_type"]; ok {
		bt, ok := backendTypeValue.(string)
		if !ok {
			return nil, fmt.Errorf("invalid backend_type: must be a string, got %T", backendTypeValue)
		}

		bt = sanitizeString(bt)
		bt = strings.TrimSpace(bt)

		if bt != "" {
			backendType = bt
			explicitBackendType = true
		}
	}

	// Step 2: Auto-detect if backend_type not provided
	if backendType == "" {
		detectedType, err := detectBackendType(configMap)
		if err != nil {
			return nil, err
		}
		backendType = detectedType
		explicitBackendType = false
	}

	// Step 3: Validate backend type (CRITICAL SECURITY CHECK)
	if err := validateBackendType(backendType); err != nil {
		return nil, err
	}

	// Step 4: If explicit, validate config matches backend type
	if explicitBackendType {
		if err := validateBackendConfigMatch(backendType, configMap); err != nil {
			return nil, err
		}
	}

	// Extract workspace field (optional, defaults to "default")
	workspace := "default"
	if workspaceValue, ok := configMap["workspace"]; ok {
		if ws, ok := workspaceValue.(string); ok && ws != "" {
			// Sanitize workspace name
			ws = sanitizeString(ws)
			ws = strings.TrimSpace(ws)

			// Validate workspace name (CRITICAL SECURITY CHECK)
			if err := validateWorkspaceName(ws); err != nil {
				return nil, err
			}

			workspace = ws
		}
	}

	// Perform backend-specific validation
	if err := validateBackendSpecificConfig(backendType, configMap); err != nil {
		return nil, err
	}

	return &Config{
		backendType: backendType,
		workspace:   workspace,
		raw:         configMap,
	}, nil
}

// validateBackendSpecificConfig performs validation specific to each backend type.
func validateBackendSpecificConfig(backendType string, configMap map[string]interface{}) error {
	switch backendType {
	case BackendTypeLocal:
		return validateLocalBackendConfig(configMap)
	case BackendTypeAzureRM:
		return validateAzureBackendConfig(configMap)
	default:
		// This should never happen due to allowlist validation above
		return fmt.Errorf("%w: %s", ErrUnsupportedBackendType, backendType)
	}
}

// validateLocalBackendConfig validates local backend-specific configuration.
func validateLocalBackendConfig(configMap map[string]interface{}) error {
	// Path is required for local backend, but we validate it here for early detection
	// The backend constructor will also validate this
	if pathValue, ok := configMap["path"]; ok {
		if path, ok := pathValue.(string); ok {
			path = sanitizeString(path)
			path = strings.TrimSpace(path)

			if err := validatePath(path); err != nil {
				return fmt.Errorf("local backend path validation: %w", err)
			}
		}
	}

	return nil
}

// validateAzureBackendConfig validates Azure backend-specific configuration.
func validateAzureBackendConfig(configMap map[string]interface{}) error {
	// Validate storage_account_name
	if saValue, ok := configMap["storage_account_name"]; ok {
		if sa, ok := saValue.(string); ok {
			sa = sanitizeString(sa)
			sa = strings.TrimSpace(sa)

			if err := validateStorageAccountName(sa); err != nil {
				return err
			}
		}
	}

	// Validate container_name
	if cnValue, ok := configMap["container_name"]; ok {
		if cn, ok := cnValue.(string); ok {
			cn = sanitizeString(cn)
			cn = strings.TrimSpace(cn)

			if err := validateContainerName(cn); err != nil {
				return err
			}
		}
	}

	// Validate key (blob key)
	if keyValue, ok := configMap["key"]; ok {
		if key, ok := keyValue.(string); ok {
			key = sanitizeString(key)
			key = strings.TrimSpace(key)

			if err := validateBlobKey(key); err != nil {
				return err
			}
		}
	}

	return nil
}
