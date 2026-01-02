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
	"sync"
	"sync/atomic"
	"time"

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

// metrics tracks RPC call counts, errors, and durations for observability.
//
// This is a simple in-memory metrics implementation for MVP. It uses atomic operations
// for thread-safe increments without requiring mutexes. All counters are monotonically
// increasing and reset only on service restart.
//
// Future Enhancement: Replace this struct with a real metrics library:
//   - Prometheus: Use prometheus.Counter and prometheus.Histogram types
//   - OpenTelemetry: Use meter.Int64Counter and meter.Float64Histogram
//   - Export endpoint: Add HTTP /metrics endpoint or OTLP exporter
//
// The current structure matches common metrics library patterns (method-specific counters)
// to make migration straightforward. When replacing, convert each atomic field to the
// corresponding metrics library primitive and add labels/attributes as needed.
type metrics struct {
	// Call counters: total invocations per RPC method
	initCalls     atomic.Int64
	fetchCalls    atomic.Int64
	infoCalls     atomic.Int64
	healthCalls   atomic.Int64
	shutdownCalls atomic.Int64

	// Error counters: failed invocations per RPC method
	initErrors     atomic.Int64
	fetchErrors    atomic.Int64
	infoErrors     atomic.Int64
	healthErrors   atomic.Int64
	shutdownErrors atomic.Int64

	// Duration tracking: sum of durations in nanoseconds
	// To get average: totalDuration / totalCalls
	// For percentiles, a real metrics library with histograms is needed
	initDurationNs     atomic.Int64
	fetchDurationNs    atomic.Int64
	infoDurationNs     atomic.Int64
	healthDurationNs   atomic.Int64
	shutdownDurationNs atomic.Int64
}

// recordCall increments the call counter and returns a function to record duration and errors.
//
// This helper function simplifies RPC method instrumentation:
//
//	done := s.metrics.recordCall(&s.metrics.initCalls, &s.metrics.initDurationNs)
//	defer done(&err, &s.metrics.initErrors)
//
// The returned function should be deferred and will:
//   - Record the call duration
//   - Increment error counter if err is not nil
func (m *metrics) recordCall(callCounter, durationCounter *atomic.Int64) func(err *error, errorCounter *atomic.Int64) {
	start := time.Now()
	callCounter.Add(1)

	return func(err *error, errorCounter *atomic.Int64) {
		duration := time.Since(start)
		durationCounter.Add(duration.Nanoseconds())

		if err != nil && *err != nil {
			errorCounter.Add(1)
		}
	}
}

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
	metrics   metrics
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
func (s *Service) Info(_ context.Context, _ *pb.InfoRequest) (resp *pb.InfoResponse, err error) {
	done := s.metrics.recordCall(&s.metrics.infoCalls, &s.metrics.infoDurationNs)
	defer done(&err, &s.metrics.infoErrors)

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
func (s *Service) Health(_ context.Context, _ *pb.HealthRequest) (resp *pb.HealthResponse, err error) {
	done := s.metrics.recordCall(&s.metrics.healthCalls, &s.metrics.healthDurationNs)
	defer done(&err, &s.metrics.healthErrors)

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
func (s *Service) Shutdown(_ context.Context, _ *pb.ShutdownRequest) (resp *pb.ShutdownResponse, err error) {
	done := s.metrics.recordCall(&s.metrics.shutdownCalls, &s.metrics.shutdownDurationNs)
	defer done(&err, &s.metrics.shutdownErrors)

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
func (s *Service) Init(ctx context.Context, req *pb.InitRequest) (resp *pb.InitResponse, err error) {
	done := s.metrics.recordCall(&s.metrics.initCalls, &s.metrics.initDurationNs)
	defer done(&err, &s.metrics.initErrors)

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

	// Get backend constructor from registry
	b, err := backend.GetBackend(ctx, cfg.Type(), cfg.Raw())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
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
func (s *Service) Fetch(ctx context.Context, req *pb.FetchRequest) (resp *pb.FetchResponse, err error) {
	done := s.metrics.recordCall(&s.metrics.fetchCalls, &s.metrics.fetchDurationNs)
	defer done(&err, &s.metrics.fetchErrors)

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

	// Convert output value to protobuf Value
	pbValue, err := structpb.NewValue(output.Value)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert output value: %v", err)
	}

	// protobuf Struct can only hold JSON objects (maps), not primitives or arrays directly.
	// To return the actual value cleanly, we wrap it in a single-field struct.
	// For objects, we return them directly.
	// For primitives/arrays, we wrap in {"value": <actual-value>} for backwards compatibility.
	var valueStruct *structpb.Struct
	if s := pbValue.GetStructValue(); s != nil {
		// Value is already an object - return it directly
		valueStruct = s
	} else {
		// Value is a primitive or array - wrap in a struct
		valueStruct = &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"value": pbValue,
			},
		}
	}

	return &pb.FetchResponse{
		Value: valueStruct,
	}, nil
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
