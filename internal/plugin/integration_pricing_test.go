//go:build integration

package plugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// TestIntegration_PricingDifferentiation verifies that EC2 pricing correctly
// differentiates between platforms (Windows vs Linux vs RHEL vs SUSE), tenancies
// (Shared vs Dedicated), and architectures (x86 vs ARM).
//
// This test validates:
//  1. US1: Windows pricing > Linux pricing for the same instance type.
//  2. US2: Dedicated Instance pricing > Shared tenancy pricing.
//  3. US3: ARM (Graviton) pricing is different (usually lower) than x86.
//  4. FR-003: RHEL and SUSE platform pricing differentiation (premium over Linux).
//
// Note: AWS pricing structure varies by instance type. Many instance types (r5, m5.large,
// c5, c6i.large) have identical Linux/Windows Shared prices. T2/T3 smaller sizes and
// larger m5/m6i types (xlarge+) typically show the expected Windows license premium.
// This test uses t2.medium which consistently shows Windows > Linux pricing (~1.39x).
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//   - No AWS credentials required (uses embedded public pricing data)
//
// Test workflow:
//  1. Build the us-east-1 binary with region_use1 build tag
//  2. Start the binary and capture the PORT announcement from stdout
//  3. Connect to the gRPC server on the announced port
//  4. Execute OS differentiation tests (Linux vs Windows, RHEL, SUSE)
//  5. Execute tenancy differentiation tests (Shared vs Dedicated)
//  6. Execute architecture differentiation tests (x86 vs ARM)
//  7. Verify pricing ratios and billing detail strings
//
// Run with: go test -tags=integration ./internal/plugin/... -run TestIntegration_PricingDifferentiation
func TestIntegration_PricingDifferentiation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 1. Build the us-east-1 binary (T005)
	t.Log("Building us-east-1 binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "finfocus-plugin-aws-public-use1")

	// Use relative path from internal/plugin to cmd
	buildCmd := exec.Command("go", "build",
		"-tags", "region_use1",
		"-o", binPath,
		"../../cmd/finfocus-plugin-aws-public")

	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, string(output))
	}

	// 2. Start the binary (T006)
	t.Log("Starting plugin binary...")
	cmd := exec.Command(binPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	// Capture stderr for trace_id verification in logs
	var stderrBuf bytes.Buffer
	var stderrMu sync.Mutex
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start binary: %v", err)
	}
	defer func() {
		if cmd.Process == nil {
			return
		}

		// Send interrupt signal for graceful shutdown
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			t.Logf("Failed to send interrupt signal: %v", err)
		}

		// Wait for process to exit with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				// Process exited with non-zero status (expected after interrupt)
				t.Logf("Process exited: %v", err)
			}
		case <-time.After(2 * time.Second):
			// Timeout: force kill and reap
			t.Logf("Graceful shutdown timed out, forcing kill")
			if err := cmd.Process.Kill(); err != nil {
				t.Logf("Failed to kill process: %v", err)
			}
			// Wait to reap the process after kill
			if err := <-done; err != nil {
				t.Logf("Process exited after kill: %v", err)
			}
		}
	}()

	// 3. Capture PORT (T006)
	port, err := waitForPort(stdout, portAnnouncementTimeout)
	if err != nil {
		t.Fatalf("Failed to get port: %v", err)
	}
	t.Logf("Plugin started on PORT=%s", port)

	// 4. Connect via gRPC (T006)
	conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)

	// Helper function to create context with trace_id metadata
	ctxWithTraceID := func(traceID string) context.Context {
		md := metadata.New(map[string]string{
			pluginsdk.TraceIDMetadataKey: traceID,
		})
		return metadata.NewOutgoingContext(ctx, md)
	}

	// Helper function to verify trace_id appears in stderr logs
	verifyTraceIDInLogs := func(t *testing.T, traceID string) {
		t.Helper()
		// Give time for logs to flush
		time.Sleep(50 * time.Millisecond)

		stderrMu.Lock()
		logOutput := stderrBuf.String()
		stderrMu.Unlock()

		if !strings.Contains(logOutput, traceID) {
			t.Errorf("Expected trace_id %q in log output, but not found", traceID)
			return
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
			if tid, ok := logEntry["trace_id"].(string); ok && tid == traceID {
				foundTraceID = true
				break
			}
		}

		if !foundTraceID {
			t.Errorf("trace_id %q not found as structured field in any log entry", traceID)
		}
	}

	// --- User Story 1: OS Differentiation (T007-T011, T017, T019) ---
	t.Run("OS_Differentiation_Windows_vs_Linux", func(t *testing.T) {
		// Use t2.medium which consistently shows Windows > Linux pricing (~1.39x ratio).
		// Note: AWS pricing structure is complex - many instance types (r5, m5.large, c5)
		// have identical Linux/Windows Shared prices, while t2/t3 smaller sizes and
		// larger m5/m6i types show the expected Windows license premium.
		instanceType := "t2.medium"
		region := "us-east-1"

		// Linux Request (T007) with trace_id propagation
		linuxTraceID := "pricing-test-linux-" + instanceType
		linuxResp, err := client.GetProjectedCost(ctxWithTraceID(linuxTraceID), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          instanceType,
				Region:       region,
				Tags:         map[string]string{"platform": "linux"},
			},
		})
		if err != nil {
			t.Fatalf("Linux request failed: %v", err)
		}
		verifyTraceIDInLogs(t, linuxTraceID)

		// Windows Request (T008) with trace_id propagation
		windowsTraceID := "pricing-test-windows-" + instanceType
		windowsResp, err := client.GetProjectedCost(ctxWithTraceID(windowsTraceID), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          instanceType,
				Region:       region,
				Tags:         map[string]string{"platform": "windows"},
			},
		})
		if err != nil {
			t.Fatalf("Windows request failed: %v", err)
		}
		verifyTraceIDInLogs(t, windowsTraceID)

		t.Logf("Instance: %s, Linux: $%.2f (%s), Windows: $%.2f (%s)",
			instanceType, linuxResp.CostPerMonth, linuxResp.BillingDetail,
			windowsResp.CostPerMonth, windowsResp.BillingDetail)

		// Assertions (T009, T010, T011)
		if linuxResp.CostPerMonth <= 0 {
			t.Errorf("Linux price should be > 0 (SC-005)")
		}
		if windowsResp.CostPerMonth <= linuxResp.CostPerMonth {
			t.Errorf("Windows price ($%.2f) should be higher than Linux price ($%.2f) (SC-001)",
				windowsResp.CostPerMonth, linuxResp.CostPerMonth)
		}

		// Ratio Check (T017, SC-002)
		ratio := windowsResp.CostPerMonth / linuxResp.CostPerMonth
		if ratio < 1.3 || ratio > 2.0 {
			t.Errorf("Windows/Linux ratio %.2fx outside expected [1.3, 2.0] range", ratio)
		}

		if !strings.Contains(linuxResp.BillingDetail, "Linux") {
			t.Errorf("Linux billing detail should mention 'Linux': %s", linuxResp.BillingDetail)
		}
		if !strings.Contains(windowsResp.BillingDetail, "Windows") {
			t.Errorf("Windows billing detail should mention 'Windows': %s", windowsResp.BillingDetail)
		}

		// RHEL and SUSE (T019, FR-003) - use t3.medium for these as they show clear differentiation
		rhelSuseInstance := "t3.medium"
		baselineTraceID := "pricing-test-baseline-" + rhelSuseInstance
		linuxBaselineResp, err := client.GetProjectedCost(ctxWithTraceID(baselineTraceID), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          rhelSuseInstance,
				Region:       region,
				Tags:         map[string]string{"platform": "linux"},
			},
		})
		if err != nil {
			t.Fatalf("Linux baseline request failed: %v", err)
		}
		verifyTraceIDInLogs(t, baselineTraceID)

		platforms := []string{"rhel", "suse"}
		for _, p := range platforms {
			platformTraceID := "pricing-test-" + p + "-" + rhelSuseInstance
			resp, err := client.GetProjectedCost(ctxWithTraceID(platformTraceID), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          rhelSuseInstance,
					Region:       region,
					Tags:         map[string]string{"platform": p},
				},
			})
			if err != nil {
				t.Errorf("%s request failed: %v", p, err)
				continue
			}
			verifyTraceIDInLogs(t, platformTraceID)

			t.Logf("%s (%s) monthly cost: $%.2f (Linux baseline: $%.2f)", p, rhelSuseInstance, resp.CostPerMonth, linuxBaselineResp.CostPerMonth)
			if resp.CostPerMonth <= 0 {
				t.Errorf("%s price should be > 0", p)
			}
			if resp.CostPerMonth <= linuxBaselineResp.CostPerMonth {
				t.Errorf("%s price ($%.2f) should be higher than Linux baseline price ($%.2f)", p, resp.CostPerMonth, linuxBaselineResp.CostPerMonth)
			}
		}
	})

	// --- User Story 2: Tenancy Differentiation (T012-T015) ---
	t.Run("Tenancy_Differentiation_Shared_vs_Dedicated", func(t *testing.T) {
		instanceType := "m5.large"
		region := "us-east-1"

		// Shared Request (T012) with trace_id propagation
		sharedTraceID := "pricing-test-shared-" + instanceType
		sharedResp, err := client.GetProjectedCost(ctxWithTraceID(sharedTraceID), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          instanceType,
				Region:       region,
				Tags:         map[string]string{"tenancy": "shared", "platform": "linux"},
			},
		})
		if err != nil {
			t.Fatalf("Shared tenancy request failed: %v", err)
		}
		verifyTraceIDInLogs(t, sharedTraceID)

		// Dedicated Request (T013) with trace_id propagation
		dedicatedTraceID := "pricing-test-dedicated-" + instanceType
		dedicatedResp, err := client.GetProjectedCost(ctxWithTraceID(dedicatedTraceID), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          instanceType,
				Region:       region,
				Tags:         map[string]string{"tenancy": "dedicated", "platform": "linux"},
			},
		})
		if err != nil {
			t.Fatalf("Dedicated tenancy request failed: %v", err)
		}
		verifyTraceIDInLogs(t, dedicatedTraceID)

		t.Logf("Instance: %s, Shared: $%.2f, Dedicated: $%.2f", instanceType, sharedResp.CostPerMonth, dedicatedResp.CostPerMonth)

		// Assertions (T014, T015)
		if dedicatedResp.CostPerMonth <= sharedResp.CostPerMonth {
			t.Errorf("Dedicated price ($%.2f) should be higher than Shared price ($%.2f) (SC-003)",
				dedicatedResp.CostPerMonth, sharedResp.CostPerMonth)
		}

		if !strings.Contains(dedicatedResp.BillingDetail, "Dedicated") {
			t.Errorf("Dedicated billing detail should mention 'Dedicated': %s", dedicatedResp.BillingDetail)
		}
	})

	// --- User Story 3: Architecture Differentiation (T016) ---
	t.Run("Architecture_Differentiation_x86_vs_ARM", func(t *testing.T) {
		region := "us-east-1"

		// x86 Request (t3.medium) with trace_id propagation
		x86TraceID := "pricing-test-x86-t3medium"
		x86Resp, err := client.GetProjectedCost(ctxWithTraceID(x86TraceID), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.medium",
				Region:       region,
			},
		})
		if err != nil {
			t.Fatalf("x86 request failed: %v", err)
		}
		verifyTraceIDInLogs(t, x86TraceID)

		// ARM Request (t4g.medium) with trace_id propagation
		armTraceID := "pricing-test-arm-t4gmedium"
		armResp, err := client.GetProjectedCost(ctxWithTraceID(armTraceID), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t4g.medium",
				Region:       region,
			},
		})
		if err != nil {
			t.Fatalf("ARM request failed: %v", err)
		}
		verifyTraceIDInLogs(t, armTraceID)

		t.Logf("x86 (t3.medium): $%.2f, ARM (t4g.medium): $%.2f", x86Resp.CostPerMonth, armResp.CostPerMonth)

		// Graviton (t4g) is cheaper than T3
		if armResp.CostPerMonth >= x86Resp.CostPerMonth {
			t.Errorf("ARM (t4g.medium) price ($%.2f) should be lower than x86 (t3.medium) price ($%.2f)",
				armResp.CostPerMonth, x86Resp.CostPerMonth)
		}
	})
}
