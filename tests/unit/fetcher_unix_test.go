//go:build !windows
// +build !windows

package unit

import (
	"fmt"
	"os"
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/fetcher"
)

// T085: Unit test for Unix case sensitivity
//
// Tests that on Unix systems (Linux, macOS), PATH and path are treated as different variables.
func TestUnixCaseSensitivity(t *testing.T) {
	t.Parallel()

	// On Unix, environment variables are case-sensitive
	testKey := fmt.Sprintf("TEST_UNIX_CASE_%d", os.Getpid())
	testKeyLower := fmt.Sprintf("test_unix_case_%d", os.Getpid())
	testValue := "UPPERCASE_VALUE"
	testValueLower := "lowercase_value"

	// Clean up
	_ = os.Unsetenv(testKey)
	_ = os.Unsetenv(testKeyLower)
	defer func() {
		_ = os.Unsetenv(testKey)
		_ = os.Unsetenv(testKeyLower)
	}()

	// Set both variables with different values
	if err := os.Setenv(testKey, testValue); err != nil {
		t.Fatalf("failed to set uppercase variable: %v", err)
	}

	if err := os.Setenv(testKeyLower, testValueLower); err != nil {
		t.Fatalf("failed to set lowercase variable: %v", err)
	}

	f := fetcher.New()

	// Fetch uppercase version
	valueUpper, errUpper := f.Fetch(testKey)
	if errUpper != nil {
		t.Fatalf("failed to fetch uppercase variable: %v", errUpper)
	}

	// Fetch lowercase version
	valueLower, errLower := f.Fetch(testKeyLower)
	if errLower != nil {
		t.Fatalf("failed to fetch lowercase variable: %v", errLower)
	}

	// On Unix, these should be different
	if valueUpper == valueLower {
		t.Errorf("On Unix, case-different variables should have different values")
		t.Errorf("Both returned: %q", valueUpper)
	}

	if valueUpper != testValue {
		t.Errorf("uppercase variable: got %q, want %q", valueUpper, testValue)
	}

	if valueLower != testValueLower {
		t.Errorf("lowercase variable: got %q, want %q", valueLower, testValueLower)
	}

	// Test with actual PATH variable (always present on Unix)
	pathValue, err := f.Fetch("PATH")
	if err != nil {
		t.Fatalf("failed to fetch PATH: %v", err)
	}

	if pathValue == "" {
		t.Error("PATH variable should not be empty on Unix")
	}

	// Attempt to fetch lowercase "path" (should not exist or be different)
	pathLowerValue, errPath := f.Fetch("path")

	// Either it doesn't exist, or it's different from PATH
	if errPath == nil {
		// If it exists, it should be different from PATH
		if pathLowerValue == pathValue {
			t.Error("On Unix, 'PATH' and 'path' should be different variables")
		}
	}

	t.Logf("Unix case sensitivity verified:")
	t.Logf("  %s = %q", testKey, valueUpper)
	t.Logf("  %s = %q", testKeyLower, valueLower)
}
