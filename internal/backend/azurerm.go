// Package backend implements Terraform remote state backends.
//
// # Azure Backend Workspace Handling
//
// The Azure backend handles workspaces differently from the local backend.
// Unlike the local backend which uses a separate "terraform.tfstate.d" directory
// structure, the Azure backend embeds workspace information directly in the
// blob key (path).
//
// The provider treats the key as an opaque string and does NOT manipulate it
// based on the workspace parameter. Users must specify the complete blob path
// including any workspace-specific path segments in their Terraform backend
// configuration.
//
// Common workspace patterns:
//
//   - Default workspace: key = "terraform.tfstate"
//   - Named workspace (env prefix): key = "env:/dev/terraform.tfstate"
//   - Named workspace (workspaces dir): key = "workspaces/dev/terraform.tfstate"
//   - Custom pattern: key = "states/production/app.tfstate"
//
// The specific pattern used depends on the user's Terraform backend configuration.
// Terraform's azurerm backend uses the "env:" prefix by default for named workspaces,
// but users may configure custom key patterns.
//
// Example Terraform configurations:
//
//	# Default workspace
//	terraform {
//	  backend "azurerm" {
//	    storage_account_name = "mystorageaccount"
//	    container_name       = "tfstate"
//	    key                  = "terraform.tfstate"
//	  }
//	}
//
//	# Named workspace with env: prefix (Terraform default)
//	# For workspace "dev", Terraform uses key "env:/dev/terraform.tfstate"
//	terraform {
//	  backend "azurerm" {
//	    storage_account_name = "mystorageaccount"
//	    container_name       = "tfstate"
//	    key                  = "terraform.tfstate"  # Base key, Terraform adds workspace prefix
//	  }
//	}
//
//	# Custom workspace pattern
//	terraform {
//	  backend "azurerm" {
//	    storage_account_name = "mystorageaccount"
//	    container_name       = "tfstate"
//	    key                  = "workspaces/${terraform.workspace}/terraform.tfstate"
//	  }
//	}
package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

func init() {
	Register("azurerm", func(ctx context.Context, config map[string]interface{}) (Backend, error) {
		// Extract storage_account_name (required)
		// Security: Validation happens in config.ParseConfig
		storageAccountValue, ok := config["storage_account_name"]
		if !ok {
			return nil, fmt.Errorf("missing required field: storage_account_name")
		}
		storageAccountName, ok := storageAccountValue.(string)
		if !ok {
			return nil, fmt.Errorf("storage_account_name must be a string")
		}

		// Extract container_name (required)
		// Security: Validation happens in config.ParseConfig
		containerValue, ok := config["container_name"]
		if !ok {
			return nil, fmt.Errorf("missing required field: container_name")
		}
		containerName, ok := containerValue.(string)
		if !ok {
			return nil, fmt.Errorf("container_name must be a string")
		}

		// Extract key (required)
		// Security: Validation happens in config.ParseConfig
		keyValue, ok := config["key"]
		if !ok {
			return nil, fmt.Errorf("missing required field: key")
		}
		key, ok := keyValue.(string)
		if !ok {
			return nil, fmt.Errorf("key must be a string")
		}

		return NewAzureBackend(ctx, AzureBackendConfig{
			StorageAccountName: storageAccountName,
			ContainerName:      containerName,
			Key:                key,
		})
	})
}

// Sentinel errors for Azure backend operations
var (
	// ErrBlobNotFound indicates the blob does not exist in the container
	ErrBlobNotFound = errors.New("blob not found")

	// ErrAuthenticationFailed indicates Azure authentication failed
	ErrAuthenticationFailed = errors.New("azure authentication failed")

	// ErrInvalidStorageAccountName indicates the storage account name is invalid
	ErrInvalidStorageAccountName = errors.New("invalid storage account name")

	// ErrInvalidContainerName indicates the container name is invalid
	ErrInvalidContainerName = errors.New("invalid container name")

	// ErrInvalidKey indicates the blob key is invalid
	ErrInvalidKey = errors.New("key cannot be empty")
)

// Storage account name validation regex (3-24 chars, lowercase alphanumeric)
var storageAccountRegex = regexp.MustCompile(`^[a-z0-9]{3,24}$`)

// Container name validation regex (3-63 chars, lowercase alphanumeric and hyphens, no consecutive hyphens)
var containerNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

// AzureBackendConfig holds configuration for the Azure Storage backend.
//
// The Azure backend reads Terraform state files from Azure Blob Storage.
// It uses DefaultAzureCredential for authentication, which supports multiple
// authentication methods including environment variables, managed identity,
// Azure CLI, and more.
//
// # Workspace Handling
//
// For the Azure backend, workspace information is embedded in the Key (blob path).
// The provider does NOT manipulate the key based on workspace parameters.
// Users must specify the complete blob path including any workspace-specific
// path segments.
//
// Examples:
//   - Default workspace: Key = "terraform.tfstate"
//   - Named workspace: Key = "env:/dev/terraform.tfstate"
//   - Workspaces directory: Key = "workspaces/production/terraform.tfstate"
//
// The specific pattern depends on how the Terraform backend is configured.
// Terraform's azurerm backend uses "env:/<workspace>/" prefix for named workspaces
// by default.
//
// Configuration validation rules:
//   - storage_account_name: 3-24 characters, lowercase alphanumeric only
//   - container_name: 3-63 characters, lowercase alphanumeric and hyphens,
//     no consecutive hyphens, cannot start or end with hyphen
//   - key: non-empty string, the blob name/path (including workspace path if applicable)
type AzureBackendConfig struct {
	// StorageAccountName is the name of the Azure Storage account.
	// Must be 3-24 characters, lowercase alphanumeric only.
	StorageAccountName string

	// ContainerName is the name of the blob container within the storage account.
	// Must be 3-63 characters, lowercase alphanumeric and hyphens.
	// Cannot have consecutive hyphens or start/end with hyphen.
	ContainerName string

	// Key is the blob name (path) within the container.
	//
	// For workspace-specific state files, this should include the complete
	// path including workspace information. The provider treats this as an
	// opaque string and does not manipulate it.
	//
	// Examples:
	//   - "terraform.tfstate" (default workspace)
	//   - "env:/dev/terraform.tfstate" (workspace "dev" with env: prefix)
	//   - "workspaces/staging/terraform.tfstate" (custom workspace pattern)
	Key string
}

