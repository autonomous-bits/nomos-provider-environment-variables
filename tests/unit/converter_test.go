package unit

import (
	"errors"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/converter"
)

// T057: Unit test for numeric conversion (integers and floats)
func TestNumericConversion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		// Integer conversions
		{"positive integer", "42", 42},
		{"negative integer", "-42", -42},
		{"zero", "0", 0},
		{"large integer", "9223372036854775807", 9223372036854775807},
		// Float conversions
		{"positive float", "3.14", 3.14},
		{"negative float", "-3.14", -3.14},
		{"float with leading zero", "0.5", 0.5},
		{"float without leading zero", ".5", 0.5},
		// Scientific notation
		{"scientific notation positive exponent", "1.23e10", 1.23e10},
		{"scientific notation negative exponent", "1.23e-10", 1.23e-10},
		{"scientific notation capital E", "1.23E10", 1.23e10},
		// Edge cases
		{"zero float", "0.0", 0.0},
		{"very small number", "0.0000001", 0.0000001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := converter.ConvertValue(tt.input, true, true)
			if err != nil {
				t.Fatalf("ConvertValue() error = %v", err)
			}

			gotNum, ok := got.(float64)
			if !ok {
				t.Fatalf("expected float64, got %T", got)
			}

			if gotNum != tt.want {
				t.Errorf("got %v, want %v", gotNum, tt.want)
			}
		})
	}
}

// T058: Unit test for boolean conversion (true/false/yes/no)
func TestBooleanConversion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Lowercase
		{"true lowercase", "true", true},
		{"false lowercase", "false", false},
		{"yes lowercase", "yes", true},
		{"no lowercase", "no", false},
		// Uppercase
		{"TRUE uppercase", "TRUE", true},
		{"FALSE uppercase", "FALSE", false},
		{"YES uppercase", "YES", true},
		{"NO uppercase", "NO", false},
		// Mixed case
		{"True mixed case", "True", true},
		{"False mixed case", "False", false},
		{"Yes mixed case", "Yes", true},
		{"No mixed case", "No", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := converter.ConvertValue(tt.input, true, true)
			if err != nil {
				t.Fatalf("ConvertValue() error = %v", err)
			}

			gotBool, ok := got.(bool)
			if !ok {
				t.Fatalf("expected bool, got %T", got)
			}

			if gotBool != tt.want {
				t.Errorf("got %v, want %v", gotBool, tt.want)
			}
		})
	}
}

// T059: Unit test for JSON parsing (objects and arrays)
func TestJSONParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result interface{})
	}{
		{
			name:  "simple object",
			input: `{"key":"value"}`,
			validate: func(t *testing.T, result interface{}) {
				m, ok := result.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map[string]interface{}, got %T", result)
				}
				if m["key"] != "value" {
					t.Errorf("expected key='value', got %v", m["key"])
				}
			},
		},
		{
			name:  "simple array",
			input: `[1,2,3]`,
			validate: func(t *testing.T, result interface{}) {
				arr, ok := result.([]interface{})
				if !ok {
					t.Fatalf("expected []interface{}, got %T", result)
				}
				if len(arr) != 3 {
					t.Errorf("expected length 3, got %d", len(arr))
				}
			},
		},
		{
			name:  "nested object",
			input: `{"database":{"host":"localhost","port":5432}}`,
			validate: func(t *testing.T, result interface{}) {
				m, ok := result.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map[string]interface{}, got %T", result)
				}
				db, ok := m["database"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected nested object, got %T", m["database"])
				}
				if db["host"] != "localhost" {
					t.Errorf("expected host='localhost', got %v", db["host"])
				}
			},
		},
		{
			name:  "empty object",
			input: `{}`,
			validate: func(t *testing.T, result interface{}) {
				m, ok := result.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map[string]interface{}, got %T", result)
				}
				if len(m) != 0 {
					t.Errorf("expected empty object, got %d keys", len(m))
				}
			},
		},
		{
			name:  "empty array",
			input: `[]`,
			validate: func(t *testing.T, result interface{}) {
				arr, ok := result.([]interface{})
				if !ok {
					t.Fatalf("expected []interface{}, got %T", result)
				}
				if len(arr) != 0 {
					t.Errorf("expected empty array, got length %d", len(arr))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := converter.ConvertValue(tt.input, true, true)
			if err != nil {
				t.Fatalf("ConvertValue() error = %v", err)
			}
			tt.validate(t, got)
		})
	}
}

