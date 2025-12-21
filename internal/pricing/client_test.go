package pricing

import (
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Verify basic initialization
	if client.Region() == "" {
		t.Error("Region() returned empty string")
	}

	if client.Currency() != "USD" {
		t.Errorf("Currency() = %q, want %q", client.Currency(), "USD")
	}
}

func TestClient_EC2OnDemandPricePerHour(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	tests := []struct {
		name         string
		instanceType string
		os           string
		tenancy      string
		wantFound    bool
		// Removed strict price check to allow multi-region testing
	}{
		{
			name:         "t3.micro Linux Shared",
			instanceType: "t3.micro",
			os:           "Linux",
			tenancy:      "Shared",
			wantFound:    true,
		},
		{
			name:         "t3.small Linux Shared",
			instanceType: "t3.small",
			os:           "Linux",
			tenancy:      "Shared",
			wantFound:    true,
		},
		{
			name:         "nonexistent instance type",
			instanceType: "t99.mega",
			os:           "Linux",
			tenancy:      "Shared",
			wantFound:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, found := client.EC2OnDemandPricePerHour(tt.instanceType, tt.os, tt.tenancy)

			if found != tt.wantFound {
				t.Errorf("EC2OnDemandPricePerHour() found = %v, want %v", found, tt.wantFound)
			}

			if tt.wantFound {
				if price <= 0 {
					t.Errorf("EC2OnDemandPricePerHour() price = %v, want > 0", price)
				}
				// Optional: strict check only for us-east-1 to verify exact parsing logic
				if client.Region() == "us-east-1" {
					var expected float64
					switch tt.instanceType {
					case "t3.micro":
						expected = 0.0104
					case "t3.small":
						expected = 0.0208
					}
					if expected > 0 && price != expected {
						t.Errorf("Region %s: Expected exact price %v, got %v", client.Region(), expected, price)
					}
				}
			} else {
				if price != 0 {
					t.Errorf("EC2OnDemandPricePerHour() price = %v, want 0", price)
				}
			}
		})
	}
}

func TestClient_EBSPricePerGBMonth(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	tests := []struct {
		name       string
		volumeType string
		wantFound  bool
	}{
		{
			name:       "gp3",
			volumeType: "gp3",
			wantFound:  true,
		},
		{
			name:       "gp2",
			volumeType: "gp2",
			wantFound:  true,
		},
		{
			name:       "nonexistent volume type",
			volumeType: "super-fast",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, found := client.EBSPricePerGBMonth(tt.volumeType)

			if found != tt.wantFound {
				t.Errorf("EBSPricePerGBMonth() found = %v, want %v", found, tt.wantFound)
			}

			if tt.wantFound {
				if price <= 0 {
					t.Errorf("EBSPricePerGBMonth() price = %v, want > 0", price)
				}
				// Optional: strict check only for us-east-1
				if client.Region() == "us-east-1" {
					var expected float64
					switch tt.volumeType {
					case "gp3":
						expected = 0.08
					case "gp2":
						expected = 0.10
					}
					if expected > 0 && price != expected {
						t.Errorf("Region %s: Expected exact price %v, got %v", client.Region(), expected, price)
					}
				}
			} else {
				if price != 0 {
					t.Errorf("EBSPricePerGBMonth() price = %v, want 0", price)
				}
			}
		})
	}
}

