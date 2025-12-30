package config

import (
	"errors"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		configMap   map[string]interface{}
		wantType    string
		wantErr     error
		wantRawKeys []string
	}{
		{
			name: "valid local backend config",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": "default",
			},
			wantType:    "local",
			wantErr:     nil,
			wantRawKeys: []string{"type", "path", "workspace"},
		},
		{
			name: "valid azurerm backend config",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantType:    "azurerm",
			wantErr:     nil,
			wantRawKeys: []string{"type", "storage_account_name", "container_name", "key"},
		},
		{
			name: "minimal valid config",
			configMap: map[string]interface{}{
				"type": "local",
			},
			wantType:    "local",
			wantErr:     nil,
			wantRawKeys: []string{"type"},
		},
		{
			name:      "nil config map",
			configMap: nil,
			wantType:  "",
			wantErr:   ErrNilConfig,
		},
		{
			name:      "empty config map",
			configMap: map[string]interface{}{},
			wantType:  "",
			wantErr:   ErrMissingType,
		},
		{
			name: "missing type field",
			configMap: map[string]interface{}{
				"path": "terraform.tfstate",
			},
			wantType: "",
			wantErr:  ErrMissingType,
		},
		{
			name: "type field is not string - integer",
			configMap: map[string]interface{}{
				"type": 123,
			},
			wantType: "",
			wantErr:  ErrInvalidType,
		},
		{
			name: "type field is not string - boolean",
			configMap: map[string]interface{}{
				"type": true,
			},
			wantType: "",
			wantErr:  ErrInvalidType,
		},
		{
			name: "type field is not string - map",
			configMap: map[string]interface{}{
				"type": map[string]string{"backend": "local"},
			},
			wantType: "",
			wantErr:  ErrInvalidType,
		},
		{
			name: "type field is empty string",
			configMap: map[string]interface{}{
				"type": "",
			},
			wantType: "",
			wantErr:  ErrInvalidType,
		},
		{
			name: "type field with whitespace only",
			configMap: map[string]interface{}{
				"type": "   ",
			},
			wantType: "",
			wantErr:  ErrUnsupportedBackendType, // After trimming, becomes empty and fails allowlist
		},
		{
			name: "config with additional fields preserved",
			configMap: map[string]interface{}{
				"type":         "azurerm",
				"custom_field": "custom_value",
				"nested": map[string]interface{}{
					"key": "value",
				},
				"array": []string{"item1", "item2"},
			},
			wantType:    "azurerm",
			wantErr:     nil,
			wantRawKeys: []string{"type", "custom_field", "nested", "array"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfig(tt.configMap)

			// Check error
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ParseConfig() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// Check no error when none expected
			if err != nil {
				t.Errorf("ParseConfig() unexpected error = %v", err)
				return
			}

			// Validate BackendConfig is not nil
			if got == nil {
				t.Error("ParseConfig() returned nil config without error")
				return
			}

			// Validate Type() method
			if got.Type() != tt.wantType {
				t.Errorf("BackendConfig.Type() = %v, want %v", got.Type(), tt.wantType)
			}

			// Validate Raw() method returns the original map
			rawMap := got.Raw()
			if rawMap == nil {
				t.Error("BackendConfig.Raw() returned nil")
				return
			}

			// Verify all expected keys are present in raw map
			for _, key := range tt.wantRawKeys {
				if _, ok := rawMap[key]; !ok {
					t.Errorf("BackendConfig.Raw() missing expected key: %s", key)
				}
			}

			// Verify raw map has correct number of keys
			if len(rawMap) != len(tt.wantRawKeys) {
				t.Errorf("BackendConfig.Raw() has %d keys, want %d", len(rawMap), len(tt.wantRawKeys))
			}
		})
	}
}

func TestConfig_Type(t *testing.T) {
	tests := []struct {
		name        string
		backendType string
	}{
		{
			name:        "local type",
			backendType: "local",
		},
		{
			name:        "azurerm type",
			backendType: "azurerm",
		},
		{
			name:        "s3 type",
			backendType: "s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				backendType: tt.backendType,
				raw:         map[string]interface{}{"type": tt.backendType},
			}

			if got := c.Type(); got != tt.backendType {
				t.Errorf("Config.Type() = %v, want %v", got, tt.backendType)
			}
		})
	}
}

