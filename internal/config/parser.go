package config

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// ParseConfig parses a protobuf Struct into a Config
func ParseConfig(pbConfig *structpb.Struct) (*Config, error) {
	cfg := DefaultConfig()

	if pbConfig == nil || pbConfig.Fields == nil {
		return cfg, nil
	}

	// Parse optional fields
	cfg.Separator = getString(pbConfig, "separator", cfg.Separator)
	cfg.CaseTransform = getString(pbConfig, "case_transform", cfg.CaseTransform)
	cfg.Prefix = getString(pbConfig, "prefix", cfg.Prefix)
	cfg.PrefixMode = getString(pbConfig, "prefix_mode", cfg.PrefixMode)
	cfg.EnableTypeConversion = getBool(pbConfig, "enable_type_conversion", cfg.EnableTypeConversion)
	cfg.EnableJSONParsing = getBool(pbConfig, "enable_json_parsing", cfg.EnableJSONParsing)

	// Parse required_variables list
	if requiredVars := getStringList(pbConfig, "required_variables"); requiredVars != nil {
		cfg.RequiredVariables = requiredVars
	}

	return cfg, nil
}
