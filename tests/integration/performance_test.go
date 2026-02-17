//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// T089: Benchmark test for SC-002 (fetch response <10ms)
//
// Success Criteria: SC-002 states fetch should respond in <10ms for cached values
// Tests gRPC roundtrip time for cached environment variable fetches
//
// Performance Target: <10ms per fetch operation (cached values)
//
// Usage:
//
//	go test -tags=integration -bench=BenchmarkFetchResponseTime -benchmem ./tests/integration/
//
// Expected Results:
//   - Each fetch operation should complete in <10ms
//   - Memory allocations should be minimal for cached reads
//   - Consistent performance across multiple iterations
func BenchmarkFetchResponseTime(b *testing.B) {
	client, cleanup := startTestServer(b)
	defer cleanup()

	ctx := context.Background()

	// Set up test environment variable
	testKey := fmt.Sprintf("BENCHMARK_VAR_%d", time.Now().Unix())
	testValue := "benchmark_value_cached"
	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Initialize provider
	configStruct, err := structpb.NewStruct(map[string]interface{}{})
	if err != nil {
		b.Fatalf("failed to create config: %v", err)
	}

	_, err = client.Init(ctx, &pb.InitRequest{
		Alias:  "benchmark-env",
		Config: configStruct,
	})
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}

	// Pre-warm cache with initial fetch
	_, err = client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{testKey},
	})
	if err != nil {
		b.Fatalf("pre-warm fetch failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Benchmark cached fetch operations
	for i := 0; i < b.N; i++ {
		_, err := client.Fetch(ctx, &pb.FetchRequest{
			Path: []string{testKey},
		})
		if err != nil {
			b.Fatalf("fetch failed on iteration %d: %v", i, err)
		}
	}

	b.StopTimer()

	// Report performance metrics
	elapsed := b.Elapsed()
	opsPerSec := float64(b.N) / elapsed.Seconds()
	avgLatency := elapsed / time.Duration(b.N)

	b.ReportMetric(opsPerSec, "ops/s")
	b.ReportMetric(float64(avgLatency.Nanoseconds())/1e6, "ms/op")

	// Validate SC-002: <10ms per operation
	if avgLatency > 10*time.Millisecond {
		b.Errorf("SC-002 FAILED: average latency %v exceeds 10ms target", avgLatency)
	} else {
		b.Logf("SC-002 PASSED: average latency %v is within 10ms target", avgLatency)
	}
}

// BenchmarkFetchResponseTime_MultipleVariables benchmarks fetch performance
// with different types of environment variables
func BenchmarkFetchResponseTime_MultipleVariables(b *testing.B) {
	client, cleanup := startTestServer(b)
	defer cleanup()

	ctx := context.Background()

	// Set up multiple test environment variables with different types
	timestamp := time.Now().Unix()
	testVars := map[string]string{
		fmt.Sprintf("BENCH_STRING_%d", timestamp):  "simple_string_value",
		fmt.Sprintf("BENCH_NUMBER_%d", timestamp):  "42",
		fmt.Sprintf("BENCH_BOOLEAN_%d", timestamp): "true",
		fmt.Sprintf("BENCH_JSON_%d", timestamp):    `{"key":"value","count":10}`,
	}

	for key, value := range testVars {
		_ = os.Setenv(key, value)
		defer func(k string) { _ = os.Unsetenv(k) }(key)
	}

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "benchmark-env",
		Config: configStruct,
	})
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}

	// Pre-warm cache
	for key := range testVars {
		_, _ = client.Fetch(ctx, &pb.FetchRequest{Path: []string{key}})
	}

	// Benchmark each variable type
	for varName, varType := range map[string]string{
		fmt.Sprintf("BENCH_STRING_%d", timestamp):  "string",
		fmt.Sprintf("BENCH_NUMBER_%d", timestamp):  "number",
		fmt.Sprintf("BENCH_BOOLEAN_%d", timestamp): "boolean",
		fmt.Sprintf("BENCH_JSON_%d", timestamp):    "json",
	} {
		b.Run(varType, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := client.Fetch(ctx, &pb.FetchRequest{
					Path: []string{varName},
				})
				if err != nil {
					b.Fatalf("fetch failed: %v", err)
				}
			}

			b.StopTimer()
			avgLatency := b.Elapsed() / time.Duration(b.N)
			b.ReportMetric(float64(avgLatency.Nanoseconds())/1e6, "ms/op")

			if avgLatency > 10*time.Millisecond {
				b.Errorf("SC-002 FAILED for %s: latency %v exceeds 10ms", varType, avgLatency)
			}
		})
	}
}

