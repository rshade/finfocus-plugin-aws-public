package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

// parseWebConfig parses environment variables to configure the web server.
// It returns a WebConfig struct and an error if the configuration is invalid.
func parseWebConfig(enabled bool, logger zerolog.Logger) (pluginsdk.WebConfig, error) {
	if !enabled {
		return pluginsdk.WebConfig{}, nil
	}

	config := pluginsdk.WebConfig{
		Enabled: true,
	}

	// FR-001: Allowed Origins
	hasWildcard := false
	if origins := os.Getenv("FINFOCUS_CORS_ALLOWED_ORIGINS"); origins != "" {
		rawOrigins := strings.Split(origins, ",")
		for _, o := range rawOrigins {
			trimmed := strings.TrimSpace(o)
			if trimmed == "*" {
				hasWildcard = true
				continue
			}
			if trimmed != "" {
				config.AllowedOrigins = append(config.AllowedOrigins, trimmed)
			}
		}

		if hasWildcard {
			// FR-003: Warn on wildcard
			logger.Warn().Msg("CORS wildcard origin (*) is insecure; use specific origins in production")
		}
	}

	// FR-004: Allow Credentials
	if strings.ToLower(os.Getenv("FINFOCUS_CORS_ALLOW_CREDENTIALS")) == "true" {
		config.AllowCredentials = true
	}

	// FR-006: Health Endpoint
	if strings.ToLower(os.Getenv("FINFOCUS_PLUGIN_HEALTH_ENDPOINT")) == "true" {
		config.EnableHealthEndpoint = true
	}

	// FR-005: Fatal error on Wildcard + Credentials
	if hasWildcard && config.AllowCredentials {
		return pluginsdk.WebConfig{}, fmt.Errorf("cannot enable credentials with wildcard origin (*); security risk")
	}

	// FR-007 & FR-008: Max Age
	maxAge := 86400 // Default
	if maxAgeStr := os.Getenv("FINFOCUS_CORS_MAX_AGE"); maxAgeStr != "" {
		if parsed, err := strconv.Atoi(maxAgeStr); err == nil && parsed >= 0 {
			maxAge = parsed
		} else {
			logger.Warn().Str("value", maxAgeStr).Msg("invalid FINFOCUS_CORS_MAX_AGE, using default")
		}
	}
	config.MaxAge = &maxAge

	// FR-009: Log configuration
	logger.Debug().
		Strs("allowed_origins", config.AllowedOrigins).
		Int("max_age", maxAge).
		Msg("CORS configuration applied")

	return config, nil
}
