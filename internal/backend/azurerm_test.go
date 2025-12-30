package backend

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

// mockBlobDownloader simulates Azure blob storage for testing.
type mockBlobDownloader struct {
	data      string
	shouldErr bool
	errType   error
}

func (m *mockBlobDownloader) DownloadBlob(_ context.Context) ([]byte, error) {
	if m.shouldErr {
		if m.errType != nil {
			return nil, m.errType
		}
		return nil, errors.New("download failed")
	}
	return []byte(m.data), nil
}

// TestAzureBackend_FetchState tests the AzureBackend.FetchState method.
func TestAzureBackend_FetchState(t *testing.T) {
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

	invalidVersionData := `{
		"version": 3,
		"terraform_version": "1.5.0",
		"serial": 1,
		"lineage": "abc-123-def-456",
		"outputs": {}
	}`

	corruptedData := `{invalid json`

	tests := []struct {
		name    string
		mock    *mockBlobDownloader
		wantErr bool
		errType error
	}{
		{
			name: "valid state file",
			mock: &mockBlobDownloader{
				data:      validStateData,
				shouldErr: false,
			},
			wantErr: false,
		},
		{
			name: "blob not found",
			mock: &mockBlobDownloader{
				shouldErr: true,
				errType:   ErrBlobNotFound,
			},
			wantErr: true,
			errType: ErrBlobNotFound,
		},
		{
			name: "invalid state version",
			mock: &mockBlobDownloader{
				data:      invalidVersionData,
				shouldErr: false,
			},
			wantErr: true,
			errType: state.ErrUnsupportedVersion,
		},
		{
			name: "corrupted state file",
			mock: &mockBlobDownloader{
				data:      corruptedData,
				shouldErr: false,
			},
			wantErr: true,
		},
		{
			name: "download error",
			mock: &mockBlobDownloader{
				shouldErr: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &AzureBackend{
				config: AzureBackendConfig{
					StorageAccountName: "testaccount",
					ContainerName:      "tfstate",
					Key:                "terraform.tfstate",
				},
				downloader: tt.mock,
			}

			ctx := context.Background()
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

// TestNewAzureBackend tests the AzureBackend constructor.
func TestNewAzureBackend(t *testing.T) {
	tests := []struct {
		name    string
		config  AzureBackendConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			wantErr: false,
		},
		{
			name: "empty storage account name",
			config: AzureBackendConfig{
				StorageAccountName: "",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid storage account name - too short",
			config: AzureBackendConfig{
				StorageAccountName: "ab",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid storage account name - too long",
			config: AzureBackendConfig{
				StorageAccountName: "abcdefghijklmnopqrstuvwxy",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid storage account name - uppercase",
			config: AzureBackendConfig{
				StorageAccountName: "MyAccount",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid storage account name - special chars",
			config: AzureBackendConfig{
				StorageAccountName: "my-account",
				ContainerName:      "tfstate",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "empty container name",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount",
				ContainerName:      "",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid container name - too short",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount",
				ContainerName:      "ab",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid container name - too long",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount",
				ContainerName:      strings.Repeat("a", 64),
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid container name - uppercase",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount",
				ContainerName:      "MyContainer",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "invalid container name - consecutive hyphens",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount",
				ContainerName:      "my--container",
				Key:                "terraform.tfstate",
			},
			wantErr: true,
		},
		{
			name: "empty key",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount",
				ContainerName:      "tfstate",
				Key:                "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAzureBackend(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAzureBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewAzureBackend() returned nil backend")
			}
		})
	}
}

// TestAzureBackendConfig_Validate tests the config validation.
func TestAzureBackendConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AzureBackendConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AzureBackendConfig{
				StorageAccountName: "validaccount123",
				ContainerName:      "tfstate-container",
				Key:                "production/terraform.tfstate",
			},
			wantErr: false,
		},
		{
			name: "valid config with minimum lengths",
			config: AzureBackendConfig{
				StorageAccountName: "abc",
				ContainerName:      "abc",
				Key:                "x",
			},
			wantErr: false,
		},
		{
			name: "valid config with maximum lengths",
			config: AzureBackendConfig{
				StorageAccountName: "abcdefghijklmnopqrstuvw",
				ContainerName:      strings.Repeat("a", 63),
				Key:                strings.Repeat("x", 1024),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
