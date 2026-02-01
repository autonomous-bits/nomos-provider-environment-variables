package fetcher

import (
	"fmt"
	"os"
	"testing"
)

// T022: Unit test for environment variable lookup (exists, not exists, empty)
func TestEnvLookup(t *testing.T) {
	tests := []struct {
		name       string
		varName    string
		varValue   string
		setupEnv   bool
		wantValue  string
		wantExists bool
	}{
		{
			name:       "variable exists",
			varName:    "TEST_VAR_EXISTS",
			varValue:   "test_value",
			setupEnv:   true,
			wantValue:  "test_value",
			wantExists: true,
		},
		{
			name:       "variable does not exist",
			varName:    "TEST_VAR_NOTEXIST",
			setupEnv:   false,
			wantValue:  "",
			wantExists: false,
		},
		{
			name:       "variable exists but empty",
			varName:    "TEST_VAR_EMPTY",
			varValue:   "",
			setupEnv:   true,
			wantValue:  "",
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean slate
			_ = os.Unsetenv(tt.varName)

			if tt.setupEnv {
				_ = os.Setenv(tt.varName, tt.varValue)
				defer func() { _ = os.Unsetenv(tt.varName) }()
			}

			value, exists := os.LookupEnv(tt.varName)

			if exists != tt.wantExists {
				t.Errorf("exists: got %v, want %v", exists, tt.wantExists)
			}

			if value != tt.wantValue {
				t.Errorf("value: got %q, want %q", value, tt.wantValue)
			}
		})
	}
}

func TestFetcherBasic(t *testing.T) {
	// Setup test environment variables
	testVars := map[string]string{
		"FETCH_TEST_VAR1": "value1",
		"FETCH_TEST_VAR2": "value2",
		"FETCH_TEST_VAR3": "",
	}

	for k, v := range testVars {
		_ = os.Setenv(k, v)
		defer func(key string) { _ = os.Unsetenv(key) }(k)
	}

	fetcher := New()

	tests := []struct {
		name      string
		varName   string
		wantValue string
		wantError bool
	}{
		{
			name:      "fetch existing variable",
			varName:   "FETCH_TEST_VAR1",
			wantValue: "value1",
			wantError: false,
		},
		{
			name:      "fetch empty variable",
			varName:   "FETCH_TEST_VAR3",
			wantValue: "",
			wantError: false,
		},
		{
			name:      "fetch non-existent variable",
			varName:   "FETCH_TEST_NONEXISTENT",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := fetcher.Fetch(tt.varName)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if value != tt.wantValue {
					t.Errorf("value: got %q, want %q", value, tt.wantValue)
				}
			}
		})
	}
}

func TestFetcherCaching(t *testing.T) {
	testVar := fmt.Sprintf("FETCH_CACHE_TEST_%d", os.Getpid())
	_ = os.Setenv(testVar, "initial_value")
	defer func() { _ = os.Unsetenv(testVar) }()

	fetcher := New()

	// First fetch
	val1, err := fetcher.Fetch(testVar)
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}

	if val1 != "initial_value" {
		t.Errorf("first fetch: got %q, want %q", val1, "initial_value")
	}

	// Change environment variable
	_ = os.Setenv(testVar, "changed_value")

	// Second fetch should return cached value
	val2, err := fetcher.Fetch(testVar)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}

	if val2 != "initial_value" {
		t.Errorf("second fetch should return cached value %q, got %q", "initial_value", val2)
	}
}
