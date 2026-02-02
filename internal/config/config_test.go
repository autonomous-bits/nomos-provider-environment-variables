package config

import "testing"

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{"default config", DefaultConfig(), false},
		{"invalid case_transform", &Config{Separator: "_", CaseTransform: "invalid", PrefixMode: "prepend"}, true},
		{"invalid prefix_mode", &Config{Separator: "_", CaseTransform: "upper", PrefixMode: "invalid"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateConfig() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// T038: Unit test for Config validation (invalid case_transform, invalid separator)
func TestConfigValidationComprehensive(t *testing.T) {
	t.Helper()

	tests := []struct {
		name       string
		config     *Config
		wantErr    bool
		errPattern string // Expected error message pattern
	}{
		{
			name: "valid uppercase",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "valid lowercase",
			config: &Config{
				Separator:     "_",
				CaseTransform: "lower",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "valid preserve",
			config: &Config{
				Separator:     "_",
				CaseTransform: "preserve",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "invalid case_transform - empty",
			config: &Config{
				Separator:     "_",
				CaseTransform: "",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "invalid case_transform",
		},
		{
			name: "invalid case_transform - uppercase variant",
			config: &Config{
				Separator:     "_",
				CaseTransform: "UPPER",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "invalid case_transform",
		},
		{
			name: "invalid case_transform - title",
			config: &Config{
				Separator:     "_",
				CaseTransform: "title",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "invalid case_transform",
		},
		{
			name: "invalid case_transform - mixed",
			config: &Config{
				Separator:     "_",
				CaseTransform: "Mixed",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "invalid case_transform",
		},
		{
			name: "invalid case_transform - random",
			config: &Config{
				Separator:     "_",
				CaseTransform: "random_value",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "invalid case_transform",
		},
		{
			name: "invalid separator - empty",
			config: &Config{
				Separator:     "",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "separator must be a single character",
		},
		{
			name: "invalid separator - multiple characters",
			config: &Config{
				Separator:     "__",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "separator must be a single character",
		},
		{
			name: "invalid separator - three characters",
			config: &Config{
				Separator:     "___",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "separator must be a single character",
		},
		{
			name: "invalid separator - word",
			config: &Config{
				Separator:     "SEP",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr:    true,
			errPattern: "separator must be a single character",
		},
		{
			name: "valid separator - dash",
			config: &Config{
				Separator:     "-",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "valid separator - dot",
			config: &Config{
				Separator:     ".",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "valid separator - colon",
			config: &Config{
				Separator:     ":",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "valid separator - forward slash",
			config: &Config{
				Separator:     "/",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "multiple validation errors - both invalid",
			config: &Config{
				Separator:     "+++",
				CaseTransform: "invalid",
				PrefixMode:    "prepend",
			},
			wantErr: true,
			// Should fail on first validation (case_transform checked first)
			errPattern: "invalid case_transform",
		},
		{
			name: "invalid prefix_mode with valid separator and case",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				PrefixMode:    "invalid_mode",
			},
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name: "valid filter_only prefix mode",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				PrefixMode:    "filter_only",
			},
			wantErr: false,
		},
		{
			name: "empty required variable",
			config: &Config{
				Separator:         "_",
				CaseTransform:     "upper",
				PrefixMode:        "prepend",
				RequiredVariables: []string{"VAR1", "", "VAR2"},
			},
			wantErr:    true,
			errPattern: "required_variables",
		},
		{
			name: "whitespace-only required variable",
			config: &Config{
				Separator:         "_",
				CaseTransform:     "upper",
				PrefixMode:        "prepend",
				RequiredVariables: []string{"VAR1", "   ", "VAR2"},
			},
			wantErr:    true,
			errPattern: "required_variables",
		},
		{
			name: "valid required variables",
			config: &Config{
				Separator:         "_",
				CaseTransform:     "upper",
				PrefixMode:        "prepend",
				RequiredVariables: []string{"DATABASE_URL", "API_KEY"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errPattern != "" {
				if err == nil {
					t.Errorf("ValidateConfig() expected error containing %q, got nil", tt.errPattern)
				} else if !contains(err.Error(), tt.errPattern) {
					t.Errorf("ValidateConfig() error = %q, expected to contain %q", err.Error(), tt.errPattern)
				}
			}
		})
	}
}

// Helper function for substring checking
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && substr != "" && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test edge cases for separator validation
func TestSeparatorValidation(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		separator string
		wantErr   bool
	}{
		{"single char underscore", "_", false},
		{"single char dash", "-", false},
		{"single char dot", ".", false},
		{"single char colon", ":", false},
		{"single char slash", "/", false},
		{"single char pipe", "|", false},
		{"single char comma", ",", false},
		{"single char semicolon", ";", false},
		{"empty string", "", true},
		{"two chars", "__", true},
		{"three chars", "___", true},
		{"word", "SEPARATOR", true},
		{"space only", " ", false}, // Single space is technically valid
		{"two spaces", "  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Separator:     tt.separator,
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			}
			err := ValidateConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() with separator %q: error = %v, wantErr %v", tt.separator, err, tt.wantErr)
			}
		})
	}
}

// Test case_transform validation exhaustively
func TestCaseTransformValidation(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		transform string
		wantErr   bool
	}{
		{"valid upper", "upper", false},
		{"valid lower", "lower", false},
		{"valid preserve", "preserve", false},
		{"invalid UPPER", "UPPER", true},
		{"invalid Upper", "Upper", true},
		{"invalid LOWER", "LOWER", true},
		{"invalid Lower", "Lower", true},
		{"invalid PRESERVE", "PRESERVE", true},
		{"invalid Preserve", "Preserve", true},
		{"invalid title", "title", true},
		{"invalid camel", "camel", true},
		{"invalid snake", "snake", true},
		{"invalid kebab", "kebab", true},
		{"empty string", "", true},
		{"whitespace", "   ", true},
		{"mixed", "UpPeR", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Separator:     "_",
				CaseTransform: tt.transform,
				PrefixMode:    "prepend",
			}
			err := ValidateConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() with case_transform %q: error = %v, wantErr %v", tt.transform, err, tt.wantErr)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Separator != "_" {
		t.Errorf("got %q, want _", cfg.Separator)
	}
}

// T050: Unit test for Config validation (invalid prefix_mode)
//
// Tests comprehensive validation of prefix_mode configuration field.
// Valid values are "prepend" and "filter_only" (case-sensitive).
// Any other value should result in a validation error.
func TestPrefixModeValidation(t *testing.T) {
	t.Helper()

	tests := []struct {
		name       string
		prefixMode string
		wantErr    bool
		errPattern string
	}{
		{
			name:       "valid prepend mode",
			prefixMode: "prepend",
			wantErr:    false,
		},
		{
			name:       "valid filter_only mode",
			prefixMode: "filter_only",
			wantErr:    false,
		},
		{
			name:       "invalid mode - empty string",
			prefixMode: "",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - PREPEND uppercase",
			prefixMode: "PREPEND",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - Prepend title case",
			prefixMode: "Prepend",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - FILTER_ONLY uppercase",
			prefixMode: "FILTER_ONLY",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - FilterOnly camelCase",
			prefixMode: "FilterOnly",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - filter-only with dash",
			prefixMode: "filter-only",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - filter_and_prepend",
			prefixMode: "filter_and_prepend",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - append",
			prefixMode: "append",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - prefix",
			prefixMode: "prefix",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - filter",
			prefixMode: "filter",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - whitespace",
			prefixMode: "   ",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - random value",
			prefixMode: "random_value",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - numeric",
			prefixMode: "123",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - boolean",
			prefixMode: "true",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - prepend with trailing space",
			prefixMode: "prepend ",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
		{
			name:       "invalid mode - prepend with leading space",
			prefixMode: " prepend",
			wantErr:    true,
			errPattern: "invalid prefix_mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Separator:     "_",
				CaseTransform: "upper",
				PrefixMode:    tt.prefixMode,
			}
			err := ValidateConfig(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() with prefix_mode %q: error = %v, wantErr %v", tt.prefixMode, err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errPattern != "" {
				if err == nil {
					t.Errorf("ValidateConfig() expected error containing %q, got nil", tt.errPattern)
				} else if !contains(err.Error(), tt.errPattern) {
					t.Errorf("ValidateConfig() error = %q, expected to contain %q", err.Error(), tt.errPattern)
				}
			}
		})
	}
}

// Test prefix_mode validation in combination with other config fields
func TestPrefixModeValidationWithOtherFields(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid prepend mode with prefix",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				Prefix:        "MYAPP_",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "valid filter_only mode with prefix",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				Prefix:        "MYAPP_",
				PrefixMode:    "filter_only",
			},
			wantErr: false,
		},
		{
			name: "invalid prefix_mode with valid prefix",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				Prefix:        "MYAPP_",
				PrefixMode:    "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid prepend mode without prefix",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				Prefix:        "",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "valid filter_only mode without prefix",
			config: &Config{
				Separator:     "_",
				CaseTransform: "upper",
				Prefix:        "",
				PrefixMode:    "filter_only",
			},
			wantErr: false,
		},
		{
			name: "prepend mode with lowercase transform",
			config: &Config{
				Separator:     "_",
				CaseTransform: "lower",
				Prefix:        "myapp_",
				PrefixMode:    "prepend",
			},
			wantErr: false,
		},
		{
			name: "filter_only mode with preserve case",
			config: &Config{
				Separator:     "_",
				CaseTransform: "preserve",
				Prefix:        "MyApp_",
				PrefixMode:    "filter_only",
			},
			wantErr: false,
		},
		{
			name: "invalid prefix_mode with dash separator",
			config: &Config{
				Separator:     "-",
				CaseTransform: "upper",
				Prefix:        "MYAPP-",
				PrefixMode:    "INVALID",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test that prefix_mode error takes precedence in validation order
func TestPrefixModeValidationOrder(t *testing.T) {
	t.Helper()

	// When both case_transform and prefix_mode are invalid,
	// case_transform should be checked first (based on current implementation)
	cfg := &Config{
		Separator:     "_",
		CaseTransform: "invalid_transform",
		PrefixMode:    "invalid_mode",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	// The error should mention case_transform since it's checked first
	if !contains(err.Error(), "invalid case_transform") {
		t.Errorf("expected error about case_transform, got: %v", err)
	}
}

// Test default prefix_mode behavior
func TestDefaultPrefixModeConfig(t *testing.T) {
	t.Helper()

	defaultCfg := DefaultConfig()
	if defaultCfg.PrefixMode != "prepend" {
		t.Errorf("DefaultConfig() PrefixMode = %q, want \"prepend\"", defaultCfg.PrefixMode)
	}

	// Default config should be valid
	err := ValidateConfig(defaultCfg)
	if err != nil {
		t.Errorf("DefaultConfig() should be valid, got error: %v", err)
	}
}
