package unit

import (
	"testing"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/config"
	"google.golang.org/protobuf/types/known/structpb"
)

// T021: Unit test for Config validation (valid minimal config)
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantError bool
	}{
		{
			name:      "default config",
			config:    config.DefaultConfig(),
			wantError: false,
		},
		{
			name: "valid custom config",
			config: &config.Config{
				Separator:            "-",
				CaseTransform:        "lower",
				Prefix:               "APP_",
				PrefixMode:           "prepend",
				RequiredVariables:    []string{},
				EnableTypeConversion: true,
				EnableJSONParsing:    true,
			},
			wantError: false,
		},
		{
			name: "invalid case_transform",
			config: &config.Config{
				Separator:     "_",
				CaseTransform: "invalid",
				PrefixMode:    "prepend",
			},
			wantError: true,
		},
		{
			name: "invalid prefix_mode",
			config: &config.Config{
				Separator:     "_",
				CaseTransform: "upper",
				PrefixMode:    "invalid",
			},
			wantError: true,
		},
		{
			name: "invalid separator (empty)",
			config: &config.Config{
				Separator:     "",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantError: true,
		},
		{
			name: "invalid separator (multi-char)",
			config: &config.Config{
				Separator:     "__",
				CaseTransform: "upper",
				PrefixMode:    "prepend",
			},
			wantError: true,
		},
		{
			name: "empty required variable",
			config: &config.Config{
				Separator:         "_",
				CaseTransform:     "upper",
				PrefixMode:        "prepend",
				RequiredVariables: []string{"", "VAR1"},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateConfig(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name     string
		pbConfig map[string]interface{}
		want     *config.Config
	}{
		{
			name:     "nil config uses defaults",
			pbConfig: nil,
			want:     config.DefaultConfig(),
		},
		{
			name:     "empty config uses defaults",
			pbConfig: map[string]interface{}{},
			want:     config.DefaultConfig(),
		},
		{
			name: "custom values",
			pbConfig: map[string]interface{}{
				"separator":              "-",
				"case_transform":         "lower",
				"prefix":                 "MYAPP_",
				"prefix_mode":            "filter_only",
				"enable_type_conversion": false,
				"enable_json_parsing":    false,
				"required_variables":     []interface{}{"VAR1", "VAR2"},
			},
			want: &config.Config{
				Separator:            "-",
				CaseTransform:        "lower",
				Prefix:               "MYAPP_",
				PrefixMode:           "filter_only",
				RequiredVariables:    []string{"VAR1", "VAR2"},
				EnableTypeConversion: false,
				EnableJSONParsing:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pbStruct *structpb.Struct
			if tt.pbConfig != nil {
				var err error
				pbStruct, err = structpb.NewStruct(tt.pbConfig)
				if err != nil {
					t.Fatalf("failed to create protobuf struct: %v", err)
				}
			}

			got, err := config.ParseConfig(pbStruct)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			if got.Separator != tt.want.Separator {
				t.Errorf("separator: got %q, want %q", got.Separator, tt.want.Separator)
			}
			if got.CaseTransform != tt.want.CaseTransform {
				t.Errorf("case_transform: got %q, want %q", got.CaseTransform, tt.want.CaseTransform)
			}
			if got.Prefix != tt.want.Prefix {
				t.Errorf("prefix: got %q, want %q", got.Prefix, tt.want.Prefix)
			}
			if got.PrefixMode != tt.want.PrefixMode {
				t.Errorf("prefix_mode: got %q, want %q", got.PrefixMode, tt.want.PrefixMode)
			}
			if got.EnableTypeConversion != tt.want.EnableTypeConversion {
				t.Errorf("enable_type_conversion: got %v, want %v", got.EnableTypeConversion, tt.want.EnableTypeConversion)
			}
			if got.EnableJSONParsing != tt.want.EnableJSONParsing {
				t.Errorf("enable_json_parsing: got %v, want %v", got.EnableJSONParsing, tt.want.EnableJSONParsing)
			}
		})
	}
}
