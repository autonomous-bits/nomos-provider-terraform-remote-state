//go:build integration

package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// Performance requirements from spec:
const (
	maxInitDuration   = 2 * time.Second        // Init RPC must complete within 2 seconds
	maxFetchDuration  = 5 * time.Second        // Fetch RPC must complete within 5 seconds (local backend)
	maxHealthDuration = 100 * time.Millisecond // Health RPC must complete within 100ms
)

// TestPerformance_InitRPC validates that Init RPC completes within performance requirements.
//
// Tests Init with:
// - Small state file (1KB)
// - Medium state file (100KB)
// - Large state file (1MB)
//
// All must complete within 2 seconds as per spec.
func TestPerformance_InitRPC(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	ctx := context.Background()

	testCases := []struct {
		name     string
		sizeDesc string
		size     int // Number of outputs to generate
	}{
		{
			name:     "small_state_1kb",
			sizeDesc: "~1KB",
			size:     10,
		},
		{
			name:     "medium_state_100kb",
			sizeDesc: "~100KB",
			size:     1000,
		},
		{
			name:     "large_state_1mb",
			sizeDesc: "~1MB",
			size:     10000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService()

			// Create state file of appropriate size
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "terraform.tfstate")
			createStateFileWithOutputs(t, stateFile, tc.size)

			// Verify file size (informational)
			info, err := os.Stat(stateFile)
			if err != nil {
				t.Fatalf("failed to stat file: %v", err)
			}
			t.Logf("State file size: %d bytes (%.2f KB)", info.Size(), float64(info.Size())/1024)

			config, err := structpb.NewStruct(map[string]interface{}{
				"backend_type": "local",
				"path":         stateFile,
			})
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			// Measure Init duration
			start := time.Now()
			_, err = service.Init(ctx, &pb.InitRequest{
				Alias:  "perf-test",
				Config: config,
			})
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}

			// Verify performance requirement
			if duration > maxInitDuration {
				t.Errorf("Init(%s) took %v, requirement < %v ❌", tc.sizeDesc, duration, maxInitDuration)
			} else {
				t.Logf("Init(%s) completed in %v (< %v) ✓", tc.sizeDesc, duration, maxInitDuration)
			}

			// Report performance metrics
			t.Logf("Performance: %d outputs initialized in %v (%.2f outputs/ms)",
				tc.size, duration, float64(tc.size)/float64(duration.Milliseconds()))
		})
	}
}

// TestPerformance_FetchRPC validates that Fetch RPC completes within performance requirements.
//
// Tests Fetch with different state file sizes to ensure consistent performance.
func TestPerformance_FetchRPC(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	ctx := context.Background()

	testCases := []struct {
		name         string
		sizeDesc     string
		outputCount  int
		fetchOutputs []string // Outputs to fetch
	}{
		{
			name:         "small_state_single_output",
			sizeDesc:     "~1KB",
			outputCount:  10,
			fetchOutputs: []string{"output_0", "output_5", "output_9"},
		},
		{
			name:         "medium_state_single_output",
			sizeDesc:     "~100KB",
			outputCount:  1000,
			fetchOutputs: []string{"output_0", "output_500", "output_999"},
		},
		{
			name:         "large_state_single_output",
			sizeDesc:     "~1MB",
			outputCount:  10000,
			fetchOutputs: []string{"output_0", "output_5000", "output_9999"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService()

			// Create and initialize provider with state file
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "terraform.tfstate")
			createStateFileWithOutputs(t, stateFile, tc.outputCount)

			config, err := structpb.NewStruct(map[string]interface{}{
				"backend_type": "local",
				"path":         stateFile,
			})
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			_, err = service.Init(ctx, &pb.InitRequest{
				Alias:  "perf-test",
				Config: config,
			})
			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}

			// Test fetching multiple outputs
			var totalDuration time.Duration
			for _, outputName := range tc.fetchOutputs {
				start := time.Now()
				resp, err := service.Fetch(ctx, &pb.FetchRequest{
					Path: []string{outputName},
				})
				duration := time.Since(start)
				totalDuration += duration

				if err != nil {
					t.Fatalf("Fetch(%s) error = %v", outputName, err)
				}

				// Verify we got the correct output
				fields := resp.Value.AsMap()
				value := fields["value"].(string)
				expectedValue := fmt.Sprintf("value_%s", outputName)
				if value != expectedValue {
					t.Errorf("Fetch(%s) value = %q, want %q", outputName, value, expectedValue)
				}

				// Verify performance requirement for each fetch
				if duration > maxFetchDuration {
					t.Errorf("Fetch(%s) from %s took %v, requirement < %v ❌",
						outputName, tc.sizeDesc, duration, maxFetchDuration)
				} else {
					t.Logf("Fetch(%s) from %s completed in %v (< %v) ✓",
						outputName, tc.sizeDesc, duration, maxFetchDuration)
				}
			}

			// Report average fetch time
			avgDuration := totalDuration / time.Duration(len(tc.fetchOutputs))
			t.Logf("Average Fetch time for %s: %v (requirement: < %v)",
				tc.sizeDesc, avgDuration, maxFetchDuration)
		})
	}
}

