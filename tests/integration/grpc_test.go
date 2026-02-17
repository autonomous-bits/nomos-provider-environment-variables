//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/logger"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/provider"
	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// startTestServer starts a test gRPC server and returns a client and cleanup function.
// Uses testing.TB interface to support both *testing.T and *testing.B.
func startTestServer(tb testing.TB) (pb.ProviderServiceClient, func()) {
	tb.Helper()

	log := logger.New(logger.ERROR)
	prov := provider.New(log)

	grpcServer := grpc.NewServer()
	pb.RegisterProviderServiceServer(grpcServer, prov)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("failed to listen: %v", err)
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		tb.Fatalf("failed to connect: %v", err)
	}

	client := pb.NewProviderServiceClient(conn)

	cleanup := func() {
		conn.Close()
		grpcServer.Stop()
	}

	return client, cleanup
}

// T016: Integration test for Init RPC with minimal config
func TestInitRPC(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		alias     string
		config    map[string]interface{}
		wantError bool
		errorCode codes.Code
	}{
		{
			name:      "minimal config",
			alias:     "test-env",
			config:    map[string]interface{}{},
			wantError: false,
		},
		{
			name:  "custom config",
			alias: "test-env-custom",
			config: map[string]interface{}{
				"separator":      "-",
				"case_transform": "lower",
			},
			wantError: false,
		},
		{
			name:  "invalid case_transform",
			alias: "test-env-invalid",
			config: map[string]interface{}{
				"case_transform": "invalid",
			},
			wantError: true,
			errorCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configStruct, err := structpb.NewStruct(tt.config)
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			req := &pb.InitRequest{
				Alias:  tt.alias,
				Config: configStruct,
			}

			_, err = client.Init(ctx, req)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC error, got: %v", err)
				}
				if st.Code() != tt.errorCode {
					t.Errorf("expected error code %v, got %v", tt.errorCode, st.Code())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// T017: Integration test for successful Fetch RPC (variable exists)
func TestFetchSuccess(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up environment variable
	testKey := fmt.Sprintf("TEST_VAR_%d", time.Now().Unix())
	testValue := "test_value_123"
	_ = os.Setenv(testKey, testValue)
	defer func() { _ = os.Unsetenv(testKey) }()

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Fetch the variable
	resp, err := client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{testKey},
	})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if resp.Value == nil {
		t.Fatal("expected value, got nil")
	}

	// Verify the value
	fields := resp.Value.AsMap()
	if val, ok := fields["value"].(string); !ok || val != testValue {
		t.Errorf("expected value %q, got %v", testValue, fields["value"])
	}
}

// T018: Integration test for failed Fetch RPC (variable not found)
func TestFetchNotFound(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Fetch non-existent variable
	nonExistentKey := fmt.Sprintf("NONEXISTENT_VAR_%d", time.Now().Unix())
	_, err = client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{nonExistentKey},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC error, got: %v", err)
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound error, got %v", st.Code())
	}
}

// T019: Integration test for Info RPC returning type and version
func TestInfoRPC(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Info(ctx, &pb.InfoRequest{})
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}

	if resp.Type != "environment-variables" {
		t.Errorf("expected type 'environment-variables', got %q", resp.Type)
	}

	if resp.Version == "" {
		t.Error("expected version to be set")
	}

	// After init, alias should be set
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err = client.Init(ctx, &pb.InitRequest{
		Alias:  "test-alias",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	resp, err = client.Info(ctx, &pb.InfoRequest{})
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}

	if resp.Alias != "test-alias" {
		t.Errorf("expected alias 'test-alias', got %q", resp.Alias)
	}
}

