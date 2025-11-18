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
		wantPrice    float64
	}{
		{
			name:         "t3.micro Linux Shared",
			instanceType: "t3.micro",
			os:           "Linux",
			tenancy:      "Shared",
			wantFound:    true,
			wantPrice:    0.0104,
		},
		{
			name:         "t3.small Linux Shared",
			instanceType: "t3.small",
			os:           "Linux",
			tenancy:      "Shared",
			wantFound:    true,
			wantPrice:    0.0208,
		},
		{
			name:         "nonexistent instance type",
			instanceType: "t99.mega",
			os:           "Linux",
			tenancy:      "Shared",
			wantFound:    false,
			wantPrice:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, found := client.EC2OnDemandPricePerHour(tt.instanceType, tt.os, tt.tenancy)

			if found != tt.wantFound {
				t.Errorf("EC2OnDemandPricePerHour() found = %v, want %v", found, tt.wantFound)
			}

			if price != tt.wantPrice {
				t.Errorf("EC2OnDemandPricePerHour() price = %v, want %v", price, tt.wantPrice)
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
		wantPrice  float64
	}{
		{
			name:       "gp3",
			volumeType: "gp3",
			wantFound:  true,
			wantPrice:  0.08,
		},
		{
			name:       "gp2",
			volumeType: "gp2",
			wantFound:  true,
			wantPrice:  0.10,
		},
		{
			name:       "nonexistent volume type",
			volumeType: "super-fast",
			wantFound:  false,
			wantPrice:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, found := client.EBSPricePerGBMonth(tt.volumeType)

			if found != tt.wantFound {
				t.Errorf("EBSPricePerGBMonth() found = %v, want %v", found, tt.wantFound)
			}

			if price != tt.wantPrice {
				t.Errorf("EBSPricePerGBMonth() price = %v, want %v", price, tt.wantPrice)
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
