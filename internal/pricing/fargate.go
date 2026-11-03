package pricing

import (
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

// Fargate CPU architectures accepted by FargateOnDemandRates.
const (
	FargateArchX86 = "x86_64"
	FargateArchARM = "arm64"
)

// FargateRates holds the on-demand Linux Fargate rates for one CPU
// architecture. Both EKS Fargate pods and ECS Fargate tasks bill at these
// rates.
type FargateRates struct {
	// VCPUPerHour is the price of one vCPU for one hour in USD.
	VCPUPerHour float64
	// GiBPerHour is the price of one GiB of memory for one hour in USD.
	GiBPerHour float64
}

// fargateUsage maps Price List usage types (region prefix removed) to the
// architecture and rate they price. Windows, ephemeral storage, and Spot
// usage types are deliberately absent.
var fargateUsage = map[string]struct {
	arch string
	vcpu bool
}{
	"Fargate-vCPU-Hours:perCPU":     {arch: FargateArchX86, vcpu: true},
	"Fargate-GB-Hours":              {arch: FargateArchX86},
	"Fargate-ARM-vCPU-Hours:perCPU": {arch: FargateArchARM, vcpu: true},
	"Fargate-ARM-GB-Hours":          {arch: FargateArchARM},
}

// parseFargatePricing parses the AmazonECS offer, which publishes the Fargate
// rates used by both ECS and EKS. Usage types are matched exactly after
// removing the optional region prefix, so Windows, ephemeral storage, and
// OS-license products are not mistaken for the Linux compute rates.
// Returns the detected region and any parsing error.
func (c *Client) parseFargatePricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse ECS JSON: %w", err)
	}

	if pricing.OfferCode != "AmazonECS" {
		c.logger.Warn().
			Str("expected", "AmazonECS").
			Str("actual", pricing.OfferCode).
			Msg("ECS pricing data has unexpected offerCode")
	}

	var region string
	rates := make(map[string]FargateRates, 2)
	for sku, prod := range pricing.Products {
		if region == "" && prod.Attributes["regionCode"] != "" {
			region = prod.Attributes["regionCode"]
		}
		applyFargateProduct(rates, &pricing, sku, prod)
	}

	for arch, r := range rates {
		if r.VCPUPerHour > 0 && r.GiBPerHour > 0 {
			c.fargateRates[arch] = r
		}
	}
	return region, nil
}

// applyFargateProduct records a Fargate compute product's on-demand hourly
// rate in rates and ignores every other product.
func applyFargateProduct(rates map[string]FargateRates, data *awsPricing, sku string, prod product) {
	attrs := prod.Attributes
	if prod.ProductFamily != "Compute" || attrs["locationType"] != locationTypeAWSRegion {
		return
	}
	usage, ok := fargateUsage[natGatewayUsage(attrs["usagetype"])]
	if !ok {
		return
	}
	rate, unit, found := getOnDemandPrice(data, sku)
	if !found || !isHourlyUnit(unit) || rate <= 0 {
		return
	}
	r := rates[usage.arch]
	if usage.vcpu {
		r.VCPUPerHour = rate
	} else {
		r.GiBPerHour = rate
	}
	rates[usage.arch] = r
}

// FargateOnDemandRates returns the on-demand Linux Fargate vCPU-hour and
// GiB-hour rates for the client's region.
//
// architecture is FargateArchX86 ("x86_64", also used for ""), or
// FargateArchARM ("arm64" or "arm"), case-insensitive. Both rates are
// positive when found is true. Returns (zero value, false) when the
// architecture is unknown or the region has no complete pair of rates.
func (c *Client) FargateOnDemandRates(architecture string) (FargateRates, bool) {
	start := time.Now()
	defer func() {
		if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "Fargate").
				Str("architecture", architecture).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return FargateRates{}, false
	}

	switch strings.ToLower(strings.TrimSpace(architecture)) {
	case "", FargateArchX86, "amd64", "x86":
		architecture = FargateArchX86
	case FargateArchARM, "arm":
		architecture = FargateArchARM
	default:
		return FargateRates{}, false
	}

	r, found := c.fargateRates[architecture]
	return r, found
}