// T037: Integration test for multi-segment path resolution
func TestMultiSegmentPathResolution(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up test environment variables
	testVars := map[string]string{
		"DATABASE_HOST":   "localhost",
		"DATABASE_PORT":   "5432",
		"APP_API_TIMEOUT": "30",
		"database-host":   "testhost",
		"MyApp_Key":       "secret",
	}
	for k, v := range testVars {
		_ = os.Setenv(k, v)
		defer func(key string) { _ = os.Unsetenv(key) }(k)
	}

	tests := []struct {
		name      string
		config    map[string]interface{}
		path      []string
		wantValue string
		wantError bool
		errorCode codes.Code
	}{
		{
			name: "uppercase underscore - two segments",
			config: map[string]interface{}{
				"separator":              "_",
				"case_transform":         "upper",
				"enable_type_conversion": false,
			},
			path:      []string{"database", "host"},
			wantValue: "localhost",
		},
		{
			name: "uppercase underscore - three segments",
			config: map[string]interface{}{
				"separator":              "_",
				"case_transform":         "upper",
				"enable_type_conversion": false,
			},
			path:      []string{"app", "api", "timeout"},
			wantValue: "30",
		},
		{
			name: "lowercase dash",
			config: map[string]interface{}{
				"separator":              "-",
				"case_transform":         "lower",
				"enable_type_conversion": false,
			},
			path:      []string{"Database", "Host"},
			wantValue: "testhost",
		},
		{
			name: "preserve case underscore",
			config: map[string]interface{}{
				"separator":              "_",
				"case_transform":         "preserve",
				"enable_type_conversion": false,
			},
			path:      []string{"MyApp", "Key"},
			wantValue: "secret",
		},
		{
			name: "multi-segment not found",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
			},
			path:      []string{"nonexistent", "variable"},
			wantError: true,
			errorCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize provider with config
			configStruct, err := structpb.NewStruct(tt.config)
			if err != nil {
				t.Fatalf("failed to create config: %v", err)
			}

			_, err = client.Init(ctx, &pb.InitRequest{
				Alias:  "test-env",
				Config: configStruct,
			})
			if err != nil {
				t.Fatalf("init failed: %v", err)
			}

			// Fetch value
			resp, err := client.Fetch(ctx, &pb.FetchRequest{
				Path: tt.path,
			})

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC error, got: %v", err)
				}
				if st.Code() != tt.errorCode {
					t.Errorf("expected error code %v, got %v", tt.errorCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}

			// Extract value from response
			if resp.Value == nil || resp.Value.Fields == nil {
				t.Fatal("response value is nil")
			}

			valueField, ok := resp.Value.Fields["value"]
			if !ok {
				t.Fatal("response missing 'value' field")
			}

			strVal, ok := valueField.Kind.(*structpb.Value_StringValue)
			if !ok {
				t.Fatalf("value is not a string, got %T", valueField.Kind)
			}

			if strVal.StringValue != tt.wantValue {
				t.Errorf("got value %q, want %q", strVal.StringValue, tt.wantValue)
			}
		})
	}
}

