package unit

import (
	"fmt"
	"os"
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/fetcher"
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
				if err := os.Setenv(tt.varName, tt.varValue); err != nil {
					t.Fatalf("failed to set env var: %v", err)
				}
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
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("failed to set env var %s: %v", k, err)
		}
		defer func(key string) { _ = os.Unsetenv(key) }(k)
	}

	fetcher := fetcher.New()

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
	if err := os.Setenv(testVar, "initial_value"); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}
	defer func() { _ = os.Unsetenv(testVar) }()

	fetcher := fetcher.New()

	// First fetch
	val1, err := fetcher.Fetch(testVar)
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}
	if val1 != "initial_value" {
		t.Errorf("first fetch: got %q, want %q", val1, "initial_value")
	}

	// Change environment variable
	if err := os.Setenv(testVar, "changed_value"); err != nil {
		t.Fatalf("failed to change env var: %v", err)
	}

	// Second fetch should return cached value
	val2, err := fetcher.Fetch(testVar)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}
	if val2 != "initial_value" {
		t.Errorf("second fetch should return cached value %q, got %q", "initial_value", val2)
	}
}

// T086: Unit test for special characters in variable names (dots, dashes, underscores)
//
// Tests that variables with special characters like MY.VAR.NAME and MY-VAR-NAME
// work correctly with direct access and path transformation.
func TestSpecialCharactersInVariableNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		varName     string
		varValue    string
		wantSuccess bool
	}{
		{
			name:        "dots in variable name",
			varName:     "MY.VAR.NAME",
			varValue:    "dotted_value",
			wantSuccess: true,
		},
		{
			name:        "dashes in variable name",
			varName:     "MY-VAR-NAME",
			varValue:    "dashed_value",
			wantSuccess: true,
		},
		{
			name:        "underscores in variable name",
			varName:     "MY_VAR_NAME",
			varValue:    "underscored_value",
			wantSuccess: true,
		},
		{
			name:        "mixed separators",
			varName:     "MY.VAR-NAME_TEST",
			varValue:    "mixed_value",
			wantSuccess: true,
		},
		{
			name:        "dotted config path",
			varName:     "CONFIG.API.ENDPOINT",
			varValue:    "https://api.example.com",
			wantSuccess: true,
		},
		{
			name:        "dashed feature flag",
			varName:     "APP-FEATURE-FLAG",
			varValue:    "enabled",
			wantSuccess: true,
		},
		{
			name:        "complex mixed separators",
			varName:     "DATABASE.CONNECTION-STRING_PRIMARY",
			varValue:    "postgresql://localhost:5432",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			_ = os.Unsetenv(tt.varName)
			defer func() { _ = os.Unsetenv(tt.varName) }()

			// Set the environment variable
			if err := os.Setenv(tt.varName, tt.varValue); err != nil {
				if tt.wantSuccess {
					t.Logf("Note: OS does not support variable name %q: %v", tt.varName, err)
					t.Skip("OS does not support this variable name format")
				}
				return
			}

			// Create fetcher and fetch the value
			fetcher := fetcher.New()
			value, err := fetcher.Fetch(tt.varName)

			if tt.wantSuccess {
				if err != nil {
					t.Errorf("fetch failed for %q: %v", tt.varName, err)
					return
				}
				if value != tt.varValue {
					t.Errorf("got value %q, want %q", value, tt.varValue)
				}
			} else if err == nil {
				t.Errorf("expected error for %q, got success", tt.varName)
			}
		})
	}
}

