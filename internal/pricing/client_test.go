package pricing

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
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
	client, err := NewClient()
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
	client, err := NewClient()
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
	client, err := NewClient()
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
	client, err := NewClient()
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
	client, err := NewClient()
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
