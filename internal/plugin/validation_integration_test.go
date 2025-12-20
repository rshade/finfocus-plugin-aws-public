//go:build integration

package plugin_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// portAnnouncementTimeout is the maximum time to wait for the plugin to announce its listening port.
// This can be overridden via the PULUMICOST_PORT_TIMEOUT environment variable (in milliseconds)
// for slower CI environments or resource-constrained systems.
var portAnnouncementTimeout = getPortAnnouncementTimeout()

// getPortAnnouncementTimeout reads the timeout from environment variable or returns default (15 seconds).
// The environment variable PULUMICOST_PORT_TIMEOUT should specify the timeout in milliseconds.
func getPortAnnouncementTimeout() time.Duration {
	if envTimeout := os.Getenv("PULUMICOST_PORT_TIMEOUT"); envTimeout != "" {
		if ms, err := strconv.ParseInt(envTimeout, 10, 64); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	// Default: 15 seconds for most environments
	return 15 * time.Second
}

func waitForPort(stdout io.Reader, timeout time.Duration) (string, error) {
	scanner := bufio.NewScanner(stdout)
	portRegex := regexp.MustCompile(`^PORT=(\d+)$`)
	portChan := make(chan string, 1)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if matches := portRegex.FindStringSubmatch(line); len(matches) == 2 {
				portChan <- matches[1]
				return
			}
		}
	}()

	select {
	case p := <-portChan:
		return p, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for PORT announcement")
	}
}

// TestIntegration_Validation verifies that the plugin enforces strict validation
// for all required parameters in GetProjectedCost (T007).
func TestIntegration_Validation(t *testing.T) {
	// Build the binary with default (fallback) pricing
	// Use a unique binary path per test to avoid collisions when tests run in parallel
	t.Log("Building plugin binary...")
	// Sanitize test name (remove slashes from subtest names like "TestIntegration_Validation/subtest")
	sanitizedTestName := strings.ReplaceAll(t.Name(), "/", "_")
	binPath := fmt.Sprintf("../../dist/test-pulumicost-plugin-validation-%s", sanitizedTestName)
	buildCmd := exec.Command("go", "build",
		"-o", binPath,
		"../../cmd/pulumicost-plugin-aws-public")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
	// Clean up binary after test completes (including subtests in parallel)
	t.Cleanup(func() {
		os.Remove(binPath)
	})

	// Start the binary
	cmd := exec.Command(binPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start binary: %v", err)
	}
	defer cmd.Process.Kill()

	// Capture port
	port, err := waitForPort(stdout, portAnnouncementTimeout)
	if err != nil {
		t.Fatalf("Failed to get port: %v", err)
	}

	// Connect using grpc.NewClient (grpc.Dial is deprecated since v1.63)
	conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)

	tests := []struct {
		name     string
		resource *pbc.ResourceDescriptor
		wantCode codes.Code
	}{
		{
			name: "missing provider",
			resource: &pbc.ResourceDescriptor{
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing resource_type",
			resource: &pbc.ResourceDescriptor{
				Provider: "aws",
				Sku:      "t3.micro",
				Region:   "us-east-1",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty request",
			resource: nil,
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: tt.resource,
			})
			if err == nil {
				t.Error("Expected error but got nil")
				return
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Expected gRPC status error, got %T", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("Code = %v, want %v. Message: %s", st.Code(), tt.wantCode, st.Message())
			}
		})
	}
}
