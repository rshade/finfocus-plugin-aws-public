package router

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

const (
	// maxRetries is the maximum number of restart attempts per request for a failed child.
	maxRetries = 3
	// shutdownTimeout is the maximum time to wait for all children to exit during shutdown.
	shutdownTimeout = 30 * time.Second
)

// ChildRegistry is a thread-safe registry of region-specific child processes.
type ChildRegistry struct {
	mu         sync.RWMutex
	children   map[string]*ChildProcess
	launchMu   map[string]*sync.Mutex // Per-region launch mutex for singleflight-style dedup
	launchMuMu sync.Mutex             // Protects launchMu map
	downloader *Downloader
	offline    bool
	logger     zerolog.Logger
}

// NewChildRegistry creates a new ChildRegistry with pre-populated idle children from discovery.
func NewChildRegistry(
	discovered map[string]string,
	downloader *Downloader,
	offline bool,
	logger zerolog.Logger,
) *ChildRegistry {
	children := make(map[string]*ChildProcess, len(discovered))
	for region, path := range discovered {
		children[region] = NewChildProcess(region, path, logger)
	}

	return &ChildRegistry{
		children:   children,
		launchMu:   make(map[string]*sync.Mutex),
		downloader: downloader,
		offline:    offline,
		logger:     logger.With().Str("component", "registry").Logger(),
	}
}

// getRegionMutex returns the per-region mutex, creating it if needed.
func (r *ChildRegistry) getRegionMutex(region string) *sync.Mutex {
	r.launchMuMu.Lock()
	defer r.launchMuMu.Unlock()

	mu, ok := r.launchMu[region]
	if !ok {
		mu = &sync.Mutex{}
		r.launchMu[region] = mu
	}
	return mu
}

// GetOrLaunch returns a ready client for the given region, launching the child if needed.
// It uses per-region locking to prevent concurrent duplicate launches.
func (r *ChildRegistry) GetOrLaunch(ctx context.Context, region string) (*pluginsdk.Client, error) {
	if err := validateRegion(region); err != nil {
		return nil, err
	}

	// In offline mode, avoid creating per-region mutexes for unknown regions.
	if r.offline {
		r.mu.RLock()
		_, exists := r.children[region]
		r.mu.RUnlock()
		if !exists {
			return nil, fmt.Errorf(
				"region %s not available. Install with: finfocus plugin install aws-public --metadata=region=%s",
				region, region,
			)
		}
	}

	// Per-region mutex ensures only one goroutine launches a child at a time
	regionMu := r.getRegionMutex(region)
	regionMu.Lock()
	defer regionMu.Unlock()

	// Fast path: check if child is already ready
	r.mu.RLock()
	child, exists := r.children[region]
	r.mu.RUnlock()

	if exists && child.State() == ChildStateReady {
		return child.Client(), nil
	}

	if exists && child.State() == ChildStateLaunching {
		return nil, fmt.Errorf("child for region %s is already launching", region)
	}

	// Handle unhealthy child with restart retry logic
	if exists && child.State() == ChildStateUnhealthy {
		return r.restartChild(ctx, child)
	}

	// Handle failed child
	if exists && child.State() == ChildStateFailed {
		return nil, fmt.Errorf("child for region %s failed permanently", region)
	}

	// Child exists in idle state — launch it
	if exists && child.State() == ChildStateIdle {
		if err := child.Launch(ctx); err != nil {
			return nil, fmt.Errorf("failed to launch child for region %s: %w", region, err)
		}
		return child.Client(), nil
	}

	// No child exists — try to download if online
	if !r.offline && r.downloader != nil {
		return r.downloadAndLaunch(ctx, region)
	}

	// Offline mode: no binary available
	return nil, fmt.Errorf(
		"region %s not available. Install with: finfocus plugin install aws-public --metadata=region=%s",
		region, region,
	)
}

// Get returns the child for a region if it exists, nil otherwise.
func (r *ChildRegistry) Get(region string) *ChildProcess {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.children[region]
}

