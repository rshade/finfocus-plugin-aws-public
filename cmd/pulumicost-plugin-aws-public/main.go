package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/plugin"
	"github.com/rshade/pulumicost-plugin-aws-public/internal/pricing"
	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
)

// version is the plugin version, set at build time via ldflags.
var version = "0.0.3"

// main is the entry point that delegates to run() and handles exit codes.
// This pattern ensures all defer statements execute properly before process exit.
func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// run contains the main application logic, returning an error on failure.
// This function configures logging, initializes the pricing client and plugin instance,
// and runs the plugin server until a shutdown signal is received.
// It reads LOG_LEVEL and PORT from the environment, validates test-mode configuration,
// logs the AWS region returned by the pricing client, and performs a graceful shutdown on
// os.Interrupt or syscall.SIGTERM.
func run() error {
	// Parse log level from environment (default: info)
	level := zerolog.InfoLevel
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		if parsed, err := zerolog.ParseLevel(lvl); err == nil {
			level = parsed
		}
	}

	// Create logger using SDK utility (outputs JSON to stderr)
	logger := pluginsdk.NewPluginLogger("aws-public", version, level, nil)

	// Validate test mode env var at startup (logs warning for invalid values)
	plugin.ValidateTestModeEnv(logger)

	// Initialize pricing client
	pricingClient, err := pricing.NewClient(logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to initialize pricing client")
		return err
	}
	region := pricingClient.Region()

	// Log startup with region info (US3: Plugin Startup Logging)
	logger.Info().
		Str("aws_region", region).
		Msg("plugin started")

	// Log PORT env var if set (debug level for troubleshooting)
	if port := os.Getenv("PORT"); port != "" {
		logger.Debug().Str("port", port).Msg("using PORT from environment")
	}

	// Create plugin instance with logger
	awsPlugin := plugin.NewAWSPublicPlugin(region, pricingClient, logger)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info().Msg("received shutdown signal")
		cancel()
	}()

	// Serve using pluginsdk
	config := pluginsdk.ServeConfig{
		Plugin: awsPlugin,
		Port:   0, // Use PORT env var or random port
	}
	if err := pluginsdk.Serve(ctx, config); err != nil {
		logger.Error().Err(err).Msg("server error")
		return err
	}

	return nil
}