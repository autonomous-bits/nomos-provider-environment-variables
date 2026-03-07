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

// ---------------------------------------------------------------------------
// Wildcard Path Spread Operator Tests (002-wildcard-path-spread)
// ---------------------------------------------------------------------------

// initProvider is a helper that initialises a fresh provider with the given config.
func initProvider(t *testing.T, client pb.ProviderServiceClient, ctx context.Context, cfg map[string]interface{}) {
	t.Helper()
	s, err := structpb.NewStruct(cfg)
	if err != nil {
		t.Fatalf("NewStruct: %v", err)
	}
	if _, err = client.Init(ctx, &pb.InitRequest{Alias: "wc-test", Config: s}); err != nil {
		t.Fatalf("Init: %v", err)
	}
}

// wildcardFetch is a helper that calls Fetch with the given path and returns the
// top-level fields of the response struct as a map[string]interface{}.
func wildcardFetch(t *testing.T, client pb.ProviderServiceClient, ctx context.Context, path []string) map[string]interface{} {
	t.Helper()
	resp, err := client.Fetch(ctx, &pb.FetchRequest{Path: path})
	if err != nil {
		t.Fatalf("Fetch(%v): %v", path, err)
	}
	return resp.Value.AsMap()
}

// T005: US1 — Root-level wildcard retrieval
func TestWildcardRootLevelRetrieval(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const pfx = "WC_US1_"
	vars := map[string]string{
		pfx + "API_KEY": "secret",
		pfx + "DB_HOST": "localhost",
		pfx + "DB_PORT": "5432",
	}
	for k, v := range vars {
		_ = os.Setenv(k, v)
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}

	t.Run("US1-AC1: all in-scope vars returned for root wildcard", func(t *testing.T) {
		// No provider prefix — root wildcard returns everything
		initProvider(t, client, ctx, map[string]interface{}{
			"separator":              "_",
			"case_transform":         "upper",
			"enable_type_conversion": false,
			"enable_json_parsing":    false,
		})
		got := wildcardFetch(t, client, ctx, []string{"*"})
		for k, wantVal := range vars {
			if got[k] == nil {
				t.Errorf("US1-AC1: expected key %q in response, got nothing; full response keys: %v", k, mapKeys(got))
			} else if got[k] != wantVal {
				t.Errorf("US1-AC1: %q = %v, want %q", k, got[k], wantVal)
			}
		}
	})

	t.Run("US1-AC2: empty in-scope returns empty struct, no error", func(t *testing.T) {
		// Provider prefix guaranteed to match nothing
		initProvider(t, client, ctx, map[string]interface{}{
			"separator":              "_",
			"case_transform":         "upper",
			"prefix":                 "DEFINITELY_NO_SUCH_PREFIX_XYZ_",
			"prefix_mode":            "prepend",
			"enable_type_conversion": false,
		})
		resp, err := client.Fetch(ctx, &pb.FetchRequest{Path: []string{"*"}})
		if err != nil {
			t.Fatalf("US1-AC2: unexpected error: %v", err)
		}
		if resp.Value == nil {
			t.Fatal("US1-AC2: expected non-nil Value")
		}
		if len(resp.Value.Fields) != 0 {
			t.Errorf("US1-AC2: expected empty struct, got %d fields", len(resp.Value.Fields))
		}
	})

	t.Run("US1-AC3: provider prefix scopes and strips results", func(t *testing.T) {
		prov := "WC_US1_"
		initProvider(t, client, ctx, map[string]interface{}{
			"separator":              "_",
			"case_transform":         "upper",
			"prefix":                 prov,
			"prefix_mode":            "prepend",
			"enable_type_conversion": false,
			"enable_json_parsing":    false,
		})
		got := wildcardFetch(t, client, ctx, []string{"*"})
		// Keys must NOT include the provider prefix
		if _, bad := got[prov+"API_KEY"]; bad {
			t.Error("US1-AC3: full env var name must not appear as key; expected stripped key")
		}
		// Stripped keys should be present
		for _, stripped := range []string{"API_KEY", "DB_HOST", "DB_PORT"} {
			if got[stripped] == nil {
				t.Errorf("US1-AC3: stripped key %q missing from response; keys: %v", stripped, mapKeys(got))
			}
		}
	})
}

