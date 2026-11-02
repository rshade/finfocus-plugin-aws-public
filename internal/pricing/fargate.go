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

// FargateWindowsRates holds the on-demand Windows Fargate rates. A Windows
// task pays the Windows vCPU and memory rates plus an OS license fee per vCPU.
type FargateWindowsRates struct {
	// VCPUPerHour is the price of one Windows vCPU for one hour in USD.
	VCPUPerHour float64
	// GiBPerHour is the price of one GiB of Windows task memory for one hour in USD.
	GiBPerHour float64
	// OSPerVCPUHour is the Windows license fee per vCPU-hour in USD.
	OSPerVCPUHour float64
}

const (
	usageFargateWindowsVCPU = "Fargate-Windows-vCPU-Hours:perCPU"
	usageFargateWindowsGiB  = "Fargate-Windows-GB-Hours"
	usageFargateWindowsOS   = "Fargate-Windows-OS-Hours:perCPU"
	usageFargateEphemeral   = "Fargate-EphemeralStorage-GB-Hours"
	unitGBHours             = "GB-Hours"
)

// fargateUsage maps Price List usage types (region prefix removed) to the
// architecture and rate they price. Windows and ephemeral storage usage types
// are handled by applyFargateExtraProduct, and Spot is not priced.
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
	var windows FargateWindowsRates
	var ephemeral float64
	for sku, prod := range pricing.Products {
		if region == "" && prod.Attributes["regionCode"] != "" {
			region = prod.Attributes["regionCode"]
		}
		applyFargateProduct(rates, &pricing, sku, prod)
		applyFargateExtraProduct(&windows, &ephemeral, &pricing, sku, prod)
	}

	for arch, r := range rates {
		if r.VCPUPerHour > 0 && r.GiBPerHour > 0 {
			c.fargateRates[arch] = r
		}
	}
	if windows.VCPUPerHour > 0 && windows.GiBPerHour > 0 && windows.OSPerVCPUHour > 0 {
		c.fargateWindows = windows
		c.fargateWindowsFound = true
	}
	c.fargateEphemeralPerGiBHour = ephemeral
	return region, nil
}

// applyFargateExtraProduct records the Windows compute, Windows license, and
// ephemeral storage rates and ignores every other product.
func applyFargateExtraProduct(
	windows *FargateWindowsRates,
	ephemeral *float64,
	data *awsPricing,
	sku string,
	prod product,
) {
	attrs := prod.Attributes
	if prod.ProductFamily != "Compute" || attrs["locationType"] != locationTypeAWSRegion {
		return
	}
	usage := natGatewayUsage(attrs["usagetype"])
	switch usage {
	case usageFargateWindowsVCPU, usageFargateWindowsGiB, usageFargateWindowsOS, usageFargateEphemeral:
	default:
		return
	}
	rate, unit, found := getOnDemandPrice(data, sku)
	if !found || rate <= 0 {
		return
	}
	if usage == usageFargateEphemeral {
		if strings.EqualFold(unit, unitGBHours) {
			*ephemeral = rate
		}
		return
	}
	if !isHourlyUnit(unit) {
		return
	}
	switch usage {
	case usageFargateWindowsVCPU:
		windows.VCPUPerHour = rate
	case usageFargateWindowsGiB:
		windows.GiBPerHour = rate
	case usageFargateWindowsOS:
		windows.OSPerVCPUHour = rate
	}
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

// FargateWindowsOnDemandRates returns the on-demand Windows Fargate vCPU-hour,
// GiB-hour, and per-vCPU license rates for the client's region. All three
// rates are positive when found is true.
func (c *Client) FargateWindowsOnDemandRates() (FargateWindowsRates, bool) {
	if err := c.init(); err != nil {
		return FargateWindowsRates{}, false
	}
	return c.fargateWindows, c.fargateWindowsFound
}

// FargateEphemeralStoragePerGiBHour returns the on-demand price in USD of one
// GiB of Fargate ephemeral storage above the free 20 GiB, for one hour.
// Returns (0, false) when the region publishes no such rate.
func (c *Client) FargateEphemeralStoragePerGiBHour() (float64, bool) {
	if err := c.init(); err != nil {
		return 0, false
	}
	return c.fargateEphemeralPerGiBHour, c.fargateEphemeralPerGiBHour > 0
}
