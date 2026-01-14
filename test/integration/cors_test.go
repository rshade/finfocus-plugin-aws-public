//go:build integration

package integration

import (
	"bufio"
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

// waitForPort parses PORT=XXXX from the reader
func waitForPort(t *testing.T, r io.Reader) string {
	scanner := bufio.NewScanner(r)
	timeout := time.After(10 * time.Second)
	found := make(chan string)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			// t.Logf("Plugin Output: %s", line) // Debug logging
			if strings.Contains(line, "PORT=") {
				re := regexp.MustCompile(`PORT=(\d+)`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					found <- matches[1]
					return
				}
			}
		}
	}()

	select {
	case port := <-found:
		return port
	case <-timeout:
		return ""
	}
}

func TestCORS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the plugin binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "plugin-server")
	rootDir, err := filepath.Abs("../..")
	require.NoError(t, err)

	cmdBuild := exec.Command("go", "build", "-tags", "region_use1", "-o", binaryPath, "./cmd/finfocus-plugin-aws-public")
	cmdBuild.Dir = rootDir
	output, err := cmdBuild.CombinedOutput()
	require.NoError(t, err, "Build failed: %s", string(output))

	tests := []struct {
		name              string
		env               []string
		origin            string
		expectCors        bool
		expectOrigin      string
		expectMaxAge      string
		expectCredentials bool
	}{
		{
			name: "Basic CORS Success",
			env: []string{
				"FINFOCUS_PLUGIN_WEB_ENABLED=true",
				"FINFOCUS_CORS_ALLOWED_ORIGINS=http://localhost:3000",
			},
			origin:       "http://localhost:3000",
			expectCors:   true,
			expectOrigin: "http://localhost:3000",
			expectMaxAge: "86400", // Default
		},
		{
			name: "CORS No Match",
			env: []string{
				"FINFOCUS_PLUGIN_WEB_ENABLED=true",
				"FINFOCUS_CORS_ALLOWED_ORIGINS=http://localhost:3000",
			},
			origin:     "http://evil.com",
			expectCors: false,
		},
		{
			name: "Custom Max Age",
			env: []string{
				"FINFOCUS_PLUGIN_WEB_ENABLED=true",
				"FINFOCUS_CORS_ALLOWED_ORIGINS=http://localhost:3000",
				"FINFOCUS_CORS_MAX_AGE=60",
			},
			origin:       "http://localhost:3000",
			expectCors:   true,
			expectOrigin: "http://localhost:3000",
			expectMaxAge: "60",
		},
		{
			name: "Credentials Success",
			env: []string{
				"FINFOCUS_PLUGIN_WEB_ENABLED=true",
				"FINFOCUS_CORS_ALLOWED_ORIGINS=http://localhost:3000",
				"FINFOCUS_CORS_ALLOW_CREDENTIALS=true",
			},
			origin:            "http://localhost:3000",
			expectCors:        true,
			expectOrigin:      "http://localhost:3000",
			expectCredentials: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start plugin with specific env
			cmd := exec.Command(binaryPath)
			cmd.Env = append(os.Environ(), tt.env...)

			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)

			err = cmd.Start()
			require.NoError(t, err)

			defer func() {
				_ = cmd.Process.Kill()
			}()

			// Wait for port
			port := waitForPort(t, stdout)
			require.NotEmpty(t, port, "failed to get port")

			// Make OPTIONS Request (Preflight)
			url := fmt.Sprintf("http://localhost:%s/finfocus.v1.CostSourceService/GetProjectedCost", port)
			req, err := http.NewRequest("OPTIONS", url, nil)
			require.NoError(t, err)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", "POST")

			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify Headers
			if tt.expectCors {
				assert.Equal(t, tt.expectOrigin, resp.Header.Get("Access-Control-Allow-Origin"))
				if tt.expectMaxAge != "" {
					assert.Equal(t, tt.expectMaxAge, resp.Header.Get("Access-Control-Max-Age"))
				}
				// Verify credentials
				if tt.expectCredentials {
					assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
				} else {
					assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
				}
			} else {
				assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
			}
		})
	}

	t.Run("Fatal Error Wildcard + Credentials", func(t *testing.T) {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(),
			"FINFOCUS_PLUGIN_WEB_ENABLED=true",
			"FINFOCUS_CORS_ALLOWED_ORIGINS=*",
			"FINFOCUS_CORS_ALLOW_CREDENTIALS=true",
		)

		output, err := cmd.CombinedOutput()
		assert.Error(t, err, "process should have failed")
		assert.Contains(t, string(output), "cannot enable credentials with wildcard origin")
	})

	t.Run("Health Endpoint Enabled", func(t *testing.T) {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(),
			"FINFOCUS_PLUGIN_WEB_ENABLED=true",
			"FINFOCUS_PLUGIN_HEALTH_ENDPOINT=true",
		)

		stdout, err := cmd.StdoutPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		defer func() {
			_ = cmd.Process.Kill()
		}()

		port := waitForPort(t, stdout)
		require.NotEmpty(t, port, "failed to get port")

		url := fmt.Sprintf("http://localhost:%s/healthz", port)
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
