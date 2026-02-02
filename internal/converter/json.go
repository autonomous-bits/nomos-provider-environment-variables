package converter

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrInvalidJSON is returned when JSON parsing fails
	ErrInvalidJSON = errors.New("invalid JSON")
	// ErrJSONTooDeep is returned when JSON nesting exceeds max depth
	ErrJSONTooDeep = errors.New("JSON nesting depth exceeds maximum of 100 levels")
)

const (
	// MaxJSONDepth is the maximum allowed JSON nesting depth
	MaxJSONDepth = 100
)

// TryJSON attempts to parse a JSON string.
// Returns the parsed value (map[string]interface{} for objects, []interface{} for arrays).
// Returns error if parsing fails or depth exceeds limit.
func TryJSON(value string) (interface{}, error) {
	var result interface{}

	// Attempt to parse JSON
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	// Validate depth
	if err := validateDepth(result, 0); err != nil {
		return nil, err
	}

	return result, nil
}

// validateDepth recursively checks JSON nesting depth to prevent stack overflow
func validateDepth(value interface{}, depth int) error {
	if depth > MaxJSONDepth {
		return ErrJSONTooDeep
	}

	switch v := value.(type) {
	case map[string]interface{}:
		for _, val := range v {
			if err := validateDepth(val, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, val := range v {
			if err := validateDepth(val, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}