func TestConfig_Raw(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]interface{}
	}{
		{
			name: "simple config",
			configMap: map[string]interface{}{
				"type": "local",
				"path": "terraform.tfstate",
			},
		},
		{
			name: "complex config with nested structures",
			configMap: map[string]interface{}{
				"type": "azurerm",
				"nested": map[string]interface{}{
					"key": "value",
				},
				"array": []string{"item1", "item2"},
			},
		},
		{
			name:      "empty config",
			configMap: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				backendType: "test",
				raw:         tt.configMap,
			}

			got := c.Raw()

			// Verify it returns the same map
			if len(got) != len(tt.configMap) {
				t.Errorf("Config.Raw() returned map with %d keys, want %d", len(got), len(tt.configMap))
			}

			// Verify all keys match
			for key := range tt.configMap {
				if _, ok := got[key]; !ok {
					t.Errorf("Config.Raw() missing key %s", key)
				}
			}
		})
	}
}

func TestConfig_Workspace(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		wantWorkspace string
	}{
		{
			name:          "default workspace",
			workspace:     "default",
			wantWorkspace: "default",
		},
		{
			name:          "production workspace",
			workspace:     "production",
			wantWorkspace: "production",
		},
		{
			name:          "dev workspace",
			workspace:     "dev",
			wantWorkspace: "dev",
		},
		{
			name:          "empty workspace",
			workspace:     "",
			wantWorkspace: "",
		},
		{
			name:          "workspace with hyphen",
			workspace:     "feature-branch",
			wantWorkspace: "feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				backendType: "local",
				workspace:   tt.workspace,
				raw: map[string]interface{}{
					"type":      "local",
					"workspace": tt.workspace,
				},
			}

			if got := c.Workspace(); got != tt.wantWorkspace {
				t.Errorf("Config.Workspace() = %v, want %v", got, tt.wantWorkspace)
			}
		})
	}
}

func TestBackendConfigInterface(t *testing.T) {
	// Verify Config implements BackendConfig interface
	var _ BackendConfig = (*Config)(nil)

	configMap := map[string]interface{}{
		"type": "local",
		"path": "terraform.tfstate",
	}

	config, err := ParseConfig(configMap)
	if err != nil {
		t.Fatalf("ParseConfig() failed: %v", err)
	}

	// Test interface methods
	if config.Type() != "local" {
		t.Errorf("BackendConfig.Type() = %v, want local", config.Type())
	}

	raw := config.Raw()
	if raw == nil {
		t.Error("BackendConfig.Raw() returned nil")
	}
}

func TestParseConfig_ErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		configMap     map[string]interface{}
		wantErrString string
	}{
		{
			name:          "nil config",
			configMap:     nil,
			wantErrString: "configuration map is nil",
		},
		{
			name:          "missing type",
			configMap:     map[string]interface{}{},
			wantErrString: "missing required field: type",
		},
		{
			name: "invalid type - integer",
			configMap: map[string]interface{}{
				"type": 123,
			},
			wantErrString: "invalid type field",
		},
		{
			name: "empty type string",
			configMap: map[string]interface{}{
				"type": "",
			},
			wantErrString: "invalid type field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig(tt.configMap)
			if err == nil {
				t.Error("ParseConfig() expected error, got nil")
				return
			}

			if err.Error() == "" {
				t.Error("ParseConfig() returned error with empty message")
			}
		})
	}
}

