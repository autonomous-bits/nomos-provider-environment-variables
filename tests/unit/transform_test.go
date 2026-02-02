package unit

import (
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/resolver"
)

// T036: Unit test for case transformation functions
func TestToUpperCase(t *testing.T) {
	t.Helper()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase to uppercase",
			input: "database",
			want:  "DATABASE",
		},
		{
			name:  "mixed case to uppercase",
			input: "DataBase",
			want:  "DATABASE",
		},
		{
			name:  "already uppercase",
			input: "DATABASE",
			want:  "DATABASE",
		},
		{
			name:  "with numbers",
			input: "database01",
			want:  "DATABASE01",
		},
		{
			name:  "with underscores",
			input: "data_base",
			want:  "DATA_BASE",
		},
		{
			name:  "single character",
			input: "a",
			want:  "A",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "special characters",
			input: "data-base.config",
			want:  "DATA-BASE.CONFIG",
		},
		{
			name:  "unicode characters",
			input: "café",
			want:  "CAFÉ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ToUpperCase(tt.input)
			if got != tt.want {
				t.Errorf("ToUpperCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToLowerCase(t *testing.T) {
	t.Helper()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "uppercase to lowercase",
			input: "DATABASE",
			want:  "database",
		},
		{
			name:  "mixed case to lowercase",
			input: "DataBase",
			want:  "database",
		},
		{
			name:  "already lowercase",
			input: "database",
			want:  "database",
		},
		{
			name:  "with numbers",
			input: "DATABASE01",
			want:  "database01",
		},
		{
			name:  "with underscores",
			input: "DATA_BASE",
			want:  "data_base",
		},
		{
			name:  "single character",
			input: "A",
			want:  "a",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "special characters",
			input: "DATA-BASE.CONFIG",
			want:  "data-base.config",
		},
		{
			name:  "unicode characters",
			input: "CAFÉ",
			want:  "café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ToLowerCase(tt.input)
			if got != tt.want {
				t.Errorf("ToLowerCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPreserveCase(t *testing.T) {
	t.Helper()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase preserved",
			input: "database",
			want:  "database",
		},
		{
			name:  "uppercase preserved",
			input: "DATABASE",
			want:  "DATABASE",
		},
		{
			name:  "mixed case preserved",
			input: "DataBase",
			want:  "DataBase",
		},
		{
			name:  "camelCase preserved",
			input: "myAppConfig",
			want:  "myAppConfig",
		},
		{
			name:  "PascalCase preserved",
			input: "MyAppConfig",
			want:  "MyAppConfig",
		},
		{
			name:  "snake_case preserved",
			input: "my_app_config",
			want:  "my_app_config",
		},
		{
			name:  "SCREAMING_SNAKE_CASE preserved",
			input: "MY_APP_CONFIG",
			want:  "MY_APP_CONFIG",
		},
		{
			name:  "kebab-case preserved",
			input: "my-app-config",
			want:  "my-app-config",
		},
		{
			name:  "with numbers preserved",
			input: "MyApp123Config",
			want:  "MyApp123Config",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.PreserveCase(tt.input)
			if got != tt.want {
				t.Errorf("PreserveCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Test case transformation with path segments
func TestTransformSegment(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		segment   string
		transform string
		want      string
	}{
		// Uppercase transformations
		{
			name:      "uppercase lowercase input",
			segment:   "database",
			transform: "upper",
			want:      "DATABASE",
		},
		{
			name:      "uppercase mixed input",
			segment:   "DataBase",
			transform: "upper",
			want:      "DATABASE",
		},
		// Lowercase transformations
		{
			name:      "lowercase uppercase input",
			segment:   "DATABASE",
			transform: "lower",
			want:      "database",
		},
		{
			name:      "lowercase mixed input",
			segment:   "DataBase",
			transform: "lower",
			want:      "database",
		},
		// Preserve transformations
		{
			name:      "preserve lowercase",
			segment:   "database",
			transform: "preserve",
			want:      "database",
		},
		{
			name:      "preserve uppercase",
			segment:   "DATABASE",
			transform: "preserve",
			want:      "DATABASE",
		},
		{
			name:      "preserve mixed",
			segment:   "DataBase",
			transform: "preserve",
			want:      "DataBase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.TransformSegment(tt.segment, tt.transform)
			if got != tt.want {
				t.Errorf("TransformSegment(%q, %q) = %q, want %q", tt.segment, tt.transform, got, tt.want)
			}
		})
	}
}

// Test batch transformation of multiple segments
func TestTransformSegments(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		segments  []string
		transform string
		want      []string
	}{
		{
			name:      "uppercase multiple segments",
			segments:  []string{"database", "host", "port"},
			transform: "upper",
			want:      []string{"DATABASE", "HOST", "PORT"},
		},
		{
			name:      "lowercase multiple segments",
			segments:  []string{"DATABASE", "HOST", "PORT"},
			transform: "lower",
			want:      []string{"database", "host", "port"},
		},
		{
			name:      "preserve multiple segments",
			segments:  []string{"Database", "Host", "Port"},
			transform: "preserve",
			want:      []string{"Database", "Host", "Port"},
		},
		{
			name:      "single segment uppercase",
			segments:  []string{"database"},
			transform: "upper",
			want:      []string{"DATABASE"},
		},
		{
			name:      "empty segments",
			segments:  []string{},
			transform: "upper",
			want:      []string{},
		},
		{
			name:      "mixed case input to uppercase",
			segments:  []string{"DataBase", "HostName", "PortNumber"},
			transform: "upper",
			want:      []string{"DATABASE", "HOSTNAME", "PORTNUMBER"},
		},
		{
			name:      "mixed case input to lowercase",
			segments:  []string{"DataBase", "HostName", "PortNumber"},
			transform: "lower",
			want:      []string{"database", "hostname", "portnumber"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.TransformSegments(tt.segments, tt.transform)

			if len(got) != len(tt.want) {
				t.Errorf("TransformSegments() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("TransformSegments()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Test case transformation edge cases
func TestCaseTransformEdgeCases(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		input     string
		transform string
		want      string
	}{
		{
			name:      "uppercase empty",
			input:     "",
			transform: "upper",
			want:      "",
		},
		{
			name:      "lowercase empty",
			input:     "",
			transform: "lower",
			want:      "",
		},
		{
			name:      "preserve empty",
			input:     "",
			transform: "preserve",
			want:      "",
		},
		{
			name:      "uppercase whitespace",
			input:     "   ",
			transform: "upper",
			want:      "   ",
		},
		{
			name:      "uppercase numbers only",
			input:     "12345",
			transform: "upper",
			want:      "12345",
		},
		{
			name:      "lowercase numbers only",
			input:     "12345",
			transform: "lower",
			want:      "12345",
		},
		{
			name:      "uppercase special chars",
			input:     "!@#$%",
			transform: "upper",
			want:      "!@#$%",
		},
		{
			name:      "lowercase special chars",
			input:     "!@#$%",
			transform: "lower",
			want:      "!@#$%",
		},
		{
			name:      "mixed alphanumeric uppercase",
			input:     "abc123def456",
			transform: "upper",
			want:      "ABC123DEF456",
		},
		{
			name:      "mixed alphanumeric lowercase",
			input:     "ABC123DEF456",
			transform: "lower",
			want:      "abc123def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			switch tt.transform {
			case "upper":
				got = resolver.ToUpperCase(tt.input)
			case "lower":
				got = resolver.ToLowerCase(tt.input)
			case "preserve":
				got = resolver.PreserveCase(tt.input)
			default:
				t.Fatalf("unknown transform: %s", tt.transform)
			}

			if got != tt.want {
				t.Errorf("transform(%q, %q) = %q, want %q", tt.input, tt.transform, got, tt.want)
			}
		})
	}
}
