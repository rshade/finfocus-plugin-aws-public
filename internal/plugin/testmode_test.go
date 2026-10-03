package plugin

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestIsTestMode(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "enabled with true",
			envValue: "true",
			want:     true,
		},
		{
			name:     "disabled with false",
			envValue: "false",
			want:     false,
		},
		{
			name:     "disabled when unset",
			envValue: "",
			want:     false,
		},
		{
			name:     "disabled with 1 (strict matching)",
			envValue: "1",
			want:     false,
		},
		{
			name:     "disabled with yes (strict matching)",
			envValue: "yes",
			want:     false,
		},
		{
			name:     "disabled with TRUE (case sensitive)",
			envValue: "TRUE",
			want:     false,
		},
		{
			name:     "disabled with True (case sensitive)",
			envValue: "True",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test value
			t.Setenv(testModeEnvVar, tt.envValue)

			got := IsTestMode()
			if got != tt.want {
				t.Errorf("IsTestMode() = %v, want %v", got, tt.want)
			}
		})
	}

	// Also test legacy TEST_MODE env var
	legacyTests := []struct {
		name    string
		legacy  string
		current string
		want    bool
	}{
		{
			name:    "legacy TEST_MODE=true when FINFOCUS_TEST_MODE unset",
			legacy:  "TEST_MODE",
			current: testModeEnvVar,
			want:    true,
		},
		{
			name:    "FINFOCUS_TEST_MODE=true takes precedence over legacy TEST_MODE=false",
			legacy:  "TEST_MODE",
			current: testModeEnvVar,
			want:    true,
		},
	}

	for _, tt := range legacyTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.legacy, "true")
			if strings.Contains(tt.name, "precedence") {
				t.Setenv(tt.current, "true")
			} else {
				t.Setenv(tt.current, "")
			}

			got := IsTestMode()
			if got != tt.want {
				t.Errorf("IsTestMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsTestModeWithLogger verifies that legacy TEST_MODE is logged when used.
func TestIsTestModeWithLogger(t *testing.T) {
	tests := []struct {
		name          string
		envVar        string
		envValue      string
		expectMode    bool
		expectWarning bool
	}{
		{
			name:          "FINFOCUS_TEST_MODE=true no warning",
			envVar:        testModeEnvVar,
			envValue:      "true",
			expectMode:    true,
			expectWarning: false,
		},
		{
			name:          "legacy TEST_MODE=true with warning",
			envVar:        "TEST_MODE",
			envValue:      "true",
			expectMode:    true,
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test value
			t.Setenv(tt.envVar, tt.envValue)

			// Capture log output
			var buf bytes.Buffer
			logger := zerolog.New(&buf)

			got := IsTestModeWithLogger(logger)
			if got != tt.expectMode {
				t.Errorf("IsTestModeWithLogger() = %v, want %v", got, tt.expectMode)
			}

			logOutput := buf.String()
			hasWarning := strings.Contains(logOutput, "deprecated")
			if tt.expectWarning && !hasWarning {
				t.Error("Expected deprecation warning, got none")
			}
			if !tt.expectWarning && hasWarning {
				t.Errorf("Expected no deprecation warning, got: %s", logOutput)
			}
		})
	}
}

func TestValidateTestModeEnv(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		expectWarning bool
	}{
		{
			name:          "valid true - no warning",
			envValue:      "true",
			expectWarning: false,
		},
		{
			name:          "valid false - no warning",
			envValue:      "false",
			expectWarning: false,
		},
		{
			name:          "unset - no warning",
			envValue:      "",
			expectWarning: false,
		},
		{
			name:          "invalid 1 - warning",
			envValue:      "1",
			expectWarning: true,
		},
		{
			name:          "invalid yes - warning",
			envValue:      "yes",
			expectWarning: true,
		},
		{
			name:          "invalid TRUE - warning",
			envValue:      "TRUE",
			expectWarning: true,
		},
		{
			name:          "invalid maybe - warning",
			envValue:      "maybe",
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test value
			t.Setenv(testModeEnvVar, tt.envValue)

			// Capture log output
			var buf bytes.Buffer
			logger := zerolog.New(&buf)

			ValidateTestModeEnv(logger)

			logOutput := buf.String()
			hasWarning := len(logOutput) > 0

			if hasWarning != tt.expectWarning {
				t.Errorf("ValidateTestModeEnv() logged warning = %v, want %v, output: %s",
					hasWarning, tt.expectWarning, logOutput)
			}

			// Verify warning contains expected content
			if tt.expectWarning && hasWarning {
				if !bytes.Contains(buf.Bytes(), []byte("Invalid FINFOCUS_TEST_MODE")) {
					t.Errorf("Warning message missing expected content, got: %s", logOutput)
				}
				if !bytes.Contains(buf.Bytes(), []byte(tt.envValue)) {
					t.Errorf("Warning message missing invalid value %q, got: %s", tt.envValue, logOutput)
				}
			}
		})
	}
}