// BenchmarkFetchResponseTime_PathResolution benchmarks fetch performance
// with multi-segment path resolution
func BenchmarkFetchResponseTime_PathResolution(b *testing.B) {
	client, cleanup := startTestServer(b)
	defer cleanup()

	ctx := context.Background()

	// Set up environment variable for path resolution
	timestamp := time.Now().Unix()
	testKey := fmt.Sprintf("DATABASE_HOST_PROD_%d", timestamp)
	testValue := "localhost"
	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Initialize provider with path transformation
	configStruct, _ := structpb.NewStruct(map[string]interface{}{
		"separator":      "_",
		"case_transform": "upper",
	})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "benchmark-env",
		Config: configStruct,
	})
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}

	// Pre-warm cache
	_, err = client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{"database", "host", "prod", fmt.Sprintf("%d", timestamp)},
	})
	if err != nil {
		b.Fatalf("pre-warm fetch failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Benchmark path resolution with cached values
	for i := 0; i < b.N; i++ {
		_, err := client.Fetch(ctx, &pb.FetchRequest{
			Path: []string{"database", "host", "prod", fmt.Sprintf("%d", timestamp)},
		})
		if err != nil {
			b.Fatalf("fetch failed: %v", err)
		}
	}

	b.StopTimer()
	avgLatency := b.Elapsed() / time.Duration(b.N)
	b.ReportMetric(float64(avgLatency.Nanoseconds())/1e6, "ms/op")

	if avgLatency > 10*time.Millisecond {
		b.Errorf("SC-002 FAILED for path resolution: latency %v exceeds 10ms", avgLatency)
	} else {
		b.Logf("SC-002 PASSED for path resolution: latency %v", avgLatency)
	}
}

// T090: Benchmark test for SC-006 (concurrent fetches)
//
// Success Criteria: SC-006 states safe concurrent fetches across 10,000 concurrent requests
// Tests provider thread safety and performance under high concurrency
//
// Performance Target: 10,000 concurrent requests with zero data races
//
// Usage:
//
//	go test -tags=integration -bench=BenchmarkConcurrentFetches -benchmem ./tests/integration/
//	go test -tags=integration -bench=BenchmarkConcurrentFetches -race ./tests/integration/
//
// Expected Results:
//   - All concurrent fetches complete successfully
//   - Zero data races when run with -race flag
//   - Consistent values returned across all goroutines
//   - Memory usage scales reasonably with concurrency
func BenchmarkConcurrentFetches(b *testing.B) {
	client, cleanup := startTestServer(b)
	defer cleanup()

	ctx := context.Background()

	// Set up test environment variable
	testKey := fmt.Sprintf("CONCURRENT_BENCH_VAR_%d", time.Now().Unix())
	testValue := "concurrent_value_123"
	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "benchmark-env",
		Config: configStruct,
	})
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}

	// Pre-warm cache
	_, err = client.Fetch(ctx, &pb.FetchRequest{Path: []string{testKey}})
	if err != nil {
		b.Fatalf("pre-warm fetch failed: %v", err)
	}

	// Test different concurrency levels
	concurrencyLevels := []int{10, 100, 1000, 5000, 10000}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency_%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				errChan := make(chan error, concurrency)
				successCount := 0
				var mu sync.Mutex

				wg.Add(concurrency)

				for j := 0; j < concurrency; j++ {
					go func(goroutineID int) {
						defer wg.Done()

						resp, err := client.Fetch(ctx, &pb.FetchRequest{
							Path: []string{testKey},
						})

						if err != nil {
							errChan <- fmt.Errorf("goroutine %d failed: %w", goroutineID, err)
							return
						}

						// Verify response consistency
						if resp.Value == nil || resp.Value.Fields == nil {
							errChan <- fmt.Errorf("goroutine %d: nil response", goroutineID)
							return
						}

						value := resp.Value.Fields["value"].GetStringValue()
						if value != testValue {
							errChan <- fmt.Errorf("goroutine %d: got %q, want %q", goroutineID, value, testValue)
							return
						}

						mu.Lock()
						successCount++
						mu.Unlock()
					}(j)
				}

				wg.Wait()
				close(errChan)

				// Check for errors
				errorCount := 0
				for err := range errChan {
					errorCount++
					if errorCount <= 5 {
						b.Errorf("Error: %v", err)
					}
				}

				if errorCount > 0 {
					b.Fatalf("SC-006 FAILED: %d out of %d concurrent requests failed", errorCount, concurrency)
				}

				if successCount != concurrency {
					b.Fatalf("SC-006 FAILED: expected %d successes, got %d", concurrency, successCount)
				}
			}

			b.StopTimer()

			// Report metrics
			totalOps := b.N * concurrency
			opsPerSec := float64(totalOps) / b.Elapsed().Seconds()
			b.ReportMetric(opsPerSec, "total_ops/s")
			b.ReportMetric(float64(concurrency), "concurrent_requests")

			if concurrency >= 10000 {
				b.Logf("SC-006 PASSED: successfully handled %d concurrent requests", concurrency)
			}
		})
	}
}

