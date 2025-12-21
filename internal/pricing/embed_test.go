//go:build region_use1 || region_usw1 || region_usw2 || region_govw1 || region_gove1 || region_euw1 || region_apse1 || region_apse2 || region_apne1 || region_aps1 || region_cac1 || region_sae1

package pricing

import (
	"encoding/json"
	"testing"
)

// Per-Service Size Thresholds (T035-T041)
//
// These thresholds prevent the v0.0.10/v0.0.11 regression where filtering
// stripped pricing data, causing $0 estimates. Each service has a minimum
// size based on actual AWS pricing data (December 2024, OnDemand terms only).
//
// Actual sizes from us-east-1 (December 2024):
//   EC2: 153.7 MB, RDS: 6.8 MB, EKS: 772 KB, Lambda: 445 KB
//   S3: 306 KB, DynamoDB: 22 KB, ELB: 13 KB
//
// DO NOT reduce these thresholds without explicit approval.
const (
	// EC2: Largest service, contains compute instances + EBS storage
	// Full us-east-1 with OnDemand only: ~154MB
	minEC2Size = 100_000_000 // 100MB minimum (T035)

	// RDS: Database instances and storage
	// Full us-east-1 with OnDemand only: ~7MB
	minRDSSize = 5_000_000 // 5MB minimum (T036)

	// EKS: Cluster management fees
	// Full us-east-1: ~772KB
	minEKSSize = 500_000 // 500KB minimum (T037)

	// Lambda: Serverless compute pricing
	// Full us-east-1: ~445KB
	minLambdaSize = 300_000 // 300KB minimum (T038)

	// S3: Storage classes and tiers
	// Full us-east-1: ~306KB
	minS3Size = 200_000 // 200KB minimum (T039)

	// DynamoDB: NoSQL throughput and storage
	// Full us-east-1: ~22KB (small but valid)
	minDynamoDBSize = 10_000 // 10KB minimum (T040)

	// ELB: Load balancer hourly and capacity rates
	// Full us-east-1: ~13KB (small but valid)
	minELBSize = 8_000 // 8KB minimum (T041)
)

// TestEmbeddedData_EC2Size verifies EC2 pricing data is not filtered (T035).
//
// CRITICAL: This test prevents the v0.0.10/v0.0.11 regression.
// EC2 is the largest service and most commonly estimated.
func TestEmbeddedData_EC2Size(t *testing.T) {
	size := len(rawEC2JSON)
	if size < minEC2Size {
		t.Fatalf("❌ CRITICAL: EC2 pricing data too small (%d bytes, %.1f MB)\n"+
			"Expected: > %d bytes (100MB) for OnDemand pricing data\n"+
			"This indicates EC2 data is being filtered incorrectly.\n"+
			"FIX: Check tools/generate-pricing/main.go",
			size, float64(size)/1_000_000, minEC2Size)
	}
	t.Logf("✓ EC2 pricing data size: %d bytes (%.1f MB)", size, float64(size)/1_000_000)
}

// TestEmbeddedData_RDSSize verifies RDS pricing data is not filtered (T036).
func TestEmbeddedData_RDSSize(t *testing.T) {
	size := len(rawRDSJSON)
	if size < minRDSSize {
		t.Fatalf("❌ RDS pricing data too small (%d bytes, %.1f MB)\n"+
			"Expected: > %d bytes (5MB)",
			size, float64(size)/1_000_000, minRDSSize)
	}
	t.Logf("✓ RDS pricing data size: %d bytes (%.1f MB)", size, float64(size)/1_000_000)
}

// TestEmbeddedData_EKSSize verifies EKS pricing data is not filtered (T037).
func TestEmbeddedData_EKSSize(t *testing.T) {
	size := len(rawEKSJSON)
	if size < minEKSSize {
		t.Fatalf("❌ EKS pricing data too small (%d bytes, %.1f KB)\n"+
			"Expected: > %d bytes (500KB)",
			size, float64(size)/1_000, minEKSSize)
	}
	t.Logf("✓ EKS pricing data size: %d bytes (%.1f KB)", size, float64(size)/1_000)
}

// TestEmbeddedData_LambdaSize verifies Lambda pricing data is not filtered (T038).
func TestEmbeddedData_LambdaSize(t *testing.T) {
	size := len(rawLambdaJSON)
	if size < minLambdaSize {
		t.Fatalf("❌ Lambda pricing data too small (%d bytes, %.1f KB)\n"+
			"Expected: > %d bytes (300KB)",
			size, float64(size)/1_000, minLambdaSize)
	}
	t.Logf("✓ Lambda pricing data size: %d bytes (%.1f KB)", size, float64(size)/1_000)
}

