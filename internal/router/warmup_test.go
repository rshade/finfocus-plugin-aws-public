package router

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestChildRegistry_WarmUp_LaunchesIdleChildren verifies that WarmUp
// launches goroutines for all idle children without blocking, and that
// children actually attempt to transition out of Idle state.
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
	start := time.Now()
	registry.WarmUp(ctx)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 100*time.Millisecond, "WarmUp should return immediately")

	// Wait for background goroutines to complete (they fail fast on nonexistent binaries)
	assert.Eventually(t, func() bool {
		for _, region := range []string{"us-east-1", "us-west-2"} {
			if registry.Get(region).State() == ChildStateIdle {
				return false
			}
		}
		return true
	}, 5*time.Second, 50*time.Millisecond, "children should transition out of Idle state")

	// Verify children attempted launch and ended up Unhealthy (binary doesn't exist)
	for _, region := range []string{"us-east-1", "us-west-2"} {
		state := registry.Get(region).State()
		assert.NotEqual(t, ChildStateIdle, state, "region %s should have left Idle state", region)
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
