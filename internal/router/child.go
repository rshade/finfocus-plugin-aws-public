package router

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

const (
	// regexSubmatchWithOneGroup is the expected number of matches from FindStringSubmatch
	// when the regex contains one capture group (full match + one capture group).
	regexSubmatchWithOneGroup = 2
)

// ChildState represents the lifecycle state of a child process.
type ChildState int

const (
	// ChildStateIdle indicates the binary was discovered but the process has not been started.
	ChildStateIdle ChildState = iota
	// ChildStateLaunching indicates the process is starting and waiting for PORT announcement.
	ChildStateLaunching
	// ChildStateReady indicates the process is healthy and accepting RPCs.
	ChildStateReady
	// ChildStateUnhealthy indicates the process crashed or an RPC failed.
	ChildStateUnhealthy
	// ChildStateFailed indicates the process exceeded its retry limit.
	ChildStateFailed
)

// String returns a human-readable name for the child state.
func (s ChildState) String() string {
	switch s {
	case ChildStateIdle:
		return "idle"
	case ChildStateLaunching:
		return "launching"
	case ChildStateReady:
		return "ready"
	case ChildStateUnhealthy:
		return "unhealthy"
	case ChildStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

const (
	// childStartTimeout is the maximum time to wait for a child to announce its PORT.
	childStartTimeout = 30 * time.Second
)

// portRegex matches the PORT=<digits> announcement from child stdout.
var portRegex = regexp.MustCompile(`PORT=(\d+)`)

// ChildProcess represents a running region-specific plugin child.
type ChildProcess struct {
	region     string
	binaryPath string
	cmd        *exec.Cmd
	port       int
	client     *pluginsdk.Client
	state      ChildState
	mu         sync.Mutex
	logger     zerolog.Logger
}

// NewChildProcess creates a new ChildProcess in Idle state for the given region and binary path.
func NewChildProcess(region, binaryPath string, logger zerolog.Logger) *ChildProcess {
	return &ChildProcess{
		region:     region,
		binaryPath: binaryPath,
		state:      ChildStateIdle,
		logger:     logger.With().Str("child_region", region).Logger(),
	}
}

// State returns the current lifecycle state of the child process.
func (c *ChildProcess) State() ChildState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Client returns the pluginsdk.Client for delegating RPCs to this child.
func (c *ChildProcess) Client() *pluginsdk.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client
}

// Launch starts the child process, captures the PORT announcement, and creates a Connect client.
// It transitions the state from Idle/Unhealthy to Launching to Ready.
func (c *ChildProcess) Launch(ctx context.Context) error {
	c.mu.Lock()
	if c.state != ChildStateIdle && c.state != ChildStateUnhealthy {
		c.mu.Unlock()
		return fmt.Errorf("cannot launch child in state %s", c.state)
	}
	c.state = ChildStateLaunching
	c.mu.Unlock()

	c.logger.Info().Str("binary", c.binaryPath).Msg("launching child process")

	// Build environment for child: enable Connect protocol, use ephemeral port
	env := os.Environ()
	env = append(env, "FINFOCUS_PLUGIN_WEB_ENABLED=true", "PORT=0")

	//nolint:gosec // G204: binaryPath is from trusted discovery, not user input
	cmd := exec.CommandContext(ctx, c.binaryPath)
	cmd.Env = env
	cmd.Stderr = os.Stderr // Inherit stderr for child logging

	stdout, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		c.setStateFailed(pipeErr)
		return fmt.Errorf("failed to create stdout pipe: %w", pipeErr)
	}

	if startErr := cmd.Start(); startErr != nil {
		c.setStateFailed(startErr)
		return fmt.Errorf("failed to start child: %w", startErr)
	}

	// Parse PORT announcement from stdout with timeout
	portChan := make(chan int, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if matches := portRegex.FindStringSubmatch(line); len(matches) == regexSubmatchWithOneGroup {
				port, parseErr := strconv.Atoi(matches[1])
				if parseErr == nil && port > 0 {
					portChan <- port
					return
				}
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			errChan <- fmt.Errorf("error reading child stdout: %w", scanErr)
		} else {
			errChan <- errors.New("child process exited without announcing PORT")
		}
	}()

	select {
	case port := <-portChan:
		c.mu.Lock()
		c.cmd = cmd
		c.port = port
		c.client = pluginsdk.NewConnectClient(fmt.Sprintf("http://127.0.0.1:%d", port))
		c.state = ChildStateReady
		c.mu.Unlock()

		c.logger.Info().Int("port", port).Msg("child process ready")
		return nil

	case childErr := <-errChan:
		_ = cmd.Process.Kill()
		c.setStateFailed(childErr)
		return fmt.Errorf("child startup failed: %w", childErr)

	case <-time.After(childStartTimeout):
		_ = cmd.Process.Kill()
		timeoutErr := fmt.Errorf("child failed to announce PORT within %s", childStartTimeout)
		c.setStateFailed(timeoutErr)
		return timeoutErr

	case <-ctx.Done():
		_ = cmd.Process.Kill()
		c.setStateFailed(ctx.Err())
		return fmt.Errorf("context cancelled during child startup: %w", ctx.Err())
	}
}

// HealthCheck sends a Name() RPC to verify the child is still responsive.
// If the check fails, the child is marked Unhealthy.
func (c *ChildProcess) HealthCheck(ctx context.Context) error {
	c.mu.Lock()
	client := c.client
	state := c.state
	c.mu.Unlock()

	if state != ChildStateReady {
		return fmt.Errorf("child is in state %s, not ready", state)
	}

	if client == nil {
		c.markUnhealthy()
		return errors.New("child client is nil")
	}

	// Check if the process has exited
	if c.cmd != nil && c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		c.markUnhealthy()
		return errors.New("child process has exited")
	}

	_, err := client.Name(ctx)
	if err != nil {
		c.markUnhealthy()
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Shutdown gracefully terminates the child process.
func (c *ChildProcess) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	cmd := c.cmd
	client := c.client
	c.mu.Unlock()

	if client != nil {
		client.Close()
	}

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	c.logger.Info().Msg("shutting down child process")

	// Send SIGTERM for graceful shutdown
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		// Process may have already exited
		c.logger.Debug().Err(err).Msg("signal failed, process may have exited")
		return nil
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		c.logger.Info().Msg("child process exited gracefully")
		return nil
	case <-ctx.Done():
		// Force kill if context expires
		c.logger.Warn().Msg("forcing child process kill")
		_ = cmd.Process.Kill()
		return nil
	}
}

// markUnhealthy transitions the child to Unhealthy state.
func (c *ChildProcess) markUnhealthy() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = ChildStateUnhealthy
	c.logger.Warn().Msg("child marked unhealthy")
}

// setStateFailed kills any running process and marks the child as failed.
func (c *ChildProcess) setStateFailed(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = ChildStateUnhealthy
	c.logger.Error().Err(err).Msg("child launch failed")
}
