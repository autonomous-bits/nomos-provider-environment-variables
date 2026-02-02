// Package provider implements the ProviderService gRPC contract.
package provider

import (
	"fmt"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/converter"
)

// convertValue applies type conversion to a string value based on provider configuration
func (p *Provider) convertValue(value string) (interface{}, error) {
	// Call the converter package which handles automatic type detection
	// Pass the config flags to control conversion behavior
	converted, _, err := converter.ConvertValue(value, p.config.EnableTypeConversion, p.config.EnableJSONParsing)
	return converted, err
}

// toProtoValue converts a Go value to a protobuf Value
func toProtoValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		return v, nil
	case bool:
		return v, nil
	case map[string]interface{}:
		// Recursively convert nested maps
		return convertMapToProto(v)
	case []interface{}:
		// Recursively convert arrays
		return convertArrayToProto(v)
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", value)
	}
}

// convertMapToProto recursively converts a map to protobuf-compatible format
func convertMapToProto(m map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range m {
		converted, err := toProtoValue(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert map value for key %s: %w", k, err)
		}
		result[k] = converted
	}
	return result, nil
}

// convertArrayToProto recursively converts an array to protobuf-compatible format
func convertArrayToProto(arr []interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(arr))
	for i, v := range arr {
		converted, err := toProtoValue(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert array element at index %d: %w", i, err)
		}
		result[i] = converted
	}
	return result, nil
}
