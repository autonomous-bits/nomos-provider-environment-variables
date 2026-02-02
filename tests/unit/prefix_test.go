package unit

import (
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/resolver"
)

// T045: Unit test for prefix prepending (prepend mode)
//
// Tests the prepend mode where the prefix is automatically added to the
// transformed variable name. In this mode, users reference variables WITHOUT
// the prefix in their CSL paths, and the provider automatically prepends it.
//
// Example: With prefix "MYAPP_" in prepend mode:
//   - Path: ["database", "host"] → Transforms to: "DATABASE_HOST"
//   - ApplyPrefix adds: "MYAPP_DATABASE_HOST"
//   - User accesses via: env["database"]["host"]
func TestPrefixPrepending(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		prefix  string
		mode    string
		want    string
		wantErr bool
	}{
		// Prepend mode tests
		{
			name:    "prepend mode - basic prefix",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "prepend",
			want:    "MYAPP_DATABASE_HOST",
			wantErr: false,
		},
		{
			name:    "prepend mode - multi-segment variable",
			varName: "APP_API_TIMEOUT",
			prefix:  "MYAPP_",
			mode:    "prepend",
			want:    "MYAPP_APP_API_TIMEOUT",
			wantErr: false,
		},
		{
			name:    "prepend mode - single segment",
			varName: "VERSION",
			prefix:  "MYAPP_",
			mode:    "prepend",
			want:    "MYAPP_VERSION",
			wantErr: false,
		},
		{
			name:    "prepend mode - no prefix configured",
			varName: "DATABASE_HOST",
			prefix:  "",
			mode:    "prepend",
			want:    "DATABASE_HOST",
			wantErr: false,
		},
		{
			name:    "prepend mode - prefix without underscore",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP",
			mode:    "prepend",
			want:    "MYAPPDATABASE_HOST",
			wantErr: false,
		},
		{
			name:    "prepend mode - lowercase prefix",
			varName: "DATABASE_HOST",
			prefix:  "myapp_",
			mode:    "prepend",
			want:    "myapp_DATABASE_HOST",
			wantErr: false,
		},
		{
			name:    "prepend mode - mixed case prefix",
			varName: "database_host",
			prefix:  "MyApp_",
			mode:    "prepend",
			want:    "MyApp_database_host",
			wantErr: false,
		},
		{
			name:    "prepend mode - numeric in variable",
			varName: "API_V2_ENDPOINT",
			prefix:  "MYAPP_",
			mode:    "prepend",
			want:    "MYAPP_API_V2_ENDPOINT",
			wantErr: false,
		},
		{
			name:    "prepend mode - special chars in prefix",
			varName: "DATABASE_HOST",
			prefix:  "MY-APP_",
			mode:    "prepend",
			want:    "MY-APP_DATABASE_HOST",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ApplyPrefix(tt.varName, tt.prefix, tt.mode)
			if got != tt.want {
				t.Errorf("ApplyPrefix() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// T046: Unit test for prefix filtering (filter_only mode)
//
// Tests the filter_only mode where the prefix is used ONLY for filtering
// and is NOT automatically prepended. Users must include the full prefix
// in their CSL paths.
//
// Example: With prefix "MYAPP_" in filter_only mode:
//   - Path: ["MYAPP_DATABASE_HOST"] → No transformation, direct lookup
//   - User must access via: env["MYAPP_DATABASE_HOST"]
//   - Variables without MYAPP_ prefix return NotFound
func TestPrefixFiltering(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		prefix  string
		mode    string
		want    string
		wantErr bool
	}{
		// Filter-only mode tests
		{
			name:    "filter_only mode - with matching prefix",
			varName: "MYAPP_DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "filter_only",
			want:    "MYAPP_DATABASE_HOST", // No transformation, returns as-is
			wantErr: false,
		},
		{
			name:    "filter_only mode - multi-segment with prefix",
			varName: "MYAPP_APP_API_TIMEOUT",
			prefix:  "MYAPP_",
			mode:    "filter_only",
			want:    "MYAPP_APP_API_TIMEOUT",
			wantErr: false,
		},
		{
			name:    "filter_only mode - no prefix configured",
			varName: "DATABASE_HOST",
			prefix:  "",
			mode:    "filter_only",
			want:    "DATABASE_HOST",
			wantErr: false,
		},
		{
			name:    "filter_only mode - variable without prefix",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "filter_only",
			want:    "DATABASE_HOST", // Returns as-is, filtering happens elsewhere
			wantErr: false,
		},
		{
			name:    "filter_only mode - different prefix",
			varName: "OTHERAPP_KEY",
			prefix:  "MYAPP_",
			mode:    "filter_only",
			want:    "OTHERAPP_KEY",
			wantErr: false,
		},
		{
			name:    "filter_only mode - prefix case sensitive",
			varName: "myapp_DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "filter_only",
			want:    "myapp_DATABASE_HOST", // Case doesn't match, returns as-is
			wantErr: false,
		},
		{
			name:    "filter_only mode - partial prefix match",
			varName: "MYAPP_DATABASE_HOST",
			prefix:  "MY_",
			mode:    "filter_only",
			want:    "MYAPP_DATABASE_HOST", // Partial match, returns as-is
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ApplyPrefix(tt.varName, tt.prefix, tt.mode)
			if got != tt.want {
				t.Errorf("ApplyPrefix() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test FilterByPrefix function for determining if a variable should be accessible
func TestFilterByPrefix(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		prefix  string
		want    bool // true = variable should be accessible
	}{
		{
			name:    "variable has matching prefix",
			varName: "MYAPP_DATABASE_HOST",
			prefix:  "MYAPP_",
			want:    true,
		},
		{
			name:    "variable has matching prefix - multi underscore",
			varName: "MYAPP_CONFIG_API_KEY",
			prefix:  "MYAPP_",
			want:    true,
		},
		{
			name:    "variable without prefix",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP_",
			want:    false,
		},
		{
			name:    "variable with different prefix",
			varName: "OTHERAPP_KEY",
			prefix:  "MYAPP_",
			want:    false,
		},
		{
			name:    "no prefix configured - all allowed",
			varName: "ANY_VARIABLE",
			prefix:  "",
			want:    true,
		},
		{
			name:    "empty variable name",
			varName: "",
			prefix:  "MYAPP_",
			want:    false,
		},
		{
			name:    "prefix case sensitive - no match",
			varName: "myapp_DATABASE_HOST",
			prefix:  "MYAPP_",
			want:    false,
		},
		{
			name:    "prefix case sensitive - match",
			varName: "MYAPP_DATABASE_HOST",
			prefix:  "MYAPP_",
			want:    true,
		},
		{
			name:    "partial prefix match - no match",
			varName: "MYAPPLICATION_KEY",
			prefix:  "MYAPP_",
			want:    false,
		},
		{
			name:    "exact prefix - no underscore",
			varName: "MYAPPKEY",
			prefix:  "MYAPP",
			want:    true,
		},
		{
			name:    "prefix longer than variable",
			varName: "MY",
			prefix:  "MYAPP_",
			want:    false,
		},
		{
			name:    "prefix equals variable",
			varName: "MYAPP_",
			prefix:  "MYAPP_",
			want:    true,
		},
		{
			name:    "system variable without prefix",
			varName: "PATH",
			prefix:  "MYAPP_",
			want:    false,
		},
		{
			name:    "system variable with no prefix configured",
			varName: "PATH",
			prefix:  "",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.FilterByPrefix(tt.varName, tt.prefix)
			if got != tt.want {
				t.Errorf("FilterByPrefix(%q, %q) got = %v, want %v", tt.varName, tt.prefix, got, tt.want)
			}
		})
	}
}

// Test edge cases for prefix handling
func TestPrefixEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		prefix  string
		mode    string
		want    string
	}{
		{
			name:    "empty prefix and variable",
			varName: "",
			prefix:  "",
			mode:    "prepend",
			want:    "",
		},
		{
			name:    "special characters in prefix",
			varName: "HOST",
			prefix:  "MY-APP.PREFIX_",
			mode:    "prepend",
			want:    "MY-APP.PREFIX_HOST",
		},
		{
			name:    "unicode in prefix",
			varName: "DATABASE_HOST",
			prefix:  "МОЁ_ПРИЛОЖЕНИЕ_",
			mode:    "prepend",
			want:    "МОЁ_ПРИЛОЖЕНИЕ_DATABASE_HOST",
		},
		{
			name:    "numeric prefix",
			varName: "DATABASE_HOST",
			prefix:  "123_",
			mode:    "prepend",
			want:    "123_DATABASE_HOST",
		},
		{
			name:    "whitespace in prefix",
			varName: "DATABASE_HOST",
			prefix:  "MY APP_",
			mode:    "prepend",
			want:    "MY APP_DATABASE_HOST",
		},
		{
			name:    "very long prefix",
			varName: "KEY",
			prefix:  "VERY_LONG_APPLICATION_NAME_PREFIX_WITH_LOTS_OF_SEGMENTS_",
			mode:    "prepend",
			want:    "VERY_LONG_APPLICATION_NAME_PREFIX_WITH_LOTS_OF_SEGMENTS_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ApplyPrefix(tt.varName, tt.prefix, tt.mode)
			if got != tt.want {
				t.Errorf("ApplyPrefix() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test invalid mode handling
func TestPrefixInvalidMode(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		prefix  string
		mode    string
		want    string // For invalid modes, should return varName unchanged
	}{
		{
			name:    "invalid mode - empty",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "",
			want:    "DATABASE_HOST", // Should default to safe behavior
		},
		{
			name:    "invalid mode - random string",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "invalid_mode",
			want:    "DATABASE_HOST",
		},
		{
			name:    "invalid mode - uppercase variant",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "PREPEND",
			want:    "DATABASE_HOST",
		},
		{
			name:    "invalid mode - mixed case",
			varName: "DATABASE_HOST",
			prefix:  "MYAPP_",
			mode:    "Prepend",
			want:    "DATABASE_HOST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ApplyPrefix(tt.varName, tt.prefix, tt.mode)
			if got != tt.want {
				t.Errorf("ApplyPrefix() with invalid mode got = %q, want %q", got, tt.want)
			}
		})
	}
}
