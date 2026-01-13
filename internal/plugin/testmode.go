package plugin

import (
	"os"
	"sync"

	"github.com/rs/zerolog"
)

// testModeEnvVar is the environment variable name for enabling test mode.
const testModeEnvVar = "FINFOCUS_TEST_MODE"

// testModeEnvVarDeprecated is the deprecated environment variable for backward compatibility.
const testModeEnvVarDeprecated = "PULUMICOST_TEST_MODE"

// testModeEnvVarLegacy is the legacy environment variable for additional backward compatibility.
const testModeEnvVarLegacy = "TEST_MODE"

// testModeDeprecationOnce ensures the deprecation warning is logged exactly once.
var testModeDeprecationOnce sync.Once

// testModeLegacyOnce ensures the legacy warning is logged exactly once.
var testModeLegacyOnce sync.Once

// IsTestMode returns true if test mode is enabled via environment variable.
// Only the exact string "true" enables test mode (strict matching per FR-011).
// Supports backward compatibility with deprecated PULUMICOST_TEST_MODE and legacy TEST_MODE.
func IsTestMode() bool {
	if os.Getenv(testModeEnvVar) == "true" {
		return true
	}
	// Backward compatibility: check deprecated env var
	if os.Getenv(testModeEnvVarDeprecated) == "true" {
		return true
	}
	// Additional backward compatibility: check legacy env var
	return os.Getenv(testModeEnvVarLegacy) == "true"
}

// IsTestModeWithLogger returns true if test mode is enabled, logging deprecation warning if needed.
func IsTestModeWithLogger(logger zerolog.Logger) bool {
	if os.Getenv(testModeEnvVar) == "true" {
		return true
	}
	// Backward compatibility: check deprecated env var
	if os.Getenv(testModeEnvVarDeprecated) == "true" {
		testModeDeprecationOnce.Do(func() {
			logger.Warn().
				Str("env_var", testModeEnvVarDeprecated).
				Str("replacement", testModeEnvVar).
				Str("deprecated_since", "v0.0.18").
				Str("removal_version", "v1.0.0").
				Msg("PULUMICOST_TEST_MODE is deprecated, use FINFOCUS_TEST_MODE instead")
		})
		return true
	}
	// Additional backward compatibility: check legacy env var
	if os.Getenv(testModeEnvVarLegacy) == "true" {
		testModeLegacyOnce.Do(func() {
			logger.Warn().
				Str("env_var", testModeEnvVarLegacy).
				Str("replacement", testModeEnvVar).
				Str("deprecated_since", "v0.0.18").
				Str("removal_version", "v1.0.0").
				Msg("TEST_MODE is deprecated, use FINFOCUS_TEST_MODE instead")
		})
		return true
	}
	return false
}

// ValidateTestModeEnv checks if FINFOCUS_TEST_MODE has an invalid value
// and logs a warning if so. Valid values are "true", "false", or unset.
// Invalid values (e.g., "1", "yes", "maybe") are treated as disabled with warning.
func ValidateTestModeEnv(logger zerolog.Logger) {
	val := os.Getenv(testModeEnvVar)
	if val != "" && val != "true" && val != "false" {
		logger.Warn().
			Str("env_var", testModeEnvVar).
			Str("value", val).
			Msg("Invalid FINFOCUS_TEST_MODE value; treating as disabled")
	}
}
