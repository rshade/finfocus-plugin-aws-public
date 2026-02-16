package router

import (
	"context"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewChildRegistry verifies that the registry is initialized with discovered children.
func TestNewChildRegistry(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	discovered := map[string]string{
		"us-east-1": "/path/to/us-east-1",
		"us-west-2": "/path/to/us-west-2",
	}

	registry := NewChildRegistry(discovered, nil, true, logger)

	assert.NotNil(t, registry.Get("us-east-1"))
	assert.NotNil(t, registry.Get("us-west-2"))
	assert.Nil(t, registry.Get("eu-west-1"))
}

// TestChildRegistry_Get_NonExistent verifies that Get returns nil for unknown regions.
func TestChildRegistry_Get_NonExistent(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	registry := NewChildRegistry(nil, nil, true, logger)

	assert.Nil(t, registry.Get("us-east-1"))
}

// TestChildRegistry_GetOrLaunch_Offline_NoChild verifies that GetOrLaunch returns
// a helpful error message when offline mode is enabled and no binary is available.
func TestChildRegistry_GetOrLaunch_Offline_NoChild(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	registry := NewChildRegistry(nil, nil, true, logger)

	_, err := registry.GetOrLaunch(context.Background(), "us-east-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "us-east-1")
	assert.Contains(t, err.Error(), "not available")
	assert.Contains(t, err.Error(), "finfocus plugin install")
}

// TestChildRegistry_GetOrLaunch_FailedChild verifies that GetOrLaunch returns
// an error for permanently failed children.
func TestChildRegistry_GetOrLaunch_FailedChild(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	registry := NewChildRegistry(nil, nil, true, logger)

	// Manually add a failed child
	child := NewChildProcess("us-east-1", "/path/to/binary", logger)
	child.mu.Lock()
	child.state = ChildStateFailed
	child.mu.Unlock()

	registry.mu.Lock()
	registry.children["us-east-1"] = child
	registry.mu.Unlock()

	_, err := registry.GetOrLaunch(context.Background(), "us-east-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed permanently")
}

// TestChildRegistry_ShutdownAll_Empty verifies that ShutdownAll handles empty registry.
func TestChildRegistry_ShutdownAll_Empty(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	registry := NewChildRegistry(nil, nil, true, logger)

	// Should not panic
	registry.ShutdownAll(context.Background())
}

// TestChildRegistry_ConcurrentGetRegionMutex verifies that per-region mutex creation
// is safe under concurrent access.
func TestChildRegistry_ConcurrentGetRegionMutex(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	registry := NewChildRegistry(nil, nil, true, logger)

	var wg sync.WaitGroup
	mutexes := make([]*sync.Mutex, 10)

	// Launch 10 goroutines requesting the same region mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mutexes[idx] = registry.getRegionMutex("us-east-1")
		}(i)
	}
	wg.Wait()

	// All should return the same mutex instance
	for i := 1; i < 10; i++ {
		assert.Same(t, mutexes[0], mutexes[i], "all goroutines should get the same mutex")
	}
}

// TestChildRegistry_ConcurrentGetOrLaunch_Offline verifies that concurrent
// GetOrLaunch calls for the same non-existent region don't cause races.
func TestChildRegistry_ConcurrentGetOrLaunch_Offline(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	registry := NewChildRegistry(nil, nil, true, logger)

	var wg sync.WaitGroup
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errors[idx] = registry.GetOrLaunch(context.Background(), "us-east-1")
		}(i)
	}
	wg.Wait()

	// All should return an error (no binary in offline mode)
	for i := 0; i < 10; i++ {
		assert.Error(t, errors[i])
	}
}