// ShutdownAll sends shutdown signals to all children and waits for them to exit.
func (r *ChildRegistry) ShutdownAll(ctx context.Context) {
	r.mu.RLock()
	children := make([]*ChildProcess, 0, len(r.children))
	for _, child := range r.children {
		children = append(children, child)
	}
	r.mu.RUnlock()

	if len(children) == 0 {
		return
	}

	r.logger.Info().Int("count", len(children)).Msg("shutting down all children")

	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, child := range children {
		wg.Add(1)
		go func(c *ChildProcess) {
			defer wg.Done()
			if err := c.Shutdown(shutdownCtx); err != nil {
				r.logger.Warn().
					Str("region", c.region).
					Err(err).
					Msg("error shutting down child")
			}
		}(child)
	}
	wg.Wait()

	r.logger.Info().Msg("all children shut down")
}

// WarmUp launches all idle children in parallel background goroutines.
// It returns immediately (fire-and-forget). Children that fail to start
// remain available for retry on first actual request via GetOrLaunch().
//
// WarmUp routes through GetOrLaunch to use the same per-region mutex and
// state machine, preventing race conditions where concurrent gRPC requests
// would receive errors during the warm-up window.
//
// Warm-up goroutines use context.Background() so they are not cancelled
// if the caller's context is cancelled (e.g., during shutdown).
func (r *ChildRegistry) WarmUp(_ context.Context) {
	r.mu.RLock()
	regions := make([]string, 0, len(r.children))
	for region, child := range r.children {
		if child.State() == ChildStateIdle {
			regions = append(regions, region)
		}
	}
	r.mu.RUnlock()

	if len(regions) == 0 {
		return
	}

	r.logger.Info().Int("count", len(regions)).Msg("warming up discovered children")

	for _, region := range regions {
		go func(rgn string) { //nolint:gosec // G118: intentional — warm-up goroutines must outlive the caller context
			if _, err := r.GetOrLaunch(context.Background(), rgn); err != nil {
				r.logger.Warn().
					Str("region", rgn).
					Err(err).
					Msg("warm-up failed, will retry on first request")
			}
		}(region)
	}
}

// restartChild attempts to restart an unhealthy child with retry limits.
func (r *ChildRegistry) restartChild(ctx context.Context, child *ChildProcess) (*pluginsdk.Client, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		r.logger.Info().
			Str("region", child.region).
			Int("attempt", attempt).
			Msg("restarting unhealthy child")

		// Kill old process if still running
		_ = child.Shutdown(ctx)

		// Reset state to allow relaunch
		child.mu.Lock()
		child.state = ChildStateIdle
		child.cmd = nil
		child.client = nil
		child.port = 0
		child.mu.Unlock()

		if err := child.Launch(ctx); err != nil {
			r.logger.Warn().
				Str("region", child.region).
				Int("attempt", attempt).
				Err(err).
				Msg("restart attempt failed")
			continue
		}

		return child.Client(), nil
	}

	// Mark as permanently failed after exhausting retries
	child.mu.Lock()
	child.state = ChildStateFailed
	child.mu.Unlock()

	return nil, fmt.Errorf("child for region %s failed to start after %d attempts", child.region, maxRetries)
}

// downloadAndLaunch downloads a region binary and launches it.
// Caller (GetOrLaunch) does NOT hold r.mu; this method acquires it to register the child.
func (r *ChildRegistry) downloadAndLaunch(ctx context.Context, region string) (*pluginsdk.Client, error) {
	r.logger.Info().Str("region", region).Msg("downloading region binary")

	binaryPath, err := r.downloader.Download(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to download binary for region %s: %w", region, err)
	}

	child := NewChildProcess(region, binaryPath, r.logger)

	if launchErr := child.Launch(ctx); launchErr != nil {
		return nil, fmt.Errorf("failed to launch downloaded binary for region %s: %w", region, launchErr)
	}

	// Register the new child
	r.mu.Lock()
	r.children[region] = child
	r.mu.Unlock()

	return child.Client(), nil
}
