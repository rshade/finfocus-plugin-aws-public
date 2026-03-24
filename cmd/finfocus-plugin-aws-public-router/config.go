package main

import (
	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

	"github.com/rshade/finfocus-plugin-aws-public/internal/webconfig"
)

// parseWebConfig delegates to the shared webconfig package.
func parseWebConfig(enabled bool, logger zerolog.Logger) (pluginsdk.WebConfig, error) {
	return webconfig.ParseWebConfig(enabled, logger)
}
