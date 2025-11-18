package plugin

import (
	"testing"
)

// mockPricingClient is a test double for pricing.PricingClient.
type mockPricingClient struct {
	region            string
	currency          string
	ec2Prices         map[string]float64 // key: "instanceType/os/tenancy"
	ebsPrices         map[string]float64 // key: "volumeType"
	ec2OnDemandCalled int
	ebsPriceCalled    int
}

// newMockPricingClient creates a new mockPricingClient with default values.
func newMockPricingClient(region, currency string) *mockPricingClient {
	return &mockPricingClient{
		region:    region,
		currency:  currency,
		ec2Prices: make(map[string]float64),
		ebsPrices: make(map[string]float64),
	}
}

func (m *mockPricingClient) Region() string {
	return m.region
}

func (m *mockPricingClient) Currency() string {
	return m.currency
}

func (m *mockPricingClient) EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool) {
	m.ec2OnDemandCalled++
	key := instanceType + "/" + os + "/" + tenancy
	price, found := m.ec2Prices[key]
	return price, found
}

func (m *mockPricingClient) EBSPricePerGBMonth(volumeType string) (float64, bool) {
	m.ebsPriceCalled++
	price, found := m.ebsPrices[volumeType]
	return price, found
}

func TestNewAWSPublicPlugin(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	plugin := NewAWSPublicPlugin("us-east-1", mock)

	// NewAWSPublicPlugin never returns nil
	if plugin.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", plugin.region)
	}

	if plugin.pricing != mock {
		t.Error("Pricing client not set correctly")
	}
}

func TestName(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	plugin := NewAWSPublicPlugin("us-east-1", mock)

	name := plugin.Name()
	expected := "pulumicost-plugin-aws-public"
	if name != expected {
		t.Errorf("Expected name %q, got %q", expected, name)
	}
}