// T048: Integration test for prefix + prepend mode
//
// Tests end-to-end workflow where:
// 1. Provider is initialized with prefix and prepend mode
// 2. Environment variables with the prefix are set
// 3. Fetch requests use paths WITHOUT the prefix
// 4. Provider automatically prepends the prefix to the transformed variable name
// 5. Only variables with the prefix are accessible
func TestPrefixPrependModeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up test environment variables
	timestamp := time.Now().Unix()
	testVars := map[string]string{
		fmt.Sprintf("MYAPP_DATABASE_HOST_%d", timestamp): "localhost",
		fmt.Sprintf("MYAPP_DATABASE_PORT_%d", timestamp): "5432",
		fmt.Sprintf("MYAPP_API_TIMEOUT_%d", timestamp):   "30",
		fmt.Sprintf("OTHERAPP_KEY_%d", timestamp):        "othervalue",
		fmt.Sprintf("SYSTEM_VAR_%d", timestamp):          "systemvalue",
	}
	for k, v := range testVars {
		_ = os.Setenv(k, v)
		defer func(key string) { _ = os.Unsetenv(key) }(k)
	}

	prefix := "MYAPP_"
	timestampStr := fmt.Sprintf("%d", timestamp)

	tests := []struct {
		name      string
		config    map[string]interface{}
		path      []string
		wantValue string
		wantError bool
		errorCode codes.Code
	}{
		{
			name: "prepend mode - fetch with prefix prepended",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "prepend",
			},
			path:      []string{"database", "host", timestampStr},
			wantValue: "localhost",
			wantError: false,
		},
		{
			name: "prepend mode - multi-segment path",
			config: map[string]interface{}{
				"separator":              "_",
				"case_transform":         "upper",
				"prefix":                 prefix,
				"prefix_mode":            "prepend",
				"enable_type_conversion": false,
			},
			path:      []string{"database", "port", timestampStr},
			wantValue: "5432",
			wantError: false,
		},
		{
			name: "prepend mode - three segment path",
			config: map[string]interface{}{
				"separator":              "_",
				"case_transform":         "upper",
				"prefix":                 prefix,
				"prefix_mode":            "prepend",
				"enable_type_conversion": false,
			},
			path:      []string{"api", "timeout", timestampStr},
			wantValue: "30",
			wantError: false,
		},
		{
			name: "prepend mode - variable without prefix not found",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "prepend",
			},
			path:      []string{"system", "var", timestampStr},
			wantError: true,
			errorCode: codes.NotFound,
		},
		{
			name: "prepend mode - different prefix not found",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "prepend",
			},
			path:      []string{"otherapp", "key", timestampStr},
			wantError: true,
			errorCode: codes.NotFound,
		},
		{
			name: "prepend mode - nonexistent variable",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "prepend",
			},
			path:      []string{"nonexistent", "variable"},
			wantError: true,
			errorCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize provider with config
			configStruct, err := structpb.NewStruct(tt.config)
			if err != nil {
				t.Fatalf("failed to create config: %v", err)
			}

			_, err = client.Init(ctx, &pb.InitRequest{
				Alias:  "test-env",
				Config: configStruct,
			})
			if err != nil {
				t.Fatalf("init failed: %v", err)
			}

			// Fetch value
			resp, err := client.Fetch(ctx, &pb.FetchRequest{
				Path: tt.path,
			})

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC error, got: %v", err)
				}
				if st.Code() != tt.errorCode {
					t.Errorf("expected error code %v, got %v", tt.errorCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}

			// Extract value from response
			if resp.Value == nil || resp.Value.Fields == nil {
				t.Fatal("response value is nil")
			}

			valueField, ok := resp.Value.Fields["value"]
			if !ok {
				t.Fatal("response missing 'value' field")
			}

			strVal, ok := valueField.Kind.(*structpb.Value_StringValue)
			if !ok {
				t.Fatalf("value is not a string, got %T", valueField.Kind)
			}

			if strVal.StringValue != tt.wantValue {
				t.Errorf("got value %q, want %q", strVal.StringValue, tt.wantValue)
			}
		})
	}
}

