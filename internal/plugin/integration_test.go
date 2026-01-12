//go:build integration

package plugin_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Note: portAnnouncementTimeout is defined in validation_integration_test.go
// with env var override support (FINFOCUS_PORT_TIMEOUT). That version is shared
// across all integration tests in this package.

// TestIntegration_APSoutheast1_Binary performs end-to-end testing of the ap-southeast-1 binary.
//
// This integration test validates the complete gRPC plugin lifecycle including:
//   - Binary compilation with region-specific build tags
//   - PORT announcement via stdout (gRPC protocol requirement)
//   - Name() RPC returning correct plugin identifier
//   - GetProjectedCost() for EC2 instances in ap-southeast-1
//   - GetProjectedCost() for EBS volumes with size tags
//   - Region mismatch rejection (us-east-1 request to ap-southeast-1 binary)
//
// Test coverage (task reference: T014):
//   - Validates pricing data embedded correctly for Singapore region
//   - Confirms monthly cost calculation (hourly × 730)
//   - Verifies region validation rejects cross-region requests
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//
// Run with: go test -tags=integration ./internal/plugin/...
func TestIntegration_APSoutheast1_Binary(t *testing.T) {
	// Build the binary with region_apse1 tag
	t.Log("Building ap-southeast-1 binary...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_apse1",
		"-o", "../../dist/test-finfocus-plugin-aws-public-ap-southeast-1",
		"../../cmd/finfocus-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("../../dist/test-finfocus-plugin-aws-public-ap-southeast-1")

	// Start the binary
	t.Log("Starting ap-southeast-1 binary...")
	cmd := exec.Command("../../dist/test-finfocus-plugin-aws-public-ap-southeast-1")
	cmd.Dir, _ = os.Getwd()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start binary: %v", err)
	}
	defer cmd.Process.Kill()

	// Read PORT announcement from stdout
	var port int
	scanner := bufio.NewScanner(stdout)
	portRegex := regexp.MustCompile(`^PORT=(\d+)$`)

	timeout := time.After(portAnnouncementTimeout)
	portChan := make(chan int, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if matches := portRegex.FindStringSubmatch(line); len(matches) == 2 {
				if p, err := strconv.Atoi(matches[1]); err == nil {
					portChan <- p
					return
				}
			}
		}
	}()

	select {
	case port = <-portChan:
		t.Logf("Binary announced PORT=%d", port)
	case <-timeout:
		t.Fatal("Timeout waiting for PORT announcement")
	}

	// Give the server a moment to start listening
	time.Sleep(500 * time.Millisecond)

	// Connect to the gRPC server
	t.Log("Connecting to gRPC server...")
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: Name()
	t.Run("Name", func(t *testing.T) {
		resp, err := client.Name(ctx, &pbc.NameRequest{})
		if err != nil {
			t.Fatalf("Name() failed: %v", err)
		}
		t.Logf("Plugin name: %s", resp.Name)
		if resp.Name != "finfocus-plugin-aws-public" {
			t.Errorf("Expected name 'finfocus-plugin-aws-public', got %q", resp.Name)
		}
	})

	// TODO(rshade): Re-enable Supports() tests once gRPC handler is implemented
	// Supports() is not exposed via gRPC by pluginsdk - it works in unit tests but not integration tests.
	// Tracking issues:
	//   - finfocus-spec#64: Add Supports() RPC method to proto
	//   - finfocus-core#160: Implement Supports() gRPC handler in pluginsdk
	// For now, GetProjectedCost() properly validates regions and rejects mismatches.

	// Test 4: GetProjectedCost() - t3.micro in ap-southeast-1
	t.Run("GetProjectedCost_t3micro_Singapore", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "ap-southeast-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("t3.micro in ap-southeast-1: $%.2f/month (hourly: $%.4f, currency: %s)",
			resp.CostPerMonth, resp.UnitPrice, resp.Currency)
		t.Logf("Billing detail: %s", resp.BillingDetail)

		if resp.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", resp.Currency)
		}
		if resp.UnitPrice <= 0 {
			t.Errorf("Expected positive unit price, got %.4f", resp.UnitPrice)
		}
		if resp.CostPerMonth <= 0 {
			t.Errorf("Expected positive monthly cost, got %.2f", resp.CostPerMonth)
		}
		// Verify it's roughly hourly_rate * 730
		expectedCost := resp.UnitPrice * 730.0
		if resp.CostPerMonth < expectedCost*0.99 || resp.CostPerMonth > expectedCost*1.01 {
			t.Errorf("Monthly cost %.2f doesn't match expected %.2f (hourly * 730)",
				resp.CostPerMonth, expectedCost)
		}
	})

	// Test 5: GetProjectedCost() - EBS gp3 in ap-southeast-1
	t.Run("GetProjectedCost_gp3_Singapore", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ebs",
				Sku:          "gp3",
				Region:       "ap-southeast-1",
				Tags: map[string]string{
					"size": "100",
				},
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("gp3 100GB in ap-southeast-1: $%.2f/month (per-GB: $%.4f)",
			resp.CostPerMonth, resp.UnitPrice)
		t.Logf("Billing detail: %s", resp.BillingDetail)

		if resp.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", resp.Currency)
		}
		if resp.UnitPrice <= 0 {
			t.Errorf("Expected positive per-GB price, got %.4f", resp.UnitPrice)
		}
		// Verify it's roughly per_gb_rate * 100
		expectedCost := resp.UnitPrice * 100.0
		if resp.CostPerMonth < expectedCost*0.99 || resp.CostPerMonth > expectedCost*1.01 {
			t.Errorf("Monthly cost %.2f doesn't match expected %.2f (per-GB * 100)",
				resp.CostPerMonth, expectedCost)
		}
	})

	// Test 6: GetProjectedCost() - Wrong region (should fail)
	t.Run("GetProjectedCost_WrongRegion", func(t *testing.T) {
		_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		})
		if err == nil {
			t.Error("Expected error for wrong region, got nil")
		} else {
			t.Logf("Correctly rejected wrong region with error: %v", err)
		}
	})

	// Test 7: GetProjectedCost() - NAT Gateway in ap-southeast-1
	t.Run("GetProjectedCost_NATGateway_Singapore", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "nat_gateway",
				Sku:          "nat_gateway",
				Region:       "ap-southeast-1",
				Tags: map[string]string{
					"data_processed_gb": "100",
				},
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("NAT Gateway 100GB in ap-southeast-1: $%.2f/month (hourly: $%.4f)",
			resp.CostPerMonth, resp.UnitPrice)
		t.Logf("Billing detail: %s", resp.BillingDetail)

		if resp.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", resp.Currency)
		}
		if resp.UnitPrice <= 0 {
			t.Errorf("Expected positive hourly price, got %.4f", resp.UnitPrice)
		}
		// Since we don't have the exact price here without generating data,
		// we just verify it's greater than hourly cost.
		minExpected := resp.UnitPrice * 730.0
		if resp.CostPerMonth <= minExpected {
			t.Errorf("Monthly cost %.2f should be > hourly cost %.2f (due to 100GB processing)",
				resp.CostPerMonth, minExpected)
		}
	})

	t.Log("Integration test completed successfully!")
}