func BenchmarkParseConfig(b *testing.B) {
	configMap := map[string]interface{}{
		"type":                 "azurerm",
		"storage_account_name": "mystorageacct",
		"container_name":       "tfstate",
		"key":                  "terraform.tfstate",
		"resource_group_name":  "myresourcegroup",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseConfig(configMap)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseConfig_Error(b *testing.B) {
	configMap := map[string]interface{}{
		"type": 123,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseConfig(configMap)
	}
}

// TestParseConfig_SecurityValidation tests security-critical input validation.
func TestParseConfig_SecurityValidation(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]interface{}
		wantErr   error
	}{
		// Backend type validation - allowlist enforcement
		{
			name: "unsupported backend type - s3",
			configMap: map[string]interface{}{
				"type": "s3",
			},
			wantErr: ErrUnsupportedBackendType,
		},
		{
			name: "unsupported backend type - gcs",
			configMap: map[string]interface{}{
				"type": "gcs",
			},
			wantErr: ErrUnsupportedBackendType,
		},
		{
			name: "unsupported backend type - consul",
			configMap: map[string]interface{}{
				"type": "consul",
			},
			wantErr: ErrUnsupportedBackendType,
		},
		{
			name: "backend type with control characters",
			configMap: map[string]interface{}{
				"type": "local\\x00",
			},
			wantErr: ErrUnsupportedBackendType, // After sanitization, "local\\x00" won't match allowlist
		},

		// Workspace validation - path traversal prevention
		{
			name: "workspace with path traversal - dot dot",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": "../etc/passwd",
			},
			wantErr: ErrPathTraversal,
		},
		{
			name: "workspace with path traversal - encoded",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": "..%2F..%2Fetc%2Fpasswd",
			},
			wantErr: ErrPathTraversal,
		},
		{
			name: "workspace with forward slash",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": "prod/staging",
			},
			wantErr: ErrInvalidWorkspace,
		},
		{
			name: "workspace with backslash",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": "prod\\\\staging",
			},
			wantErr: ErrInvalidWorkspace,
		},
		{
			name: "workspace with null byte",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": "prod\\x00",
			},
			wantErr: ErrInvalidWorkspace,
		},
		{
			name: "workspace exceeding max length",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": strings.Repeat("a", 101), // maxWorkspaceNameLength is 100
			},
			wantErr: ErrInvalidWorkspace,
		},

		// Local backend path validation
		{
			name: "local path with path traversal",
			configMap: map[string]interface{}{
				"type": "local",
				"path": "../../etc/passwd",
			},
			wantErr: ErrPathTraversal,
		},
		{
			name: "local path with encoded path traversal",
			configMap: map[string]interface{}{
				"type": "local",
				"path": "..%2F..%2Fetc%2Fpasswd",
			},
			wantErr: ErrPathTraversal,
		},
		{
			name: "local path with backslashes",
			configMap: map[string]interface{}{
				"type": "local",
				"path": "C:\\\\Windows\\\\System32\\\\config",
			},
			wantErr: ErrInvalidPath,
		},
		{
			name: "local path with null byte",
			configMap: map[string]interface{}{
				"type": "local",
				"path": "terraform\\x00.tfstate",
			},
			wantErr: ErrInvalidPath,
		},
		{
			name: "local path exceeding max length",
			configMap: map[string]interface{}{
				"type": "local",
				"path": strings.Repeat("a", 1025), // maxPathLength is 1024
			},
			wantErr: ErrInvalidPath,
		},

		// Azure backend validation
		{
			name: "azure storage account with uppercase",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "MyStorageAccount",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantErr: ErrInvalidStorageAccountName,
		},
		{
			name: "azure storage account too short",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "ab",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantErr: ErrInvalidStorageAccountName,
		},
		{
			name: "azure storage account too long",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "abcdefghijklmnopqrstuvwxyz",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantErr: ErrInvalidStorageAccountName,
		},
		{
			name: "azure storage account with special chars",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "my-storage-account",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantErr: ErrInvalidStorageAccountName,
		},
		{
			name: "azure container name with uppercase",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "TfState",
				"key":                  "terraform.tfstate",
			},
			wantErr: ErrInvalidContainerName,
		},
		{
			name: "azure container name too short",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "ab",
				"key":                  "terraform.tfstate",
			},
			wantErr: ErrInvalidContainerName,
		},
		{
			name: "azure container name with consecutive hyphens",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tf--state",
				"key":                  "terraform.tfstate",
			},
			wantErr: ErrInvalidContainerName,
		},
		{
			name: "azure blob key with path traversal",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  "../../secrets.tfstate",
			},
			wantErr: ErrPathTraversal,
		},
		{
			name: "azure blob key with backslash",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  "dir\\\\terraform.tfstate",
			},
			wantErr: ErrInvalidBlobKey,
		},
		{
			name: "azure blob key starting with slash",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  "/terraform.tfstate",
			},
			wantErr: ErrInvalidBlobKey,
		},
		{
			name: "azure blob key ending with slash",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate/",
			},
			wantErr: ErrInvalidBlobKey,
		},
		{
			name: "azure blob key with null byte",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  "terraform\\x00.tfstate",
			},
			wantErr: ErrInvalidBlobKey,
		},
		{
			name: "azure blob key exceeding max length",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  strings.Repeat("a", 1025), // maxBlobKeyLength is 1024
			},
			wantErr: ErrInvalidBlobKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig(tt.configMap)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ParseConfig() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestParseConfig_ValidInputsSanitized tests that valid inputs pass validation after sanitization.
