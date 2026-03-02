package router

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
