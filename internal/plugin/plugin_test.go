package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mockPricingClient is a test double for pricing.PricingClient.
type mockPricingClient struct {
	region                string
	currency              string
	ec2Prices             map[string]float64 // key: "instanceType/os/tenancy"
	ebsPrices             map[string]float64 // key: "volumeType"
	s3Prices              map[string]float64 // key: "storageClass"
	rdsInstancePrices     map[string]float64 // key: "instanceType/engine"
	rdsStoragePrices      map[string]float64 // key: "volumeType"
	eksStandardPrice      float64            // EKS cluster standard support hourly rate
	eksExtendedPrice      float64            // EKS cluster extended support hourly rate
	ec2OnDemandCalled     int
	ebsPriceCalled        int
	s3PriceCalled         int
	rdsOnDemandCalled     int
	rdsStoragePriceCalled int
	eksPriceCalled        int
}

// newMockPricingClient creates a new mockPricingClient with default values.
func newMockPricingClient(region, currency string) *mockPricingClient {
	return &mockPricingClient{
		region:            region,
		currency:          currency,
		ec2Prices:         make(map[string]float64),
		ebsPrices:         make(map[string]float64),
		s3Prices:          make(map[string]float64),
		rdsInstancePrices: make(map[string]float64),
		rdsStoragePrices:  make(map[string]float64),
	}
}

func (m *mockPricingClient) Region() string {
	return m.region
}

func (m *mockPricingClient) Currency() string {
	return m.currency
}

func (m *mockPricingClient) EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool) {
	m.ec2OnDemandCalled++
	key := instanceType + "/" + os + "/" + tenancy
	price, found := m.ec2Prices[key]
	return price, found
}

func (m *mockPricingClient) EBSPricePerGBMonth(volumeType string) (float64, bool) {
	m.ebsPriceCalled++
	price, found := m.ebsPrices[volumeType]
	return price, found
}

func (m *mockPricingClient) S3PricePerGBMonth(storageClass string) (float64, bool) {
	m.s3PriceCalled++
	price, found := m.s3Prices[storageClass]
	return price, found
}

func (m *mockPricingClient) RDSOnDemandPricePerHour(instanceType, engine string) (float64, bool) {
	m.rdsOnDemandCalled++
	key := instanceType + "/" + engine
	price, found := m.rdsInstancePrices[key]
	return price, found
}

func (m *mockPricingClient) RDSStoragePricePerGBMonth(volumeType string) (float64, bool) {
	m.rdsStoragePriceCalled++
	price, found := m.rdsStoragePrices[volumeType]
	return price, found
}

func (m *mockPricingClient) EKSClusterPricePerHour(extendedSupport bool) (float64, bool) {
	m.eksPriceCalled++
	if extendedSupport {
		if m.eksExtendedPrice > 0 {
			return m.eksExtendedPrice, true
		}
		return 0, false
	}
	if m.eksStandardPrice > 0 {
		return m.eksStandardPrice, true
	}
	return 0, false
}

func TestNewAWSPublicPlugin(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// NewAWSPublicPlugin never returns nil
	if plugin.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", plugin.region)
	}

	if plugin.pricing != mock {
		t.Error("Pricing client not set correctly")
	}
}

func TestName(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	name := plugin.Name()
	expected := "pulumicost-plugin-aws-public"
	if name != expected {
		t.Errorf("Expected name %q, got %q", expected, name)
	}
}

// BenchmarkLoggingOverhead benchmarks the logging overhead to verify SC-005 (<1ms per request).
func BenchmarkLoggingOverhead(b *testing.B) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = plugin.GetProjectedCost(context.Background(), req)
	}
}

// T017: Test trace_id propagation with provided trace_id in gRPC metadata
func TestTraceIDPropagationWithProvidedTraceID(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Create context with trace_id in gRPC metadata
	expectedTraceID := "test-trace-id-12345"
	md := metadata.New(map[string]string{
		pluginsdk.TraceIDMetadataKey: expectedTraceID,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	}

	_, err := plugin.GetProjectedCost(ctx, req)
	if err != nil {
		t.Fatalf("GetProjectedCost() error: %v", err)
	}

	// Parse log output and verify trace_id
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, expectedTraceID) {
		t.Errorf("Log output should contain trace_id %q, got: %s", expectedTraceID, logOutput)
	}

	// Verify structured field in JSON
	scanner := bufio.NewScanner(&logBuf)
	found := false
	for scanner.Scan() {
		var logEntry map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &logEntry); err != nil {
			continue
		}
		if traceID, ok := logEntry["trace_id"].(string); ok {
			if traceID == expectedTraceID {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("trace_id %q not found in any log entry", expectedTraceID)
	}
}

// T018: Test UUID generation when trace_id is missing from context
func TestTraceIDGenerationWhenMissing(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Create context WITHOUT trace_id
	ctx := context.Background()

	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	}

	_, err := plugin.GetProjectedCost(ctx, req)
	if err != nil {
		t.Fatalf("GetProjectedCost() error: %v", err)
	}

	// Parse log output and verify a UUID-format trace_id was generated
	var logEntry map[string]interface{}
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	traceID, ok := logEntry["trace_id"].(string)
	if !ok || traceID == "" {
		t.Error("trace_id should be present in log output even when not provided")
	}

	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 chars with hyphens)
	if len(traceID) != 36 {
		t.Errorf("Generated trace_id %q should be UUID format (36 chars)", traceID)
	}
}