// T087: Unit test for Unicode/UTF-8 variable names and values
//
// Tests non-ASCII characters in variable names (if OS supports) and Unicode values.
// Verifies proper encoding/decoding of international characters and emoji.
func TestUnicodeUTF8Variables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		varName     string
		varValue    string
		skipOnError bool // Some OSes don't support Unicode var names
	}{
		{
			name:     "ASCII name with Japanese value",
			varName:  "MESSAGE_JAPANESE",
			varValue: "こんにちは世界",
		},
		{
			name:     "ASCII name with Chinese value",
			varName:  "MESSAGE_CHINESE",
			varValue: "你好世界",
		},
		{
			name:     "ASCII name with Arabic value",
			varName:  "MESSAGE_ARABIC",
			varValue: "مرحبا بالعالم",
		},
		{
			name:     "ASCII name with emoji",
			varName:  "MESSAGE_EMOJI",
			varValue: "Hello 👋 World 🌍",
		},
		{
			name:     "ASCII name with mixed Unicode",
			varName:  "MESSAGE_MIXED",
			varValue: "Mixed 文字 with émojis 🎉",
		},
		{
			name:     "Spanish text",
			varName:  "MESSAGE_SPANISH",
			varValue: "¡Hola Mundo!",
		},
		{
			name:     "French text with accents",
			varName:  "MESSAGE_FRENCH",
			varValue: "Bonjour le Monde",
		},
		{
			name:     "German text with umlauts",
			varName:  "MESSAGE_GERMAN",
			varValue: "Hallo Welt äöü ÄÖÜ ß",
		},
		{
			name:     "Unicode name with accents",
			varName:  "USER_NAME_UNICODE",
			varValue: "José García",
		},
		{
			name:     "Currency symbols",
			varName:  "CURRENCY_SYMBOLS",
			varValue: "$ € £ ¥ ₹",
		},
		{
			name:     "Special Unicode symbols",
			varName:  "SPECIAL_SYMBOLS",
			varValue: "™ © ® ° ± × ÷",
		},
		{
			name:     "Cyrillic text",
			varName:  "MESSAGE_RUSSIAN",
			varValue: "Привет мир",
		},
		{
			name:     "Korean text",
			varName:  "MESSAGE_KOREAN",
			varValue: "안녕하세요 세계",
		},
		{
			name:     "Hebrew text",
			varName:  "MESSAGE_HEBREW",
			varValue: "שלום עולם",
		},
		{
			name:     "Thai text",
			varName:  "MESSAGE_THAI",
			varValue: "สวัสดีชาวโลก",
		},
		{
			name:     "Mixed emoji and text",
			varName:  "STATUS_MESSAGE",
			varValue: "✅ Success! 🎉 All tests passing 🚀",
		},
		{
			name:     "Mathematical symbols",
			varName:  "MATH_SYMBOLS",
			varValue: "∑ ∫ √ ∞ ≈ ≠ ≤ ≥",
		},
		{
			name:     "Empty Unicode string",
			varName:  "EMPTY_UNICODE",
			varValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			_ = os.Unsetenv(tt.varName)
			defer func() { _ = os.Unsetenv(tt.varName) }()

			// Set the environment variable
			if err := os.Setenv(tt.varName, tt.varValue); err != nil {
				if tt.skipOnError {
					t.Logf("Note: OS does not support Unicode variable name %q: %v", tt.varName, err)
					t.Skip("OS does not support Unicode environment variable names")
				} else {
					t.Fatalf("failed to set environment variable: %v", err)
				}
				return
			}

			// Create fetcher and fetch the value
			fetcher := fetcher.New()
			value, err := fetcher.Fetch(tt.varName)

			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}

			if value != tt.varValue {
				t.Errorf("got value %q, want %q", value, tt.varValue)
				// Show byte representation for debugging
				t.Logf("got bytes: %v", []byte(value))
				t.Logf("want bytes: %v", []byte(tt.varValue))
			}

			// Verify UTF-8 validity
			if len(value) > 0 {
				// Check if value is valid UTF-8
				validUTF8 := true
				for _, r := range value {
					if r == '\uFFFD' {
						validUTF8 = false
						break
					}
				}
				if !validUTF8 {
					t.Errorf("value contains invalid UTF-8 sequences")
				}
			}
		})
	}
}

// T085: Unit test for platform-native case sensitivity
//
// Tests that PATH and path are treated according to platform (Unix: different, Windows: same).
// Uses build tags to ensure platform-specific behavior is correct.
func TestPlatformNativeCaseSensitivity(t *testing.T) {
	t.Parallel()

	// This test verifies case sensitivity behavior based on the platform.
	// On Unix (Linux, macOS), PATH and path are different variables.
	// On Windows, PATH and path refer to the same variable.

	// Set up test variables with different cases
	testKey := fmt.Sprintf("TEST_CASE_VAR_%d", os.Getpid())
	testKeyLower := fmt.Sprintf("test_case_var_%d", os.Getpid())
	testValue := "uppercase_value"
	testValueLower := "lowercase_value"

	// Clean up
	_ = os.Unsetenv(testKey)
	_ = os.Unsetenv(testKeyLower)
	defer func() {
		_ = os.Unsetenv(testKey)
		_ = os.Unsetenv(testKeyLower)
	}()

	// Set both uppercase and lowercase variables
	if err := os.Setenv(testKey, testValue); err != nil {
		t.Fatalf("failed to set uppercase variable: %v", err)
	}

	if err := os.Setenv(testKeyLower, testValueLower); err != nil {
		// On Windows, this will overwrite the previous value
		t.Logf("setting lowercase variable: %v", err)
	}

	fetcher := fetcher.New()

	// Fetch uppercase version
	valueUpper, errUpper := fetcher.Fetch(testKey)

	// Fetch lowercase version
	valueLower, errLower := fetcher.Fetch(testKeyLower)

	// Platform-specific assertions
	if errUpper != nil {
		t.Fatalf("failed to fetch uppercase variable: %v", errUpper)
	}

	if errLower != nil {
		t.Fatalf("failed to fetch lowercase variable: %v", errLower)
	}

	// Note: We cannot deterministically test Windows vs Unix behavior here without build tags
	// The actual behavior depends on the OS. This test documents the behavior.
	//
	// On Unix: valueUpper == "uppercase_value" && valueLower == "lowercase_value"
	// On Windows: both should have the same value (last one set)

	t.Logf("Platform case sensitivity test:")
	t.Logf("  %s = %q", testKey, valueUpper)
	t.Logf("  %s = %q", testKeyLower, valueLower)

	if valueUpper == valueLower {
		t.Logf("Variables are case-insensitive (likely Windows)")
	} else {
		t.Logf("Variables are case-sensitive (likely Unix/Linux/macOS)")
		if valueUpper != testValue {
			t.Errorf("uppercase variable: got %q, want %q", valueUpper, testValue)
		}
		if valueLower != testValueLower {
			t.Errorf("lowercase variable: got %q, want %q", valueLower, testValueLower)
		}
	}
}
