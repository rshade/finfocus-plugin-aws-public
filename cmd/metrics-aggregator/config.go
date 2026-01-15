package main

import (
	"flag"
	"time"

	"github.com/rs/zerolog/log"
)

// Config holds settings for the metrics aggregator server.
// StartPort and EndPort define the scraping port range, ListenAddr specifies
// the listen address for the HTTP server, and Timeout sets the per-scrape limit.
type Config struct {
	StartPort  int
	EndPort    int
	ListenAddr string
	Timeout    time.Duration
}

// parseConfig constructs and returns a *Config populated from command-line flags.
// Supported flags: -start-port (default 8001), -end-port (default 8012), -listen (default ":9090"), -timeout (default 5s).
// The returned *Config contains the values parsed from those flags after flag.Parse().
// Validates that start-port <= end-port and timeout > 0, exiting if invalid.
func parseConfig() *Config {
	config := &Config{}

	flag.IntVar(&config.StartPort, "start-port", 8001, "Starting port number to scrape")
	flag.IntVar(&config.EndPort, "end-port", 8012, "Ending port number to scrape")
	flag.StringVar(&config.ListenAddr, "listen", ":9090", "Address to listen on for metrics endpoint")
	flag.DurationVar(&config.Timeout, "timeout", 5*time.Second, "Timeout for scraping individual endpoints")

	flag.Parse()

	// Validate configuration
	if config.EndPort < config.StartPort {
		log.Fatal().
			Int("startPort", config.StartPort).
			Int("endPort", config.EndPort).
			Msg("invalid configuration: end-port must be >= start-port")
	}
	if config.Timeout <= 0 {
		log.Fatal().
			Dur("timeout", config.Timeout).
			Msg("invalid configuration: timeout must be positive")
	}

	return config
}