// BenchmarkConcurrentFetches_MixedVariables benchmarks concurrent fetches
// with multiple different environment variables
func BenchmarkConcurrentFetches_MixedVariables(b *testing.B) {
	client, cleanup := startTestServer(b)
	defer cleanup()

	ctx := context.Background()

	// Set up multiple test environment variables
	timestamp := time.Now().Unix()
	numVars := 10
	testVars := make(map[string]string)

	for i := 0; i < numVars; i++ {
		key := fmt.Sprintf("CONCURRENT_VAR_%d_%d", i, timestamp)
		value := fmt.Sprintf("value_%d", i)
		testVars[key] = value
		_ = os.Setenv(key, value)
		defer func(k string) { _ = os.Unsetenv(k) }(key)
	}

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "benchmark-env",
		Config: configStruct,
	})
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}

	// Pre-warm cache
	for key := range testVars {
		_, _ = client.Fetch(ctx, &pb.FetchRequest{Path: []string{key}})
	}

	const concurrency = 1000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errChan := make(chan error, concurrency)

		wg.Add(concurrency)

		// Each goroutine fetches a different variable (round-robin)
		varKeys := make([]string, 0, len(testVars))
		for k := range testVars {
			varKeys = append(varKeys, k)
		}

		for j := 0; j < concurrency; j++ {
			go func(goroutineID int) {
				defer wg.Done()

				// Select variable based on goroutine ID (round-robin)
				varKey := varKeys[goroutineID%len(varKeys)]
				expectedValue := testVars[varKey]

				resp, err := client.Fetch(ctx, &pb.FetchRequest{
					Path: []string{varKey},
				})

				if err != nil {
					errChan <- fmt.Errorf("goroutine %d failed: %w", goroutineID, err)
					return
				}

				value := resp.Value.Fields["value"].GetStringValue()
				if value != expectedValue {
					errChan <- fmt.Errorf("goroutine %d: got %q, want %q", goroutineID, value, expectedValue)
				}
			}(j)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			b.Errorf("Error: %v", err)
		}
	}

	b.StopTimer()

	opsPerSec := float64(b.N*concurrency) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "ops/s")
	b.ReportMetric(float64(concurrency), "concurrent_goroutines")
}

// BenchmarkConcurrentFetches_WithPathResolution benchmarks concurrent fetches
// with path transformation and resolution
func BenchmarkConcurrentFetches_WithPathResolution(b *testing.B) {
	client, cleanup := startTestServer(b)
	defer cleanup()

	ctx := context.Background()

	// Set up test environment variables with path-based names
	timestamp := time.Now().Unix()
	testVars := map[string][]string{
		fmt.Sprintf("DATABASE_HOST_%d", timestamp):   {"database", "host", fmt.Sprintf("%d", timestamp)},
		fmt.Sprintf("DATABASE_PORT_%d", timestamp):   {"database", "port", fmt.Sprintf("%d", timestamp)},
		fmt.Sprintf("API_ENDPOINT_%d", timestamp):    {"api", "endpoint", fmt.Sprintf("%d", timestamp)},
		fmt.Sprintf("CACHE_TIMEOUT_%d", timestamp):   {"cache", "timeout", fmt.Sprintf("%d", timestamp)},
		fmt.Sprintf("FEATURE_ENABLED_%d", timestamp): {"feature", "enabled", fmt.Sprintf("%d", timestamp)},
	}

	for key := range testVars {
		_ = os.Setenv(key, "test_value")
		defer func(k string) { _ = os.Unsetenv(k) }(key)
	}

	// Initialize provider with path transformation
	configStruct, _ := structpb.NewStruct(map[string]interface{}{
		"separator":      "_",
		"case_transform": "upper",
	})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "benchmark-env",
		Config: configStruct,
	})
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}

	// Pre-warm cache
	for _, path := range testVars {
		_, _ = client.Fetch(ctx, &pb.FetchRequest{Path: path})
	}

	const concurrency = 1000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errChan := make(chan error, concurrency)

		wg.Add(concurrency)

		// Convert map to slice for indexing
		paths := make([][]string, 0, len(testVars))
		for _, path := range testVars {
			paths = append(paths, path)
		}

		for j := 0; j < concurrency; j++ {
			go func(goroutineID int) {
				defer wg.Done()

				// Select path based on goroutine ID (round-robin)
				path := paths[goroutineID%len(paths)]

				_, err := client.Fetch(ctx, &pb.FetchRequest{
					Path: path,
				})

				if err != nil {
					errChan <- fmt.Errorf("goroutine %d failed: %w", goroutineID, err)
				}
			}(j)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			b.Errorf("Error: %v", err)
		}
	}

	b.StopTimer()

	opsPerSec := float64(b.N*concurrency) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "ops/s")
}

