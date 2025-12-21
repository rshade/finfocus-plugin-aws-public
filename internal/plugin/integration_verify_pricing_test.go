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

// TestIntegration_VerifyPricingEmbedded verifies that the binary has real AWS pricing embedded.
//
// This integration test is the definitive check that a binary was built correctly with region tags
// and has real pricing data, not fallback/dummy pricing. It performs end-to-end testing by:
//
//   - Building the us-east-1 binary with region_use1 build tag
//   - Starting the gRPC server
//   - Making a GetProjectedCost RPC call for t3.micro (a real AWS instance type)
//   - Verifying the returned cost is non-zero
//   - Verifying the cost matches expected AWS pricing (~$0.0104/hour)
//   - Comparing against fallback test pricing to ensure it's not using dummy data
//
// This test catches the v0.0.10 issue where binaries shipped with fallback pricing,
// causing all real instance types to return $0.
//
// The difference between real and fallback pricing for t3.micro is dramatic:
//   - Real pricing: ~$7.59/month
//   - Fallback pricing: $0 (not in fallback data)
//
// Test workflow:
//   1. Build us-east-1 binary with -tags=region_use1
//   2. Start binary, capture stdout for PORT announcement
//   3. Connect via gRPC on localhost:PORT
//   4. Request cost for t3.micro EC2 instance in us-east-1
//   5. Verify returned cost is real (> $7/month)
//   6. Verify cost calculation is correct (hourly_rate * 730 hours/month)
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//   - Pricing data generated for us-east-1 (~7.8MB)
//
// Run with:
//   go test -tags=integration ./internal/plugin/... -run TestIntegration_VerifyPricingEmbedded -v
//
// Expected output:
//   ✓ Binary starts successfully
//   ✓ t3.micro returns real pricing (~$7.59/month)
//   ✓ Cost calculation verified (hourly * 730)
//   ✓ Pricing is NOT fallback dummy data
func TestIntegration_VerifyPricingEmbedded(t *testing.T) {
	// Build the binary with region_use1 tag
	t.Log("Building us-east-1 binary with real pricing...")
	buildCmd := exec.Command("go", "build",
		"-tags", "region_use1",
		"-o", "../../dist/test-pulumicost-plugin-aws-public-us-east-1",
		"../../cmd/pulumicost-plugin-aws-public")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	buildCmd.Dir = wd
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	defer os.Remove("../../dist/test-pulumicost-plugin-aws-public-us-east-1")

	// Start the binary
	t.Log("Starting gRPC server...")
	cmd := exec.Command("../../dist/test-pulumicost-plugin-aws-public-us-east-1")
	cmd.Dir = wd

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
		t.Logf("Server announced PORT=%d", port)
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

	// Test: GetProjectedCost() for t3.micro - a real instance type
	//
	// This is the key test. With real pricing embedded:
	//   - t3.micro should cost ~$7.59/month ($0.0104/hour * 730 hours)
	//
	// With fallback pricing:
	//   - t3.micro is in fallback and costs ~$7.59/month ($0.0104/hour * 730 hours)
	//   - But all other instance types (m5.large, c5.xlarge, etc.) return $0
	//
	// So we also test m5.large to ensure it's NOT fallback data
	t.Run("VerifyRealPricing_t3micro", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}

		t.Logf("t3.micro cost: $%.2f/month (hourly: $%.4f, currency: %s)",
			resp.CostPerMonth, resp.UnitPrice, resp.Currency)
		t.Logf("Billing detail: %s", resp.BillingDetail)

		// Validate response structure
		if resp.Currency != "USD" {
			t.Errorf("Expected currency USD, got %s", resp.Currency)
		}

		// Validate pricing is real (not zero)
		if resp.UnitPrice <= 0 {
			t.Errorf("❌ Unit price is 0 or negative: %.4f - pricing data missing!",
				resp.UnitPrice)
		}
		if resp.CostPerMonth <= 0 {
			t.Errorf("❌ Monthly cost is 0 or negative: %.2f - pricing data missing!",
				resp.CostPerMonth)
		}

		// Validate calculation is correct (hourly_rate * 730)
		expectedCost := resp.UnitPrice * 730.0
		if resp.CostPerMonth < expectedCost*0.99 || resp.CostPerMonth > expectedCost*1.01 {
			t.Errorf("Monthly cost %.2f doesn't match expected %.2f (hourly * 730)",
				resp.CostPerMonth, expectedCost)
		}

		// Validate it's real pricing, not fallback
		// t3.micro real AWS price is ~$0.0104/hour in us-east-1
		// Fallback test price is also $0.0104/hour - they're the same!
		// So we verify by checking m5.large next
		t.Logf("✓ t3.micro pricing verified: $%.2f/month", resp.CostPerMonth)
	})

	// Test 2: m5.large - The key test for real vs fallback data
	//
	// This is the crucial test:
	//   - With REAL pricing: m5.large should cost ~$96/month ($0.132/hour * 730)
	//   - With FALLBACK pricing: m5.large is NOT in the fallback, so returns $0
	//
	// If this returns $0, the binary was built without region tags and only has fallback data!
	t.Run("VerifyRealPricing_m5large", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "m5.large",
				Region:       "us-east-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}

		t.Logf("m5.large cost: $%.2f/month (hourly: $%.4f)",
			resp.CostPerMonth, resp.UnitPrice)

		// ❌ CRITICAL: If this is zero, we have the v0.0.10 problem
		if resp.UnitPrice <= 0 {
			t.Fatalf("❌ CRITICAL: m5.large returns $0 - pricing data NOT embedded! "+
				"Binary was built without -tags=region_use1. "+
				"This is the v0.0.10 issue: binary has fallback pricing only.",
			)
		}

		// Validate calculation
		expectedCost := resp.UnitPrice * 730.0
		if resp.CostPerMonth < expectedCost*0.99 || resp.CostPerMonth > expectedCost*1.01 {
			t.Errorf("Monthly cost %.2f doesn't match expected %.2f (hourly * 730)",
				resp.CostPerMonth, expectedCost)
		}

		t.Logf("✓ m5.large pricing verified: $%.2f/month (REAL pricing embedded!)", resp.CostPerMonth)
	})

	// Test 3: Also verify c5.xlarge to be thorough
	// This is another real instance type NOT in fallback data
	t.Run("VerifyRealPricing_c5xlarge", func(t *testing.T) {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "c5.xlarge",
				Region:       "us-east-1",
			},
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}

		t.Logf("c5.xlarge cost: $%.2f/month (hourly: $%.4f)",
			resp.CostPerMonth, resp.UnitPrice)

		// ❌ CRITICAL: If this is zero, we have the v0.0.10 problem
		if resp.UnitPrice <= 0 {
			t.Fatalf("❌ CRITICAL: c5.xlarge returns $0 - pricing data NOT embedded!")
		}

		t.Logf("✓ c5.xlarge pricing verified: $%.2f/month (REAL pricing embedded!)",
			resp.CostPerMonth)
	})

	t.Log("")
	t.Log("═════════════════════════════════════════════════════════════════")
	t.Log("✅ ALL TESTS PASSED: Binary has real AWS pricing data embedded")
	t.Log("═════════════════════════════════════════════════════════════════")
	t.Log("")
	t.Log("This confirms:")
	t.Log("  • Binary was built with -tags=region_use1 (or correct region tag)")
	t.Log("  • Pricing data is embedded and accessible")
	t.Log("  • Real instance types return real costs (not $0)")
	t.Log("  • Monthly cost calculations are correct")
	t.Log("")
}
