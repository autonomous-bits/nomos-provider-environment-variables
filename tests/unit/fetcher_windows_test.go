//go:build windows
// +build windows

package unit

import (
	"fmt"
	"os"
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/fetcher"
)

// T085: Unit test for Windows case insensitivity
//
// Tests that on Windows, PATH and path are treated as the same variable.
func TestWindowsCaseInsensitivity(t *testing.T) {
	t.Parallel()

	// On Windows, environment variables are case-insensitive
	testKey := fmt.Sprintf("TEST_WINDOWS_CASE_%d", os.Getpid())
	testKeyLower := fmt.Sprintf("test_windows_case_%d", os.Getpid())
	testValue := "test_value"

	// Clean up
	_ = os.Unsetenv(testKey)
	_ = os.Unsetenv(testKeyLower)
	defer func() {
		_ = os.Unsetenv(testKey)
		_ = os.Unsetenv(testKeyLower)
	}()

	// Set uppercase variable
	if err := os.Setenv(testKey, testValue); err != nil {
		t.Fatalf("failed to set variable: %v", err)
	}

	fetcher := fetcher.New()

	// Fetch with original case
	valueOriginal, err := fetcher.Fetch(testKey)
	if err != nil {
		t.Fatalf("failed to fetch original case: %v", err)
	}

	// Fetch with different case
	valueLower, err := fetcher.Fetch(testKeyLower)
	if err != nil {
		t.Fatalf("failed to fetch lowercase case: %v", err)
	}

	// On Windows, both should return the same value
	if valueOriginal != valueLower {
		t.Errorf("On Windows, case variations should return same value")
		t.Errorf("  %s = %q", testKey, valueOriginal)
		t.Errorf("  %s = %q", testKeyLower, valueLower)
	}

	if valueOriginal != testValue {
		t.Errorf("got value %q, want %q", valueOriginal, testValue)
	}

	// Test with PATH variable
	pathVariations := []string{"PATH", "Path", "path", "PaTh"}
	var pathValue string

	for i, variation := range pathVariations {
		value, err := fetcher.Fetch(variation)
		if err != nil {
			t.Fatalf("failed to fetch %q: %v", variation, err)
		}

		if i == 0 {
			pathValue = value
		} else {
			if value != pathValue {
				t.Errorf("PATH case variation %q returned different value", variation)
				t.Errorf("  Expected: %q", pathValue)
				t.Errorf("  Got:      %q", value)
			}
		}
	}

	t.Logf("Windows case insensitivity verified:")
	t.Logf("  %s == %s (both return %q)", testKey, testKeyLower, valueOriginal)
}
