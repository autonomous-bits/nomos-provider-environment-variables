package unit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/logger"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/provider"
	pb "github.com/autonomous-bits/nomos-provider-environment-variables/proto/providerv1"
)

// T072: Unit test for single required variable validation (exists)
func TestSingleRequiredVariableExists(t *testing.T) {
	t.Helper()

	// Create a unique variable name to avoid conflicts
	timestamp := time.Now().Unix()
	varName := fmt.Sprintf("TEST_REQUIRED_VAR_%d", timestamp)
	varValue := "test_value"

	// Set the environment variable
	if err := os.Setenv(varName, varValue); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer func() {
				if err := os.Unsetenv(varName); err != nil {
					t.Logf("cleanup failed: %v", err)
				}
			}()

	tests := []struct {
		name              string
		requiredVariables []string
		wantError         bool
	}{
		{
			name:              "single required variable exists",
			requiredVariables: []string{varName},
			wantError:         false,
		},
		{
			name:              "required variable PATH exists (system var)",
			requiredVariables: []string{"PATH"},
			wantError:         false,
		},
		{
			name:              "required variable HOME exists (system var)",
			requiredVariables: []string{"HOME"},
			wantError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.ERROR)
			prov := provider.New(log)

			config := map[string]interface{}{
				"required_variables": convertToInterfaceSlice(tt.requiredVariables),
			}

			configStruct, err := structpb.NewStruct(config)
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			req := &pb.InitRequest{
				Alias:  "test-provider",
				Config: configStruct,
			}

			_, err = prov.Init(context.Background(), req)

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

// T073: Unit test for single required variable validation (missing)
func TestSingleRequiredVariableMissing(t *testing.T) {
	t.Helper()

	// Use a variable name that definitely doesn't exist
	timestamp := time.Now().Unix()
	nonExistentVar := fmt.Sprintf("NONEXISTENT_REQUIRED_VAR_%d", timestamp)

	tests := []struct {
		name              string
		requiredVariables []string
		wantErrorCode     codes.Code
		wantErrorContains string
	}{
		{
			name:              "single required variable missing",
			requiredVariables: []string{nonExistentVar},
			wantErrorCode:     codes.InvalidArgument,
			wantErrorContains: "required environment variables missing",
		},
		{
			name:              "missing variable error includes variable name",
			requiredVariables: []string{nonExistentVar},
			wantErrorCode:     codes.InvalidArgument,
			wantErrorContains: nonExistentVar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.ERROR)
			prov := provider.New(log)

			config := map[string]interface{}{
				"required_variables": convertToInterfaceSlice(tt.requiredVariables),
			}

			configStruct, err := structpb.NewStruct(config)
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			req := &pb.InitRequest{
				Alias:  "test-provider",
				Config: configStruct,
			}

			_, err = prov.Init(context.Background(), req)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}

			if st.Code() != tt.wantErrorCode {
				t.Errorf("expected error code %v, got %v", tt.wantErrorCode, st.Code())
			}

			if tt.wantErrorContains != "" {
				errMsg := st.Message()
				if !strings.Contains(errMsg, tt.wantErrorContains) {
					t.Errorf("expected error message to contain %q, got: %q", tt.wantErrorContains, errMsg)
				}
			}
		})
	}
}