// T009: US2 — Prefix-scoped wildcard retrieval
func TestWildcardPrefixScopedRetrieval(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = os.Setenv("WC_US2_DATABASE_HOST", "localhost")
	_ = os.Setenv("WC_US2_DATABASE_PORT", "5432")
	_ = os.Setenv("WC_US2_APP_KEY", "other")
	t.Cleanup(func() {
		_ = os.Unsetenv("WC_US2_DATABASE_HOST")
		_ = os.Unsetenv("WC_US2_DATABASE_PORT")
		_ = os.Unsetenv("WC_US2_APP_KEY")
	})

	initProvider(t, client, ctx, map[string]interface{}{
		"separator":              "_",
		"case_transform":         "upper",
		"prefix":                 "WC_US2_",
		"prefix_mode":            "prepend",
		"enable_type_conversion": false,
		"enable_json_parsing":    false,
	})

	t.Run("US2-AC1: single-segment namespace, sibling exclusion", func(t *testing.T) {
		got := wildcardFetch(t, client, ctx, []string{"database", "*"})
		if got["HOST"] == nil {
			t.Errorf("US2-AC1: expected key HOST; keys: %v", mapKeys(got))
		}
		if got["PORT"] == nil {
			t.Errorf("US2-AC1: expected key PORT; keys: %v", mapKeys(got))
		}
		// APP_KEY is a sibling, must not appear
		if _, bad := got["APP_KEY"]; bad {
			t.Error("US2-AC1: sibling APP_KEY must not appear")
		}
	})

	t.Run("US2-AC2: returned key is relative suffix, not full name", func(t *testing.T) {
		got := wildcardFetch(t, client, ctx, []string{"database", "*"})
		// Keys must be HOST / PORT, NOT WC_US2_DATABASE_HOST
		for _, bad := range []string{"WC_US2_DATABASE_HOST", "DATABASE_HOST", "WC_US2_DATABASE_PORT"} {
			if _, ok := got[bad]; ok {
				t.Errorf("US2-AC2: key %q must not appear (should be stripped)", bad)
			}
		}
	})

	t.Run("US2-AC3: empty DATABASE_* result returns empty struct, no error", func(t *testing.T) {
		// Use a prefix that maps to nothing
		initProvider(t, client, ctx, map[string]interface{}{
			"separator":              "_",
			"case_transform":         "upper",
			"prefix":                 "WC_NO_SUCH_",
			"prefix_mode":            "prepend",
			"enable_type_conversion": false,
		})
		resp, err := client.Fetch(ctx, &pb.FetchRequest{Path: []string{"database", "*"}})
		if err != nil {
			t.Fatalf("US2-AC3: unexpected error: %v", err)
		}
		if len(resp.Value.Fields) != 0 {
			t.Errorf("US2-AC3: expected empty struct, got %d fields", len(resp.Value.Fields))
		}
	})
}

