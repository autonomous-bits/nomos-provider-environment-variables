// Package converter provides automatic type conversion from string values to appropriate Go types.
package converter

import (
	"errors"
	"strconv"
	"strings"
)

var (
	// ErrValueTooLarge is returned when the value exceeds maximum size
	ErrValueTooLarge = errors.New("value exceeds maximum size of 1MB")
)

const (
	// MaxValueSize is the maximum allowed size for a value (1MB)
	MaxValueSize = 1 * 1024 * 1024
)

// ConvertValue applies automatic type conversion to a string value.
// Conversion precedence: JSON (if starts with { or [) → Number → Boolean → String.
// enableTypeConversion controls number/boolean conversion, enableJSONParsing controls JSON parsing.
// Returns the converted value as interface{}, type string, and error if conversion fails.
func ConvertValue(value string, enableTypeConversion, enableJSONParsing bool) (result interface{}, typeStr string, err error) {
	// Check size limit
	if len(value) > MaxValueSize {
		return nil, "", ErrValueTooLarge
	}

	// Empty strings remain empty strings
	if value == "" {
		return value, "string", nil
	}

	// Check JSON parsing first (if enabled and value starts with { or [)
	trimmed := strings.TrimSpace(value)
	if enableJSONParsing && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
		result, err := TryJSON(value)
		if err != nil {
			return nil, "", err
		}
		// Determine type from result
		typ := "object"
		if _, isArray := result.([]interface{}); isArray {
			typ = "array"
		}
		return result, typ, nil
	}

	// Skip type conversion if disabled
	if !enableTypeConversion {
		return value, "string", nil
	}

	// Try numeric conversion
	if num, ok := TryNumeric(value); ok {
		return num, "number", nil
	}

	// Try boolean conversion
	if b, ok := TryBoolean(value); ok {
		return b, "boolean", nil
	}

	// Default to string
	return value, "string", nil
}

// TryNumeric attempts to parse a numeric value.
// Returns the numeric value as float64 and true if successful, 0 and false otherwise.
// Integers are converted to float64 for consistent typing in JSON/protobuf.
func TryNumeric(value string) (float64, bool) {
	// Try to parse as float (handles both integers and floats)
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// TryBoolean attempts to parse a boolean value.
// Supports: true, false, yes, no (case-insensitive).
// Returns the boolean value and true if successful, false and false otherwise.
func TryBoolean(value string) (result, ok bool) {
	lower := strings.ToLower(strings.TrimSpace(value))

	switch lower {
	case "true", "yes":
		return true, true
	case "false", "no":
		return false, true
	default:
		return false, false
	}
}
