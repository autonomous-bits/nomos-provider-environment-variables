// Package config provides configuration management for the environment variables provider.
package config

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// Config represents the provider configuration
type Config struct {
	Separator            string
	CaseTransform        string
	Prefix               string
	PrefixMode           string
	RequiredVariables    []string
	EnableTypeConversion bool
	EnableJSONParsing    bool
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Separator:            "_",
		CaseTransform:        "upper",
		Prefix:               "",
		PrefixMode:           "prepend",
		RequiredVariables:    []string{},
		EnableTypeConversion: true,
		EnableJSONParsing:    true,
	}
}

// ValidateConfig validates the configuration
func ValidateConfig(c *Config) error {
	// Validate case_transform
	validCaseTransforms := map[string]bool{
		"upper": true, "lower": true, "preserve": true,
	}
	if !validCaseTransforms[c.CaseTransform] {
		return fmt.Errorf("invalid case_transform: %s (must be upper, lower, or preserve)", c.CaseTransform)
	}

	// Validate prefix_mode
	validPrefixModes := map[string]bool{
		"prepend": true, "filter_only": true,
	}
	if !validPrefixModes[c.PrefixMode] {
		return fmt.Errorf("invalid prefix_mode: %s (must be prepend or filter_only)", c.PrefixMode)
	}

	// Validate separator
	if len(c.Separator) != 1 {
		return fmt.Errorf("separator must be a single character, got: %q", c.Separator)
	}

	// Validate required_variables (non-empty strings)
	for i, varName := range c.RequiredVariables {
		if strings.TrimSpace(varName) == "" {
			return fmt.Errorf("required_variables[%d] is empty", i)
		}
	}

	return nil
}

// getString extracts a string value from a protobuf Struct
func getString(m *structpb.Struct, key string, defaultVal string) string {
	if m == nil || m.Fields == nil {
		return defaultVal
	}
	val, ok := m.Fields[key]
	if !ok {
		return defaultVal
	}
	strVal, ok := val.Kind.(*structpb.Value_StringValue)
	if !ok {
		return defaultVal
	}
	return strVal.StringValue
}

// getBool extracts a boolean value from a protobuf Struct
func getBool(m *structpb.Struct, key string, defaultVal bool) bool {
	if m == nil || m.Fields == nil {
		return defaultVal
	}
	val, ok := m.Fields[key]
	if !ok {
		return defaultVal
	}
	boolVal, ok := val.Kind.(*structpb.Value_BoolValue)
	if !ok {
		return defaultVal
	}
	return boolVal.BoolValue
}

// getStringList extracts a string array from a protobuf Struct
func getStringList(m *structpb.Struct, key string) []string {
	if m == nil || m.Fields == nil {
		return nil
	}
	val, ok := m.Fields[key]
	if !ok {
		return nil
	}
	listVal, ok := val.Kind.(*structpb.Value_ListValue)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(listVal.ListValue.Values))
	for _, item := range listVal.ListValue.Values {
		strVal, ok := item.Kind.(*structpb.Value_StringValue)
		if ok {
			result = append(result, strVal.StringValue)
		}
	}
	return result
}