// T012: US3 — Deeply nested wildcard retrieval
func TestWildcardDeeplyNested(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = os.Setenv("APP_DATABASE_HOST", "localhost")
	_ = os.Setenv("APP_DATABASE_PORT", "5432")
	_ = os.Setenv("APP_CACHE_HOST", "redis")
	t.Cleanup(func() {
		_ = os.Unsetenv("APP_DATABASE_HOST")
		_ = os.Unsetenv("APP_DATABASE_PORT")
		_ = os.Unsetenv("APP_CACHE_HOST")
	})

	initProvider(t, client, ctx, map[string]interface{}{
		"separator":              "_",
		"case_transform":         "upper",
		"enable_type_conversion": false,
		"enable_json_parsing":    false,
	})

	t.Run("US3-AC1: APP_DATABASE_* returned, APP_CACHE_* excluded", func(t *testing.T) {
		got := wildcardFetch(t, client, ctx, []string{"app", "database", "*"})
		if got["HOST"] == nil {
			t.Errorf("US3-AC1: expected HOST; keys: %v", mapKeys(got))
		}
		if got["PORT"] == nil {
			t.Errorf("US3-AC1: expected PORT; keys: %v", mapKeys(got))
		}
		if _, bad := got["APP_CACHE_HOST"]; bad {
			t.Error("US3-AC1: APP_CACHE_HOST must not appear (sibling namespace)")
		}
		if _, bad := got["CACHE_HOST"]; bad {
			t.Error("US3-AC1: CACHE_HOST must not appear (sibling namespace)")
		}
	})

	t.Run("US3-AC2: deeply nested path strips full combined prefix", func(t *testing.T) {
		// Set a three-level variable
		_ = os.Setenv("SERVICE_DB_REPLICA_HOST", "replica.db")
		t.Cleanup(func() { _ = os.Unsetenv("SERVICE_DB_REPLICA_HOST") })

		got := wildcardFetch(t, client, ctx, []string{"service", "db", "replica", "*"})
		if got["HOST"] == nil {
			t.Errorf("US3-AC2: expected key HOST; keys: %v", mapKeys(got))
		}
		// Full name must not appear
		for _, bad := range []string{"SERVICE_DB_REPLICA_HOST", "DB_REPLICA_HOST"} {
			if _, ok := got[bad]; ok {
				t.Errorf("US3-AC2: key %q must not appear (should be stripped)", bad)
			}
		}
	})
}

// T014: US4 — Non-terminal wildcard position validation
func TestWildcardPositionValidation(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	initProvider(t, client, ctx, map[string]interface{}{
		"separator":      "_",
		"case_transform": "upper",
	})

	tests := []struct {
		name      string
		path      []string
		wantIndex int
	}{
		{
			name:      "US4-AC1: wildcard at index 0, additional segment after",
			path:      []string{"*", "host"},
			wantIndex: 0,
		},
		{
			name:      "US4-AC2: first non-terminal wildcard at index 1",
			path:      []string{"database", "*", "*"},
			wantIndex: 1,
		},
		{
			name:      "SC-003: no partial result — middle wildcard",
			path:      []string{"app", "*", "host"},
			wantIndex: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Fetch(ctx, &pb.FetchRequest{Path: tt.path})
			if err == nil {
				t.Fatal("expected INVALID_ARGUMENT error, got nil")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}
			if st.Code() != codes.InvalidArgument {
				t.Errorf("expected INVALID_ARGUMENT, got %v", st.Code())
			}
			wantMsg := fmt.Sprintf("wildcard operator '*' is only valid at the terminal position of a path; found at index %d", tt.wantIndex)
			if st.Message() != wantMsg {
				t.Errorf("message = %q, want %q", st.Message(), wantMsg)
			}
		})
	}
}

