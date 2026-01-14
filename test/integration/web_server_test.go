//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebServer performs an end-to-end integration test of the plugin's HTTP web server.
// This test builds the plugin binary, starts it with web serving enabled, waits for it to
// listen on a port, and then makes an HTTP request to the GetProjectedCost endpoint.
// It verifies that the plugin can serve cost estimation requests over HTTP.
//
// The test covers:
// - Binary compilation with region-specific build tags
// - Plugin startup and port allocation
// - HTTP endpoint availability and response format
// - Cost calculation for a sample EC2 instance (t3.micro)
//
// This test is skipped in short test mode due to its integration nature.
func TestWebServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the plugin binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "plugin-server")

	// We build the main package. assuming we are in test/integration, so main is ../../cmd/finfocus-plugin-aws-public
	rootDir, err := filepath.Abs("../..")
	require.NoError(t, err)

	t.Log("Building plugin binary...")
	cmdBuild := exec.Command("go", "build", "-tags", "region_use1", "-o", binaryPath, "./cmd/finfocus-plugin-aws-public")
	cmdBuild.Dir = rootDir
	output, err := cmdBuild.CombinedOutput()
	require.NoError(t, err, "Build failed: %s", string(output))

	// Run the plugin
	t.Log("Starting plugin server...")
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "FINFOCUS_PLUGIN_WEB_ENABLED=true")
	
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		stdoutW.Close()
		stderrW.Close()
		_ = cmd.Process.Kill()
	}()

	// Wait for port
	var port string
	var stdoutBuf, stderrBuf strings.Builder
	done := make(chan struct{})
	go func() {
		defer stdoutR.Close()
		defer stderrR.Close()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.After(5 * time.Second)

		buf := make([]byte, 1024)
		for {
			select {
			case <-done:
				return
			case <-timeout:
				return
			case <-ticker.C:
				// Read available stdout data
				for {
					n, err := stdoutR.Read(buf)
					if err != nil {
						if err != io.EOF {
							t.Logf("Error reading stdout: %v", err)
						}
						break
					}
					stdoutBuf.Write(buf[:n])
				}

				// Read available stderr data
				for {
					n, err := stderrR.Read(buf)
					if err != nil {
						if err != io.EOF {
							t.Logf("Error reading stderr: %v", err)
						}
						break
					}
					stderrBuf.Write(buf[:n])
				}

				out := stdoutBuf.String()
				if strings.Contains(out, "PORT=") {
					re := regexp.MustCompile(`PORT=(\d+)`)
					matches := re.FindStringSubmatch(out)
					if len(matches) > 1 {
						port = matches[1]
						close(done)
						return
					}
				}
			}
		}
	}()

	select {
	case <-done:
		t.Logf("Plugin listening on port %s", port)
	case <-time.After(6 * time.Second):
		t.Fatalf("Timed out waiting for port. Stdout: %s, Stderr: %s", stdoutBuf.String(), stderrBuf.String())
	}

	// Make HTTP Request
	url := fmt.Sprintf("http://localhost:%s/finfocus.v1.CostSourceService/GetProjectedCost", port)
	reqBody := []byte(`{
		"resource": {
			"provider": "aws",
			"resource_type": "ec2",
			"sku": "t3.micro",
			"region": "us-east-1"
		}
	}`)

	t.Logf("Sending POST request to %s", url)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("Response Status: %s", resp.Status)
	t.Logf("Response Body: %s", string(body))

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Fallback pricing should return a cost for t3.micro
	assert.Contains(t, string(body), "costPerMonth")
	assert.Contains(t, string(body), "USD")

	// Test GetPluginInfo
	infoUrl := fmt.Sprintf("http://localhost:%s/finfocus.v1.CostSourceService/GetPluginInfo", port)
	t.Logf("Sending POST request to %s", infoUrl)
	// Empty JSON body for GetPluginInfo
	respInfo, err := http.Post(infoUrl, "application/json", bytes.NewBuffer([]byte("{}")))
	require.NoError(t, err)
	defer respInfo.Body.Close()

	bodyInfo, err := io.ReadAll(respInfo.Body)
	require.NoError(t, err)

	t.Logf("Response Status: %s", respInfo.Status)
	t.Logf("Response Body: %s", string(bodyInfo))

	assert.Equal(t, http.StatusOK, respInfo.StatusCode)
	assert.Contains(t, string(bodyInfo), "finfocus-plugin-aws-public")
}
