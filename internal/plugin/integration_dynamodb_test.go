//go:build integration

package plugin_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// TestIntegration_DynamoDB_Provisioned performs end-to-end testing of DynamoDB provisioned mode cost estimation.
//
// This integration test validates the complete gRPC plugin lifecycle for DynamoDB provisioned tables:
//   - Binary compilation with region_use1 build tag
//   - PORT announcement via stdout (gRPC protocol requirement)
//   - GetProjectedCost() for DynamoDB provisioned tables with RCU/WCU/storage tags
//   - Verification of non-zero cost calculation
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//
// Run with: go test -tags=integration,region_use1 ./internal/plugin/... -run TestIntegration_DynamoDB_Provisioned
func TestIntegration_DynamoDB_Provisioned(t *testing.T) {
	// Build the binary with region_use1 tag
	t.Log("Building binary with region_use1 tag...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_use1",
		"-o", "../../dist/test-finfocus-plugin-aws-public-region-use1",
		"../../cmd/finfocus-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("../../dist/test-finfocus-plugin-aws-public-region-use1")

	// Start the binary
	t.Log("Starting binary...")
	cmd := exec.Command("../../dist/test-finfocus-plugin-aws-public-region-use1")
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

	// Connect to gRPC server
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)

	// Test DynamoDB provisioned cost estimation
	md := metadata.New(map[string]string{
		pluginsdk.TraceIDMetadataKey: "test-dynamodb-provisioned",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "dynamodb",
			Sku:          "provisioned",
			Region:       "us-east-1",
			Tags: map[string]string{
				"read_capacity_units":  "100",
				"write_capacity_units": "50",
				"storage_gb":           "50",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify non-zero cost
	if resp.CostPerMonth <= 0 {
		t.Errorf("Expected positive CostPerMonth, got: %v", resp.CostPerMonth)
	}

	if resp.UnitPrice <= 0 {
		t.Errorf("Expected positive UnitPrice, got: %v", resp.UnitPrice)
	}

	if resp.BillingDetail == "" {
		t.Error("Expected non-empty BillingDetail")
	}

	t.Logf("DynamoDB provisioned cost: $%.2f/month, unit price: $%.8f, detail: %s",
		resp.CostPerMonth, resp.UnitPrice, resp.BillingDetail)
}

// TestIntegration_DynamoDB_OnDemand performs end-to-end testing of DynamoDB on-demand mode cost estimation.
//
// This integration test validates the complete gRPC plugin lifecycle for DynamoDB on-demand tables:
//   - Binary compilation with region_use1 build tag
//   - PORT announcement via stdout (gRPC protocol requirement)
//   - GetProjectedCost() for DynamoDB on-demand tables with request/storage tags
//   - Verification of non-zero cost calculation
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//
// Run with: go test -tags=integration,region_use1 ./internal/plugin/... -run TestIntegration_DynamoDB_OnDemand
func TestIntegration_DynamoDB_OnDemand(t *testing.T) {
	// Build the binary with region_use1 tag
	t.Log("Building binary with region_use1 tag...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_use1",
		"-o", "../../dist/test-finfocus-plugin-aws-public-region-use1-ondemand",
		"../../cmd/finfocus-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("../../dist/test-finfocus-plugin-aws-public-region-use1-ondemand")

	// Start the binary
	t.Log("Starting binary...")
	cmd := exec.Command("../../dist/test-finfocus-plugin-aws-public-region-use1-ondemand")
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

	// Connect to gRPC server
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)

	// Test DynamoDB on-demand cost estimation
	md := metadata.New(map[string]string{
		pluginsdk.TraceIDMetadataKey: "test-dynamodb-ondemand",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "dynamodb",
			Sku:          "on-demand",
			Region:       "us-east-1",
			Tags: map[string]string{
				"read_requests_per_month":  "10000000",
				"write_requests_per_month": "1000000",
				"storage_gb":               "50",
			},
		},
	})

	if err != nil {
		t.Fatalf("GetProjectedCost() returned error: %v", err)
	}

	// Verify non-zero cost
	if resp.CostPerMonth <= 0 {
		t.Errorf("Expected positive CostPerMonth, got: %v", resp.CostPerMonth)
	}

	if resp.UnitPrice <= 0 {
		t.Errorf("Expected positive UnitPrice, got: %v", resp.UnitPrice)
	}

	if resp.BillingDetail == "" {
		t.Error("Expected non-empty BillingDetail")
	}

	t.Logf("DynamoDB on-demand cost: $%.2f/month, unit price: $%.8f, detail: %s",
		resp.CostPerMonth, resp.UnitPrice, resp.BillingDetail)
}