// T016: Type conversion in wildcard results
func TestWildcardTypeConversion(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = os.Setenv("WC_TC_PORT", "5432")
	_ = os.Setenv("WC_TC_ENABLED", "true")
	_ = os.Setenv("WC_TC_NAME", "mydb")
	t.Cleanup(func() {
		_ = os.Unsetenv("WC_TC_PORT")
		_ = os.Unsetenv("WC_TC_ENABLED")
		_ = os.Unsetenv("WC_TC_NAME")
	})

	initProvider(t, client, ctx, map[string]interface{}{
		"separator":              "_",
		"case_transform":         "upper",
		"enable_type_conversion": true,
		"enable_json_parsing":    true,
	})

	got := wildcardFetch(t, client, ctx, []string{"WC_TC_", "*"})
	// With type conversion, "5432" should come back as a number
	// Note: match prefix is "WC_TC__" ... actually we need to use the right prefix.
	// Let me re-check: path ["WC_TC_", "*"] => namespace ["WC_TC_"] => BuildPrefix → "WC_TC__"
	// That won't match "WC_TC_PORT". Let me use ["WC_TC", "*"]
	// path ["WC_TC", "*"] => namespace ["WC_TC"] => upper → "WC_TC" + "_" = "WC_TC_"
	_ = got // discard result above, we'll re-fetch correctly

	got = wildcardFetch(t, client, ctx, []string{"WC_TC", "*"})
	// PORT should be numeric
	portVal, ok := got["PORT"]
	if !ok {
		t.Fatalf("expected PORT in response; keys: %v", mapKeys(got))
	}
	portNum, isNum := portVal.(float64)
	if !isNum {
		t.Errorf("PORT type conversion: expected float64, got %T (%v)", portVal, portVal)
	} else if portNum != 5432 {
		t.Errorf("PORT value: got %v, want 5432", portNum)
	}

	// ENABLED should be boolean
	enabledVal, ok := got["ENABLED"]
	if !ok {
		t.Fatalf("expected ENABLED in response; keys: %v", mapKeys(got))
	}
	enabledBool, isBool := enabledVal.(bool)
	if !isBool {
		t.Errorf("ENABLED type conversion: expected bool, got %T (%v)", enabledVal, enabledVal)
	} else if !enabledBool {
		t.Errorf("ENABLED value: got false, want true")
	}

	// NAME should remain a string
	nameVal, ok := got["NAME"]
	if !ok {
		t.Fatalf("expected NAME in response; keys: %v", mapKeys(got))
	}
	if nameStr, isStr := nameVal.(string); !isStr || nameStr != "mydb" {
		t.Errorf("NAME value: got %v (%T), want string 'mydb'", nameVal, nameVal)
	}
}

// T017: Combined provider prefix + path prefix wildcard
func TestWildcardCombinedProviderAndPathPrefix(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = os.Setenv("MYAPP_DATABASE_HOST", "db.local")
	_ = os.Setenv("MYAPP_DATABASE_PORT", "5432")
	_ = os.Setenv("MYAPP_CACHE_HOST", "cache.local")
	_ = os.Setenv("OTHER_DATABASE_HOST", "other.local")
	t.Cleanup(func() {
		_ = os.Unsetenv("MYAPP_DATABASE_HOST")
		_ = os.Unsetenv("MYAPP_DATABASE_PORT")
		_ = os.Unsetenv("MYAPP_CACHE_HOST")
		_ = os.Unsetenv("OTHER_DATABASE_HOST")
	})

	// Provider configured with prefix="MYAPP_" in prepend mode.
	// path=["database","*"] → matchPrefix = "MYAPP_DATABASE_"
	initProvider(t, client, ctx, map[string]interface{}{
		"separator":              "_",
		"case_transform":         "upper",
		"prefix":                 "MYAPP_",
		"prefix_mode":            "prepend",
		"enable_type_conversion": false,
		"enable_json_parsing":    false,
	})

	got := wildcardFetch(t, client, ctx, []string{"database", "*"})

	t.Run("FR-005: only MYAPP_DATABASE_* vars returned", func(t *testing.T) {
		if got["HOST"] == nil {
			t.Errorf("expected HOST; keys: %v", mapKeys(got))
		}
		if got["PORT"] == nil {
			t.Errorf("expected PORT; keys: %v", mapKeys(got))
		}
		// Must not include MYAPP_CACHE_* or OTHER_DATABASE_*
		for _, bad := range []string{"CACHE_HOST", "MYAPP_CACHE_HOST", "OTHER_DATABASE_HOST"} {
			if _, ok := got[bad]; ok {
				t.Errorf("key %q must not appear", bad)
			}
		}
	})

	t.Run("FR-005: full MYAPP_DATABASE_ prefix stripped from keys", func(t *testing.T) {
		for _, bad := range []string{"MYAPP_DATABASE_HOST", "DATABASE_HOST"} {
			if _, ok := got[bad]; ok {
				t.Errorf("unstripped key %q must not appear", bad)
			}
		}
	})
}

