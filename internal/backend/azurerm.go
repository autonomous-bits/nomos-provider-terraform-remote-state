package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

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
// Configuration validation rules:
//   - storage_account_name: 3-24 characters, lowercase alphanumeric only
//   - container_name: 3-63 characters, lowercase alphanumeric and hyphens,
//     no consecutive hyphens, cannot start or end with hyphen
//   - key: non-empty string, the blob name/path
type AzureBackendConfig struct {
	// StorageAccountName is the name of the Azure Storage account.
	// Must be 3-24 characters, lowercase alphanumeric only.
	StorageAccountName string

	// ContainerName is the name of the blob container within the storage account.
	// Must be 3-63 characters, lowercase alphanumeric and hyphens.
	// Cannot have consecutive hyphens or start/end with hyphen.
	ContainerName string

	// Key is the blob name (path) within the container.
	// This is the path to the terraform.tfstate file.
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
	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Download the blob
	data, err := b.downloader.DownloadBlob(ctx)
	if err != nil {
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
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return stateFile, nil
}