// T060: Unit test for conversion precedence (number before boolean)
func TestConversionPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{"number string '1' converts to number not boolean", "1", "number"},
		{"number string '0' converts to number not boolean", "0", "number"},
		{"word 'true' converts to boolean", "true", "boolean"},
		{"word 'false' converts to boolean", "false", "boolean"},
		{"JSON object takes precedence", `{"value":"123"}`, "object"},
		{"JSON array takes precedence", `[1,2,3]`, "array"},
		{"string that looks numeric with text is string", "123abc", "string"},
		{"string that looks boolean with text is string", "trueish", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := converter.ConvertValue(tt.input, true, true)
			if err != nil {
				t.Fatalf("ConvertValue() error = %v", err)
			}

			switch tt.wantType {
			case "number":
				if _, ok := got.(float64); !ok {
					t.Errorf("expected float64, got %T", got)
				}
			case "boolean":
				if _, ok := got.(bool); !ok {
					t.Errorf("expected bool, got %T", got)
				}
			case "string":
				if _, ok := got.(string); !ok {
					t.Errorf("expected string, got %T", got)
				}
			case "object":
				if _, ok := got.(map[string]interface{}); !ok {
					t.Errorf("expected map[string]interface{}, got %T", got)
				}
			case "array":
				if _, ok := got.([]interface{}); !ok {
					t.Errorf("expected []interface{}, got %T", got)
				}
			}
		})
	}
}

// T061: Unit test for JSON parsing errors
func TestJSONParsingErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError error
	}{
		{"malformed object - missing closing brace", `{"key":"value"`, converter.ErrInvalidJSON},
		{"malformed array - missing closing bracket", `[1,2,3`, converter.ErrInvalidJSON},
		{"malformed object - missing quotes", `{key:value}`, converter.ErrInvalidJSON},
		{"malformed array - trailing comma", `[1,2,3,]`, converter.ErrInvalidJSON},
		{"malformed object - trailing comma", `{"key":"value",}`, converter.ErrInvalidJSON},
		{"invalid JSON - single quotes", `{'key':'value'}`, converter.ErrInvalidJSON},
		{"unclosed string in object", `{"key":"value}`, converter.ErrInvalidJSON},
		{"nested object exceeds max depth", createDeeplyNestedJSON(101), converter.ErrJSONTooDeep},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := converter.ConvertValue(tt.input, true, true)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.wantError) {
				t.Errorf("expected error %v, got %v", tt.wantError, err)
			}
		})
	}
}

// T063: Unit test for empty string handling
func TestEmptyStringHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"whitespace only", "   ", "   "},
		{"tab character", "\t", "\t"},
		{"newline character", "\n", "\n"},
		{"mixed whitespace", " \t\n ", " \t\n "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := converter.ConvertValue(tt.input, true, true)
			if err != nil {
				t.Fatalf("ConvertValue() error = %v", err)
			}

			gotStr, ok := got.(string)
			if !ok {
				t.Fatalf("expected string, got %T", got)
			}

			if gotStr != tt.want {
				t.Errorf("got %q, want %q", gotStr, tt.want)
			}
		})
	}
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError error
	}{
		{"value exceeding 1MB limit", strings.Repeat("a", 1024*1024+1), converter.ErrValueTooLarge},
		{"value at 1MB limit", strings.Repeat("a", 1024*1024), nil},
		{"very large number", "999999999999999999999999999999", nil},
		{"unicode characters", "Hello 世界 🌍", nil},
		{"JSON with unicode", `{"message":"Hello 世界"}`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := converter.ConvertValue(tt.input, true, true)

			if tt.wantError != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tt.wantError) {
					t.Errorf("expected error %v, got %v", tt.wantError, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Test JSON depth validation
func TestJSONDepthValidation(t *testing.T) {
	tests := []struct {
		name      string
		depth     int
		wantError bool
	}{
		{"depth 1 - valid", 1, false},
		{"depth 50 - valid", 50, false},
		{"depth 100 - at limit, valid", 100, false},
		{"depth 101 - exceeds limit, invalid", 101, true},
		{"depth 200 - far exceeds limit, invalid", 200, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createDeeplyNestedJSON(tt.depth)
			_, _, err := converter.ConvertValue(input, true, true)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error for deep nesting, got nil")
				}
				if !errors.Is(err, converter.ErrJSONTooDeep) {
					t.Errorf("expected ErrJSONTooDeep, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Helper function to create deeply nested JSON for testing
func createDeeplyNestedJSON(depth int) string {
	if depth <= 0 {
		return `"value"`
	}
	var builder strings.Builder
	for i := 0; i < depth; i++ {
		builder.WriteString(`{"nested":`)
	}
	builder.WriteString(`"value"`)
	for i := 0; i < depth; i++ {
		builder.WriteString(`}`)
	}
	return builder.String()
}
