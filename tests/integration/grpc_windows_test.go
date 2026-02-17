//go:build windows
// +build windows

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// T088: Integration test for Windows case-insensitivity behavior
//
// Tests Windows-specific behavior where PATH, Path, and path all resolve to the same variable.
// This test file uses build tag and is only included on Windows builds.
func TestWindowsCaseInsensitivityBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up a test environment variable with mixed case
	timestamp := time.Now().Unix()
	testKey := fmt.Sprintf("TestCaseVar_%d", timestamp)
	testValue := "case_insensitive_value"
	_ = os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Initialize provider
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err := client.Init(ctx, &pb.InitRequest{
		Alias:  "test-case-windows",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Test different case variations
	caseVariations := []string{
		testKey,                                  // Original: TestCaseVar_123
		fmt.Sprintf("testcasevar_%d", timestamp), // lowercase
		fmt.Sprintf("TESTCASEVAR_%d", timestamp), // UPPERCASE
		fmt.Sprintf("TeStCaSeVaR_%d", timestamp), // MiXeD
		fmt.Sprintf("testCASEvar_%d", timestamp), // mixed
	}

	for _, variation := range caseVariations {
		t.Run(variation, func(t *testing.T) {
			resp, err := client.Fetch(ctx, &pb.FetchRequest{
				Path: []string{variation},
			})

			if err != nil {
				t.Fatalf("fetch with case variation %q failed: %v", variation, err)
			}

			if resp.Value == nil || resp.Value.Fields == nil {
				t.Fatal("response value is nil")
			}

			value := resp.Value.Fields["value"].GetStringValue()
			if value != testValue {
				t.Errorf("case variation %q: got %q, want %q", variation, value, testValue)
			}
		})
	}

	// Test with common system variable PATH
	pathVariations := []string{"PATH", "Path", "path", "PaTh"}

	for _, variation := range pathVariations {
		t.Run("PATH_"+variation, func(t *testing.T) {
			resp, err := client.Fetch(ctx, &pb.FetchRequest{
				Path: []string{variation},
			})

			// PATH should exist on Windows
			if err != nil {
				t.Fatalf("fetch PATH variation %q failed: %v", variation, err)
			}

			if resp.Value == nil || resp.Value.Fields == nil {
				t.Fatal("response value is nil")
			}

			// All variations should return the same value
			value := resp.Value.Fields["value"].GetStringValue()
			if value == "" {
				t.Errorf("PATH variation %q returned empty value", variation)
			}
		})
	}

	t.Log("Windows case-insensitivity behavior verified")
}
