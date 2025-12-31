// Package main provides the entry point for the Terraform Remote State provider server.
//
// This binary implements a gRPC server that exposes the ProviderService interface,
// allowing Nomos to interact with Terraform Remote State backends. The server:
//   - Listens on a random TCP port for subprocess discovery
//   - Prints PROVIDER_PORT to stdout for Nomos tooling discovery
//   - Handles graceful shutdown via OS signals (SIGINT, SIGTERM)
//   - Provides structured logging for debugging and monitoring
//
// Usage:
//
//	nomos-provider-terraform-remote-state
//
// The server runs until interrupted, at which point it performs graceful shutdown.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/provider"
	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// Build-time variables set via ldflags
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting terraform-remote-state provider",
		"version", version,
		"commit", commit,
		"buildTime", buildTime,
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run server and handle errors
	if err := run(ctx, logger); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server shutdown complete")
}

// run starts the gRPC server and handles graceful shutdown.
//
// This function:
//  1. Creates and configures the gRPC server with appropriate parameters
//  2. Listens on a random port (":0") for subprocess discovery
//  3. Prints PROVIDER_PORT to stdout (required by Nomos tooling)
//  4. Registers the provider service
//  5. Starts the server in a goroutine
//  6. Waits for shutdown signals (SIGINT, SIGTERM)
//  7. Performs graceful shutdown with timeout
//
// Returns error if server setup or execution fails.
func run(ctx context.Context, logger *slog.Logger) error {
	// Create gRPC server with production-ready configuration
	grpcServer := grpc.NewServer(
		// Max message sizes (10MB for large state files)
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.MaxSendMsgSize(10*1024*1024),

		// Keepalive parameters for long-lived connections
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Minute,
			Timeout:               20 * time.Second,
		}),

		// Keepalive enforcement
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Minute,
			PermitWithoutStream: true,
		}),
	)

	// Create and register provider service
	providerService := provider.NewService()
	provider.SetVersion(version)
	pb.RegisterProviderServiceServer(grpcServer, providerService)

	// Listen on random port (required for Nomos subprocess discovery)
	// gosec G102: Binding to all interfaces is intentional for subprocess discovery
	listener, err := net.Listen("tcp", ":0") //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Extract and print port for Nomos discovery
	// This MUST be printed to stdout for Nomos tooling to discover the provider
	addr := listener.Addr().(*net.TCPAddr)
	fmt.Printf("PROVIDER_PORT=%d\n", addr.Port)

	logger.Info("gRPC server listening",
		"address", addr.String(),
		"port", addr.Port,
	)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("context canceled, shutting down")
	case sig := <-sigChan:
		logger.Info("received shutdown signal", "signal", sig)
	case err := <-serverErr:
		return err
	}

	// Graceful shutdown with timeout
	logger.Info("initiating graceful shutdown")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Create channel to signal when graceful stop completes
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-done:
		logger.Info("graceful shutdown completed")
	case <-shutdownCtx.Done():
		logger.Warn("graceful shutdown timeout, forcing stop")
		grpcServer.Stop()
	}

	return nil
}