// TestPerformance_HealthRPC validates that Health RPC completes within performance requirements.
//
// Health checks must be fast (< 100ms) as they may be called frequently.
func TestPerformance_HealthRPC(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	service := NewService()
	ctx := context.Background()

	// Run multiple iterations to get consistent measurements
	iterations := 100
	var totalDuration time.Duration
	var slowest time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := service.Health(ctx, &pb.HealthRequest{})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Health() iteration %d error = %v", i, err)
		}

		totalDuration += duration
		if duration > slowest {
			slowest = duration
		}

		// Each individual call must meet requirement
		if duration > maxHealthDuration {
			t.Errorf("Health() iteration %d took %v, requirement < %v ❌", i, duration, maxHealthDuration)
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)

	t.Logf("Health RPC Performance (%d iterations):", iterations)
	t.Logf("  Average: %v (requirement: < %v)", avgDuration, maxHealthDuration)
	t.Logf("  Slowest: %v (requirement: < %v)", slowest, maxHealthDuration)
	t.Logf("  Total: %v", totalDuration)

	// Verify average meets requirement with headroom
	if avgDuration > maxHealthDuration/2 {
		t.Logf("Warning: Average Health time (%v) exceeds 50%% of requirement (%v)",
			avgDuration, maxHealthDuration/2)
	}

	if avgDuration < maxHealthDuration {
		t.Logf("Health RPC performance: ✓ PASS")
	}
}

// TestPerformance_InfoRPC validates Info RPC performance.
//
// Info RPC should also be fast as it provides metadata.
func TestPerformance_InfoRPC(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	service := NewService()
	ctx := context.Background()

	// Run multiple iterations
	iterations := 100
	var totalDuration time.Duration
	maxExpected := 100 * time.Millisecond

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := service.Info(ctx, &pb.InfoRequest{})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Info() iteration %d error = %v", i, err)
		}

		totalDuration += duration

		if duration > maxExpected {
			t.Errorf("Info() iteration %d took %v, expected < %v", i, duration, maxExpected)
		}
	}

	avgDuration := totalDuration / time.Duration(iterations)
	t.Logf("Info RPC average time: %v (< %v) ✓", avgDuration, maxExpected)
}

