package pluginsettings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"

	"github.com/rshade/finfocus-plugin-aws-public/internal/pluginsettings"
)

// TestSettingsAreValid verifies each advertised list satisfies the spec's
// descriptor rules, so the SDK never rejects GetPluginInfo at runtime.
func TestSettingsAreValid(t *testing.T) {
	tests := []struct {
		name     string
		settings []*pbc.PluginSetting
	}{
		{"region", pluginsettings.Region()},
		{"router", pluginsettings.Router()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotEmpty(t, tt.settings)
			require.NoError(t, plugintesting.ValidatePluginSettings(tt.settings))
		})
	}
}

// TestSettingsEnvNames pins the environment variables the host will inject.
// A change here must match the os.Getenv call that reads the variable.
func TestSettingsEnvNames(t *testing.T) {
	envFor := func(settings []*pbc.PluginSetting) map[string]string {
		out := make(map[string]string, len(settings))
		for _, s := range settings {
			out[s.GetKey()] = s.GetEnv()
		}
		return out
	}

	common := map[string]string{
		"log_level":              "FINFOCUS_LOG_LEVEL",
		"web_enabled":            "FINFOCUS_PLUGIN_WEB_ENABLED",
		"cors_allowed_origins":   "FINFOCUS_CORS_ALLOWED_ORIGINS",
		"cors_allow_credentials": "FINFOCUS_CORS_ALLOW_CREDENTIALS",
		"cors_max_age":           "FINFOCUS_CORS_MAX_AGE",
		"health_endpoint":        "FINFOCUS_PLUGIN_HEALTH_ENDPOINT",
	}

	region := envFor(pluginsettings.Region())
	for key, env := range common {
		assert.Equal(t, env, region[key], "region %s", key)
	}
	assert.Equal(t, "FINFOCUS_MAX_BATCH_SIZE", region["max_batch_size"])
	assert.Equal(t, "FINFOCUS_STRICT_VALIDATION", region["strict_validation"])

	router := envFor(pluginsettings.Router())
	for key, env := range common {
		assert.Equal(t, env, router[key], "router %s", key)
	}
	assert.Equal(t, "FINFOCUS_PLUGIN_OFFLINE", router["offline"])
	assert.Equal(t, "FINFOCUS_PLUGIN_EAGER_WARMUP", router["eager_warmup"])
}

// TestSettingsHoldNoSecrets guards the advisory secret flag: this plugin reads
// no credentials, so nothing should be required or marked secret.
func TestSettingsHoldNoSecrets(t *testing.T) {
	for _, s := range append(pluginsettings.Region(), pluginsettings.Router()...) {
		assert.False(t, s.GetSecret(), s.GetKey())
		assert.False(t, s.GetRequired(), s.GetKey())
	}
}

// TestSettingsAreIndependentCopies verifies callers cannot mutate shared state.
func TestSettingsAreIndependentCopies(t *testing.T) {
	first := pluginsettings.Region()
	first[0].Description = "mutated"
	assert.NotEqual(t, "mutated", pluginsettings.Region()[0].GetDescription())
}