func TestClient_ConcurrentAccess(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Test thread safety by accessing pricing from multiple goroutines
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			_, _ = client.EC2OnDemandPricePerHour("t3.micro", "Linux", "Shared")
			_, _ = client.EBSPricePerGBMonth("gp3")
			_ = client.Region()
			_ = client.Currency()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestClient_APSoutheast1 tests pricing data loading for ap-southeast-1 (T012)
// Note: This test validates the structure; actual region will depend on build tag
func TestClient_APSoutheast1_DataStructure(t *testing.T) {
	client, err := NewClient(zerolog.Nop()) // Pass zerolog.Nop() here
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Verify client initialization works
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Verify region is set (could be any region depending on build tag)
	region := client.Region()
	if region == "" {
		t.Error("Region() returned empty string")
	}

	// Verify currency is USD
	if client.Currency() != "USD" {
		t.Errorf("Currency() = %q, want %q", client.Currency(), "USD")
	}

	// Verify EC2 pricing lookup works (returns found or not found)
	_, found := client.EC2OnDemandPricePerHour("t3.micro", "Linux", "Shared")
	// Don't check found value, as it depends on build tag and pricing data
	_ = found

	// Verify EBS pricing lookup works
	_, found = client.EBSPricePerGBMonth("gp3")
	_ = found
}

// TestClient_RegionSpecificPricing tests that region-specific pricing is loaded correctly (T012)
func TestClient_RegionSpecificPricing(t *testing.T) {
	client, err := NewClient(zerolog.Nop()) // Pass zerolog.Nop() here
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	region := client.Region()

	// This test validates that pricing data is properly loaded for whatever region
	// the binary is built for. Specific pricing values depend on build tag.
	t.Logf("Testing pricing data for region: %s", region)

	// For any region, we should be able to look up pricing (even if not found)
	// The important thing is that the lookup doesn't crash
	testInstances := []string{"t3.micro", "t3.small", "m5.large"}
	for _, instance := range testInstances {
		price, found := client.EC2OnDemandPricePerHour(instance, "Linux", "Shared")
		if found {
			t.Logf("Region %s: %s hourly price = $%.4f", region, instance, price)
			if price < 0 {
				t.Errorf("Negative price for %s: %v", instance, price)
			}
		}
	}

	testVolumes := []string{"gp3", "gp2", "io2"}
	for _, volume := range testVolumes {
		price, found := client.EBSPricePerGBMonth(volume)
		if found {
			t.Logf("Region %s: %s GB-month price = $%.4f", region, volume, price)
			if price < 0 {
				t.Errorf("Negative price for %s: %v", volume, price)
			}
		}
	}
}

func TestClient_EKSClusterPricePerHour(t *testing.T) {
	t.Log("Starting EKS test")
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	region := client.Region()
	t.Logf("Client region: %s", region)

	// Skip "should be found" checks for fallback data (region == "unknown")
	isFallback := region == "unknown"

	// Test standard support pricing
	standardPrice, standardFound := client.EKSClusterPricePerHour(false)
	t.Logf("EKS standard support price lookup: found=%v, price=%v", standardFound, standardPrice)

	// Standard support pricing should be available for known regions
	if !standardFound && !isFallback {
		t.Errorf("EKSClusterPricePerHour(false) should return found=true for standard support, region=%s", region)
	} else if standardFound {
		// Verify standard price is reasonable (should be around $0.10/hour)
		if standardPrice <= 0 {
			t.Errorf("EKS standard price should be positive, got: %v", standardPrice)
		}
		if standardPrice > 0.20 {
			t.Errorf("EKS standard price seems too high: %v (expected ~$0.10/hour)", standardPrice)
		}
		t.Logf("EKS cluster standard support hourly price = $%.4f", standardPrice)
	}

	// Test extended support pricing
	extendedPrice, extendedFound := client.EKSClusterPricePerHour(true)
	t.Logf("EKS extended support price lookup: found=%v, price=%v", extendedFound, extendedPrice)

	// Extended support pricing may or may not be available depending on the pricing data
	if extendedFound {
		// Verify extended price is reasonable (should be around $0.50/hour)
		if extendedPrice <= 0 {
			t.Errorf("EKS extended price should be positive, got: %v", extendedPrice)
		}
		if extendedPrice < standardPrice {
			t.Errorf("EKS extended price (%v) should be >= standard price (%v)", extendedPrice, standardPrice)
		}
		t.Logf("EKS cluster extended support hourly price = $%.4f", extendedPrice)
	} else {
		t.Logf("Extended support pricing not available for region %s (this may be expected)", client.Region())
	}
}

func TestClient_DynamoDBPricing(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	region := client.Region()
	t.Logf("Testing DynamoDB pricing for region: %s", region)

	// Test On-Demand Read Price
	readPrice, readFound := client.DynamoDBOnDemandReadPrice()
	if readFound {
		if readPrice <= 0 {
			t.Errorf("DynamoDB On-Demand read price should be positive, got: %v", readPrice)
		}
		t.Logf("DynamoDB On-Demand read price = $%.8f", readPrice)
	}

	// Test On-Demand Write Price
	writePrice, writeFound := client.DynamoDBOnDemandWritePrice()
	if writeFound {
		if writePrice <= 0 {
			t.Errorf("DynamoDB On-Demand write price should be positive, got: %v", writePrice)
		}
		t.Logf("DynamoDB On-Demand write price = $%.8f", writePrice)
	}

	// Test Storage Price
	storagePrice, storageFound := client.DynamoDBStoragePricePerGBMonth()
	if storageFound {
		if storagePrice <= 0 {
			t.Errorf("DynamoDB Storage price should be positive, got: %v", storagePrice)
		}
		t.Logf("DynamoDB Storage price = $%.4f", storagePrice)
	}

	// Test Provisioned RCU Price
	rcuPrice, rcuFound := client.DynamoDBProvisionedRCUPrice()
	if rcuFound {
		if rcuPrice <= 0 {
			t.Errorf("DynamoDB Provisioned RCU price should be positive, got: %v", rcuPrice)
		}
		t.Logf("DynamoDB Provisioned RCU price = $%.8f", rcuPrice)
	}

	// Test Provisioned WCU Price
	wcuPrice, wcuFound := client.DynamoDBProvisionedWCUPrice()
	if wcuFound {
		if wcuPrice <= 0 {
			t.Errorf("DynamoDB Provisioned WCU price should be positive, got: %v", wcuPrice)
		}
		t.Logf("DynamoDB Provisioned WCU price = $%.8f", wcuPrice)
	}

	// Optional: strict check only for us-east-1
	// Note: AWS pricing is per-request unit, not per-million.
	// Actual prices from AWS API (December 2024):
	//   - On-Demand Read: ~$0.00000012/request ($0.25/million reads)
	//   - On-Demand Write: ~$0.00000063/request ($1.25/million writes)
	//   - Storage: ~$0.10/GB-month (varies by region)
	if region == "us-east-1" {
		// These are reasonable range checks, not exact value checks,
		// since AWS prices can change slightly over time.
		if readFound && (readPrice < 1e-8 || readPrice > 1e-5) {
			t.Errorf("us-east-1: On-Demand read price out of expected range: %v", readPrice)
		}
		if writeFound && (writePrice < 1e-8 || writePrice > 1e-5) {
			t.Errorf("us-east-1: On-Demand write price out of expected range: %v", writePrice)
		}
		if storageFound && (storagePrice < 0.05 || storagePrice > 0.50) {
			t.Errorf("us-east-1: Storage price out of expected range: %v", storagePrice)
		}
		if rcuFound && (rcuPrice < 0.00005 || rcuPrice > 0.001) {
			t.Errorf("us-east-1: Provisioned RCU price out of expected range: %v", rcuPrice)
		}
		if wcuFound && (wcuPrice < 0.0001 || wcuPrice > 0.005) {
			t.Errorf("us-east-1: Provisioned WCU price out of expected range: %v", wcuPrice)
		}
	}
}

func TestClient_ELBPricing(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	region := client.Region()
	t.Logf("Testing ELB pricing for region: %s", region)

	// Skip "should be found" checks for fallback data (region == "unknown")
	// Fallback data only has minimal test data, not full ELB pricing
	isFallback := region == "unknown"

	// Test ALB Hourly Rate
	albHourly, albHourlyFound := client.ALBPricePerHour()
	if albHourlyFound {
		if albHourly <= 0 {
			t.Errorf("ALB hourly price should be positive, got: %v", albHourly)
		}
		t.Logf("ALB hourly price = $%.4f", albHourly)
	} else if !isFallback {
		t.Errorf("ALB hourly price should be found in region %s", region)
	}

	// Test ALB LCU Rate
	albLCU, albLCUFound := client.ALBPricePerLCU()
	if albLCUFound {
		if albLCU <= 0 {
			t.Errorf("ALB LCU price should be positive, got: %v", albLCU)
		}
		t.Logf("ALB LCU price = $%.4f", albLCU)
	} else if !isFallback {
		t.Errorf("ALB LCU price should be found in region %s", region)
	}

	// Test NLB Hourly Rate
	nlbHourly, nlbHourlyFound := client.NLBPricePerHour()
	if nlbHourlyFound {
		if nlbHourly <= 0 {
			t.Errorf("NLB hourly price should be positive, got: %v", nlbHourly)
		}
		t.Logf("NLB hourly price = $%.4f", nlbHourly)
	} else if !isFallback {
		t.Errorf("NLB hourly price should be found in region %s", region)
	}

	// Test NLB NLCU Rate
	nlbNLCU, nlbNLCUFound := client.NLBPricePerNLCU()
	if nlbNLCUFound {
		if nlbNLCU <= 0 {
			t.Errorf("NLB NLCU price should be positive, got: %v", nlbNLCU)
		}
		t.Logf("NLB NLCU price = $%.4f", nlbNLCU)
	} else if !isFallback {
		t.Errorf("NLB NLCU price should be found in region %s", region)
	}

	// Optional: strict check only for us-east-1
	if region == "us-east-1" {
		if albHourlyFound && albHourly != 0.0225 {
			t.Errorf("us-east-1: Expected ALB hourly price 0.0225, got %v", albHourly)
		}
		if albLCUFound && albLCU != 0.008 {
			t.Errorf("us-east-1: Expected ALB LCU price 0.008, got %v", albLCU)
		}
		if nlbHourlyFound && nlbHourly != 0.0225 {
			t.Errorf("us-east-1: Expected NLB hourly price 0.0225, got %v", nlbHourly)
		}
		if nlbNLCUFound && nlbNLCU != 0.006 {
			t.Errorf("us-east-1: Expected NLB NLCU price 0.006, got %v", nlbNLCU)
		}
	}
}

// TestEmbeddedData_EC2OfferCode verifies that EC2 embedded data contains correct offerCode (T028).
// This ensures service isolation - each embedded file contains only its service's data.
func TestEmbeddedData_EC2OfferCode(t *testing.T) {
	// Parse the raw EC2 JSON to verify offerCode
	var pricing struct {
		OfferCode string `json:"offerCode"`
	}
	if err := json.Unmarshal(rawEC2JSON, &pricing); err != nil {
		t.Fatalf("Failed to parse rawEC2JSON: %v", err)
	}

	if pricing.OfferCode != "AmazonEC2" {
		t.Errorf("EC2 offerCode = %q, want %q", pricing.OfferCode, "AmazonEC2")
	}
}

// TestEmbeddedData_ELBOfferCode verifies that ELB embedded data contains correct offerCode (T029).
// This ensures service isolation - each embedded file contains only its service's data.
func TestEmbeddedData_ELBOfferCode(t *testing.T) {
	// Parse the raw ELB JSON to verify offerCode
	var pricing struct {
		OfferCode string `json:"offerCode"`
	}
	if err := json.Unmarshal(rawELBJSON, &pricing); err != nil {
		t.Fatalf("Failed to parse rawELBJSON: %v", err)
	}

	if pricing.OfferCode != "AWSELB" {
		t.Errorf("ELB offerCode = %q, want %q", pricing.OfferCode, "AWSELB")
	}
}

// TestEmbeddedData_MetadataPreservation verifies AWS metadata preservation (T032).
// This test ensures version and publicationDate fields are present in embedded pricing data.
// These fields are critical for debugging and knowing which AWS pricing version is embedded.
func TestEmbeddedData_MetadataPreservation(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pricing struct {
				FormatVersion   string `json:"formatVersion"`
				Version         string `json:"version"`
				PublicationDate string `json:"publicationDate"`
				Disclaimer      string `json:"disclaimer"`
			}
			if err := json.Unmarshal(tt.data, &pricing); err != nil {
				t.Fatalf("Failed to parse %s JSON: %v", tt.name, err)
			}

			// Version should be a timestamp-like string (e.g., "20251218235654")
			if pricing.Version == "" {
				t.Errorf("%s: version field is empty, should contain AWS pricing version", tt.name)
			} else {
				t.Logf("%s: version = %s", tt.name, pricing.Version)
			}

			// PublicationDate should be an ISO timestamp (e.g., "2025-12-18T23:56:54Z")
			if pricing.PublicationDate == "" {
				t.Errorf("%s: publicationDate field is empty, should contain ISO timestamp", tt.name)
			} else {
				t.Logf("%s: publicationDate = %s", tt.name, pricing.PublicationDate)
			}

			// FormatVersion should be present (e.g., "v1.0")
			if pricing.FormatVersion == "" {
				t.Errorf("%s: formatVersion field is empty", tt.name)
			}

			// Disclaimer should be present
			if pricing.Disclaimer == "" {
				t.Errorf("%s: disclaimer field is empty", tt.name)
			}
		})
	}
}

