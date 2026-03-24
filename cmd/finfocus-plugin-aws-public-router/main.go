package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

	"github.com/rshade/finfocus-plugin-aws-public/internal/router"
)

// version is the plugin version, set at build time via ldflags.
var version = "dev"

// main is the entry point that delegates to run() and handles exit codes.
// This pattern ensures all defer statements execute properly before process exit.
func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// run contains the router application logic.
// It configures logging, determines its own binary directory for sibling
// discovery, creates the router plugin, optionally warms up discovered
// children, and serves gRPC until a shutdown signal is received.
func run() error {
	flag.Parse()

	level := zerolog.InfoLevel
	if lvl := pluginsdk.GetLogLevel(); lvl != "" {
		if parsed, err := zerolog.ParseLevel(lvl); err == nil {
			level = parsed
		}
	}

	logger := pluginsdk.NewPluginLogger("aws-public-router", version, level, nil)

	// Determine binary directory from own executable path
	execPath, err := os.Executable()
	if err != nil {
		logger.Error().Err(err).Msg("failed to determine executable path")
		return err
	}
	binaryDir := filepath.Dir(execPath)

	// Parse environment variables
	offline := strings.ToLower(os.Getenv("FINFOCUS_PLUGIN_OFFLINE")) == "true"
	eagerWarmup := strings.ToLower(os.Getenv("FINFOCUS_PLUGIN_EAGER_WARMUP")) != "false"
	webEnabled := strings.ToLower(os.Getenv("FINFOCUS_PLUGIN_WEB_ENABLED")) == "true"

	// Create downloader (nil if offline)
	var downloader *router.Downloader
	if !offline {
		downloader = router.NewDownloader(version, binaryDir, logger)
	}

	// Create router plugin
	routerPlugin := router.NewPlugin(version, logger, binaryDir, offline, downloader)

	logger.Info().
		Str("binary_dir", binaryDir).
		Bool("offline", offline).
		Bool("eager_warmup", eagerWarmup).
		Msg("router starting")

	// Setup context for graceful shutdown using signal.NotifyContext.
	// When SIGINT/SIGTERM is received, ctx is cancelled and Serve exits.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Shutdown children synchronously when run() returns (signal or serve error).
	// Use context.Background() so ShutdownAll gets its own independent 30s timeout
	// rather than inheriting the already-cancelled ctx.
	defer func() {
		logger.Info().Msg("shutting down child processes")
		routerPlugin.ShutdownAll(context.Background())
	}()

	// Eager warm-up: launch discovered children before serving
	if eagerWarmup {
		routerPlugin.WarmUp(ctx)
	}

	// Determine port
	port := pluginsdk.ParsePortFlag()
	if port == 0 {
		port = pluginsdk.GetPort()
	}

	// Configure web serving
	webConfig, err := parseWebConfig(webEnabled, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse web configuration")
		return err
	}

	config := pluginsdk.ServeConfig{
		Plugin: routerPlugin,
		Port:   port,
		PluginInfo: &pluginsdk.PluginInfo{
			Name:        "finfocus-plugin-aws-public",
			Version:     version,
			SpecVersion: pluginsdk.SpecVersion,
			Providers:   []string{"aws"},
			Metadata: map[string]string{
				"type": "multi-region-router",
			},
		},
	}

	if webConfig.Enabled {
		config.Web = webConfig
		logger.Info().Msg("web serving enabled with multi-protocol support")
	}

	if serveErr := pluginsdk.Serve(ctx, config); serveErr != nil {
		logger.Error().Err(serveErr).Msg("server error")
		return serveErr
	}

	return nil
}