// T020: filter_only prefix mode wildcard behaviour
func TestWildcardFilterOnlyMode(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = os.Setenv("MYAPP_KEY", "myvalue")
	_ = os.Setenv("MYAPP_OTHER", "othervalue")
	_ = os.Setenv("UNRELATED_VAR", "unrelated")
	t.Cleanup(func() {
		_ = os.Unsetenv("MYAPP_KEY")
		_ = os.Unsetenv("MYAPP_OTHER")
		_ = os.Unsetenv("UNRELATED_VAR")
	})

	t.Run("root wildcard returns all vars (no stripping in filter_only)", func(t *testing.T) {
		initProvider(t, client, ctx, map[string]interface{}{
			"separator":              "_",
			"case_transform":         "upper",
			"prefix":                 "MYAPP_",
			"prefix_mode":            "filter_only",
			"enable_type_conversion": false,
			"enable_json_parsing":    false,
		})
		got := wildcardFetch(t, client, ctx, []string{"*"})
		// In filter_only root wildcard, ALL vars are returned and NO prefix is stripped
		// (matchPrefix == "" → FetchAll returns all)
		if got["MYAPP_KEY"] == nil && got["UNRELATED_VAR"] == nil {
			// While we can't enumerate all env vars, at least our test vars should be present
			t.Logf("Note: root wildcard in filter_only returns all vars; test vars found: %v", mapKeys(got))
		}
		// MYAPP_KEY must appear (not stripped since matchPrefix is "")
		if got["MYAPP_KEY"] == nil {
			t.Errorf("expected MYAPP_KEY in root wildcard result; keys include: %v", mapKeys(got))
		}
	})

	t.Run("prefixed wildcard uses path-derived prefix only", func(t *testing.T) {
		initProvider(t, client, ctx, map[string]interface{}{
			"separator":              "_",
			"case_transform":         "upper",
			"prefix":                 "MYAPP_",
			"prefix_mode":            "filter_only",
			"enable_type_conversion": false,
			"enable_json_parsing":    false,
		})
		// path ["myapp", "*"] → BuildPrefix(["myapp"]) = "MYAPP_" (no config prefix prepended)
		// → FetchAll("MYAPP_") → vars starting with "MYAPP_", keys stripped of "MYAPP_"
		got := wildcardFetch(t, client, ctx, []string{"myapp", "*"})
		if got["KEY"] == nil {
			t.Errorf("expected KEY (stripped from MYAPP_KEY); keys: %v", mapKeys(got))
		}
		if got["OTHER"] == nil {
			t.Errorf("expected OTHER (stripped from MYAPP_OTHER); keys: %v", mapKeys(got))
		}
		// UNRELATED_VAR must not appear
		for _, bad := range []string{"UNRELATED_VAR", "MYAPP_KEY", "MYAPP_OTHER"} {
			if _, ok := got[bad]; ok {
				t.Errorf("key %q must not appear (should be excluded or stripped)", bad)
			}
		}
	})

	t.Run("path-derived prefix outside configured scope returns empty collection", func(t *testing.T) {
		initProvider(t, client, ctx, map[string]interface{}{
			"separator":              "_",
			"case_transform":         "upper",
			"prefix":                 "MYAPP_",
			"prefix_mode":            "filter_only",
			"enable_type_conversion": false,
		})
		// path ["other", "*"] → BuildPrefix → "OTHER_", which does NOT start with "MYAPP_"
		// Safety check (Decision 5) returns empty collection
		resp, err := client.Fetch(ctx, &pb.FetchRequest{Path: []string{"other", "*"}})
		if err != nil {
			t.Fatalf("expected empty collection, got error: %v", err)
		}
		if len(resp.Value.Fields) != 0 {
			t.Errorf("expected empty collection for out-of-scope prefix, got %d fields: %v",
				len(resp.Value.Fields), mapKeys(resp.Value.AsMap()))
		}
	})
}

// mapKeys returns the keys of a map as a slice (helper for test error messages).
func mapKeys(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