// TestEmbeddedData_AllServicesOfferCodes verifies all services have correct offerCodes.
// This is a comprehensive check that all per-service files contain only their service's data.
func TestEmbeddedData_AllServicesOfferCodes(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		wantOfferCode string
	}{
		{"EC2", rawEC2JSON, "AmazonEC2"},
		{"S3", rawS3JSON, "AmazonS3"},
		{"RDS", rawRDSJSON, "AmazonRDS"},
		{"EKS", rawEKSJSON, "AmazonEKS"},
		{"Lambda", rawLambdaJSON, "AWSLambda"},
		{"DynamoDB", rawDynamoDBJSON, "AmazonDynamoDB"},
		{"ELB", rawELBJSON, "AWSELB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pricing struct {
				OfferCode string `json:"offerCode"`
			}
			if err := json.Unmarshal(tt.data, &pricing); err != nil {
				t.Fatalf("Failed to parse %s JSON: %v", tt.name, err)
			}
			if pricing.OfferCode != tt.wantOfferCode {
				t.Errorf("%s offerCode = %q, want %q", tt.name, pricing.OfferCode, tt.wantOfferCode)
			}
		})
	}
}

// BenchmarkNewClient measures the initialization performance of the pricing client.
//
// This benchmark tracks the time to parse all embedded pricing data (EC2, S3, RDS,
// EKS, Lambda, DynamoDB, ELB) using parallel goroutines. Use this to detect
// performance regressions when modifying the parsing logic.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkNewClient -benchmem ./internal/pricing/...
//
// Expected baseline (us-east-1, ~174MB embedded data):
//   - Time: ~500-800ms per initialization
//   - Allocations: Proportional to product count (~90k EC2 products)
func BenchmarkNewClient(b *testing.B) {
	logger := zerolog.Nop()

	// Track memory allocations for regression detection
	b.ReportAllocs()

	// Reset timer after setup
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, err := NewClient(logger)
		if err != nil {
			b.Fatalf("NewClient() failed: %v", err)
		}
		// Prevent compiler optimization
		if client.Region() == "" {
			b.Fatal("unexpected empty region")
		}
	}
}

// BenchmarkNewClient_Parallel measures initialization under concurrent load.
//
// This simulates multiple plugin instances starting simultaneously,
// which can happen in multi-resource Pulumi deployments.
//
// Run with: go test -tags=region_use1 -bench=BenchmarkNewClient_Parallel -benchmem ./internal/pricing/...
func BenchmarkNewClient_Parallel(b *testing.B) {
	logger := zerolog.Nop()

	// Track memory allocations for regression detection
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client, err := NewClient(logger)
			if err != nil {
				b.Fatalf("NewClient() failed: %v", err)
			}
			if client.Region() == "" {
				b.Fatal("unexpected empty region")
			}
		}
	})
}
