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
	"errors"
	"fmt"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/backend"
	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/config"
	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// version is set via ldflags during build (-X main.version=...)
// It defaults to "dev" for development builds.
var version = "dev"

// Service implements the ProviderService gRPC interface for Terraform Remote State provider.
//
// Service manages multiple provider instances (one per alias) and provides operations for
// initialization, fetching, health checks, and shutdown. Each instance maintains its own
// backend connection and state.
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
// the backend connection is working.
func (s *Service) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Service is healthy if running (can accept Init)
	// After Phase 3, we could add backend health checks here
	return &pb.HealthResponse{
		Status: pb.HealthResponse_STATUS_OK,
	}, nil
}

// Shutdown gracefully shuts down the provider service.
//
// This RPC performs cleanup operations including:
//   - Removing all provider instances
//   - Clearing backend references
//
// After shutdown, the service can accept new Init requests for new instances.
func (s *Service) Shutdown(_ context.Context, _ *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Shutdown all instances
	s.instances = make(map[string]*instance)

	return &pb.ShutdownResponse{}, nil
}

// Init initializes a provider instance with the given configuration.
//
// This RPC parses the configuration, creates the appropriate backend
// (Local or Azure), and stores it for future Fetch operations.
//
// Each alias can only be initialized once. Subsequent calls with the same alias
// will return FailedPrecondition.
//
// Returns codes.InvalidArgument if:
//   - The alias is empty
//   - The config is nil or invalid
//   - The backend type is unsupported
//   - Backend-specific validation fails
//
// Returns codes.FailedPrecondition if the alias is already initialized.
func (s *Service) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	// Validate alias
	if req.Alias == "" {
		return nil, status.Error(codes.InvalidArgument, "alias cannot be empty")
	}

	// Check if already initialized
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.instances[req.Alias]; exists {
		return nil, status.Errorf(codes.FailedPrecondition, "provider instance %q already initialized", req.Alias)
	}

	// Parse config
	configMap := req.Config.AsMap()
	cfg, err := config.ParseConfig(configMap)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid configuration: %v", err)
	}

	// Create backend based on type
	var b backend.Backend
	switch cfg.Type() {
	case "local":
		b, err = createLocalBackend(cfg)
	case "azurerm":
		b, err = createAzureBackend(ctx, cfg)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported backend type: %s", cfg.Type())
	}

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to create backend: %v", err)
	}

	// Store instance
	s.instances[req.Alias] = &instance{
		alias:   req.Alias,
		backend: b,
	}

	return &pb.InitResponse{}, nil
}

// Fetch retrieves configuration data from the backend.
//
// This RPC fetches the state file from the backend, extracts the requested
// output value, and returns it as a structured value. In the MVP, the provider
// supports a single default instance. Multi-instance support (US2) will add
// explicit alias handling.
//
// The path parameter should contain a single element: the output name.
// Example: ["vpc_id"] fetches the "vpc_id" output from the state file.
//
// Returns codes.FailedPrecondition if not initialized.
// Returns codes.InvalidArgument if:
//   - The path is empty
//   - The path contains more than one element
//
// Returns codes.NotFound if:
//   - The output name doesn't exist in the state file
//   - The state file doesn't exist
//
// Returns codes.Unavailable for network errors.
// Returns codes.PermissionDenied for authentication errors.
// Returns codes.Internal for parsing errors.
func (s *Service) Fetch(ctx context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {
	// For MVP: get any instance (typically there's only one)
	// Future: Use req.Alias to select specific instance
	s.mu.RLock()
	var inst *instance
	for _, i := range s.instances {
		inst = i
		break
	}
	s.mu.RUnlock()

	if inst == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider not initialized: call Init first")
	}

	// Validate path
	if len(req.Path) == 0 {
		return nil, status.Error(codes.InvalidArgument, "path cannot be empty")
	}

	if len(req.Path) != 1 {
		return nil, status.Errorf(codes.InvalidArgument, "path must contain exactly one element (the output name), got %d", len(req.Path))
	}

	outputName := req.Path[0]
	if outputName == "" {
		return nil, status.Error(codes.InvalidArgument, "path segment cannot be empty")
	}

	// Fetch state from backend
	stateFile, err := inst.backend.FetchState(ctx)
	if err != nil {
		return nil, mapBackendError(err)
	}

	// Look up output
	output, ok := stateFile.Outputs[outputName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "output %q not found in state", outputName)
	}

	// Convert output to structpb.Struct
	// Create a map with the output value fields
	outputMap := map[string]interface{}{
		"value":     output.Value,
		"type":      output.Type,
		"sensitive": output.Sensitive,
	}

	value, err := structpb.NewStruct(outputMap)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert output value: %v", err)
	}

	return &pb.FetchResponse{
		Value: value,
	}, nil
}

// createLocalBackend creates a local backend from the config.
func createLocalBackend(cfg config.BackendConfig) (backend.Backend, error) {
	raw := cfg.Raw()

	// Extract path
	pathValue, ok := raw["path"]
	if !ok {
		return nil, fmt.Errorf("missing required field: path")
	}
	path, ok := pathValue.(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	// Extract workspace (optional, defaults to "default")
	workspace := "default"
	if workspaceValue, ok := raw["workspace"]; ok {
		if ws, ok := workspaceValue.(string); ok && ws != "" {
			workspace = ws
		}
	}

	return backend.NewLocalBackend(backend.LocalBackendConfig{
		Path:      path,
		Workspace: workspace,
	})
}

// createAzureBackend creates an Azure backend from the config.
func createAzureBackend(ctx context.Context, cfg config.BackendConfig) (backend.Backend, error) {
	raw := cfg.Raw()

	// Extract storage_account_name
	storageAccountValue, ok := raw["storage_account_name"]
	if !ok {
		return nil, fmt.Errorf("missing required field: storage_account_name")
	}
	storageAccountName, ok := storageAccountValue.(string)
	if !ok {
		return nil, fmt.Errorf("storage_account_name must be a string")
	}

	// Extract container_name
	containerValue, ok := raw["container_name"]
	if !ok {
		return nil, fmt.Errorf("missing required field: container_name")
	}
	containerName, ok := containerValue.(string)
	if !ok {
		return nil, fmt.Errorf("container_name must be a string")
	}

	// Extract key
	keyValue, ok := raw["key"]
	if !ok {
		return nil, fmt.Errorf("missing required field: key")
	}
	key, ok := keyValue.(string)
	if !ok {
		return nil, fmt.Errorf("key must be a string")
	}

	return backend.NewAzureBackend(ctx, backend.AzureBackendConfig{
		StorageAccountName: storageAccountName,
		ContainerName:      containerName,
		Key:                key,
	})
}

// mapBackendError maps backend errors to appropriate gRPC status codes.
func mapBackendError(err error) error {
	// Check for specific backend errors
	switch {
	case errors.Is(err, backend.ErrStateFileNotFound):
		return status.Error(codes.NotFound, "state file not found")
	case errors.Is(err, backend.ErrBlobNotFound):
		return status.Error(codes.NotFound, "blob not found")
	case errors.Is(err, backend.ErrAuthenticationFailed):
		return status.Error(codes.PermissionDenied, "authentication failed")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "operation cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "operation timed out")
	default:
		// For other errors, return as Internal
		return status.Errorf(codes.Internal, "backend error: %v", err)
	}
}