func TestParseConfig_ValidInputsSanitized(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]interface{}
		wantType  string
	}{
		{
			name: "backend type with whitespace trimmed",
			configMap: map[string]interface{}{
				"type": "  local  ",
			},
			wantType: "local",
		},
		{
			name: "workspace with whitespace trimmed",
			configMap: map[string]interface{}{
				"type":      "local",
				"path":      "terraform.tfstate",
				"workspace": "  production  ",
			},
			wantType: "local",
		},
		{
			name: "valid local backend",
			configMap: map[string]interface{}{
				"type": "local",
				"path": "terraform.tfstate",
			},
			wantType: "local",
		},
		{
			name: "valid azure backend",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct123",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			wantType: "azurerm",
		},
		{
			name: "valid azure backend with workspace path",
			configMap: map[string]interface{}{
				"type":                 "azurerm",
				"storage_account_name": "mystorageacct",
				"container_name":       "tfstate",
				"key":                  "env:/production/terraform.tfstate",
			},
			wantType: "azurerm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseConfig(tt.configMap)
			if err != nil {
				t.Errorf("ParseConfig() unexpected error = %v", err)
				return
			}

			if config.Type() != tt.wantType {
				t.Errorf("Config.Type() = %v, want %v", config.Type(), tt.wantType)
			}
		})
	}
}

// TestValidatePath tests path validation in isolation.
func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid relative path", "terraform.tfstate", false},
		{"valid absolute path", "/var/terraform/terraform.tfstate", false},
		{"valid nested path", "infra/terraform.tfstate", false},
		{"valid path with dots", "my.terraform.tfstate", false},
		{"path traversal with ..", "../../../etc/passwd", true},
		{"path traversal in middle", "/var/../../../etc/passwd", true},
		{"backslashes", "C:\\\\Windows\\\\System32", true},
		{"null byte", "terraform\\x00.tfstate", true},
		{"empty path", "", true},
		{"path too long", strings.Repeat("a", 1025), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateWorkspaceName tests workspace name validation in isolation.
func TestValidateWorkspaceName(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		wantErr   bool
	}{
		{"valid workspace - default", "default", false},
		{"valid workspace - production", "production", false},
		{"valid workspace with hyphen", "prod-env", false},
		{"valid workspace with underscore", "prod_env", false},
		{"empty workspace", "", false}, // Defaults to "default"
		{"workspace with slash", "prod/staging", true},
		{"workspace with backslash", "prod\\\\staging", true},
		{"workspace with dots", "prod..staging", true},
		{"workspace too long", strings.Repeat("a", 101), true},
		{"workspace with null byte", "prod\\x00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkspaceName(tt.workspace)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWorkspaceName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateBlobKey tests Azure blob key validation in isolation.
func TestValidateBlobKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid simple key", "terraform.tfstate", false},
		{"valid nested key", "env/production/terraform.tfstate", false},
		{"valid env: prefix", "env:/production/terraform.tfstate", false},
		{"key with path traversal", "../secrets.tfstate", true},
		{"key with backslash", "dir\\\\terraform.tfstate", true},
		{"key starting with slash", "/terraform.tfstate", true},
		{"key ending with slash", "terraform.tfstate/", true},
		{"empty key", "", true},
		{"key too long", strings.Repeat("a", 1025), true},
		{"key with null byte", "terraform\\x00.tfstate", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBlobKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBlobKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateStorageAccountName tests Azure storage account name validation.
func TestValidateStorageAccountName(t *testing.T) {
	tests := []struct {
		name    string
		account string
		wantErr bool
	}{
		{"valid account name", "mystorageacct", false},
		{"valid account with numbers", "mystorageacct123", false},
		{"minimum length", "abc", false},
		{"maximum length", "abcdefghijklmnopqrstuvwx", false},
		{"too short", "ab", true},
		{"too long", "abcdefghijklmnopqrstuvwxyz", true},
		{"uppercase letters", "MyStorageAccount", true},
		{"special characters", "my-storage-account", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStorageAccountName(tt.account)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStorageAccountName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateContainerName tests Azure container name validation.
func TestValidateContainerName(t *testing.T) {
	tests := []struct {
		name      string
		container string
		wantErr   bool
	}{
		{"valid container name", "tfstate", false},
		{"valid with hyphen", "tf-state", false},
		{"minimum length", "abc", false},
		{"maximum length", "a" + strings.Repeat("-a", 30) + "b", false},
		{"too short", "ab", true},
		{"uppercase letters", "TfState", true},
		{"consecutive hyphens", "tf--state", true},
		{"starts with hyphen", "-tfstate", true},
		{"ends with hyphen", "tfstate-", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContainerName(tt.container)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContainerName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
