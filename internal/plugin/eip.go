package plugin

import (
	"fmt"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
)

const (
	// eipStateInUse and eipStateIdle name the two public IPv4 rates in billing detail.
	eipStateInUse = "in-use"
	eipStateIdle  = "idle"

	// eipGrowthKey is the classification key for Elastic IP growth hints.
	eipGrowthKey = "aws:ec2:eip"
)

// eipAssociationKeys are the canonical input names that show an Elastic IP is
// associated with a resource. Any non-empty one selects the in-use rate.
var eipAssociationKeys = []string{"instance", "network_interface", "association_id"}

// isElasticIPType reports whether a lower-cased Pulumi type is an Elastic IP,
// such as "aws:ec2/eip:eip". The match ends at the type separator, so
// "ec2/eipassociation" is not an Elastic IP.
func isElasticIPType(rt string) bool {
	rest, ok := strings.CutPrefix(rt, "aws:ec2/eip")
	return ok && (rest == "" || rest[0] == ':')
}

// eipAssociated reports whether the descriptor names a resource the address is
// associated with, on attributes or tags.
func eipAssociated(resource *pbc.ResourceDescriptor) bool {
	tags := pricingTags(resource)
	for _, key := range eipAssociationKeys {
		if tags[key] != "" {
			return true
		}
	}
	return false
}

// eipRate returns the hourly rate and the state name that apply to resource.
func eipRate(price *pricing.PublicIPv4Price, resource *pbc.ResourceDescriptor) (float64, string) {
	if eipAssociated(resource) {
		return price.InUseHourly, eipStateInUse
	}
	return price.IdleHourly, eipStateIdle
}

// estimateEIP calculates the projected monthly cost of an Elastic IP.
//
// AWS bills every public IPv4 address by the hour, whether or not it is
// associated, so the cost is the hourly rate times 730. The in-use rate applies
// when the descriptor names an instance, network interface, or association;
// otherwise the idle rate applies. The EC2 free tier and data transfer are
// not included.
func (p *AWSPublicPlugin) estimateEIP(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	price, found := p.pricing.PublicIPv4Price()
	if !found {
		return nil, &PricingUnavailableError{
			Service:       "PublicIPv4",
			SKU:           p.region,
			BillingDetail: fmt.Sprintf(PricingUnavailableTemplate, "Elastic IP", p.region),
		}
	}

	rate, state := eipRate(price, resource)
	totalCost := rate * carbon.HoursPerMonth

	detail := fmt.Sprintf(
		"Elastic IP (public IPv4, %s), %d hrs/month ($%.3f/hr) (free tier and data transfer not included)",
		state, int(carbon.HoursPerMonth), rate,
	)

	p.logger.Debug().
		Str(pluginsdk.FieldTraceID, traceID).
		Str("state", state).
		Float64("hourly_rate", rate).
		Float64("total_cost", totalCost).
		Msg("Elastic IP cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  totalCost,
		UnitPrice:     rate,
		Currency:      currencyUSD,
		BillingDetail: detail,
	}

	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), eipGrowthKey, resp)

	return resp, nil
}

// eipPricingSpec returns the pricing specification for an Elastic IP.
func (p *AWSPublicPlugin) eipPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	price, found := p.pricing.PublicIPv4Price()
	if !found || price == nil {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          resource.GetSku(),
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerHour,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Description:  "Elastic IP pricing not found in embedded data",
			Source:       sourceAWSPublic,
			Assumptions:  []string{"Public IPv4 pricing data not available"},
		}
	}

	rate, state := eipRate(price, resource)
	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerHour,
		RatePerUnit:  rate,
		Currency:     currencyUSD,
		Unit:         unitHour,
		Description:  "Elastic IP (public IPv4 address)",
		Source:       sourceAWSPublic,
		Assumptions: []string{
			fmt.Sprintf("Hourly rate: $%.4f (%s address)", rate, state),
			fmt.Sprintf("%d hours per month", int(carbon.HoursPerMonth)),
			"Billed whether or not the address is associated",
			"EC2 free tier of 750 public IPv4 hours per month not applied",
			"Data transfer billed separately",
		},
	}
}