// T049: Integration test for prefix + filter_only mode
//
// Tests end-to-end workflow where:
// 1. Provider is initialized with prefix and filter_only mode
// 2. Environment variables with the prefix are set
// 3. Fetch requests must include the full prefix in the path
// 4. Provider does NOT automatically prepend the prefix
// 5. Only variables with the prefix are accessible
func TestPrefixFilterOnlyModeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up test environment variables
	timestamp := time.Now().Unix()
	testVars := map[string]string{
		fmt.Sprintf("MYAPP_DATABASE_HOST_%d", timestamp): "localhost",
		fmt.Sprintf("MYAPP_DATABASE_PORT_%d", timestamp): "5432",
		fmt.Sprintf("MYAPP_API_KEY_%d", timestamp):       "secret123",
		fmt.Sprintf("OTHERAPP_KEY_%d", timestamp):        "othervalue",
		fmt.Sprintf("SYSTEM_VAR_%d", timestamp):          "systemvalue",
	}
	for k, v := range testVars {
		_ = os.Setenv(k, v)
		defer func(key string) { _ = os.Unsetenv(key) }(k)
	}

	prefix := "MYAPP_"
	timestampStr := fmt.Sprintf("%d", timestamp)

	tests := []struct {
		name      string
		config    map[string]interface{}
		path      []string
		wantValue string
		wantError bool
		errorCode codes.Code
	}{
		{
			name: "filter_only mode - direct access with full prefix in path",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "filter_only",
			},
			path:      []string{"myapp", "database", "host", timestampStr},
			wantValue: "localhost",
			wantError: false,
		},
		{
			name: "filter_only mode - multi-segment with prefix",
			config: map[string]interface{}{
				"separator":              "_",
				"case_transform":         "upper",
				"prefix":                 prefix,
				"prefix_mode":            "filter_only",
				"enable_type_conversion": false,
			},
			path:      []string{"myapp", "database", "port", timestampStr},
			wantValue: "5432",
			wantError: false,
		},
		{
			name: "filter_only mode - api key with prefix",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "filter_only",
			},
			path:      []string{"myapp", "api", "key", timestampStr},
			wantValue: "secret123",
			wantError: false,
		},
		{
			name: "filter_only mode - path without prefix not found",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "filter_only",
			},
			path:      []string{"database", "host", timestampStr},
			wantError: true,
			errorCode: codes.NotFound,
		},
		{
			name: "filter_only mode - variable with different prefix not found",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "filter_only",
			},
			path:      []string{"otherapp", "key", timestampStr},
			wantError: true,
			errorCode: codes.NotFound,
		},
		{
			name: "filter_only mode - system variable not found",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "filter_only",
			},
			path:      []string{"system", "var", timestampStr},
			wantError: true,
			errorCode: codes.NotFound,
		},
		{
			name: "filter_only mode - nonexistent variable with prefix",
			config: map[string]interface{}{
				"separator":      "_",
				"case_transform": "upper",
				"prefix":         prefix,
				"prefix_mode":    "filter_only",
			},
			path:      []string{"myapp", "nonexistent", "variable"},
			wantError: true,
			errorCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize provider with config
			configStruct, err := structpb.NewStruct(tt.config)
			if err != nil {
				t.Fatalf("failed to create config: %v", err)
			}

			_, err = client.Init(ctx, &pb.InitRequest{
				Alias:  "test-env",
				Config: configStruct,
			})
			if err != nil {
				t.Fatalf("init failed: %v", err)
			}

			// Fetch value
			resp, err := client.Fetch(ctx, &pb.FetchRequest{
				Path: tt.path,
			})

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC error, got: %v", err)
				}
				if st.Code() != tt.errorCode {
					t.Errorf("expected error code %v, got %v", tt.errorCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}

			// Extract value from response
			if resp.Value == nil || resp.Value.Fields == nil {
				t.Fatal("response value is nil")
			}

			valueField, ok := resp.Value.Fields["value"]
			if !ok {
				t.Fatal("response missing 'value' field")
			}

			strVal, ok := valueField.Kind.(*structpb.Value_StringValue)
			if !ok {
				t.Fatalf("value is not a string, got %T", valueField.Kind)
			}

			if strVal.StringValue != tt.wantValue {
				t.Errorf("got value %q, want %q", strVal.StringValue, tt.wantValue)
			}
		})
	}
}

