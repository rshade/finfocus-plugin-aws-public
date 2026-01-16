package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// main starts the metrics aggregator HTTP server.
// It registers an endpoint for Prometheus metrics ("/metrics") and an aggregated metrics endpoint ("/metrics/aggregated"),
// begins listening on the configured address, and performs a graceful shutdown when SIGINT or SIGTERM is received using a 10-second timeout.
// The function logs startup information and exits on unrecoverable server errors.
func main() {
	config := parseConfig()

	// Create HTTP client with timeout from config
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 12,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/metrics/aggregated", func(w http.ResponseWriter, r *http.Request) {
		aggregatedMetricsHandler(w, r, config, httpClient)
	})

	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}

	shutdownDone := make(chan struct{})
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		<-signalChan

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Shutdown failed")
		}
		close(shutdownDone)
	}()

	log.Info().Str("addr", config.ListenAddr).Msg("Starting metrics aggregator")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed")
	}
	<-shutdownDone
}

// aggregatedMetricsHandler collects Prometheus metrics from a range of local ports and writes the concatenated result to the HTTP response.
//
// aggregatedMetricsHandler creates a context with the timeout specified by config.Timeout, then iterates from config.StartPort to config.EndPort,
// attempting to fetch /metrics from each localhost port. Metrics successfully retrieved are appended (separated by newlines) and served with
// Content-Type "text/plain; charset=utf-8". If fetching metrics for a specific port fails, the error is logged and the handler continues with the next port.
//
// Error Handling:
// If more than 50% of regions fail to respond (success_count < total_regions / 2),
// the handler returns HTTP 503 (Service Unavailable) to alert monitoring systems of degraded state.
// Partial metrics are still returned in this case so operators can investigate.
//
// Parameters:
//  - w: the http.ResponseWriter used to write the aggregated metrics response.
//  - r: the incoming HTTP request (unused except for context lifecycle).
//  - config: configuration specifying StartPort, EndPort, and Timeout used for collection.
//  - httpClient: HTTP client with configured timeout for making requests.
func aggregatedMetricsHandler(w http.ResponseWriter, r *http.Request, config *Config, httpClient *http.Client) {
	ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
	defer cancel()

	var allMetrics strings.Builder
	successCount := 0
	totalRegions := config.EndPort - config.StartPort + 1

	for port := config.StartPort; port <= config.EndPort; port++ {
		metrics, err := fetchMetrics(ctx, port, httpClient)
		if err != nil {
			log.Error().Err(err).Int("port", port).Msg("Failed to fetch metrics")
			continue
		}
		successCount++
		allMetrics.WriteString(metrics)
		allMetrics.WriteString("\n")
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Return HTTP 503 if more than 50% of regions failed
	if successCount*2 < totalRegions {
		w.WriteHeader(http.StatusServiceUnavailable)
		log.Warn().Int("success", successCount).Int("total", totalRegions).Msg("Metrics aggregation degraded: >50% of regions failed")
	}

	if _, err := w.Write([]byte(allMetrics.String())); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
		return
	}
}

// fetchMetrics fetches the Prometheus metrics text from the local /metrics endpoint on the given port.
//
// The ctx controls the request lifetime. The port selects the localhost TCP port to query.
// The httpClient is used to perform the HTTP request.
//
// It returns the response body as a string containing the metrics exposition on success, or an error
// if the request fails, the response status is not 200 OK, or the response body cannot be read.
func fetchMetrics(ctx context.Context, port int, httpClient *http.Client) (string, error) {
	url := fmt.Sprintf("http://localhost:%d/metrics", port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
