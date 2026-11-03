//go:build integration

package integration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
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
		// Reap the process before closing the pipes. Closing them first
		// makes Wait return a closed-pipe error from the stdout copy.
		stopPluginProcess(t, cmd)
		stdoutW.Close()
		stderrW.Close()
	}()

	// A pipe read blocks until the next write, so copy in the background
	// and wait until the PORT= line is copied. The old ticker called Read
	// in a loop and never reached the PORT check.
	stdoutBuf := newPortBuf()
	var stderrBuf streamBuf
	go func() { _, _ = io.Copy(stdoutBuf, stdoutR) }()
	go func() { _, _ = io.Copy(&stderrBuf, stderrR) }()

	var port string
	select {
	case port = <-stdoutBuf.port:
		t.Logf("Plugin listening on port %s", port)
	case <-time.After(5 * time.Second):
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

// portBuf copies plugin stdout and reports the PORT= announcement once.
type portBuf struct {
	streamBuf
	re   *regexp.Regexp
	port chan string
	once sync.Once
}

func newPortBuf() *portBuf {
	return &portBuf{
		re:   regexp.MustCompile(`PORT=(\d+)`),
		port: make(chan string, 1),
	}
}

func (p *portBuf) Write(data []byte) (int, error) {
	n, err := p.streamBuf.Write(data)
	if matches := p.re.FindStringSubmatch(p.String()); len(matches) > 1 {
		p.once.Do(func() { p.port <- matches[1] })
	}
	return n, err
}

// streamBuf is a bytes.Buffer that can be written and read from different goroutines.
type streamBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *streamBuf) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(data)
}

func (s *streamBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// stopPluginProcess asks the plugin to exit and reaps it.
//
// SIGTERM is sent first. Wait runs in one goroutine so the process is
// reaped exactly once, and the caller returns as soon as it exits.
// 200ms allows the gRPC server to shut down gracefully before Kill.
// Cleanup problems are written with t.Logf, so they show up under -v
// and do not fail the test. "already finished" is expected when the
// process exits before the next signal.
func stopPluginProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil && !processAlreadyFinished(err) {
		t.Logf("cleanup: SIGTERM failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		logCleanupWait(t, "SIGTERM", err)
	case <-time.After(200 * time.Millisecond):
		if err := cmd.Process.Kill(); err != nil && !processAlreadyFinished(err) {
			t.Logf("cleanup: Kill failed: %v", err)
		}
		logCleanupWait(t, "Kill", <-done)
	}
}

func processAlreadyFinished(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already finished")
}

func logCleanupWait(t *testing.T, step string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		t.Logf("cleanup: process wait after %s exited with code %d: %v", step, exitErr.ExitCode(), err)
		return
	}
	t.Logf("cleanup: process wait after %s failed: %v", step, err)
}
