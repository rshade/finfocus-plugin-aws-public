package router

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestChildRegistry_WarmUp_LaunchesIdleChildren verifies that WarmUp
// launches goroutines for all idle children without blocking.
func TestChildRegistry_WarmUp_LaunchesIdleChildren(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	discovered := map[string]string{
		"us-east-1": "/nonexistent/binary-1",
		"us-west-2": "/nonexistent/binary-2",
	}

	registry := NewChildRegistry(discovered, nil, true, logger)

	// Verify children start as Idle
	assert.Equal(t, ChildStateIdle, registry.Get("us-east-1").State())
	assert.Equal(t, ChildStateIdle, registry.Get("us-west-2").State())

	// WarmUp should return immediately (fire-and-forget goroutines)
	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		registry.WarmUp(ctx)
		close(done)
	}()

	select {
	case <-done:
		// WarmUp returned without blocking - correct
	case <-time.After(1 * time.Second):
		t.Fatal("WarmUp blocked for more than 1 second")
	}
}

// TestChildRegistry_WarmUp_Empty verifies WarmUp is a no-op with no children.
func TestChildRegistry_WarmUp_Empty(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	registry := NewChildRegistry(nil, nil, true, logger)

	// Should not panic
	registry.WarmUp(context.Background())
}

// TestChildRegistry_WarmUp_SkipsNonIdle verifies WarmUp only
// launches children in Idle state.
func TestChildRegistry_WarmUp_SkipsNonIdle(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	discovered := map[string]string{
		"us-east-1": "/nonexistent/binary-1",
	}

	registry := NewChildRegistry(discovered, nil, true, logger)

	// Mark child as Failed
	child := registry.Get("us-east-1")
	child.mu.Lock()
	child.state = ChildStateFailed
	child.mu.Unlock()

	// WarmUp should skip it
	registry.WarmUp(context.Background())

	// State should remain Failed (not reset)
	assert.Equal(t, ChildStateFailed, child.State())
}
