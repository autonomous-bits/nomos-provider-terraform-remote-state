// Package backend provides abstractions for retrieving Terraform state from various backend types.
//
// The backend package defines the core interface that all Terraform backend implementations
// must satisfy. This abstraction allows the provider to support multiple backend types
// (local filesystem, Azure Storage, S3, etc.) through a consistent interface.
//
// Backend implementations are responsible for:
//   - Connecting to the underlying storage system
//   - Retrieving raw state data
//   - Parsing state data into structured StateFile objects
//   - Handling backend-specific errors and converting them to standard errors
//
// The Backend interface uses context.Context for cancellation and timeout support,
// ensuring that long-running fetch operations can be properly controlled.
package backend

import (
	"context"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

// Backend represents a Terraform backend that can retrieve state files.
//
// Backend implementations handle the storage-specific logic for fetching
// Terraform state data. Each implementation (local, azurerm, s3, etc.) is
// responsible for authenticating, connecting to storage, and retrieving
// the raw state file data.
//
// All Backend implementations must:
//   - Accept context for cancellation and timeout control
//   - Return properly parsed StateFile objects
//   - Convert backend-specific errors to appropriate gRPC status codes
//   - Handle concurrent requests safely
type Backend interface {
	// FetchState retrieves the Terraform state file from the backend.
	//
	// The context parameter enables cancellation and timeout control for
	// long-running fetch operations. Implementations should check ctx.Done()
	// before and during expensive I/O operations.
	//
	// Returns the parsed StateFile on success, or an error if:
	//   - The state file cannot be found (NotFound error)
	//   - Authentication fails (PermissionDenied error)
	//   - The network is unavailable (Unavailable error)
	//   - The state file is invalid or corrupted (InvalidArgument error)
	//   - The context is cancelled (Canceled error)
	//   - An unexpected error occurs (Internal error)
	FetchState(ctx context.Context) (*state.StateFile, error)
}
