package pricing

import (
	"testing"

	"github.com/rs/zerolog"
)

func publicIPv4Product(sku, usageType, unit, usd string) fixtureSKU {
	return fixtureSKU{
		sku: sku,
		attrs: map[string]string{
			"usagetype":  usageType,
			"group":      "VPCPublicIPv4Address",
			"regionCode": "us-west-2",
		},
		dims: []fixtureDim{{begin: "0", end: "Inf", unit: unit, usd: usd}},
	}
}

// TestParsePublicIPv4Pricing verifies that the Elastic IP hourly rates are read
// from the AmazonVPC offer by usage type.
//
// The real products have no productFamily, so the match is on the usage type
// after the region prefix is removed. The IPAM "ContiguousBlock" product costs
// more and must never be taken for an Elastic IP. A price is served only when
// both the in-use and idle rates are present, so a partial price cannot reach
// an estimate. The in-use and idle rates are set apart so a swap is detectable.
func TestParsePublicIPv4Pricing(t *testing.T) {
	tests := []struct {
		name     string
		products []fixtureSKU
		wantInUse,
		wantIdle float64
		wantFound bool
	}{
		{
			name: "in-use, idle, and the IPAM block product",
			products: []fixtureSKU{
				publicIPv4Product("INUSE", "USW2-PublicIPv4:InUseAddress", "Hrs", "0.0050000000"),
				publicIPv4Product("IDLE", "USW2-PublicIPv4:IdleAddress", "Hrs", "0.0070000000"),
				publicIPv4Product("BLOCK", "USW2-PublicIPv4:ContiguousBlock", "Hrs", "0.0080000000"),
			},
			wantInUse: 0.005,
			wantIdle:  0.007,
			wantFound: true,
		},
		{
			name: "usage type without a region prefix",
			products: []fixtureSKU{
				publicIPv4Product("INUSE", "PublicIPv4:InUseAddress", "Hrs", "0.005"),
				publicIPv4Product("IDLE", "PublicIPv4:IdleAddress", "Hrs", "0.006"),
			},
			wantInUse: 0.005,
			wantIdle:  0.006,
			wantFound: true,
		},
		{
			name: "missing idle rate is not served",
			products: []fixtureSKU{
				publicIPv4Product("INUSE", "USW2-PublicIPv4:InUseAddress", "Hrs", "0.005"),
			},
		},
		{
			name: "a non-hourly unit is ignored",
			products: []fixtureSKU{
				publicIPv4Product("INUSE", "USW2-PublicIPv4:InUseAddress", "GB", "0.005"),
				publicIPv4Product("IDLE", "USW2-PublicIPv4:IdleAddress", "Hrs", "0.005"),
			},
		},
		{
			name: "the IPAM block alone is not an Elastic IP",
			products: []fixtureSKU{
				publicIPv4Product("BLOCK", "USW2-PublicIPv4:ContiguousBlock", "Hrs", "0.008"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := buildPriceList(t, "AmazonVPC", tt.products)
			c := newParserTestClient()
			if err := c.parsePublicIPv4Pricing(raw); err != nil {
				t.Fatalf("parsePublicIPv4Pricing() error: %v", err)
			}

			got := c.publicIPv4Pricing
			if !tt.wantFound {
				if got != nil {
					t.Fatalf("publicIPv4Pricing = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("publicIPv4Pricing not indexed")
			}
			if got.InUseHourly != tt.wantInUse || got.IdleHourly != tt.wantIdle {
				t.Fatalf("rates = in-use %v / idle %v, want %v / %v",
					got.InUseHourly, got.IdleHourly, tt.wantInUse, tt.wantIdle)
			}
			if got.Currency != "USD" {
				t.Errorf("Currency = %q, want USD", got.Currency)
			}
		})
	}
}

// TestClient_PublicIPv4Price_Fallback verifies the fallback embed carries
// public IPv4 rates, so tests that build a real client without a region tag
// can price an Elastic IP.
func TestClient_PublicIPv4Price_Fallback(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	price, found := client.PublicIPv4Price()
	if !found || price == nil {
		t.Fatal("PublicIPv4Price() not found in fallback data")
	}
	if price.InUseHourly != 0.005 || price.IdleHourly != 0.005 {
		t.Errorf("rates = %v / %v, want 0.005 / 0.005", price.InUseHourly, price.IdleHourly)
	}
}