// BenchmarkConcurrentFetches_DataRaceValidation is a special benchmark
// designed to be run with -race flag to validate SC-006 zero data race requirement
//
// Usage:
//
//	go test -tags=integration -bench=BenchmarkConcurrentFetches_DataRaceValidation -race ./tests/integration/
//
// This benchmark specifically tests the scenario described in SC-006:
// - 10,000 concurrent fetch requests
// - Same environment variable accessed by all goroutines
// - Must have zero data races
func BenchmarkConcurrentFetches_DataRaceValidation(b *testing.B) {
	client, cleanup := startTestServer(b)
	defer cleanup()

	ctx := context.Background()

	// Set up test environment variable
	testKey := fmt.Sprintf("RACE_TEST_VAR_%d", time.Now().Unix())
	testValue := "race_test_value"
	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "benchmark-env",
		Config: configStruct,
	})
	if err != nil {
		b.Fatalf("init failed: %v", err)
	}

	// Pre-warm cache
	_, err = client.Fetch(ctx, &pb.FetchRequest{Path: []string{testKey}})
	if err != nil {
		b.Fatalf("pre-warm fetch failed: %v", err)
	}

	// SC-006 requires 10,000 concurrent requests
	const targetConcurrency = 10000

	b.ReportAllocs()
	b.ResetTimer()

	var wg sync.WaitGroup
	errChan := make(chan error, targetConcurrency)
	successCount := 0
	var mu sync.Mutex

	wg.Add(targetConcurrency)

	// Launch 10,000 concurrent fetch requests
	for j := 0; j < targetConcurrency; j++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Each goroutine performs multiple operations to increase
			// the likelihood of detecting race conditions
			for k := 0; k < 10; k++ {
				resp, err := client.Fetch(ctx, &pb.FetchRequest{
					Path: []string{testKey},
				})

				if err != nil {
					errChan <- fmt.Errorf("goroutine %d iteration %d failed: %w", goroutineID, k, err)
					return
				}

				// Verify response consistency (accessing shared data)
				if resp.Value == nil || resp.Value.Fields == nil {
					errChan <- fmt.Errorf("goroutine %d iteration %d: nil response", goroutineID, k)
					return
				}

				value := resp.Value.Fields["value"].GetStringValue()
				if value != testValue {
					errChan <- fmt.Errorf("goroutine %d iteration %d: inconsistent value %q", goroutineID, k, value)
					return
				}
			}

			// Thread-safe counter increment
			mu.Lock()
			successCount++
			mu.Unlock()
		}(j)
	}

	wg.Wait()
	close(errChan)
	b.StopTimer()

	// Check for errors
	errorCount := 0
	for err := range errChan {
		errorCount++
		if errorCount <= 10 {
			b.Errorf("Error: %v", err)
		}
	}

	if errorCount > 0 {
		b.Fatalf("SC-006 FAILED: %d errors detected in %d concurrent requests", errorCount, targetConcurrency)
	}

	if successCount != targetConcurrency {
		b.Fatalf("SC-006 FAILED: expected %d successful goroutines, got %d", targetConcurrency, successCount)
	}

	// Report results
	totalOps := targetConcurrency * 10 // 10 iterations per goroutine
	b.ReportMetric(float64(totalOps), "total_fetch_ops")
	b.ReportMetric(float64(targetConcurrency), "concurrent_goroutines")

	b.Logf("SC-006 VALIDATION PASSED: %d concurrent fetch requests completed successfully", targetConcurrency)
	b.Logf("Run with -race flag to validate zero data races requirement")
	b.Logf("Command: go test -tags=integration -bench=BenchmarkConcurrentFetches_DataRaceValidation -race ./tests/integration/")
}
