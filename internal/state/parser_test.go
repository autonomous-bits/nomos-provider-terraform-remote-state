package state

import (
	"errors"
	"testing"
)

// TestParseStateFile tests the ParseStateFile function with various inputs
// following table-driven test pattern for comprehensive coverage.
func TestParseStateFile(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		want     *StateFile
		wantErr  bool
		errCheck func(error) bool // Optional: specific error validation
	}{
		// Valid state files with various output types
		{
			name: "valid_state_v4_with_string_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 42,
				"lineage": "abc-123-def-456",
				"outputs": {
					"vpc_id": {
						"value": "vpc-12345",
						"type": "string",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           42,
				Lineage:          "abc-123-def-456",
				Outputs: map[string]*OutputValue{
					"vpc_id": {
						Value:     "vpc-12345",
						Type:      "string",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_number_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {
					"instance_count": {
						"value": 3,
						"type": "number",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs: map[string]*OutputValue{
					"instance_count": {
						Value:     float64(3), // JSON numbers unmarshal to float64
						Type:      "number",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_bool_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {
					"enable_feature": {
						"value": true,
						"type": "bool",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs: map[string]*OutputValue{
					"enable_feature": {
						Value:     true,
						Type:      "bool",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_list_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {
					"availability_zones": {
						"value": ["us-east-1a", "us-east-1b", "us-east-1c"],
						"type": "list(string)",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs: map[string]*OutputValue{
					"availability_zones": {
						Value:     []interface{}{"us-east-1a", "us-east-1b", "us-east-1c"},
						Type:      "list(string)",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_map_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {
					"tags": {
						"value": {"Environment": "production", "Owner": "platform-team"},
						"type": "map(string)",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs: map[string]*OutputValue{
					"tags": {
						Value: map[string]interface{}{
							"Environment": "production",
							"Owner":       "platform-team",
						},
						Type:      "map(string)",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_object_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {
					"network_config": {
						"value": {
							"vpc_id": "vpc-12345",
							"cidr": "10.0.0.0/16",
							"enable_dns": true
						},
						"type": "object({vpc_id=string, cidr=string, enable_dns=bool})",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs: map[string]*OutputValue{
					"network_config": {
						Value: map[string]interface{}{
							"vpc_id":     "vpc-12345",
							"cidr":       "10.0.0.0/16",
							"enable_dns": true,
						},
						Type:      "object({vpc_id=string, cidr=string, enable_dns=bool})",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_null_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {
					"optional_value": {
						"value": null,
						"type": "string",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs: map[string]*OutputValue{
					"optional_value": {
						Value:     nil,
						Type:      "string",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_sensitive_output",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {
					"db_password": {
						"value": "super-secret-password",
						"type": "string",
						"sensitive": true
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs: map[string]*OutputValue{
					"db_password": {
						Value:     "super-secret-password",
						Type:      "string",
						Sensitive: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_multiple_outputs",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 42,
				"lineage": "abc-123-def-456",
				"outputs": {
					"vpc_id": {
						"value": "vpc-12345",
						"type": "string",
						"sensitive": false
					},
					"instance_count": {
						"value": 3,
						"type": "number",
						"sensitive": false
					},
					"enable_monitoring": {
						"value": true,
						"type": "bool",
						"sensitive": false
					}
				}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           42,
				Lineage:          "abc-123-def-456",
				Outputs: map[string]*OutputValue{
					"vpc_id": {
						Value:     "vpc-12345",
						Type:      "string",
						Sensitive: false,
					},
					"instance_count": {
						Value:     float64(3),
						Type:      "number",
						Sensitive: false,
					},
					"enable_monitoring": {
						Value:     true,
						Type:      "bool",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_with_empty_outputs",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs:          map[string]*OutputValue{},
			},
			wantErr: false,
		},
		{
			name: "valid_state_v4_higher_version",
			data: []byte(`{
				"version": 5,
				"terraform_version": "2.0.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want: &StateFile{
				Version:          5,
				TerraformVersion: "2.0.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Outputs:          map[string]*OutputValue{},
			},
			wantErr: false,
		},

		// Invalid JSON cases
		{
			name:    "invalid_json_malformed",
			data:    []byte(`{"version": 4, "terraform_version": "1.5.0"`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid_json_not_json_at_all",
			data:    []byte(`this is not json`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid_json_truncated",
			data:    []byte(`{"version": 4, "terraform_version": "1.5.0", "serial": 1, "lineage": "test", "outputs": {"vpc_id": {"value": "vpc-123`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid_json_wrong_type_version_string",
			data:    []byte(`{"version": "4", "terraform_version": "1.5.0", "serial": 1, "lineage": "test", "outputs": {}}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid_json_wrong_type_serial_string",
			data:    []byte(`{"version": 4, "terraform_version": "1.5.0", "serial": "1", "lineage": "test", "outputs": {}}`),
			want:    nil,
			wantErr: true,
		},

		// Unsupported version < 4
		{
			name: "unsupported_version_3",
			data: []byte(`{
				"version": 3,
				"terraform_version": "1.0.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				// Should return a FailedPrecondition-type error
				// This will be validated once error types are defined
				return err != nil && errors.Is(err, ErrUnsupportedVersion)
			},
		},
		{
			name: "unsupported_version_2",
			data: []byte(`{
				"version": 2,
				"terraform_version": "0.12.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrUnsupportedVersion)
			},
		},
		{
			name: "unsupported_version_1",
			data: []byte(`{
				"version": 1,
				"terraform_version": "0.11.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrUnsupportedVersion)
			},
		},
		{
			name: "unsupported_version_0",
			data: []byte(`{
				"version": 0,
				"terraform_version": "0.10.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrUnsupportedVersion)
			},
		},

		// Empty data
		{
			name:    "empty_data",
			data:    []byte{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "nil_data",
			data:    nil,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "whitespace_only",
			data:    []byte("   \n\t  "),
			want:    nil,
			wantErr: true,
		},

		// Missing required fields
		{
			name: "missing_version",
			data: []byte(`{
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrMissingRequiredField)
			},
		},
		{
			name: "missing_terraform_version",
			data: []byte(`{
				"version": 4,
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrMissingRequiredField)
			},
		},
		{
			name: "missing_serial",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrMissingRequiredField)
			},
		},
		{
			name: "missing_lineage",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrMissingRequiredField)
			},
		},
		{
			name: "missing_outputs",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "test-lineage"
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrMissingRequiredField)
			},
		},

		// Edge cases with field values
		{
			name: "zero_serial",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 0,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want: &StateFile{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           0,
				Lineage:          "test-lineage",
				Outputs:          map[string]*OutputValue{},
			},
			wantErr: false,
		},
		{
			name: "empty_lineage",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": 1,
				"lineage": "",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrInvalidField)
			},
		},
		{
			name: "empty_terraform_version",
			data: []byte(`{
				"version": 4,
				"terraform_version": "",
				"serial": 1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrInvalidField)
			},
		},
		{
			name: "negative_serial",
			data: []byte(`{
				"version": 4,
				"terraform_version": "1.5.0",
				"serial": -1,
				"lineage": "test-lineage",
				"outputs": {}
			}`),
			want:    nil,
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil && errors.Is(err, ErrInvalidField)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStateFile(tt.data)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If specific error check is provided, validate it
			if tt.errCheck != nil && err != nil {
				if !tt.errCheck(err) {
					t.Errorf("ParseStateFile() error validation failed for error: %v", err)
				}
			}

			// If no error expected, compare results
			if !tt.wantErr {
				if got == nil {
					t.Errorf("ParseStateFile() returned nil, want %v", tt.want)
					return
				}

				// Compare basic fields
				if got.Version != tt.want.Version {
					t.Errorf("ParseStateFile() Version = %v, want %v", got.Version, tt.want.Version)
				}
				if got.TerraformVersion != tt.want.TerraformVersion {
					t.Errorf("ParseStateFile() TerraformVersion = %v, want %v", got.TerraformVersion, tt.want.TerraformVersion)
				}
				if got.Serial != tt.want.Serial {
					t.Errorf("ParseStateFile() Serial = %v, want %v", got.Serial, tt.want.Serial)
				}
				if got.Lineage != tt.want.Lineage {
					t.Errorf("ParseStateFile() Lineage = %v, want %v", got.Lineage, tt.want.Lineage)
				}

				// Compare outputs
				if len(got.Outputs) != len(tt.want.Outputs) {
					t.Errorf("ParseStateFile() Outputs length = %v, want %v", len(got.Outputs), len(tt.want.Outputs))
					return
				}

				for key, wantOutput := range tt.want.Outputs {
					gotOutput, exists := got.Outputs[key]
					if !exists {
						t.Errorf("ParseStateFile() missing output key: %v", key)
						continue
					}

					if !compareOutputValues(gotOutput, wantOutput) {
						t.Errorf("ParseStateFile() output[%s] = %+v, want %+v", key, gotOutput, wantOutput)
					}
				}
			}
		})
	}
}

// compareOutputValues compares two OutputValue structures for equality.
// This helper is needed because OutputValue.Value is interface{} and requires
// deep comparison for maps and slices.
func compareOutputValues(got, want *OutputValue) bool {
	t := &testing.T{} // Dummy for t.Helper() compatibility
	_ = t

	if got == nil && want == nil {
		return true
	}
	if got == nil || want == nil {
		return false
	}

	if got.Type != want.Type {
		return false
	}
	if got.Sensitive != want.Sensitive {
		return false
	}

	// Compare values based on type
	switch wantVal := want.Value.(type) {
	case nil:
		return got.Value == nil
	case string:
		gotStr, ok := got.Value.(string)
		return ok && gotStr == wantVal
	case float64:
		gotNum, ok := got.Value.(float64)
		return ok && gotNum == wantVal
	case bool:
		gotBool, ok := got.Value.(bool)
		return ok && gotBool == wantVal
	case []interface{}:
		gotSlice, ok := got.Value.([]interface{})
		if !ok || len(gotSlice) != len(wantVal) {
			return false
		}
		for i := range wantVal {
			if gotSlice[i] != wantVal[i] {
				return false
			}
		}
		return true
	case map[string]interface{}:
		gotMap, ok := got.Value.(map[string]interface{})
		if !ok || len(gotMap) != len(wantVal) {
			return false
		}
		for k, v := range wantVal {
			if gotMap[k] != v {
				return false
			}
		}
		return true
	default:
		return false
	}
}
