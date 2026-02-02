package unit

import (
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/resolver"
)

// T033: Unit test for path transformation (uppercase + underscore)
func TestPathTransformUppercaseUnderscore(t *testing.T) {
	tests := []struct {
		name      string
		path      []string
		separator string
		transform string
		want      string
		wantErr   bool
	}{
		// Uppercase tests
		{
			name:      "single segment",
			path:      []string{"database"},
			separator: "_",
			transform: "upper",
			want:      "DATABASE",
			wantErr:   false,
		},
		{
			name:      "two segments",
			path:      []string{"database", "host"},
			separator: "_",
			transform: "upper",
			want:      "DATABASE_HOST",
			wantErr:   false,
		},
		{
			name:      "three segments",
			path:      []string{"app", "api", "timeout"},
			separator: "_",
			transform: "upper",
			want:      "APP_API_TIMEOUT",
			wantErr:   false,
		},
		{
			name:      "mixed case input",
			path:      []string{"Database", "HOST"},
			separator: "_",
			transform: "upper",
			want:      "DATABASE_HOST",
			wantErr:   false,
		},
		// Error cases
		{
			name:      "empty path",
			path:      []string{},
			separator: "_",
			transform: "upper",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "path with empty segment",
			path:      []string{"database", "", "host"},
			separator: "_",
			transform: "upper",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "single character segments",
			path:      []string{"a", "b", "c"},
			separator: "_",
			transform: "upper",
			want:      "A_B_C",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resolver.NewResolver(tt.separator, tt.transform, "", "prepend")
			got, err := r.Transform(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Transform() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// T034: Unit test for path transformation (lowercase + underscore)
func TestPathTransformLowercaseUnderscore(t *testing.T) {
	tests := []struct {
		name      string
		path      []string
		separator string
		transform string
		want      string
		wantErr   bool
	}{
		{
			name:      "single segment lowercase",
			path:      []string{"DATABASE"},
			separator: "_",
			transform: "lower",
			want:      "database",
			wantErr:   false,
		},
		{
			name:      "two segments lowercase",
			path:      []string{"Database", "Host"},
			separator: "_",
			transform: "lower",
			want:      "database_host",
			wantErr:   false,
		},
		{
			name:      "three segments lowercase",
			path:      []string{"APP", "API", "TIMEOUT"},
			separator: "_",
			transform: "lower",
			want:      "app_api_timeout",
			wantErr:   false,
		},
		{
			name:      "already lowercase",
			path:      []string{"database", "port"},
			separator: "_",
			transform: "lower",
			want:      "database_port",
			wantErr:   false,
		},
		{
			name:      "mixed case to lowercase",
			path:      []string{"MyApp", "ConfigKey"},
			separator: "_",
			transform: "lower",
			want:      "myapp_configkey",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resolver.NewResolver(tt.separator, tt.transform, "", "prepend")
			got, err := r.Transform(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Transform() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// T035: Unit test for path transformation (preserve case + custom separator)
func TestPathTransformCustomSeparator(t *testing.T) {
	tests := []struct {
		name      string
		path      []string
		separator string
		transform string
		want      string
		wantErr   bool
	}{
		{
			name:      "preserve case with underscore",
			path:      []string{"MyApp", "ConfigKey"},
			separator: "_",
			transform: "preserve",
			want:      "MyApp_ConfigKey",
			wantErr:   false,
		},
		{
			name:      "preserve case with dash",
			path:      []string{"Database", "Host"},
			separator: "-",
			transform: "preserve",
			want:      "Database-Host",
			wantErr:   false,
		},
		{
			name:      "preserve case with dot",
			path:      []string{"app", "api", "timeout"},
			separator: ".",
			transform: "preserve",
			want:      "app.api.timeout",
			wantErr:   false,
		},
		{
			name:      "preserve case with colon",
			path:      []string{"section", "key"},
			separator: ":",
			transform: "preserve",
			want:      "section:key",
			wantErr:   false,
		},
		{
			name:      "uppercase with dash separator",
			path:      []string{"database", "host"},
			separator: "-",
			transform: "upper",
			want:      "DATABASE-HOST",
			wantErr:   false,
		},
		{
			name:      "lowercase with dot separator",
			path:      []string{"APP", "CONFIG"},
			separator: ".",
			transform: "lower",
			want:      "app.config",
			wantErr:   false,
		},
		{
			name:      "slash separator",
			path:      []string{"path", "to", "value"},
			separator: "/",
			transform: "preserve",
			want:      "path/to/value",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resolver.NewResolver(tt.separator, tt.transform, "", "prepend")
			got, err := r.Transform(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Transform() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// T047: Unit test for prefix with path transformation
//
// Tests the combination of path transformation (uppercase/lowercase/preserve + separator)
// with prefix handling in both prepend and filter_only modes.
//
// In prepend mode:
//  1. Path segments are transformed: ["database", "host"] → "DATABASE_HOST"
//  2. Prefix is prepended: "MYAPP_" + "DATABASE_HOST" → "MYAPP_DATABASE_HOST"
//
// In filter_only mode:
//  1. Path segments are transformed: ["myapp", "database", "host"] → "MYAPP_DATABASE_HOST"
//  2. No additional prefix prepending (user includes prefix in path)
func TestPrefixWithPathTransformation(t *testing.T) {
	tests := []struct {
		name      string
		path      []string
		separator string
		transform string
		prefix    string
		mode      string
		want      string
		wantErr   bool
	}{
		// Prepend mode with uppercase transformation
		{
			name:      "prepend mode - uppercase underscore",
			path:      []string{"database", "host"},
			separator: "_",
			transform: "upper",
			prefix:    "MYAPP_",
			mode:      "prepend",
			want:      "MYAPP_DATABASE_HOST",
			wantErr:   false,
		},
		{
			name:      "prepend mode - uppercase three segments",
			path:      []string{"app", "api", "timeout"},
			separator: "_",
			transform: "upper",
			prefix:    "MYAPP_",
			mode:      "prepend",
			want:      "MYAPP_APP_API_TIMEOUT",
			wantErr:   false,
		},
		{
			name:      "prepend mode - uppercase single segment",
			path:      []string{"version"},
			separator: "_",
			transform: "upper",
			prefix:    "MYAPP_",
			mode:      "prepend",
			want:      "MYAPP_VERSION",
			wantErr:   false,
		},

		// Prepend mode with lowercase transformation
		{
			name:      "prepend mode - lowercase underscore",
			path:      []string{"Database", "Host"},
			separator: "_",
			transform: "lower",
			prefix:    "myapp_",
			mode:      "prepend",
			want:      "myapp_database_host",
			wantErr:   false,
		},
		{
			name:      "prepend mode - lowercase dash separator",
			path:      []string{"App", "Config"},
			separator: "-",
			transform: "lower",
			prefix:    "myapp-",
			mode:      "prepend",
			want:      "myapp-app-config",
			wantErr:   false,
		},

		// Prepend mode with preserve case
		{
			name:      "prepend mode - preserve case",
			path:      []string{"MyApp", "ConfigKey"},
			separator: "_",
			transform: "preserve",
			prefix:    "MyPrefix_",
			mode:      "prepend",
			want:      "MyPrefix_MyApp_ConfigKey",
			wantErr:   false,
		},

		// Filter-only mode with transformation
		{
			name:      "filter_only mode - uppercase with prefix in path",
			path:      []string{"myapp", "database", "host"},
			separator: "_",
			transform: "upper",
			prefix:    "MYAPP_",
			mode:      "filter_only",
			want:      "MYAPP_DATABASE_HOST",
			wantErr:   false,
		},
		{
			name:      "filter_only mode - lowercase with prefix in path",
			path:      []string{"MyApp", "Database", "Host"},
			separator: "_",
			transform: "lower",
			prefix:    "myapp_",
			mode:      "filter_only",
			want:      "myapp_database_host",
			wantErr:   false,
		},
		{
			name:      "filter_only mode - preserve case",
			path:      []string{"MyApp", "ConfigKey"},
			separator: "_",
			transform: "preserve",
			prefix:    "MyApp_",
			mode:      "filter_only",
			want:      "MyApp_ConfigKey",
			wantErr:   false,
		},

		// Custom separators with prefix
		{
			name:      "prepend mode - dash separator",
			path:      []string{"database", "host"},
			separator: "-",
			transform: "upper",
			prefix:    "MYAPP-",
			mode:      "prepend",
			want:      "MYAPP-DATABASE-HOST",
			wantErr:   false,
		},
		{
			name:      "prepend mode - dot separator",
			path:      []string{"app", "config"},
			separator: ".",
			transform: "lower",
			prefix:    "myapp.",
			mode:      "prepend",
			want:      "myapp.app.config",
			wantErr:   false,
		},

		// No prefix scenarios
		{
			name:      "no prefix - uppercase",
			path:      []string{"database", "host"},
			separator: "_",
			transform: "upper",
			prefix:    "",
			mode:      "prepend",
			want:      "DATABASE_HOST",
			wantErr:   false,
		},
		{
			name:      "no prefix - filter_only",
			path:      []string{"database", "host"},
			separator: "_",
			transform: "upper",
			prefix:    "",
			mode:      "filter_only",
			want:      "DATABASE_HOST",
			wantErr:   false,
		},

		// Mixed case input with different transformations
		{
			name:      "prepend mode - mixed input to uppercase",
			path:      []string{"DataBase", "HOST", "url"},
			separator: "_",
			transform: "upper",
			prefix:    "MYAPP_",
			mode:      "prepend",
			want:      "MYAPP_DATABASE_HOST_URL",
			wantErr:   false,
		},
		{
			name:      "prepend mode - mixed input to lowercase",
			path:      []string{"DataBase", "HOST", "url"},
			separator: "_",
			transform: "lower",
			prefix:    "myapp_",
			mode:      "prepend",
			want:      "myapp_database_host_url",
			wantErr:   false,
		},

		// Error cases
		{
			name:      "empty path with prefix",
			path:      []string{},
			separator: "_",
			transform: "upper",
			prefix:    "MYAPP_",
			mode:      "prepend",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "path with empty segment and prefix",
			path:      []string{"database", "", "host"},
			separator: "_",
			transform: "upper",
			prefix:    "MYAPP_",
			mode:      "prepend",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Transform the path
			r := resolver.NewResolver(tt.separator, tt.transform, tt.prefix, tt.mode)
			transformed, err := r.Transform(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Skip validation if transform failed
			}

			// The Transform method now applies prefix automatically
			if transformed != tt.want {
				t.Errorf("Transform(%v) got = %q, want %q", tt.path, transformed, tt.want)
			}
		})
	}
}

// Test that prefix handling respects different transformation strategies
func TestPrefixWithAllTransformationStrategies(t *testing.T) {
	tests := []struct {
		name      string
		path      []string
		separator string
		transform string
		prefix    string
		mode      string
		want      string
	}{
		// Same path, different transformations, prepend mode
		{
			name:      "uppercase strategy",
			path:      []string{"database", "host"},
			separator: "_",
			transform: "upper",
			prefix:    "APP_",
			mode:      "prepend",
			want:      "APP_DATABASE_HOST",
		},
		{
			name:      "lowercase strategy",
			path:      []string{"database", "host"},
			separator: "_",
			transform: "lower",
			prefix:    "app_",
			mode:      "prepend",
			want:      "app_database_host",
		},
		{
			name:      "preserve strategy",
			path:      []string{"Database", "Host"},
			separator: "_",
			transform: "preserve",
			prefix:    "App_",
			mode:      "prepend",
			want:      "App_Database_Host",
		},

		// Same path, different transformations, filter_only mode
		{
			name:      "uppercase strategy - filter_only",
			path:      []string{"app", "database", "host"},
			separator: "_",
			transform: "upper",
			prefix:    "APP_",
			mode:      "filter_only",
			want:      "APP_DATABASE_HOST",
		},
		{
			name:      "lowercase strategy - filter_only",
			path:      []string{"APP", "DATABASE", "HOST"},
			separator: "_",
			transform: "lower",
			prefix:    "app_",
			mode:      "filter_only",
			want:      "app_database_host",
		},
		{
			name:      "preserve strategy - filter_only",
			path:      []string{"App", "Database", "Host"},
			separator: "_",
			transform: "preserve",
			prefix:    "App_",
			mode:      "filter_only",
			want:      "App_Database_Host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resolver.NewResolver(tt.separator, tt.transform, tt.prefix, tt.mode)
			got, err := r.Transform(tt.path)
			if err != nil {
				t.Fatalf("Transform() unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}