// T074: Unit test for multiple required variables validation
func TestMultipleRequiredVariablesValidation(t *testing.T) {
	t.Helper()

	timestamp := time.Now().Unix()

	// Set up test environment variables
	existingVar1 := fmt.Sprintf("TEST_EXISTING_VAR1_%d", timestamp)
	existingVar2 := fmt.Sprintf("TEST_EXISTING_VAR2_%d", timestamp)
	existingVar3 := fmt.Sprintf("TEST_EXISTING_VAR3_%d", timestamp)
	missingVar1 := fmt.Sprintf("TEST_MISSING_VAR1_%d", timestamp)
	missingVar2 := fmt.Sprintf("TEST_MISSING_VAR2_%d", timestamp)

	if err := os.Setenv(existingVar1, "value1"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.Setenv(existingVar2, "value2"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.Setenv(existingVar3, "value3"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer func() {
		if err := os.Unsetenv(existingVar1); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
		if err := os.Unsetenv(existingVar2); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
		if err := os.Unsetenv(existingVar3); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	}()

	tests := []struct {
		name              string
		requiredVariables []string
		wantError         bool
		wantErrorCode     codes.Code
		checkMissing      []string // Variables that should be listed as missing
	}{
		{
			name:              "all required variables present",
			requiredVariables: []string{existingVar1, existingVar2, existingVar3},
			wantError:         false,
		},
		{
			name:              "some required variables missing",
			requiredVariables: []string{existingVar1, missingVar1, existingVar2, missingVar2},
			wantError:         true,
			wantErrorCode:     codes.InvalidArgument,
			checkMissing:      []string{missingVar1, missingVar2},
		},
		{
			name:              "all required variables missing",
			requiredVariables: []string{missingVar1, missingVar2},
			wantError:         true,
			wantErrorCode:     codes.InvalidArgument,
			checkMissing:      []string{missingVar1, missingVar2},
		},
		{
			name:              "mix of system vars and custom vars - all present",
			requiredVariables: []string{"PATH", existingVar1, "HOME", existingVar2},
			wantError:         false,
		},
		{
			name:              "mix of system vars and custom vars - some missing",
			requiredVariables: []string{"PATH", missingVar1, existingVar1},
			wantError:         true,
			wantErrorCode:     codes.InvalidArgument,
			checkMissing:      []string{missingVar1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.ERROR)
			prov := provider.New(log)

			config := map[string]interface{}{
				"required_variables": convertToInterfaceSlice(tt.requiredVariables),
			}

			configStruct, err := structpb.NewStruct(config)
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			req := &pb.InitRequest{
				Alias:  "test-provider",
				Config: configStruct,
			}

			_, err = prov.Init(context.Background(), req)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}

				if st.Code() != tt.wantErrorCode {
					t.Errorf("expected error code %v, got %v", tt.wantErrorCode, st.Code())
				}

				// Verify all missing variables are listed in error message
				errMsg := st.Message()
				for _, missingVar := range tt.checkMissing {
					if !strings.Contains(errMsg, missingVar) {
						t.Errorf("expected error message to contain missing variable %q, got: %q", missingVar, errMsg)
					}
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// T076: Unit test for required variables with prefix
func TestRequiredVariablesWithPrefix(t *testing.T) {
	t.Helper()

	timestamp := time.Now().Unix()

	// Set up prefixed environment variables
	prefix := "MYAPP_"
	prefixedVar1 := fmt.Sprintf("%sDATABASE_HOST_%d", prefix, timestamp)
	prefixedVar2 := fmt.Sprintf("%sAPI_KEY_%d", prefix, timestamp)
	unprefixedVar := fmt.Sprintf("SYSTEM_VAR_%d", timestamp)

	if err := os.Setenv(prefixedVar1, "localhost"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.Setenv(prefixedVar2, "secret123"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := os.Setenv(unprefixedVar, "systemvalue"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	defer func() {
		if err := os.Unsetenv(prefixedVar1); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
		if err := os.Unsetenv(prefixedVar2); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
		if err := os.Unsetenv(unprefixedVar); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	}()

	tests := []struct {
		name              string
		prefix            string
		prefixMode        string
		requiredVariables []string
		wantError         bool
		wantErrorCode     codes.Code
	}{
		{
			name:              "required variables with prefix - prepend mode - all present",
			prefix:            prefix,
			prefixMode:        "prepend",
			requiredVariables: []string{prefixedVar1, prefixedVar2},
			wantError:         false,
		},
		{
			name:              "required variables with prefix - filter_only mode - all present",
			prefix:            prefix,
			prefixMode:        "filter_only",
			requiredVariables: []string{prefixedVar1, prefixedVar2},
			wantError:         false,
		},
		{
			name:       "required variables with prefix - some missing",
			prefix:     prefix,
			prefixMode: "prepend",
			requiredVariables: []string{
				prefixedVar1,
				fmt.Sprintf("%sNONEXISTENT_%d", prefix, timestamp),
			},
			wantError:     true,
			wantErrorCode: codes.InvalidArgument,
		},
		{
			name:              "required variables with prefix - unprefixed var present",
			prefix:            prefix,
			prefixMode:        "prepend",
			requiredVariables: []string{prefixedVar1, unprefixedVar},
			wantError:         false, // Validation checks actual env vars, not filtered
		},
		{
			name:              "required variables specified by original names",
			prefix:            prefix,
			prefixMode:        "filter_only",
			requiredVariables: []string{prefixedVar1, prefixedVar2}, // Full names including prefix
			wantError:         false,
		},
		{
			name:              "mix of prefixed and unprefixed required vars",
			prefix:            prefix,
			prefixMode:        "prepend",
			requiredVariables: []string{prefixedVar1, "PATH", unprefixedVar},
			wantError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.ERROR)
			prov := provider.New(log)

			config := map[string]interface{}{
				"prefix":             tt.prefix,
				"prefix_mode":        tt.prefixMode,
				"required_variables": convertToInterfaceSlice(tt.requiredVariables),
			}

			configStruct, err := structpb.NewStruct(config)
			if err != nil {
				t.Fatalf("failed to create config struct: %v", err)
			}

			req := &pb.InitRequest{
				Alias:  "test-provider",
				Config: configStruct,
			}

			_, err = prov.Init(context.Background(), req)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}

				if st.Code() != tt.wantErrorCode {
					t.Errorf("expected error code %v, got %v", tt.wantErrorCode, st.Code())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Helper function to convert []string to []interface{} for protobuf
func convertToInterfaceSlice(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}