// TestIntegration_TraceIDPropagation verifies end-to-end trace_id propagation through the gRPC server.
//
// This test validates that when a client sends a request with a trace_id in gRPC metadata
// (using pluginsdk.TraceIDMetadataKey), the server extracts and includes that trace_id
// in all structured log entries. This is critical for distributed tracing and request
// correlation in production environments.
//
// Test workflow:
//  1. Builds the ap-southeast-1 binary with region_apse1 tag
//  2. Starts the binary, capturing stderr (where JSON logs are written)
//  3. Connects via gRPC and sends a request with trace_id in outgoing metadata
//  4. Parses the captured stderr and verifies trace_id appears in log JSON
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//
// Run with: go test -tags=integration ./internal/plugin/... -run TestIntegration_TraceIDPropagation
func TestIntegration_TraceIDPropagation(t *testing.T) {
	// Build the binary with region_apse1 tag
	t.Log("Building ap-southeast-1 binary for trace_id test...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_apse1",
		"-o", "../../dist/test-traceid-binary",
		"../../cmd/finfocus-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("../../dist/test-traceid-binary")

	// Start the binary and capture stderr for log verification
	t.Log("Starting binary...")
	cmd := exec.Command("../../dist/test-traceid-binary")
	cmd.Dir, _ = os.Getwd()
	cmd.Env = append(os.Environ(), "LOG_LEVEL=info")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start binary: %v", err)
	}
	defer cmd.Process.Kill()

	// Read PORT announcement from stdout
	var port int
	scanner := bufio.NewScanner(stdout)
	portRegex := regexp.MustCompile(`^PORT=(\d+)$`)

	timeout := time.After(portAnnouncementTimeout)
	portChan := make(chan int, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if matches := portRegex.FindStringSubmatch(line); len(matches) == 2 {
				if p, err := strconv.Atoi(matches[1]); err == nil {
					portChan <- p
					return
				}
			}
		}
	}()

	select {
	case port = <-portChan:
		t.Logf("Binary announced PORT=%d", port)
	case <-timeout:
		t.Fatal("Timeout waiting for PORT announcement")
	}

	// Give the server a moment to start listening
	time.Sleep(500 * time.Millisecond)

	// Connect to the gRPC server
	t.Log("Connecting to gRPC server...")
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)

	// Test: Send request with trace_id in metadata
	t.Run("TraceIDInMetadataPropagates", func(t *testing.T) {
		expectedTraceID := "integration-test-trace-12345"

		// Create context with trace_id in gRPC metadata
		md := metadata.New(map[string]string{
			pluginsdk.TraceIDMetadataKey: expectedTraceID,
		})
		ctx := metadata.NewOutgoingContext(context.Background(), md)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Make request
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "ap-southeast-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("Got response: $%.2f/month", resp.CostPerMonth)

		// Give time for logs to flush
		time.Sleep(100 * time.Millisecond)

		// Verify trace_id appears in stderr (log output)
		logOutput := stderrBuf.String()
		if !strings.Contains(logOutput, expectedTraceID) {
			t.Errorf("Expected trace_id %q in log output, but not found.\nLog output:\n%s",
				expectedTraceID, logOutput)
		}

		// Parse JSON log lines and verify trace_id field
		foundTraceID := false
		for _, line := range strings.Split(logOutput, "\n") {
			if line == "" {
				continue
			}
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				continue // Skip non-JSON lines
			}
			if traceID, ok := logEntry["trace_id"].(string); ok && traceID == expectedTraceID {
				foundTraceID = true
				t.Logf("Found trace_id in log entry: %s", line)

				// Also verify other expected fields
				if op, ok := logEntry["operation"].(string); ok && op == "GetProjectedCost" {
					t.Log("Verified operation field is present")
				}
				break
			}
		}

		if !foundTraceID {
			t.Errorf("trace_id %q not found as structured field in any log entry", expectedTraceID)
		}
	})

	t.Log("Trace ID propagation integration test completed!")
}

