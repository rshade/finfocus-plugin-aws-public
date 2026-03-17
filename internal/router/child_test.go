package router

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChildState_String verifies string representation of all child states.
func TestChildState_String(t *testing.T) {
	tests := []struct {
		state    ChildState
		expected string
	}{
		{ChildStateIdle, "idle"},
		{ChildStateLaunching, "launching"},
		{ChildStateReady, "ready"},
		{ChildStateUnhealthy, "unhealthy"},
		{ChildStateFailed, "failed"},
		{ChildState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

// TestNewChildProcess verifies that a new child process is created in Idle state.
func TestNewChildProcess(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	child := NewChildProcess("us-east-1", "/path/to/binary", logger)

	assert.Equal(t, "us-east-1", child.region)
	assert.Equal(t, "/path/to/binary", child.binaryPath)
	assert.Equal(t, ChildStateIdle, child.State())
	assert.Nil(t, child.Client())
}

// TestChildProcess_StateTransitions verifies that state transitions work correctly.
func TestChildProcess_StateTransitions(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	child := NewChildProcess("us-east-1", "/path/to/binary", logger)
	assert.Equal(t, ChildStateIdle, child.State())

	// Mark unhealthy
	child.markUnhealthy()
	assert.Equal(t, ChildStateUnhealthy, child.State())

	// Reset to idle
	child.mu.Lock()
	child.state = ChildStateIdle
	child.mu.Unlock()
	assert.Equal(t, ChildStateIdle, child.State())

	// Mark failed
	child.mu.Lock()
	child.state = ChildStateFailed
	child.mu.Unlock()
	assert.Equal(t, ChildStateFailed, child.State())
}

// TestChildProcess_HealthCheck_NotReady verifies that health check returns error
// when child is not in Ready state.
func TestChildProcess_HealthCheck_NotReady(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	child := NewChildProcess("us-east-1", "/path/to/binary", logger)

	err := child.HealthCheck(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}

// TestChildProcess_HealthCheck_NilClient verifies that health check marks child
// unhealthy when client is nil.
func TestChildProcess_HealthCheck_NilClient(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	child := NewChildProcess("us-east-1", "/path/to/binary", logger)
	child.mu.Lock()
	child.state = ChildStateReady
	child.mu.Unlock()

	err := child.HealthCheck(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client is nil")
	assert.Equal(t, ChildStateUnhealthy, child.State())
}

// TestChildProcess_Shutdown_CancelsContext verifies that Shutdown cancels the child context,
// which terminates the child process via exec.CommandContext. This prevents orphan processes
// when the router shuts down.
//
// Test workflow:
//  1. Starts a long-running "sleep" process using exec.CommandContext with a cancellable context
//  2. Simulates the child being in Ready state with the process running
//  3. Calls Shutdown with a short timeout to force context cancellation path
//  4. Verifies the process exits promptly (Shutdown returns without hanging)
func TestChildProcess_Shutdown_CancelsContext(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create a child with a real sleep process to verify context cancellation kills it
	child := NewChildProcess("us-east-1", "/bin/sleep", logger)

	childCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(childCtx, "/bin/sleep", "300")
	cmd.Stderr = os.Stderr

	require.NoError(t, cmd.Start(), "failed to start sleep process")
	pid := cmd.Process.Pid

	child.mu.Lock()
	child.cmd = cmd
	child.cancel = cancel
	child.state = ChildStateReady
	child.mu.Unlock()

	t.Logf("started sleep process with pid %d", pid)

	// Use a very short shutdown context to trigger the force-kill path:
	// Shutdown sends SIGINT, but sleep(300) won't exit in 50ms, so the
	// shutdown context expires and cancel() is called — which triggers
	// exec.CommandContext to SIGKILL the process.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer shutdownCancel()

	// Shutdown must complete promptly (context cancellation kills the process)
	done := make(chan error, 1)
	go func() {
		done <- child.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not complete within 5 seconds — child process likely leaked")
	}
}

// TestChildProcess_CancelFunc_StoredOnLaunchFailure verifies that the cancel function
// is stored on the struct even if the process fails to start, ensuring no context leak.
func TestChildProcess_CancelFunc_StoredOnLaunchFailure(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	child := NewChildProcess("us-east-1", "/nonexistent/binary", logger)

	err := child.Launch(context.Background())
	assert.Error(t, err)

	// Even on failure, the cancel func should have been set (and can be safely called)
	child.mu.Lock()
	cancel := child.cancel
	child.mu.Unlock()

	assert.NotNil(t, cancel, "cancel func should be stored even on launch failure")
	// Calling cancel should be safe (no panic)
	cancel()
}

// TestChildProcess_Shutdown_NilCancel verifies that Shutdown handles a nil cancel func
// gracefully (e.g., if called before Launch).
func TestChildProcess_Shutdown_NilCancel(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	child := NewChildProcess("us-east-1", "/path/to/binary", logger)

	// Shutdown before Launch — cancel is nil, should not panic
	err := child.Shutdown(context.Background())
	assert.NoError(t, err)
}
