//go:build region_use1 || region_usw1 || region_usw2 || region_govw1 || region_gove1 || region_euw1 || region_apse1 || region_apse2 || region_apne1 || region_aps1 || region_cac1 || region_sae1

package pricing

import (
	"encoding/json"
	"testing"
)

// TestEmbeddedPricingDataSize verifies that real AWS pricing data is embedded.
//
// This test fails if the binary was built without region tags (e.g., with `make build`).
// When no region tag is specified, the fallback dummy pricing is used, which is < 1MB.
// This test ensures only real pricing (> 1MB) is embedded.
//
// Run with: go test -tags=region_use1 -run TestEmbeddedPricingDataSize ./internal/pricing/...
//
// This is critical for catching the v0.0.10 build issue where binaries shipped without
// region tags, resulting in all EC2 pricing returning $0.
func TestEmbeddedPricingDataSize(t *testing.T) {
	const minPricingSize = 1000000 // 1MB minimum for real data (real ~7MB)
	if len(rawPricingJSON) < minPricingSize {
		t.Fatalf("Pricing data too small (%d bytes) - missing region tag? Real pricing should be ~7MB. "+
			"Did you forget -tags=region_use1? Run: go test -tags=region_use1 ./internal/pricing/...",
			len(rawPricingJSON))
	}

	t.Logf("✓ Embedded pricing data size: %d bytes (OK)", len(rawPricingJSON))
}

// TestEmbeddedPricingDataIsValid verifies pricing data is valid AWS Price List JSON.
//
// Parses the embedded JSON and checks for expected AWS pricing structure.
// This catches build errors or corrupted embedded data.
//
// Run with: go test -tags=region_use1 -run TestEmbeddedPricingDataIsValid ./internal/pricing/...
func TestEmbeddedPricingDataIsValid(t *testing.T) {
	var data struct {
		Products map[string]interface{} `json:"products"`
		Terms    map[string]interface{} `json:"terms"`
	}

	err := json.Unmarshal(rawPricingJSON, &data)
	if err != nil {
		t.Fatalf("Failed to parse embedded pricing JSON: %v", err)
	}

	if len(data.Products) == 0 {
		t.Fatal("Embedded pricing has no products - corrupted or fallback data?")
	}

	// Check for presence of real AWS SKUs (vs fallback dummy data)
	if len(data.Products) < 100 {
		t.Logf("WARNING: Pricing has only %d products - may be fallback data. Real pricing has thousands.", len(data.Products))
	}

	t.Logf("✓ Embedded pricing: %d products, valid JSON structure (OK)", len(data.Products))
}