// Test prefix mode switching between prepend and filter_only
func TestPrefixModeSwitching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up test environment variable
	timestamp := time.Now().Unix()
	testKey := fmt.Sprintf("MYAPP_TEST_KEY_%d", timestamp)
	testValue := "test_value"
	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	prefix := "MYAPP_"
	timestampStr := fmt.Sprintf("%d", timestamp)

	// Test 1: Initialize with prepend mode
	configPrepend, _ := structpb.NewStruct(map[string]interface{}{
		"separator":      "_",
		"case_transform": "upper",
		"prefix":         prefix,
		"prefix_mode":    "prepend",
	})

	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env-prepend",
		Config: configPrepend,
	})
	if err != nil {
		t.Fatalf("init with prepend mode failed: %v", err)
	}

	// Fetch without prefix in path (prepend mode)
	resp, err := client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{"test", "key", timestampStr},
	})
	if err != nil {
		t.Fatalf("fetch in prepend mode failed: %v", err)
	}
	if resp.Value.Fields["value"].GetStringValue() != testValue {
		t.Errorf("prepend mode: got %q, want %q", resp.Value.Fields["value"].GetStringValue(), testValue)
	}

	// Test 2: Re-initialize with filter_only mode
	configFilterOnly, _ := structpb.NewStruct(map[string]interface{}{
		"separator":      "_",
		"case_transform": "upper",
		"prefix":         prefix,
		"prefix_mode":    "filter_only",
	})

	_, err = client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env-filter",
		Config: configFilterOnly,
	})
	if err != nil {
		t.Fatalf("init with filter_only mode failed: %v", err)
	}

	// Fetch with full prefix in path (filter_only mode)
	resp, err = client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{"myapp", "test", "key", timestampStr},
	})
	if err != nil {
		t.Fatalf("fetch in filter_only mode failed: %v", err)
	}
	if resp.Value.Fields["value"].GetStringValue() != testValue {
		t.Errorf("filter_only mode: got %q, want %q", resp.Value.Fields["value"].GetStringValue(), testValue)
	}
}

// T084: Integration test for concurrent Fetch calls (thread safety)
//
// Tests that the provider can handle 100+ concurrent Fetch calls without data races
// and that all calls return consistent values.
func TestConcurrentFetchThreadSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set up test environment variables
	timestamp := time.Now().Unix()
	testVars := map[string]string{
		fmt.Sprintf("CONCURRENT_VAR1_%d", timestamp): "value1",
		fmt.Sprintf("CONCURRENT_VAR2_%d", timestamp): "value2",
		fmt.Sprintf("CONCURRENT_VAR3_%d", timestamp): "value3",
	}

	for k, v := range testVars {
		_ = os.Setenv(k, v)
		defer func(key string) { _ = os.Unsetenv(key) }(k)
	}

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-concurrent",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Test concurrent access to same variable
	const numGoroutines = 100
	const requestsPerGoroutine = 10
	testKey := fmt.Sprintf("CONCURRENT_VAR1_%d", timestamp)
	expectedValue := "value1"

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*requestsPerGoroutine)
	results := make(chan string, numGoroutines*requestsPerGoroutine)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				resp, err := client.Fetch(ctx, &pb.FetchRequest{
					Path: []string{testKey},
				})

				if err != nil {
					errChan <- fmt.Errorf("worker %d request %d failed: %w", workerID, j, err)
					continue
				}

				if resp.Value == nil || resp.Value.Fields == nil {
					errChan <- fmt.Errorf("worker %d request %d: nil response", workerID, j)
					continue
				}

				value := resp.Value.Fields["value"].GetStringValue()
				results <- value
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	close(results)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("encountered %d errors during concurrent fetches:", len(errors))
		for _, err := range errors {
			t.Errorf("  - %v", err)
		}
	}

	// Verify all results are consistent
	inconsistentCount := 0
	totalResults := 0
	for value := range results {
		totalResults++
		if value != expectedValue {
			inconsistentCount++
			if inconsistentCount <= 5 {
				t.Errorf("inconsistent value: got %q, want %q", value, expectedValue)
			}
		}
	}

	if inconsistentCount > 0 {
		t.Errorf("%d out of %d results were inconsistent", inconsistentCount, totalResults)
	}

	expectedTotalResults := numGoroutines * requestsPerGoroutine
	if totalResults != expectedTotalResults {
		t.Errorf("expected %d results, got %d", expectedTotalResults, totalResults)
	}

	t.Logf("Successfully processed %d concurrent Fetch requests", totalResults)
}