// TestEmbeddedData_S3Size verifies S3 pricing data is not filtered (T039).
func TestEmbeddedData_S3Size(t *testing.T) {
	size := len(rawS3JSON)
	if size < minS3Size {
		t.Fatalf("❌ S3 pricing data too small (%d bytes, %.1f KB)\n"+
			"Expected: > %d bytes (200KB)",
			size, float64(size)/1_000, minS3Size)
	}
	t.Logf("✓ S3 pricing data size: %d bytes (%.1f KB)", size, float64(size)/1_000)
}

// TestEmbeddedData_DynamoDBSize verifies DynamoDB pricing data is not filtered (T040).
func TestEmbeddedData_DynamoDBSize(t *testing.T) {
	size := len(rawDynamoDBJSON)
	if size < minDynamoDBSize {
		t.Fatalf("❌ DynamoDB pricing data too small (%d bytes, %.1f KB)\n"+
			"Expected: > %d bytes (10KB)",
			size, float64(size)/1_000, minDynamoDBSize)
	}
	t.Logf("✓ DynamoDB pricing data size: %d bytes (%.1f KB)", size, float64(size)/1_000)
}

// TestEmbeddedData_ELBSize verifies ELB pricing data is not filtered (T041).
func TestEmbeddedData_ELBSize(t *testing.T) {
	size := len(rawELBJSON)
	if size < minELBSize {
		t.Fatalf("❌ ELB pricing data too small (%d bytes, %.1f KB)\n"+
			"Expected: > %d bytes (8KB)",
			size, float64(size)/1_000, minELBSize)
	}
	t.Logf("✓ ELB pricing data size: %d bytes (%.1f KB)", size, float64(size)/1_000)
}

// TestEmbeddedData_AllServicesValid verifies all per-service pricing files are valid JSON.
//
// This catches build errors or corrupted embedded data at test time,
// rather than runtime when customers are using the plugin.
func TestEmbeddedData_AllServicesValid(t *testing.T) {
	services := []struct {
		name string
		data []byte
	}{
		{"EC2", rawEC2JSON},
		{"S3", rawS3JSON},
		{"RDS", rawRDSJSON},
		{"EKS", rawEKSJSON},
		{"Lambda", rawLambdaJSON},
		{"DynamoDB", rawDynamoDBJSON},
		{"ELB", rawELBJSON},
	}

	for _, svc := range services {
		t.Run(svc.name, func(t *testing.T) {
			var data struct {
				Products  map[string]interface{} `json:"products"`
				Terms     map[string]interface{} `json:"terms"`
				OfferCode string                 `json:"offerCode"`
			}

			if err := json.Unmarshal(svc.data, &data); err != nil {
				t.Fatalf("Failed to parse %s pricing JSON: %v", svc.name, err)
			}

			if len(data.Products) == 0 {
				t.Errorf("%s: No products found - corrupted or fallback data?", svc.name)
			}

			if data.OfferCode == "" {
				t.Errorf("%s: Missing offerCode in pricing data", svc.name)
			}

			t.Logf("✓ %s: %d products, offerCode=%s", svc.name, len(data.Products), data.OfferCode)
		})
	}
}

// TestEmbeddedData_EC2ProductCount verifies EC2 has sufficient products (T035 detail).
//
// This complements the size check by verifying product count.
// EC2 us-east-1 should have ~90k+ products including compute instances and EBS.
func TestEmbeddedData_EC2ProductCount(t *testing.T) {
	var data struct {
		Products map[string]interface{} `json:"products"`
		Terms    struct {
			OnDemand map[string]interface{} `json:"OnDemand"`
		} `json:"terms"`
	}

	if err := json.Unmarshal(rawEC2JSON, &data); err != nil {
		t.Fatalf("Failed to parse EC2 JSON: %v", err)
	}

	const minProducts = 50_000 // Minimum EC2 products
	const minTerms = 30_000   // Minimum OnDemand terms

	if len(data.Products) < minProducts {
		t.Fatalf("❌ EC2 product count too low: %d (expected >%d)", len(data.Products), minProducts)
	}

	if len(data.Terms.OnDemand) < minTerms {
		t.Fatalf("❌ EC2 OnDemand term count too low: %d (expected >%d)", len(data.Terms.OnDemand), minTerms)
	}

	t.Logf("✓ EC2: %d products, %d OnDemand terms", len(data.Products), len(data.Terms.OnDemand))
}
