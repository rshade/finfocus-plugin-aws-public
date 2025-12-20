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

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestIntegration_CACentral1_Binary performs end-to-end testing of the ca-central-1 binary.
//
// This integration test validates the complete gRPC plugin lifecycle for Canada (Central):
//   - Binary compilation with region_cac1 build tag
//   - PORT announcement via stdout (gRPC protocol requirement)
//   - Name() RPC returning correct plugin identifier
//   - GetProjectedCost() for EC2 instances in ca-central-1
//   - GetProjectedCost() for EBS volumes with size tags
//   - Region mismatch rejection (us-east-1 request to ca-central-1 binary)
//
// Test coverage validates:
//   - Pricing data embedded correctly for Canada region
//   - Monthly cost calculation (hourly × 730)
//   - Region validation rejects cross-region requests
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//   - Pricing data generated for ca-central-1
//
// Run with: go test -tags=integration ./internal/plugin/... -run TestIntegration_CACentral1
func TestIntegration_CACentral1_Binary(t *testing.T) {
	// Build the binary with region_cac1 tag
	t.Log("Building ca-central-1 binary...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_cac1",
		"-o", "../../dist/test-pulumicost-plugin-aws-public-ca-central-1",
		"../../cmd/pulumicost-plugin-aws-public")
	buildCmd.Dir, _ = os.Getwd()
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("../../dist/test-pulumicost-plugin-aws-public-ca-central-1")

	// Start the binary
	t.Log("Starting ca-central-1 binary...")
	cmd := exec.Command("../../dist/test-pulumicost-plugin-aws-public-ca-central-1")
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

	timeout := time.After(5 * time.Second)
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
		if resp.Name != "pulumicost-plugin-aws-public" {
			t.Errorf("Expected name 'pulumicost-plugin-aws-public', got %q", resp.Name)
		}
	})

	// Test 2: GetProjectedCost() - t3.micro in ca-central-1
	t.Run("GetProjectedCost_t3micro_Canada", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "ca-central-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("t3.micro in ca-central-1: $%.2f/month (hourly: $%.4f, currency: %s)",
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

	// Test 3: GetProjectedCost() - m5.large in ca-central-1
	t.Run("GetProjectedCost_m5large_Canada", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "m5.large",
				Region:       "ca-central-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("m5.large in ca-central-1: $%.2f/month (hourly: $%.4f)",
			resp.CostPerMonth, resp.UnitPrice)

		if resp.UnitPrice <= 0 {
			t.Errorf("Expected positive unit price for m5.large, got %.4f", resp.UnitPrice)
		}
	})

	// Test 4: GetProjectedCost() - EBS gp3 in ca-central-1
	t.Run("GetProjectedCost_gp3_Canada", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ebs",
				Sku:          "gp3",
				Region:       "ca-central-1",
				Tags: map[string]string{
					"size": "100",
				},
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("gp3 100GB in ca-central-1: $%.2f/month (per-GB: $%.4f)",
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

	// Test 5: GetProjectedCost() - EBS gp2 in ca-central-1
	t.Run("GetProjectedCost_gp2_Canada", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ebs",
				Sku:          "gp2",
				Region:       "ca-central-1",
				Tags: map[string]string{
					"size": "50",
				},
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}
		t.Logf("gp2 50GB in ca-central-1: $%.2f/month (per-GB: $%.4f)",
			resp.CostPerMonth, resp.UnitPrice)

		if resp.UnitPrice <= 0 {
			t.Errorf("Expected positive per-GB price for gp2, got %.4f", resp.UnitPrice)
		}
	})

	// Test 6: GetProjectedCost() - Wrong region (should fail)
	t.Run("GetProjectedCost_WrongRegion_USEast1", func(t *testing.T) {
		_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		})
		if err == nil {
			t.Error("Expected error for wrong region (us-east-1), got nil")
		} else {
			t.Logf("Correctly rejected us-east-1 request: %v", err)
		}
	})

	// Test 7: GetProjectedCost() - Wrong region sa-east-1 (should fail)
	t.Run("GetProjectedCost_WrongRegion_SAEast1", func(t *testing.T) {
		_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "sa-east-1",
			},
		})
		if err == nil {
			t.Error("Expected error for wrong region (sa-east-1), got nil")
		} else {
			t.Logf("Correctly rejected sa-east-1 request: %v", err)
		}
	})

	t.Log("ca-central-1 integration test completed successfully!")
}
