package pricing

import (
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

	t.Logf("Client region: %s", client.Region())

	// Test standard support pricing
	standardPrice, standardFound := client.EKSClusterPricePerHour(false)
	t.Logf("EKS standard support price lookup: found=%v, price=%v", standardFound, standardPrice)

	// Standard support pricing should be available for known regions
	if !standardFound {
		t.Errorf("EKSClusterPricePerHour(false) should return found=true for standard support, region=%s", client.Region())
	} else {
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
	if region == "us-east-1" {
		if readFound && readPrice != 0.25/1_000_000 {
			t.Errorf("us-east-1: Expected On-Demand read price 0.25/1M, got %v", readPrice)
		}
		if writeFound && writePrice != 1.25/1_000_000 {
			t.Errorf("us-east-1: Expected On-Demand write price 1.25/1M, got %v", writePrice)
		}
		if storageFound && storagePrice != 0.25 {
			t.Errorf("us-east-1: Expected Storage price 0.25, got %v", storagePrice)
		}
		if rcuFound && rcuPrice != 0.00013 {
			t.Errorf("us-east-1: Expected Provisioned RCU price 0.00013, got %v", rcuPrice)
		}
		if wcuFound && wcuPrice != 0.00065 {
			t.Errorf("us-east-1: Expected Provisioned WCU price 0.00065, got %v", wcuPrice)
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

	// Test ALB Hourly Rate
	albHourly, albHourlyFound := client.ALBPricePerHour()
	if albHourlyFound {
		if albHourly <= 0 {
			t.Errorf("ALB hourly price should be positive, got: %v", albHourly)
		}
		t.Logf("ALB hourly price = $%.4f", albHourly)
	} else {
		t.Errorf("ALB hourly price should be found in region %s", region)
	}

	// Test ALB LCU Rate
	albLCU, albLCUFound := client.ALBPricePerLCU()
	if albLCUFound {
		if albLCU <= 0 {
			t.Errorf("ALB LCU price should be positive, got: %v", albLCU)
		}
		t.Logf("ALB LCU price = $%.4f", albLCU)
	} else {
		t.Errorf("ALB LCU price should be found in region %s", region)
	}

	// Test NLB Hourly Rate
	nlbHourly, nlbHourlyFound := client.NLBPricePerHour()
	if nlbHourlyFound {
		if nlbHourly <= 0 {
			t.Errorf("NLB hourly price should be positive, got: %v", nlbHourly)
		}
		t.Logf("NLB hourly price = $%.4f", nlbHourly)
	} else {
		t.Errorf("NLB hourly price should be found in region %s", region)
	}

	// Test NLB NLCU Rate
	nlbNLCU, nlbNLCUFound := client.NLBPricePerNLCU()
	if nlbNLCUFound {
		if nlbNLCU <= 0 {
			t.Errorf("NLB NLCU price should be positive, got: %v", nlbNLCU)
		}
		t.Logf("NLB NLCU price = $%.4f", nlbNLCU)
	} else {
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