// Validate validates the Azure backend configuration.
//
// Returns ErrInvalidStorageAccountName if the storage account name is invalid.
// Returns ErrInvalidContainerName if the container name is invalid.
// Returns ErrInvalidKey if the key is empty.
func (c AzureBackendConfig) Validate() error {
	// Validate storage account name
	if !storageAccountRegex.MatchString(c.StorageAccountName) {
		return fmt.Errorf("%w: must be 3-24 lowercase alphanumeric characters", ErrInvalidStorageAccountName)
	}

	// Validate container name
	if !containerNameRegex.MatchString(c.ContainerName) {
		return fmt.Errorf("%w: must be 3-63 characters, lowercase alphanumeric and hyphens, no consecutive hyphens", ErrInvalidContainerName)
	}

	// Check for consecutive hyphens in container name
	if strings.Contains(c.ContainerName, "--") {
		return fmt.Errorf("%w: cannot contain consecutive hyphens", ErrInvalidContainerName)
	}

	// Validate key is non-empty
	if c.Key == "" {
		return ErrInvalidKey
	}

	return nil
}

// blobDownloader is an interface for downloading blobs (for testing).
type blobDownloader interface {
	DownloadBlob(ctx context.Context) ([]byte, error)
}

// azureBlobClient wraps the Azure SDK blob client for downloading.
type azureBlobClient struct {
	client *azblob.Client
	config AzureBackendConfig
}

// DownloadBlob downloads a blob from Azure Storage.
func (a *azureBlobClient) DownloadBlob(ctx context.Context) ([]byte, error) {
	// Download the blob
	resp, err := a.client.DownloadStream(ctx, a.config.ContainerName, a.config.Key, nil)
	if err != nil {
		// Check if blob not found
		if strings.Contains(err.Error(), "BlobNotFound") || strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("%w: %s", ErrBlobNotFound, a.config.Key)
		}
		return nil, fmt.Errorf("failed to download blob: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Best effort close, errors ignored
	}()

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob data: %w", err)
	}

	return data, nil
}

// AzureBackend implements the Backend interface for Azure Blob Storage.
//
// AzureBackend reads Terraform state files from Azure Blob Storage using
// the Azure SDK for Go. It uses DefaultAzureCredential for authentication,
// which supports multiple authentication methods.
//
// Authentication methods supported (in order):
//   - Environment variables (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)
//   - Managed Identity (when running in Azure)
//   - Azure CLI (az login)
//   - Visual Studio Code credentials
//   - Azure PowerShell credentials
//
// The backend performs validation to ensure:
//   - Configuration is valid (storage account name, container name, key)
//   - The blob exists and is readable
//   - The state file format is valid (version 4+)
//   - Context cancellation is respected
type AzureBackend struct {
	config     AzureBackendConfig
	downloader blobDownloader
}

// NewAzureBackend creates a new Azure Storage backend.
//
// The function validates the configuration and creates an Azure blob client
// using DefaultAzureCredential for authentication.
//
// Returns ErrInvalidStorageAccountName if the storage account name is invalid.
// Returns ErrInvalidContainerName if the container name is invalid.
// Returns ErrInvalidKey if the key is empty.
// Returns ErrAuthenticationFailed if Azure authentication fails.
func NewAzureBackend(_ context.Context, cfg AzureBackendConfig) (*AzureBackend, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create Azure credential using DefaultAzureCredential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("%w: authentication failed: %w", ErrAuthenticationFailed, err)
	}

	// Construct the blob service URL
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", cfg.StorageAccountName)

	// Create the blob client
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure blob client: %w", err)
	}

	return &AzureBackend{
		config: cfg,
		downloader: &azureBlobClient{
			client: client,
			config: cfg,
		},
	}, nil
}

// FetchState retrieves the Terraform state file from Azure Blob Storage.
//
// The method downloads the blob from the configured container and key,
// then parses it as a Terraform state file.
//
// Returns ErrBlobNotFound if the blob doesn't exist.
// Returns state.ErrUnsupportedVersion if the state version is < 4.
// Returns context.Canceled if the context is cancelled before download completes.
func (b *AzureBackend) FetchState(ctx context.Context) (*state.StateFile, error) {
	slog.InfoContext(ctx, "fetching state from azure backend",
		"storage_account", b.config.StorageAccountName,
		"container", b.config.ContainerName,
		"key", b.config.Key)

	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Download the blob
	data, err := b.downloader.DownloadBlob(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to download blob",
			"storage_account", b.config.StorageAccountName,
			"container", b.config.ContainerName,
			"key", b.config.Key,
			"error", err)
		return nil, err
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
		slog.ErrorContext(ctx, "failed to parse state file",
			"storage_account", b.config.StorageAccountName,
			"container", b.config.ContainerName,
			"key", b.config.Key,
			"error", err)
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	slog.InfoContext(ctx, "successfully fetched state from azure backend",
		"storage_account", b.config.StorageAccountName,
		"container", b.config.ContainerName,
		"key", b.config.Key,
		"state_version", stateFile.Version)
	return stateFile, nil
}