// T075: Integration test for Init failure with missing required variables
//
// Tests end-to-end workflow where:
// 1. Provider is initialized with required_variables configuration
// 2. One or more required variables are missing from environment
// 3. Init RPC fails with InvalidArgument status code
// 4. Error message includes list of missing variable names
func TestInitFailureWithMissingRequiredVariables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create unique variable names that definitely don't exist
	timestamp := time.Now().Unix()
	missingVar1 := fmt.Sprintf("REQUIRED_MISSING_VAR1_%d", timestamp)
	missingVar2 := fmt.Sprintf("REQUIRED_MISSING_VAR2_%d", timestamp)
	existingVar := fmt.Sprintf("REQUIRED_EXISTING_VAR_%d", timestamp)

	// Set one existing variable
	_ = os.Setenv(existingVar, "test_value")
	defer os.Unsetenv(existingVar)

	tests := []struct {
		name              string
		requiredVariables []interface{}
		wantErrorCode     codes.Code
		checkMissing      []string
	}{
		{
			name:              "single missing required variable",
			requiredVariables: []interface{}{missingVar1},
			wantErrorCode:     codes.InvalidArgument,
			checkMissing:      []string{missingVar1},
		},
		{
			name:              "multiple missing required variables",
			requiredVariables: []interface{}{missingVar1, missingVar2},
			wantErrorCode:     codes.InvalidArgument,
			checkMissing:      []string{missingVar1, missingVar2},
		},
		{
			name:              "mix of present and missing required variables",
			requiredVariables: []interface{}{existingVar, missingVar1, missingVar2},
			wantErrorCode:     codes.InvalidArgument,
			checkMissing:      []string{missingVar1, missingVar2},
		},
		{
			name:              "all required variables present - no error",
			requiredVariables: []interface{}{existingVar, "PATH"},
			wantErrorCode:     codes.OK,
			checkMissing:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"required_variables": tt.requiredVariables,
			}

			configStruct, err := structpb.NewStruct(config)
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			req := &pb.InitRequest{
				Alias:  "test-required-vars",
				Config: configStruct,
			}

			_, err = client.Init(ctx, req)

			if tt.wantErrorCode == codes.OK {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}

			// Expect error
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}

			if st.Code() != tt.wantErrorCode {
				t.Errorf("expected error code %v, got %v", tt.wantErrorCode, st.Code())
			}

			// Verify error message contains all missing variables
			errMsg := st.Message()
			if errMsg == "" {
				t.Error("expected error message, got empty string")
			}

			// Check that error message contains "required environment variables missing"
			if !contains(errMsg, "required environment variables missing") {
				t.Errorf("expected error message to contain 'required environment variables missing', got: %q", errMsg)
			}

			// Verify all missing variables are listed
			for _, missingVar := range tt.checkMissing {
				if !contains(errMsg, missingVar) {
					t.Errorf("expected error message to contain missing variable %q, got: %q", missingVar, errMsg)
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// T062: Integration test for type-converted values in Fetch response
//
// Tests end-to-end workflow where:
// 1. Environment variables with different types are set (numeric, boolean, JSON, string)
// 2. Provider fetches these variables
// 3. Response contains properly type-converted values (not just strings)
// 4. Validates conversion precedence: JSON > Number > Boolean > String
func TestTypeConversionInFetchResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up test environment variables with different types
	timestamp := time.Now().Unix()
	testVars := map[string]struct {
		value        string
		expectedType string
		validate     func(t *testing.T, field *structpb.Value)
	}{
		fmt.Sprintf("TEST_INTEGER_%d", timestamp): {
			value:        "42",
			expectedType: "number",
			validate: func(t *testing.T, field *structpb.Value) {
				numVal := field.GetNumberValue()
				if numVal != 42 {
					t.Errorf("expected number 42, got %v", numVal)
				}
			},
		},
		fmt.Sprintf("TEST_FLOAT_%d", timestamp): {
			value:        "3.14",
			expectedType: "number",
			validate: func(t *testing.T, field *structpb.Value) {
				numVal := field.GetNumberValue()
				if numVal != 3.14 {
					t.Errorf("expected number 3.14, got %v", numVal)
				}
			},
		},
		fmt.Sprintf("TEST_NEGATIVE_%d", timestamp): {
			value:        "-99",
			expectedType: "number",
			validate: func(t *testing.T, field *structpb.Value) {
				numVal := field.GetNumberValue()
				if numVal != -99 {
					t.Errorf("expected number -99, got %v", numVal)
				}
			},
		},
		fmt.Sprintf("TEST_BOOLEAN_TRUE_%d", timestamp): {
			value:        "true",
			expectedType: "boolean",
			validate: func(t *testing.T, field *structpb.Value) {
				boolVal := field.GetBoolValue()
				if boolVal != true {
					t.Errorf("expected boolean true, got %v", boolVal)
				}
			},
		},
		fmt.Sprintf("TEST_BOOLEAN_FALSE_%d", timestamp): {
			value:        "false",
			expectedType: "boolean",
			validate: func(t *testing.T, field *structpb.Value) {
				boolVal := field.GetBoolValue()
				if boolVal != false {
					t.Errorf("expected boolean false, got %v", boolVal)
				}
			},
		},
		fmt.Sprintf("TEST_BOOLEAN_YES_%d", timestamp): {
			value:        "yes",
			expectedType: "boolean",
			validate: func(t *testing.T, field *structpb.Value) {
				boolVal := field.GetBoolValue()
				if boolVal != true {
					t.Errorf("expected boolean true, got %v", boolVal)
				}
			},
		},
		fmt.Sprintf("TEST_JSON_OBJECT_%d", timestamp): {
			value:        `{"host":"localhost","port":5432}`,
			expectedType: "object",
			validate: func(t *testing.T, field *structpb.Value) {
				structVal := field.GetStructValue()
				if structVal == nil {
					t.Fatal("expected struct value, got nil")
				}
				host := structVal.Fields["host"].GetStringValue()
				if host != "localhost" {
					t.Errorf("expected host='localhost', got %q", host)
				}
				port := structVal.Fields["port"].GetNumberValue()
				if port != 5432 {
					t.Errorf("expected port=5432, got %v", port)
				}
			},
		},
		fmt.Sprintf("TEST_JSON_ARRAY_%d", timestamp): {
			value:        `[1,2,3,4,5]`,
			expectedType: "array",
			validate: func(t *testing.T, field *structpb.Value) {
				listVal := field.GetListValue()
				if listVal == nil {
					t.Fatal("expected list value, got nil")
				}
				if len(listVal.Values) != 5 {
					t.Errorf("expected array length 5, got %d", len(listVal.Values))
				}
			},
		},
		fmt.Sprintf("TEST_PLAIN_STRING_%d", timestamp): {
			value:        "hello world",
			expectedType: "string",
			validate: func(t *testing.T, field *structpb.Value) {
				strVal := field.GetStringValue()
				if strVal != "hello world" {
					t.Errorf("expected string 'hello world', got %q", strVal)
				}
			},
		},
		fmt.Sprintf("TEST_EMPTY_STRING_%d", timestamp): {
			value:        "",
			expectedType: "string",
			validate: func(t *testing.T, field *structpb.Value) {
				strVal := field.GetStringValue()
				if strVal != "" {
					t.Errorf("expected empty string, got %q", strVal)
				}
			},
		},
		fmt.Sprintf("TEST_PRECEDENCE_NUMBER_%d", timestamp): {
			value:        "1",
			expectedType: "number",
			validate: func(t *testing.T, field *structpb.Value) {
				// Should be number, not boolean
				numVal := field.GetNumberValue()
				if numVal != 1 {
					t.Errorf("expected number 1, got %v", numVal)
				}
			},
		},
	}

	// Set environment variables
	for key, tc := range testVars {
		_ = os.Setenv(key, tc.value)
		defer func(k string) { _ = os.Unsetenv(k) }(key)
	}

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Test each variable
	for key, tc := range testVars {
		t.Run(key, func(t *testing.T) {
			resp, err := client.Fetch(ctx, &pb.FetchRequest{
				Path: []string{key},
			})
			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}

			if resp.Value == nil || resp.Value.Fields == nil {
				t.Fatal("response value is nil")
			}

			valueField, ok := resp.Value.Fields["value"]
			if !ok {
				t.Fatal("response missing 'value' field")
			}

			tc.validate(t, valueField)
		})
	}
}

