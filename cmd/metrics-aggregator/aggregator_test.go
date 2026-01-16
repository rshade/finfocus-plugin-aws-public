package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchMetrics(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("# HELP test_metric Test metric\n# TYPE test_metric gauge\ntest_metric 1\n")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Extract port from server URL
	portStr := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	port := 0
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatalf("Failed to parse port: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	httpClient := &http.Client{
		Timeout: 1 * time.Second,
	}

	metrics, err := fetchMetrics(ctx, port, httpClient)
	if err != nil {
		t.Fatalf("fetchMetrics failed: %v", err)
	}

	if !strings.Contains(metrics, "test_metric") {
		t.Errorf("Expected metrics to contain 'test_metric', got: %s", metrics)
	}
}

func TestAggregatedMetricsHandler(t *testing.T) {
	// Test the handler with unavailable regions (expected to return 503 due to >50% failure)
	config := &Config{
		StartPort:  8001,
		EndPort:    8001,
		ListenAddr: ":9090",
		Timeout:    1 * time.Second,
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	req := httptest.NewRequest("GET", "/metrics/aggregated", nil)
	w := httptest.NewRecorder()

	aggregatedMetricsHandler(w, req, config, httpClient)

	resp := w.Result()
	// When all regions fail (1 region, 0 successes), handler returns 503
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("Expected status 503 for all region failures, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if body != "" && !strings.Contains(body, "#") {
		t.Fatalf("Expected empty body or metrics content, got: %s", body)
	}
}
