// Package provider implements the gRPC ProviderService for the Terraform Remote State provider.
//
// This package provides the core service implementation that handles provider lifecycle
// management including initialization, health checks, and graceful shutdown. The provider
// supports multiple concurrent instances identified by alias, allowing a single server
// process to manage multiple backend connections.
//
// The Service struct implements the nomos.provider.v1.ProviderService interface and
// provides thread-safe access to provider instances through a mutex-protected map.
package provider

import (
	"context"
	"sync"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/backend"
	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// version is set via ldflags during build (-X main.version=...)
// It defaults to "dev" for development builds.
var version = "dev"

// Service implements the ProviderService gRPC interface for Terraform Remote State provider.
//
// Service manages multiple provider instances (one per alias) and provides thread-safe
// operations for initialization, fetching, health checks, and shutdown. Each instance
// maintains its own backend connection and state.
//
// The Service embeds UnimplementedProviderServiceServer to ensure forward compatibility
// when new RPC methods are added to the ProviderService interface.
type Service struct {
	pb.UnimplementedProviderServiceServer
	mu        sync.RWMutex
	instances map[string]*instance
}

// instance represents a single provider instance identified by an alias.
//
// Each instance manages its own backend connection and tracks initialization state.
// Instances are stored in the Service's instances map and accessed by alias.
type instance struct {
	alias   string
	backend backend.Backend
}

// NewService creates a new ProviderService implementation.
//
// The returned Service is ready to handle gRPC requests and manages provider instances
// in a thread-safe manner. Initially, no instances exist - they are created via Init RPC.
func NewService() *Service {
	return &Service{
		instances: make(map[string]*instance),
	}
}

// SetVersion sets the version string for the provider.
//
// This function should be called from main before starting the gRPC server to set
// the version from build metadata. If not called, the version defaults to "dev".
func SetVersion(v string) {
	version = v
}

// Info returns metadata about the provider including its type and version.
//
// This RPC can be called at any time and does not require prior initialization.
// It provides information needed by the Nomos tooling to identify and version the provider.
//
// Returns:
//   - type: "terraform-remote-state"
//   - version: Build version from ldflags or "dev" for development builds
//   - alias: Empty string (instance-specific aliases are tracked per instance)
func (s *Service) Info(_ context.Context, _ *pb.InfoRequest) (*pb.InfoResponse, error) {
	return &pb.InfoResponse{
		Type:    "terraform-remote-state",
		Version: version,
		Alias:   "", // Instance-specific, not applicable at service level
	}, nil
}

// Health checks the health status of the provider service.
//
// Returns STATUS_OK if the service is healthy and ready to accept requests.
// The service is considered healthy if it has been properly initialized and
// can manage provider instances.
//
// Note: In Phase 2, this provides basic service-level health. In Phase 3,
// this will be enhanced to check individual instance health and backend connectivity.
func (s *Service) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Basic service health: service is always healthy if running
	// In Phase 3, we'll add instance-specific health checks
	return &pb.HealthResponse{
		Status: pb.HealthResponse_STATUS_OK,
	}, nil
}

// Shutdown gracefully shuts down the provider service.
//
// This RPC performs cleanup operations including:
//   - Cleaning up all provider instances
//   - Closing all backend connections
//   - Clearing the instance map
//
// After shutdown, the service can still accept new Init requests to create
// new instances. This is a service-level operation that affects all instances.
//
// Note: In Phase 2, we don't have backend cleanup. This will be enhanced in
// Phase 3 when backends need connection cleanup.
func (s *Service) Shutdown(_ context.Context, _ *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Future: Add backend cleanup here when backends need it
	// For now, backends don't hold resources that need explicit cleanup

	// Clear all instances
	// In Phase 3, we'll iterate and close each backend connection
	s.instances = make(map[string]*instance)

	return &pb.ShutdownResponse{}, nil
}

// Init initializes a new provider instance with the given configuration.
//
// NOT IMPLEMENTED IN PHASE 2: This method will be implemented in Phase 3 (User Story 1)
// as part of the backend initialization and configuration parsing work.
//
// The UnimplementedProviderServiceServer provides a default implementation that returns
// Unimplemented status, which is appropriate for Phase 2.
//
// Phase 3 Implementation will:
//   - Parse and validate configuration
//   - Create appropriate backend (Local or Azure)
//   - Store instance in instances map
//   - Prevent duplicate initialization

// Fetch retrieves configuration data from the backend.
//
// NOT IMPLEMENTED IN PHASE 2: This method will be implemented in Phase 3 (User Story 1)
// as part of the backend integration and data retrieval work.
//
// The UnimplementedProviderServiceServer provides a default implementation that returns
// Unimplemented status, which is appropriate for Phase 2.
//
// Phase 3 Implementation will:
//   - Look up instance by alias
//   - Fetch state from backend
//   - Parse and extract outputs
//   - Return data in Nomos format
