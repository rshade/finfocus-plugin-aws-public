package pricing

import "testing"

func natProduct(sku, usageType, operation, unit, usd string) fixtureSKU {
	return fixtureSKU{
		sku:    sku,
		family: "NAT Gateway",
		attrs: map[string]string{
			"usagetype":    usageType,
			"operation":    operation,
			"group":        "NGW:NatGateway",
			"locationType": "AWS Region",
			"regionCode":   "us-east-1",
		},
		dims: []fixtureDim{{begin: "0", end: "Inf", unit: unit, usd: usd}},
	}
}

// natGatewayVariants mirrors the NAT Gateway products in the AmazonEC2 offer
// (us-east-1, 2026-09-24). Variant prices are distinct from the standard ones
// so a variant overwriting the standard rate is detectable; in the real data
// RegionalNatGateway-Hours costs the same $0.045 as NatGateway-Hours, which
// is how a substring match hid the collision.
func natGatewayVariants(standardPrefix string) []fixtureSKU {
	return []fixtureSKU{
		natProduct("HOURS", standardPrefix+"NatGateway-Hours", "NatGateway", "Hrs", "0.045"),
		natProduct("BYTES", standardPrefix+"NatGateway-Bytes", "NatGateway", "GB", "0.046"),
		natProduct("REG-HOURS", "RegionalNatGateway-Hours", "RegionalNatGateway", "Hrs", "0.051"),
		natProduct("REG-BYTES", "USE1-RegionalNatGateway-Bytes", "RegionalNatGateway", "GB", "0.052"),
		natProduct("PRVD-GBPS", "NatGateway-Prvd-Gbps", "NatGateway", "Gbps-hrs", "1.076"),
		natProduct("PRVD-BYTES", "NatGateway-Prvd-Bytes", "NatGateway", "GB", "0.0000000000"),
	}
}

func assertNATGatewayPrice(t *testing.T, run int, got *NATGatewayPrice, wantHourly, wantData float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("run %d: NAT Gateway price not indexed", run)
	}
	if got.HourlyRate != wantHourly || got.DataProcessingRate != wantData {
		t.Fatalf("run %d: NAT Gateway = hourly %v / data %v, want %v / %v",
			run, got.HourlyRate, got.DataProcessingRate, wantHourly, wantData)
	}
}

// TestParseEC2Pricing_IndexesNATGateway verifies NAT Gateway prices are read
// from the AmazonEC2 offer, where AWS now publishes them (#389).
//
// The fixture carries the regional NAT gateway and provisioned-bandwidth
// products alongside the standard ones; only the standard NatGateway-Hours and
// NatGateway-Bytes usage types may be indexed, on every parse.
func TestParseEC2Pricing_IndexesNATGateway(t *testing.T) {
	raw := buildPriceList(t, "AmazonEC2", natGatewayVariants(""))

	for i := range parseRepetitions {
		c := newParserTestClient()
		region, _, err := c.parseEC2Pricing(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if region != "us-east-1" {
			t.Fatalf("parsed region = %q, want us-east-1", region)
		}
		assertNATGatewayPrice(t, i, c.natGatewayEC2, 0.045, 0.046)
	}
}

// TestParseNATGatewayPricing_VPCOfferExactUsageMatch verifies the legacy
// AmazonVPC path uses the same exact usage-type matching as the EC2 path, and
// accepts the region-code prefix used outside us-east-1 (ca-central-1 lists
// "CAN1-NatGateway-Hours" in the AmazonEC2 offer).
func TestParseNATGatewayPricing_VPCOfferExactUsageMatch(t *testing.T) {
	raw := buildPriceList(t, "AmazonVPC", natGatewayVariants("CAN1-"))

	for i := range parseRepetitions {
		c := newParserTestClient()
		assertParsedRegion(t, c.parseNATGatewayPricing, raw)
		assertNATGatewayPrice(t, i, c.natGatewayVPC, 0.045, 0.046)
	}
}

// TestParseEC2Pricing_NoNATGatewayProducts verifies an EC2 offer without NAT
// Gateway products (older data, or the fallback build) leaves the EC2 NAT
// price unset so the VPC offer can supply it.
func TestParseEC2Pricing_NoNATGatewayProducts(t *testing.T) {
	raw := buildPriceList(t, "AmazonEC2", []fixtureSKU{{
		sku:    "EBS",
		family: "Storage",
		attrs:  map[string]string{"volumeApiName": "gp3", "regionCode": "us-east-1"},
		dims:   gbMonth("0.08"),
	}})

	c := newParserTestClient()
	if _, _, err := c.parseEC2Pricing(raw); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.natGatewayEC2 != nil {
		t.Fatalf("natGatewayEC2 = %+v, want nil when the offer has no NAT Gateway products", c.natGatewayEC2)
	}
}

// TestNATGatewayUsage verifies region-prefix stripping for NAT Gateway usage
// types. Cases are real usage types from the AmazonEC2 offer (us-east-1 and
// ca-central-1, 2026-09-24), including us-east-1's inconsistent prefixes.
func TestNATGatewayUsage(t *testing.T) {
	tests := []struct {
		usageType string
		want      string
	}{
		{"NatGateway-Hours", "NatGateway-Hours"},
		{"CAN1-NatGateway-Hours", "NatGateway-Hours"},
		{"CAN1-NatGateway-Bytes", "NatGateway-Bytes"},
		{"CAN1-RegionalNatGateway-Hours", "RegionalNatGateway-Hours"},
		{"USE1-RegionalNatGateway-Bytes", "RegionalNatGateway-Bytes"},
		{"RegionalNatGateway-Hours", "RegionalNatGateway-Hours"},
		{"NatGateway-Prvd-Bytes", "NatGateway-Prvd-Bytes"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := natGatewayUsage(tt.usageType); got != tt.want {
			t.Errorf("natGatewayUsage(%q) = %q, want %q", tt.usageType, got, tt.want)
		}
	}
}

// TestSelectNATGatewayPrice verifies the EC2 offer is preferred and the VPC
// offer is the fallback, and that a price without an hourly rate is ignored.
func TestSelectNATGatewayPrice(t *testing.T) {
	ec2 := &NATGatewayPrice{HourlyRate: 0.045, DataProcessingRate: 0.045, Currency: currencyUSD}
	vpc := &NATGatewayPrice{HourlyRate: 0.040, DataProcessingRate: 0.040, Currency: currencyUSD}
	noHourly := &NATGatewayPrice{DataProcessingRate: 0.045, Currency: currencyUSD}

	tests := []struct {
		name     string
		ec2, vpc *NATGatewayPrice
		want     *NATGatewayPrice
	}{
		{"both present prefers EC2", ec2, vpc, ec2},
		{"EC2 only", ec2, nil, ec2},
		{"VPC fallback", nil, vpc, vpc},
		{"EC2 without hourly rate falls back to VPC", noHourly, vpc, vpc},
		{"neither", nil, nil, nil},
		{"only incomplete prices", noHourly, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectNATGatewayPrice(tt.ec2, tt.vpc); got != tt.want {
				t.Fatalf("selectNATGatewayPrice() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