// T062b: Integration test for nested JSON structures in type conversion
func TestNestedJSONTypeConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up complex nested JSON environment variable
	timestamp := time.Now().Unix()
	testKey := fmt.Sprintf("TEST_NESTED_CONFIG_%d", timestamp)
	testValue := `{
		"database": {
			"host": "localhost",
			"port": 5432,
			"ssl": true,
			"replicas": ["db1.example.com", "db2.example.com"]
		},
		"cache": {
			"enabled": false,
			"ttl": 300
		},
		"features": ["feature1", "feature2", "feature3"]
	}`

	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Fetch the nested JSON
	resp, err := client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{testKey},
	})
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if resp.Value == nil || resp.Value.Fields == nil {
		t.Fatal("response value is nil")
	}

	valueField, ok := resp.Value.Fields["value"]
	if !ok {
		t.Fatal("response missing 'value' field")
	}

	// Validate it's a struct
	structVal := valueField.GetStructValue()
	if structVal == nil {
		t.Fatal("expected struct value, got nil")
	}

	// Validate nested database object
	databaseField, ok := structVal.Fields["database"]
	if !ok {
		t.Fatal("missing 'database' field")
	}
	database := databaseField.GetStructValue()
	if database == nil {
		t.Fatal("'database' is not a struct")
	}

	// Check database fields
	if host := database.Fields["host"].GetStringValue(); host != "localhost" {
		t.Errorf("expected host='localhost', got %q", host)
	}
	if port := database.Fields["port"].GetNumberValue(); port != 5432 {
		t.Errorf("expected port=5432, got %v", port)
	}
	if ssl := database.Fields["ssl"].GetBoolValue(); ssl != true {
		t.Errorf("expected ssl=true, got %v", ssl)
	}

	// Check replicas array
	replicas := database.Fields["replicas"].GetListValue()
	if replicas == nil {
		t.Fatal("expected replicas array, got nil")
	}
	if len(replicas.Values) != 2 {
		t.Errorf("expected 2 replicas, got %d", len(replicas.Values))
	}

	// Validate cache object
	cacheField, ok := structVal.Fields["cache"]
	if !ok {
		t.Fatal("missing 'cache' field")
	}
	cache := cacheField.GetStructValue()
	if cache == nil {
		t.Fatal("'cache' is not a struct")
	}
	if enabled := cache.Fields["enabled"].GetBoolValue(); enabled != false {
		t.Errorf("expected enabled=false, got %v", enabled)
	}
	if ttl := cache.Fields["ttl"].GetNumberValue(); ttl != 300 {
		t.Errorf("expected ttl=300, got %v", ttl)
	}

	// Validate features array
	featuresField, ok := structVal.Fields["features"]
	if !ok {
		t.Fatal("missing 'features' field")
	}
	features := featuresField.GetListValue()
	if features == nil {
		t.Fatal("expected features array, got nil")
	}
	if len(features.Values) != 3 {
		t.Errorf("expected 3 features, got %d", len(features.Values))
	}
}

// T062c: Integration test for type conversion error handling
func TestTypeConversionErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up environment variable with malformed JSON
	timestamp := time.Now().Unix()
	testKey := fmt.Sprintf("TEST_MALFORMED_JSON_%d", timestamp)
	testValue := `{"key":"value"` // Missing closing brace

	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Fetch should return error for malformed JSON
	_, err = client.Fetch(ctx, &pb.FetchRequest{
		Path: []string{testKey},
	})

	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC error, got: %v", err)
	}

	// Should return InvalidArgument or Internal error for malformed JSON
	if st.Code() != codes.InvalidArgument && st.Code() != codes.Internal {
		t.Errorf("expected InvalidArgument or Internal error, got %v", st.Code())
	}
}
