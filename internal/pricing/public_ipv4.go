package pricing

import (
	"fmt"

	"github.com/goccy/go-json"
)

// Usage types for the public IPv4 address charge, after the region prefix is
// removed. AWS bills an Elastic IP under one of these two names depending on
// whether the address is associated with a resource.
const (
	usagePublicIPv4InUse = "PublicIPv4:InUseAddress"
	usagePublicIPv4Idle  = "PublicIPv4:IdleAddress"
)

// PublicIPv4Price represents the regional hourly charge for a public IPv4
// address, which is how an Elastic IP is billed. Derived from the AmazonVPC
// offer. The products carry no productFamily, so they are matched by usage
// type; "PublicIPv4:ContiguousBlock" is IPAM and is deliberately ignored.
type PublicIPv4Price struct {
	// InUseHourly is the hourly rate for an address associated with a resource.
	// Source: usageType "PublicIPv4:InUseAddress".
	InUseHourly float64

	// IdleHourly is the hourly rate for an address that is not associated.
	// Source: usageType "PublicIPv4:IdleAddress".
	IdleHourly float64

	// Currency code (e.g., "USD")
	Currency string
}

// parsePublicIPv4Pricing parses public IPv4 address rates from the AmazonVPC
// offer. Both usage types must be present for the price to be served, so a
// partial price never reaches an estimate. Returns any parsing error.
func (c *Client) parsePublicIPv4Pricing(data []byte) error {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return fmt.Errorf("failed to parse VPC JSON: %w", err)
	}

	price := &PublicIPv4Price{Currency: currencyUSD}
	for sku, prod := range pricing.Products {
		usage := natGatewayUsage(prod.Attributes["usagetype"])
		if usage != usagePublicIPv4InUse && usage != usagePublicIPv4Idle {
			continue
		}
		rate, unit, found := getOnDemandPrice(&pricing, sku)
		if !found || !isHourlyUnit(unit) {
			continue
		}
		if usage == usagePublicIPv4InUse {
			price.InUseHourly = rate
		} else {
			price.IdleHourly = rate
		}
	}
	if price.InUseHourly > 0 && price.IdleHourly > 0 {
		c.publicIPv4Pricing = price
	}
	return nil
}

// PublicIPv4Price returns the public IPv4 address rates for the plugin region.
// Returns (price, true) if found, (nil, false) if not found.
func (c *Client) PublicIPv4Price() (*PublicIPv4Price, bool) {
	if err := c.init(); err != nil {
		return nil, false
	}
	if c.publicIPv4Pricing == nil {
		return nil, false
	}
	return c.publicIPv4Pricing, true
}