// T019: Test concurrent requests maintain separate trace_ids.
// Per coding guidelines, this test uses 100+ goroutines to validate concurrent RPC handling.
func TestConcurrentRequestsWithDifferentTraceIDs(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine has its own trace_id using numeric format for 100+ support
			traceID := fmt.Sprintf("trace-%03d", id)
			md := metadata.New(map[string]string{
				pluginsdk.TraceIDMetadataKey: traceID,
			})
			ctx := metadata.NewIncomingContext(context.Background(), md)

			// Verify getTraceID returns the correct value for this context
			extractedID := plugin.getTraceID(ctx)
			results <- extractedID
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect all results
	traceIDs := make(map[string]bool)
	for id := range results {
		traceIDs[id] = true
	}

	// Should have numGoroutines unique trace_ids
	if len(traceIDs) != numGoroutines {
		t.Errorf("Expected %d unique trace_ids, got %d", numGoroutines, len(traceIDs))
	}
}

// T029: Test error logs contain error_code field
func TestErrorLogsContainErrorCode(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(&logBuf).Level(zerolog.ErrorLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Request with region mismatch to trigger error
	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "eu-west-1", // Wrong region
		},
	}

	_, err := plugin.GetProjectedCost(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for region mismatch")
	}

	// Parse log output and verify error_code field
	var logEntry map[string]interface{}
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	errorCode, ok := logEntry["error_code"].(string)
	if !ok || errorCode == "" {
		t.Error("error_code field should be present in error log")
	}

	if !strings.Contains(errorCode, "UNSUPPORTED_REGION") {
		t.Errorf("error_code = %q, should contain UNSUPPORTED_REGION", errorCode)
	}

	// Verify operation field is also present
	operation, ok := logEntry["operation"].(string)
	if !ok || operation != "GetProjectedCost" {
		t.Errorf("operation = %q, want %q", operation, "GetProjectedCost")
	}
}

// TestErrorResponseContainsTraceID verifies that gRPC error responses include trace_id in details.
//
// This test validates that when an error occurs, the error response includes the trace_id
// in the ErrorDetail.Details map. This allows clients to correlate error responses with
// server-side log entries for debugging and troubleshooting.
//
// Test workflow:
//  1. Create plugin with mock pricing client
//  2. Send request with trace_id in gRPC metadata that will trigger an error
//  3. Extract ErrorDetail from gRPC status
//  4. Verify trace_id appears in Details map
func TestErrorResponseContainsTraceID(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(&logBuf).Level(zerolog.ErrorLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	expectedTraceID := "test-error-trace-12345"

	// Create context with trace_id in gRPC metadata
	md := metadata.New(map[string]string{
		pluginsdk.TraceIDMetadataKey: expectedTraceID,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Request with region mismatch to trigger error with details
	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "eu-west-1", // Wrong region
		},
	}

	_, err := plugin.GetProjectedCost(ctx, req)
	if err == nil {
		t.Fatal("Expected error for region mismatch")
	}

	// Extract gRPC status from error
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	// Check for ErrorDetail in status details
	var foundTraceID bool
	for _, detail := range st.Details() {
		if errDetail, ok := detail.(*pbc.ErrorDetail); ok {
			if traceID, exists := errDetail.Details["trace_id"]; exists {
				if traceID == expectedTraceID {
					foundTraceID = true
					t.Logf("Found trace_id in error details: %s", traceID)
				} else {
					t.Errorf("trace_id = %q, want %q", traceID, expectedTraceID)
				}
			}
		}
	}

	if !foundTraceID {
		t.Error("trace_id should be present in error response details")
	}
}

// T033: Test startup log format contains required fields
func TestStartupLogFormat(t *testing.T) {
	var logBuf bytes.Buffer
	// Simulate what main.go does
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel).With().
		Str("plugin_name", "aws-public").
		Str("plugin_version", "0.0.3").
		Logger()

	// Log startup message
	logger.Info().
		Str("aws_region", "us-east-1").
		Msg("plugin started")

	// Parse and verify
	var logEntry map[string]interface{}
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Check required fields
	requiredFields := []string{"plugin_name", "aws_region", "message"}
	for _, field := range requiredFields {
		if _, ok := logEntry[field]; !ok {
			t.Errorf("Startup log missing required field: %s", field)
		}
	}

	if msg, ok := logEntry["message"].(string); ok {
		if msg != "plugin started" {
			t.Errorf("message = %q, want %q", msg, "plugin started")
		}
	}
}
