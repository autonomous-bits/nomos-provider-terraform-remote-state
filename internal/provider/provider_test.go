package provider

import (
	"context"
	"testing"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// TestNewService verifies that NewService creates a properly initialized Service.
func TestNewService(t *testing.T) {
	service := NewService()

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.instances == nil {
		t.Error("instances map should be initialized")
	}

	if len(service.instances) != 0 {
		t.Errorf("expected 0 instances, got %d", len(service.instances))
	}
}

// TestSetVersion verifies that SetVersion sets the package-level version variable.
func TestSetVersion(t *testing.T) {
	originalVersion := version
	defer func() { version = originalVersion }()

	testVersion := "1.0.0-test"
	SetVersion(testVersion)

	if version != testVersion {
		t.Errorf("expected version %q, got %q", testVersion, version)
	}
}

// TestInfo verifies that Info RPC returns correct provider metadata.
func TestInfo(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Set test version
	originalVersion := version
	version = "0.1.0-test"
	defer func() { version = originalVersion }()

	resp, err := service.Info(ctx, &pb.InfoRequest{})
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if resp.Type != "terraform-remote-state" {
		t.Errorf("expected type %q, got %q", "terraform-remote-state", resp.Type)
	}

	if resp.Version != "0.1.0-test" {
		t.Errorf("expected version %q, got %q", "0.1.0-test", resp.Version)
	}

	// Alias should be empty at service level
	if resp.Alias != "" {
		t.Errorf("expected empty alias, got %q", resp.Alias)
	}
}

// TestHealth verifies that Health RPC returns STATUS_OK.
func TestHealth(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	resp, err := service.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}

	if resp.Status != pb.HealthResponse_STATUS_OK {
		t.Errorf("expected STATUS_OK, got %v", resp.Status)
	}
}

// TestShutdown verifies that Shutdown RPC clears instances.
func TestShutdown(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Add a mock instance
	service.instances["test-alias"] = &instance{
		alias:   "test-alias",
		backend: nil,
	}

	if len(service.instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(service.instances))
	}

	// Call Shutdown
	resp, err := service.Shutdown(ctx, &pb.ShutdownRequest{})
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if resp == nil {
		t.Error("expected non-nil response")
	}

	// Verify instances were cleared
	if len(service.instances) != 0 {
		t.Errorf("expected 0 instances after shutdown, got %d", len(service.instances))
	}
}

// TestConcurrentAccess verifies thread-safety of Service operations.
func TestConcurrentAccess(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Run multiple Info calls concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := service.Info(ctx, &pb.InfoRequest{})
			if err != nil {
				t.Errorf("concurrent Info call failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Run multiple Health calls concurrently
	for i := 0; i < 10; i++ {
		go func() {
			_, err := service.Health(ctx, &pb.HealthRequest{})
			if err != nil {
				t.Errorf("concurrent Health call failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestShutdownEmptyService verifies Shutdown works on empty service.
func TestShutdownEmptyService(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	resp, err := service.Shutdown(ctx, &pb.ShutdownRequest{})
	if err != nil {
		t.Fatalf("Shutdown on empty service failed: %v", err)
	}

	if resp == nil {
		t.Error("expected non-nil response")
	}

	// Verify no instances after shutdown
	if len(service.instances) != 0 {
		t.Errorf("expected 0 instances after shutdown, got %d", len(service.instances))
	}
}
