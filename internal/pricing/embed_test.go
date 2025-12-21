//go:build region_use1 || region_usw1 || region_usw2 || region_govw1 || region_gove1 || region_euw1 || region_apse1 || region_apse2 || region_apne1 || region_aps1 || region_cac1 || region_sae1

package pricing

import (
	"encoding/json"
	"testing"
)

// TestEmbeddedPricingDataSize verifies that FULL AWS pricing data is embedded without filtering.
//
// CRITICAL: This test prevents the v0.0.10/v0.0.11 regression where aggressive filtering
// in tools/generate-pricing stripped 85% of pricing data, causing:
//   - EC2 products reduced from ~90,000 to ~12,000
//   - EBS volume pricing missing
//   - Many instance types returning $0
//
// The minimum size threshold (100MB) ensures the FULL AWS pricing JSON is embedded,
// not a filtered subset. DO NOT reduce this threshold without explicit approval.
//
// If this test fails:
//  1. Check tools/generate-pricing/main.go for any filtering logic
//  2. Ensure generateCombinedPricingData() merges ALL products without filtering
//  3. Verify the pricing JSON file size before embedding (~150MB for us-east-1)
//
// Run with: go test -tags=region_use1 -run TestEmbeddedPricingDataSize ./internal/pricing/...
func TestEmbeddedPricingDataSize(t *testing.T) {
	// IMMUTABLE THRESHOLD: 100MB minimum for full unfiltered pricing data
	// Real us-east-1 pricing is ~155MB. DO NOT reduce this threshold.
	// v0.0.10/v0.0.11 had filtered data at ~5MB which passed the old 1MB threshold.
	const minPricingSize = 100_000_000 // 100MB minimum for FULL pricing data

	if len(rawPricingJSON) < minPricingSize {
		t.Fatalf("❌ CRITICAL: Pricing data too small (%d bytes, %.1f MB)\n"+
			"Expected: > %d bytes (100MB) for FULL unfiltered pricing data\n"+
			"This indicates pricing data is being FILTERED in tools/generate-pricing.\n"+
			"The v0.0.10/v0.0.11 bug stripped 85%% of pricing data.\n\n"+
			"FIX: Check tools/generate-pricing/main.go - it must merge ALL products\n"+
			"without filtering by ProductFamily, attributes, or any other criteria.\n"+
			"See: https://github.com/rshade/pulumicost-plugin-aws-public/issues/XXX",
			len(rawPricingJSON), float64(len(rawPricingJSON))/1_000_000, minPricingSize)
	}

	t.Logf("✓ Embedded pricing data size: %d bytes (%.1f MB) - FULL data embedded",
		len(rawPricingJSON), float64(len(rawPricingJSON))/1_000_000)
}

// TestEmbeddedPricingProductCount verifies the FULL product count is embedded.
//
// CRITICAL: This test prevents filtering regression. The minimum thresholds are based on
// actual AWS pricing data as of December 2024:
//   - us-east-1: ~98,000 products total across all services
//   - Other regions: ~50,000-80,000 products
//
// DO NOT reduce these thresholds. If the test fails, the generate-pricing tool
// is likely filtering products which breaks cost estimation.
//
// Run with: go test -tags=region_use1 -run TestEmbeddedPricingProductCount ./internal/pricing/...
func TestEmbeddedPricingProductCount(t *testing.T) {
	var data struct {
		Products map[string]interface{} `json:"products"`
		Terms    struct {
			OnDemand map[string]interface{} `json:"OnDemand"`
		} `json:"terms"`
	}

	if err := json.Unmarshal(rawPricingJSON, &data); err != nil {
		t.Fatalf("Failed to parse embedded pricing JSON: %v", err)
	}

	// IMMUTABLE THRESHOLDS: Based on actual AWS pricing data (December 2024)
	// DO NOT reduce these thresholds without explicit approval.
	const minProducts = 50_000  // Minimum products for any region (smallest regions have ~50k)
	const minTerms = 30_000    // Minimum OnDemand terms (not all products have OnDemand)

	productCount := len(data.Products)
	termCount := len(data.Terms.OnDemand)

	if productCount < minProducts {
		t.Fatalf("❌ CRITICAL: Product count too low (%d products)\n"+
			"Expected: > %d products for FULL unfiltered pricing\n"+
			"v0.0.10/v0.0.11 had only ~16,000 products due to filtering.\n"+
			"Real us-east-1 has ~98,000 products.\n\n"+
			"FIX: Check tools/generate-pricing/main.go for filtering logic",
			productCount, minProducts)
	}

	if termCount < minTerms {
		t.Fatalf("❌ CRITICAL: OnDemand term count too low (%d terms)\n"+
			"Expected: > %d OnDemand terms\n"+
			"This indicates the generate-pricing tool is filtering terms.\n\n"+
			"FIX: Ensure all OnDemand terms are merged without filtering",
			termCount, minTerms)
	}

	t.Logf("✓ Embedded pricing: %d products, %d OnDemand terms - FULL data verified",
		productCount, termCount)
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

	t.Logf("✓ Embedded pricing: %d products, valid JSON structure (OK)", len(data.Products))
}
