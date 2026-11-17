package pricing

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestClient_PricingMetadata verifies every embedded service file keeps the
// Price List header that its parser already unmarshals.
//
// Fallback builds report version "fallback" and publicationDate
// 2024-01-01T00:00:00Z. A region-tagged build must parse as RFC3339 and must
// not still carry that fallback date.
func TestClient_PricingMetadata(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var oldest string
	for _, file := range Files() {
		meta, ok := client.PricingMetadata(file)
		if !ok {
			t.Errorf("PricingMetadata(%q) missing", file)
			continue
		}
		published, parseErr := time.Parse(time.RFC3339, meta.PublicationDate)
		if parseErr != nil {
			t.Errorf("%s publicationDate %q: %v", file, meta.PublicationDate, parseErr)
			continue
		}
		if meta.Version == "" || meta.OfferCode == "" {
			t.Errorf("%s metadata = %+v, want version and offerCode", file, meta)
		}
		if meta.Version == PricingVersionFallback && meta.PublicationDate != "2024-01-01T00:00:00Z" {
			t.Errorf("%s fallback publicationDate = %q", file, meta.PublicationDate)
		}
		if meta.Version != PricingVersionFallback && meta.PublicationDate == "2024-01-01T00:00:00Z" {
			t.Errorf("%s region build still has the fallback publication date", file)
		}
		if oldest == "" || published.Before(mustParse(t, oldest)) {
			oldest = meta.PublicationDate
		}
	}
	if got := client.PricingAsOf(); got != oldest {
		t.Errorf("PricingAsOf() = %q, want oldest %q", got, oldest)
	}
	if _, ok := client.PricingMetadata("not-a-service"); ok {
		t.Error("PricingMetadata(unknown) = true, want false")
	}
}

func mustParse(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}
	return parsed
}
