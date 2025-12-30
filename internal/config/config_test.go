package config

import (
	"errors"
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
			wantType:    "   ",
			wantErr:     nil,
			wantRawKeys: []string{"type"},
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