// TestIntegration_EKS_UseEast1_Binary performs end-to-end testing of EKS cost estimation.
//
// This integration test validates:
//   - EKS resource type support in Supports() RPC
//   - EKS cost calculation in GetProjectedCost() RPC
//   - Standard support pricing ($0.10/hour)
//   - Extended support pricing via SKU ($0.50/hour)
//   - Extended support pricing via tags
//   - Proper billing details mentioning control plane only
//
// Test coverage (task reference: T009):
//   - Validates EKS pricing data embedded correctly
//   - Confirms monthly cost calculation (hourly × 730)
//   - Verifies support type detection (standard vs extended)
//   - Tests both SKU and tag-based extended support detection
//
// Production scenarios validated:
//  1. Standard support (default): Common case for new clusters using default SKU
//  2. Extended support via SKU: Explicitly requesting extended tier via resource SKU
//  3. Extended support via tag: Infrastructure-as-code pattern using feature flags in tags
//
// Prerequisites:
//   - Go toolchain available for building
//   - us-east-1 binary with EKS pricing data
//
// Run with: go test -tags=integration ./internal/plugin/...
func TestIntegration_EKS_UseEast1_Binary(t *testing.T) {
	// Build the binary with region_use1 tag
	t.Log("Building us-east-1 binary for EKS testing...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_use1",
		"-o", "../../dist/test-finfocus-plugin-aws-public-us-east-1-eks",
		"../../cmd/finfocus-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, string(output))
	}

	// Ensure cleanup
	defer func() {
		if err := os.Remove("../../dist/test-finfocus-plugin-aws-public-us-east-1-eks"); err != nil {
			t.Logf("Warning: failed to cleanup test binary: %v", err)
		}
	}()

	// Start the binary
	t.Log("Starting us-east-1 binary...")
	cmd := exec.Command("../../dist/test-finfocus-plugin-aws-public-us-east-1-eks")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start binary: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("Warning: failed to kill process: %v", err)
		}
	}()

	// Read PORT announcement
	portChan := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PORT=") {
				port := strings.TrimPrefix(line, "PORT=")
				portChan <- port
				break
			}
		}
	}()

	// Wait for port announcement with timeout
	select {
	case port := <-portChan:
		t.Logf("Binary announced port: %s", port)

		// Connect to gRPC server
		conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("Failed to connect to gRPC server: %v", err)
		}
		defer func() {
			if err := conn.Close(); err != nil {
				t.Logf("Warning: failed to close connection: %v", err)
			}
		}()

		client := pbc.NewCostSourceServiceClient(conn)

		// Test 1: EKS Supports (standard support)
		t.Run("EKS_Supports_Standard", func(t *testing.T) {
			resp, err := client.Supports(context.Background(), &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "eks",
					Sku:          "cluster",
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("Supports() failed: %v", err)
			}

			if !resp.Supported {
				t.Errorf("EKS should be supported, got supported=false, reason: %s", resp.Reason)
			}
		})

		// Test 2: EKS GetProjectedCost (standard support)
		t.Run("EKS_GetProjectedCost_Standard", func(t *testing.T) {
			resp, err := client.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "eks",
					Sku:          "cluster",
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() failed: %v", err)
			}

			// Verify cost calculation: 0.10 * 730 = 73.00
			expectedCost := 0.10 * 730.0
			if resp.CostPerMonth != expectedCost {
				t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
			}

			if resp.UnitPrice != 0.10 {
				t.Errorf("UnitPrice = %v, want 0.10", resp.UnitPrice)
			}

			if resp.Currency != "USD" {
				t.Errorf("Currency = %q, want %q", resp.Currency, "USD")
			}

			// Verify billing detail
			expectedDetail := "EKS cluster (standard support), 730 hrs/month (control plane only, excludes worker nodes)"
			if resp.BillingDetail != expectedDetail {
				t.Errorf("BillingDetail = %q, want %q", resp.BillingDetail, expectedDetail)
			}
		})

		// Test 3: EKS GetProjectedCost (extended support via SKU)
		t.Run("EKS_GetProjectedCost_Extended_SKU", func(t *testing.T) {
			resp, err := client.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "eks",
					Sku:          "cluster-extended",
					Region:       "us-east-1",
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() failed: %v", err)
			}

			// Verify cost calculation: 0.50 * 730 = 365.00
			expectedCost := 0.50 * 730.0
			if resp.CostPerMonth != expectedCost {
				t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
			}

			if resp.UnitPrice != 0.50 {
				t.Errorf("UnitPrice = %v, want 0.50", resp.UnitPrice)
			}

			// Verify billing detail mentions extended support
			expectedDetail := "EKS cluster (extended support), 730 hrs/month (control plane only, excludes worker nodes)"
			if resp.BillingDetail != expectedDetail {
				t.Errorf("BillingDetail = %q, want %q", resp.BillingDetail, expectedDetail)
			}
		})

		// Test 4: EKS GetProjectedCost (extended support via tags)
		t.Run("EKS_GetProjectedCost_Extended_Tags", func(t *testing.T) {
			resp, err := client.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "eks",
					Sku:          "cluster",
					Region:       "us-east-1",
					Tags: map[string]string{
						"support_type": "extended",
					},
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() failed: %v", err)
			}

			// Verify cost calculation: 0.50 * 730 = 365.00
			expectedCost := 0.50 * 730.0
			if resp.CostPerMonth != expectedCost {
				t.Errorf("CostPerMonth = %v, want %v", resp.CostPerMonth, expectedCost)
			}

			// Verify billing detail mentions extended support
			expectedDetail := "EKS cluster (extended support), 730 hrs/month (control plane only, excludes worker nodes)"
			if resp.BillingDetail != expectedDetail {
				t.Errorf("BillingDetail = %q, want %q", resp.BillingDetail, expectedDetail)
			}
		})

	case <-time.After(portAnnouncementTimeout):
		t.Fatal("Timeout waiting for PORT announcement")
	}

	t.Log("EKS integration test completed!")
}

