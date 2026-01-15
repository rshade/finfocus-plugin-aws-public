package main

import (
	"flag"
	"time"
)

type Config struct {
	StartPort  int
	EndPort    int
	ListenPort string
	Timeout    time.Duration
}

// parseConfig parses command-line flags into a Config and returns the populated instance.
// Recognized flags: -start-port (default 8001), -end-port (default 8012), -listen (default ":9090"), and -timeout (default 5s).
func parseConfig() *Config {
	config := &Config{}

	flag.IntVar(&config.StartPort, "start-port", 8001, "Starting port number to scrape")
	flag.IntVar(&config.EndPort, "end-port", 8012, "Ending port number to scrape")
	flag.StringVar(&config.ListenPort, "listen", ":9090", "Port to listen on for metrics endpoint")
	flag.DurationVar(&config.Timeout, "timeout", 5*time.Second, "Timeout for scraping individual endpoints")

	flag.Parse()

	return config
}