// Package pluginsettings declares the environment variables this plugin reads,
// so GetPluginInfo can advertise them to the host as PluginSetting entries.
//
// Each entry names the variable the host injects and mirrors the default the
// reading code applies. The plugin reads no credentials, so no entry is
// required or secret.
package pluginsettings

import (
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// defaultOff is the default for boolean settings that are disabled unless set.
const defaultOff = "false"

// Region returns the settings read by a region-specific plugin binary.
// Every call returns fresh messages, so callers may keep or modify the result.
func Region() []*pbc.PluginSetting {
	settings := []*pbc.PluginSetting{
		{
			Key:          "max_batch_size",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT,
			DefaultValue: "100",
			Env:          "FINFOCUS_MAX_BATCH_SIZE",
			Description:  "Maximum resources per GetRecommendations batch. Values above 100 are capped to 100.",
		},
		{
			Key:          "strict_validation",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
			DefaultValue: defaultOff,
			Env:          "FINFOCUS_STRICT_VALIDATION",
			Description:  "Fail a recommendations batch on the first invalid resource instead of skipping it.",
		},
	}
	return append(settings, shared()...)
}

// Router returns the settings read by the multi-region router binary.
// Every call returns fresh messages, so callers may keep or modify the result.
func Router() []*pbc.PluginSetting {
	settings := []*pbc.PluginSetting{
		{
			Key:          "offline",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
			DefaultValue: defaultOff,
			Env:          "FINFOCUS_PLUGIN_OFFLINE",
			Description:  "Disable runtime download of region binaries. Only region binaries already on disk are used.",
		},
		{
			Key:          "eager_warmup",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
			DefaultValue: "true",
			Env:          "FINFOCUS_PLUGIN_EAGER_WARMUP",
			Description:  "Start every available region binary at launch instead of on first request.",
		},
	}
	return append(settings, shared()...)
}

// shared returns the settings both binaries read: logging and web serving.
func shared() []*pbc.PluginSetting {
	return []*pbc.PluginSetting{
		{
			Key:          "log_level",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_ENUM,
			DefaultValue: "info",
			Env:          "FINFOCUS_LOG_LEVEL",
			EnumValues:   []string{"trace", "debug", "info", "warn", "error"},
			Description:  "Minimum severity written to the plugin's stderr log.",
		},
		{
			Key:          "web_enabled",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
			DefaultValue: defaultOff,
			Env:          "FINFOCUS_PLUGIN_WEB_ENABLED",
			Description:  "Serve gRPC-Web and Connect protocols for browser clients alongside gRPC.",
		},
		{
			Key:         "cors_allowed_origins",
			Type:        pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
			Env:         "FINFOCUS_CORS_ALLOWED_ORIGINS",
			Description: "Comma-separated CORS origins, or * for all. Applies only when web serving is enabled.",
		},
		{
			Key:          "cors_allow_credentials",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
			DefaultValue: defaultOff,
			Env:          "FINFOCUS_CORS_ALLOW_CREDENTIALS",
			Description:  "Allow credentialed CORS requests. Startup fails if combined with a * origin.",
		},
		{
			Key:          "cors_max_age",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT,
			DefaultValue: "86400",
			Env:          "FINFOCUS_CORS_MAX_AGE",
			Description:  "CORS preflight cache duration in seconds. Applies only when web serving is enabled.",
		},
		{
			Key:          "health_endpoint",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
			DefaultValue: defaultOff,
			Env:          "FINFOCUS_PLUGIN_HEALTH_ENDPOINT",
			Description:  "Expose an HTTP health endpoint. Applies only when web serving is enabled.",
		},
	}
}