// TestPerformance_ConcurrentFetch validates performance with concurrent Fetch operations.
//
// Simulates realistic usage where multiple fetches may occur simultaneously.
func TestPerformance_ConcurrentFetch(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	ctx := context.Background()
	service := NewService()

	// Create state file with medium number of outputs
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	outputCount := 100
	createStateFileWithOutputs(t, stateFile, outputCount)

	// Initialize provider
	config, err := structpb.NewStruct(map[string]interface{}{
		"backend_type": "local",
		"path":         stateFile,
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	_, err = service.Init(ctx, &pb.InitRequest{
		Alias:  "perf-test",
		Config: config,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Run concurrent fetches
	concurrency := 10
	fetchesPerGoroutine := 10

	done := make(chan time.Duration, concurrency)
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		i := i
		go func() {
			goroutineStart := time.Now()
			for j := 0; j < fetchesPerGoroutine; j++ {
				outputName := fmt.Sprintf("output_%d", (i*fetchesPerGoroutine+j)%outputCount)
				_, err := service.Fetch(ctx, &pb.FetchRequest{
					Path: []string{outputName},
				})
				if err != nil {
					t.Errorf("concurrent Fetch(%s) error = %v", outputName, err)
				}
			}
			done <- time.Since(goroutineStart)
		}()
	}

	// Wait for all goroutines and collect durations
	var totalGoroutineDuration time.Duration
	for i := 0; i < concurrency; i++ {
		duration := <-done
		totalGoroutineDuration += duration
	}

	totalWallClock := time.Since(start)
	totalFetches := concurrency * fetchesPerGoroutine
	avgFetchTime := totalWallClock / time.Duration(totalFetches)

	t.Logf("Concurrent Fetch Performance:")
	t.Logf("  Concurrency: %d goroutines", concurrency)
	t.Logf("  Total fetches: %d", totalFetches)
	t.Logf("  Wall clock time: %v", totalWallClock)
	t.Logf("  Average fetch time: %v", avgFetchTime)
	t.Logf("  Throughput: %.2f fetches/sec", float64(totalFetches)/totalWallClock.Seconds())

	// Verify all fetches completed within reasonable time
	maxExpected := maxFetchDuration * time.Duration(fetchesPerGoroutine) * 2 // With concurrency, allow 2x
	if totalWallClock > maxExpected {
		t.Errorf("Concurrent fetches took %v, expected < %v", totalWallClock, maxExpected)
	}
}

// BenchmarkInitLocalBackend benchmarks Init RPC with local backend.
func BenchmarkInitLocalBackend(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	createStateFileWithOutputs(b, stateFile, 100)

	config, err := structpb.NewStruct(map[string]interface{}{
		"backend_type": "local",
		"path":         stateFile,
	})
	if err != nil {
		b.Fatalf("failed to create config struct: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service := NewService()
		_, err := service.Init(ctx, &pb.InitRequest{
			Alias:  fmt.Sprintf("bench-%d", i),
			Config: config,
		})
		if err != nil {
			b.Fatalf("Init() error = %v", err)
		}
	}
}

// BenchmarkFetchLocalBackend benchmarks Fetch RPC with different state sizes.
func BenchmarkFetchLocalBackend(b *testing.B) {
	sizes := []struct {
		name        string
		outputCount int
	}{
		{"small_1kb", 10},
		{"medium_100kb", 1000},
		{"large_1mb", 10000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			ctx := context.Background()
			service := NewService()

			tmpDir := b.TempDir()
			stateFile := filepath.Join(tmpDir, "terraform.tfstate")
			createStateFileWithOutputs(b, stateFile, size.outputCount)

			config, err := structpb.NewStruct(map[string]interface{}{
				"backend_type": "local",
				"path":         stateFile,
			})
			if err != nil {
				b.Fatalf("failed to create config struct: %v", err)
			}

			_, err = service.Init(ctx, &pb.InitRequest{
				Alias:  "bench",
				Config: config,
			})
			if err != nil {
				b.Fatalf("Init() error = %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				outputName := fmt.Sprintf("output_%d", i%size.outputCount)
				_, err := service.Fetch(ctx, &pb.FetchRequest{
					Path: []string{outputName},
				})
				if err != nil {
					b.Fatalf("Fetch() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkHealthRPC benchmarks Health RPC.
func BenchmarkHealthRPC(b *testing.B) {
	service := NewService()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.Health(ctx, &pb.HealthRequest{})
		if err != nil {
			b.Fatalf("Health() error = %v", err)
		}
	}
}

// BenchmarkInfoRPC benchmarks Info RPC.
func BenchmarkInfoRPC(b *testing.B) {
	service := NewService()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.Info(ctx, &pb.InfoRequest{})
		if err != nil {
			b.Fatalf("Info() error = %v", err)
		}
	}
}

// BenchmarkStateFileParsing benchmarks state file parsing with different sizes.
func BenchmarkStateFileParsing(b *testing.B) {
	sizes := []struct {
		name        string
		outputCount int
	}{
		{"small_1kb", 10},
		{"medium_100kb", 1000},
		{"large_1mb", 10000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			ctx := context.Background()
			tmpDir := b.TempDir()
			stateFile := filepath.Join(tmpDir, "terraform.tfstate")
			createStateFileWithOutputs(b, stateFile, size.outputCount)

			service := NewService()
			config, err := structpb.NewStruct(map[string]interface{}{
				"backend_type": "local",
				"path":         stateFile,
			})
			if err != nil {
				b.Fatalf("failed to create config struct: %v", err)
			}

			_, err = service.Init(ctx, &pb.InitRequest{
				Alias:  "bench",
				Config: config,
			})
			if err != nil {
				b.Fatalf("Init() error = %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			// Each fetch requires re-reading and parsing the state file (no caching)
			for i := 0; i < b.N; i++ {
				_, err := service.Fetch(ctx, &pb.FetchRequest{
					Path: []string{"output_0"},
				})
				if err != nil {
					b.Fatalf("Fetch() error = %v", err)
				}
			}
		})
	}
}

// createStateFileWithOutputs creates a state file with the specified number of outputs.
//
// This helper generates realistic state files for performance testing.
func createStateFileWithOutputs(tb testing.TB, path string, outputCount int) {
	tb.Helper()

	var outputs strings.Builder
	outputs.WriteString("{\n")
	outputs.WriteString(`  "version": 4,` + "\n")
	outputs.WriteString(`  "terraform_version": "1.5.0",` + "\n")
	outputs.WriteString(`  "serial": 1,` + "\n")
	outputs.WriteString(`  "lineage": "test-lineage",` + "\n")
	outputs.WriteString(`  "outputs": {` + "\n")

	for i := 0; i < outputCount; i++ {
		outputName := fmt.Sprintf("output_%d", i)
		outputValue := fmt.Sprintf("value_%s", outputName)

		outputs.WriteString(fmt.Sprintf(`    "%s": {`+"\n", outputName))
		outputs.WriteString(fmt.Sprintf(`      "value": "%s",`+"\n", outputValue))
		outputs.WriteString(`      "type": "string",` + "\n")
		outputs.WriteString(`      "sensitive": false` + "\n")

		if i < outputCount-1 {
			outputs.WriteString("    },\n")
		} else {
			outputs.WriteString("    }\n")
		}
	}

	outputs.WriteString("  }\n")
	outputs.WriteString("}\n")

	if err := os.WriteFile(path, []byte(outputs.String()), 0600); err != nil {
		tb.Fatalf("failed to create state file: %v", err)
	}
}
