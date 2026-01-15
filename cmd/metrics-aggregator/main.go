package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// main starts the metrics aggregator HTTP server, registers the Prometheus handler at
// /metrics and an aggregated metrics endpoint at /metrics/aggregated, and listens on the
// port specified by the configuration.
func main() {
	config := parseConfig()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/metrics/aggregated", func(w http.ResponseWriter, r *http.Request) {
		aggregatedMetricsHandler(w, r, config)
	})

	log.Printf("Starting metrics aggregator on %s", config.ListenPort)
	log.Fatal(http.ListenAndServe(config.ListenPort, nil))
}

// "text/plain; charset=utf-8".
func aggregatedMetricsHandler(w http.ResponseWriter, r *http.Request, config *Config) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	var allMetrics strings.Builder

	for port := config.StartPort; port <= config.EndPort; port++ {
		metrics, err := fetchMetrics(ctx, port)
		if err != nil {
			log.Printf("Failed to fetch metrics from port %d: %v", port, err)
			continue
		}
		allMetrics.WriteString(metrics)
		allMetrics.WriteString("\n")
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(allMetrics.String()))
}

// fetchMetrics fetches Prometheus metrics from the /metrics endpoint on localhost at the specified port.
// It performs an HTTP GET using the provided context and returns the response body as a string.
// ctx provides cancellation and deadline control for the request; port is the target TCP port.
// The returned error indicates request creation or execution failure, a non-200 HTTP response status, or a response read error.
func fetchMetrics(ctx context.Context, port int) (string, error) {
	url := fmt.Sprintf("http://localhost:%d/metrics", port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}