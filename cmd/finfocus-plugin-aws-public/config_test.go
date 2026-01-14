package main

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWebConfig(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter())

	tests := []struct {
		name          string
		enabled       bool
		env           map[string]string
		expectedError string
		validate      func(t *testing.T, config pluginsdk.WebConfig)
	}{
		{
			name:    "Disabled",
			enabled: false,
			env:     nil,
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.False(t, config.Enabled)
				assert.Empty(t, config.AllowedOrigins)
				assert.False(t, config.AllowCredentials)
				assert.Nil(t, config.MaxAge)
				assert.False(t, config.EnableHealthEndpoint)
			},
		},
		{
			name:    "Enabled Default",
			enabled: true,
			env:     nil,
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.True(t, config.Enabled)
			},
		},
		{
			name:    "Allowed Origins - Specific",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOWED_ORIGINS": "http://localhost:3000,https://app.example.com",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.Equal(t, []string{"http://localhost:3000", "https://app.example.com"}, config.AllowedOrigins)
			},
		},
		{
			name:    "Allowed Origins - Wildcard",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOWED_ORIGINS": "*",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.Empty(t, config.AllowedOrigins)
			},
		},
		{
			name:    "Allowed Origins - Mixed Wildcard",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOWED_ORIGINS": "foo.com, *, bar.com",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.Equal(t, []string{"foo.com", "bar.com"}, config.AllowedOrigins)
			},
		},
		{
			name:    "Allowed Origins - Whitespace",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOWED_ORIGINS": " a.com , b.com ",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.Equal(t, []string{"a.com", "b.com"}, config.AllowedOrigins)
			},
		},
		{
			name:    "Max Age - Valid",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_MAX_AGE": "3600",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				require.NotNil(t, config.MaxAge)
				assert.Equal(t, 3600, *config.MaxAge)
			},
		},
		{
			name:    "Max Age - Invalid",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_MAX_AGE": "invalid",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				require.NotNil(t, config.MaxAge)
				assert.Equal(t, 86400, *config.MaxAge) // Default
			},
		},
		{
			name:    "Credentials - True",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOW_CREDENTIALS": "true",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.True(t, config.AllowCredentials)
			},
		},
		{
			name:    "Credentials - False",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOW_CREDENTIALS": "false",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.False(t, config.AllowCredentials)
			},
		},
		{
			name:    "Credentials - Case Insensitive",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOW_CREDENTIALS": "TRUE",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.True(t, config.AllowCredentials)
			},
		},
		{
			name:    "Fatal - Wildcard + Credentials",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_CORS_ALLOWED_ORIGINS":   "*",
				"FINFOCUS_CORS_ALLOW_CREDENTIALS": "true",
			},
			expectedError: "cannot enable credentials with wildcard origin",
		},
		{
			name:    "Health Endpoint - True",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_PLUGIN_HEALTH_ENDPOINT": "true",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.True(t, config.EnableHealthEndpoint)
			},
		},
		{
			name:    "Health Endpoint - False",
			enabled: true,
			env: map[string]string{
				"FINFOCUS_PLUGIN_HEALTH_ENDPOINT": "false",
			},
			validate: func(t *testing.T, config pluginsdk.WebConfig) {
				assert.False(t, config.EnableHealthEndpoint)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			// Execute
			config, err := parseWebConfig(tt.enabled, logger)

			// Validate
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}
