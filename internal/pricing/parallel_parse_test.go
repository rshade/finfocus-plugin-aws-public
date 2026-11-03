package pricing

import (
	"errors"
	"sync"
	"testing"

	"github.com/rs/zerolog"
)

// TestParallelParsing_NoRace verifies that constructing pricing clients
// concurrently is race-free.
//
// Each NewClient parses every embedded service file on its own goroutines.
// Running several clients at once overlaps those parses. The test fails if
// initialization returns an error. Run it with the race detector:
//
//	go test -race -count=1 -run TestParallelParsing_NoRace ./internal/pricing/...
//
// With no build tag this uses the fallback embed. After pricing data is
// generated, -tags=region_use1 runs the same test against the us-east-1 files.
func TestParallelParsing_NoRace(t *testing.T) {
	logger := zerolog.Nop()
	const clients = 10

	var wg sync.WaitGroup
	errs := make(chan error, clients)
	for range clients {
		wg.Go(func() {
			client, err := NewClient(logger)
			if err != nil {
				errs <- err
				return
			}
			if client == nil || client.Region() == "" {
				errs <- errors.New("NewClient returned no region")
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