// TestIntegration_Lambda_UseEast1_Binary performs end-to-end testing of Lambda cost estimation.
func TestIntegration_Lambda_UseEast1_Binary(t *testing.T) {
	// Build the binary with region_use1 tag
	t.Log("Building us-east-1 binary for Lambda testing...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_use1",
		"-o", "../../dist/test-finfocus-plugin-aws-public-us-east-1-lambda",
		"../../cmd/finfocus-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, string(output))
	}

	// Ensure cleanup
	defer func() {
		if err := os.Remove("../../dist/test-finfocus-plugin-aws-public-us-east-1-lambda"); err != nil {
			t.Logf("Warning: failed to cleanup test binary: %v", err)
		}
	}()

	// Start the binary
	t.Log("Starting us-east-1 binary...")
	cmd := exec.Command("../../dist/test-finfocus-plugin-aws-public-us-east-1-lambda")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start binary: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("Warning: failed to kill process: %v", err)
		}
	}()

	// Read PORT announcement
	portChan := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PORT=") {
				port := strings.TrimPrefix(line, "PORT=")
				portChan <- port
				break
			}
		}
	}()

	// Wait for port announcement with timeout
	select {
	case port := <-portChan:
		t.Logf("Binary announced port: %s", port)

		// Connect to gRPC server
		conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("Failed to connect to gRPC server: %v", err)
		}
		defer func() {
			if err := conn.Close(); err != nil {
				t.Logf("Warning: failed to close connection: %v", err)
			}
		}()

		client := pbc.NewCostSourceServiceClient(conn)

		// Test: Lambda GetProjectedCost
		t.Run("Lambda_GetProjectedCost", func(t *testing.T) {
			resp, err := client.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "lambda",
					Sku:          "512",
					Region:       "us-east-1",
					Tags: map[string]string{
						"requests_per_month": "1000000",
						"avg_duration_ms":    "200",
					},
				},
			})

			if err != nil {
				t.Fatalf("GetProjectedCost() failed: %v", err)
			}

			// Expected: ~$1.87
			if resp.CostPerMonth <= 0 {
				t.Errorf("Expected positive cost, got %v", resp.CostPerMonth)
			}

			if !strings.Contains(resp.BillingDetail, "Lambda 512MB") {
				t.Errorf("Billing detail missing info: %s", resp.BillingDetail)
			}
		})

	case <-time.After(portAnnouncementTimeout):
		t.Fatal("Timeout waiting for PORT announcement")
	}

	t.Log("Lambda integration test completed!")
}

