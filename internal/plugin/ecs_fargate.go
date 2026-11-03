package plugin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
)

const ecsFargateService = "ECS Fargate"

// isECSServiceType reports whether a lowercased resource type is the Pulumi
// ECS service token. The trailing colon keeps other ecs/ types out.
func isECSServiceType(lowerResourceType string) bool {
	return strings.HasPrefix(lowerResourceType, "aws:ecs/service:") ||
		lowerResourceType == "aws:ecs/service"
}

// fargateHourlyRate returns the per-task hourly cost for the task configuration.
func (p *AWSPublicPlugin) fargateHourlyRate(
	resource *pbc.ResourceDescriptor,
	attrs FargateAttributes,
) (float64, error) {
	var hourly float64
	switch {
	case attrs.Windows:
		win, found := p.pricing.FargateWindowsOnDemandRates()
		if !found {
			return 0, &PricingUnavailableError{
				Service:       ecsFargateService,
				SKU:           resource.GetSku(),
				BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "ECS Fargate Windows rates", resource.GetRegion()),
			}
		}
		hourly = attrs.VCPU*(win.VCPUPerHour+win.OSPerVCPUHour) + attrs.MemoryGiB*win.GiBPerHour
	default:
		arch := pricing.FargateArchX86
		if attrs.ARM {
			arch = pricing.FargateArchARM
		}
		rates, found := p.pricing.FargateOnDemandRates(arch)
		if !found {
			return 0, &PricingUnavailableError{
				Service:       ecsFargateService,
				SKU:           resource.GetSku(),
				BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "ECS Fargate "+arch+" rates", resource.GetRegion()),
			}
		}
		hourly = attrs.VCPU*rates.VCPUPerHour + attrs.MemoryGiB*rates.GiBPerHour
	}

	if extra := attrs.EphemeralGiB - fargateFreeEphemeralGiB; extra > 0 {
		rate, found := p.pricing.FargateEphemeralStoragePerGiBHour()
		if !found {
			return 0, &PricingUnavailableError{
				Service: ecsFargateService,
				SKU:     resource.GetSku(),
				BillingDetail: fmt.Sprintf(
					PricingNotFoundTemplate,
					"ECS Fargate ephemeral storage rate",
					resource.GetRegion(),
				),
			}
		}
		hourly += extra * rate
	}
	return hourly, nil
}

// estimateECSFargate calculates the projected monthly cost of an ECS service
// that runs on Fargate.
//
// Cost formula, per task and hour:
//
//	vcpu * vcpu_rate + memory_gib * gib_rate
//	  + max(0, ephemeral_gib - 20) * ephemeral_rate
//	  + (windows ? vcpu * windows_os_rate : 0)
//
// The monthly cost is that figure times desired_count times 730 hours. A
// service on the EC2 or EXTERNAL launch type returns $0 because its cost
// belongs to the EC2 instances or the ASG. Fargate Spot is not in the Price
// List API, so a Spot request is priced at the on-demand rate with a note.
// See ExtractFargateAttributes for the accepted inputs and their defaults.
func (p *AWSPublicPlugin) estimateECSFargate(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	logger := p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger()

	reason, err := p.ecsLaunchRouting(traceID, resource)
	if err != nil {
		return nil, err
	}
	if reason != "" {
		resp := &pbc.GetProjectedCostResponse{Currency: currencyUSD, BillingDetail: reason}
		setGrowthHint(logger, "aws:ecs:service", resp)
		return resp, nil
	}

	attrs, dt, err := p.ExtractFargateAttributes(traceID, resource)
	if err != nil {
		return nil, err
	}

	hourly, err := p.fargateHourlyRate(resource, attrs)
	if err != nil {
		return nil, err
	}
	monthly := hourly * float64(attrs.DesiredCount) * HoursPerMonthProd

	osName, archName := "Linux", "x86_64"
	switch {
	case attrs.Windows:
		osName = "Windows"
	case attrs.ARM:
		archName = "arm64"
	}
	detail := fmt.Sprintf(
		"ECS Fargate: %d tasks x (%s vCPU + %s GiB), %s %s, 730 hrs/month (on-demand; carbon estimate not available)",
		attrs.DesiredCount,
		strconv.FormatFloat(attrs.VCPU, 'f', -1, 64),
		strconv.FormatFloat(attrs.MemoryGiB, 'f', -1, 64),
		osName, archName,
	)
	if extra := attrs.EphemeralGiB - fargateFreeEphemeralGiB; extra > 0 {
		detail += fmt.Sprintf(", %s GiB ephemeral storage", strconv.FormatFloat(attrs.EphemeralGiB, 'f', -1, 64))
	}
	if attrs.DesiredCount == 0 {
		detail += " (zero tasks)"
	}
	if attrs.SpotRequested {
		detail += " (Fargate Spot not priced; on-demand rate shown)"
	}

	p.traceLogger(traceID, "GetProjectedCost").Debug().
		Float64("vcpu", attrs.VCPU).
		Float64("memory_gib", attrs.MemoryGiB).
		Int("desired_count", attrs.DesiredCount).
		Float64("hourly_rate", hourly).
		Float64("monthly_cost", monthly).
		Msg("ECS Fargate cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  monthly,
		UnitPrice:     hourly,
		Currency:      currencyUSD,
		BillingDetail: detail,
		Metadata:      dt.Metadata(),
	}
	setGrowthHint(logger, "aws:ecs:service", resp)
	return resp, nil
}

// fargatePricingSpec returns the pricing specification for an ECS service.
// The rate is the per-task hourly cost at the service's configuration.
func (p *AWSPublicPlugin) fargatePricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	spec := &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModeOnDemand,
		Currency:     currencyUSD,
		Unit:         unitPerTaskHour,
		Source:       sourceAWSPublic,
	}

	reason, err := p.ecsLaunchRouting("", resource)
	if err != nil {
		spec.Description = status.Convert(err).Message()
		return spec
	}
	if reason != "" {
		spec.BillingMode = billingModeZeroCost
		spec.Description = reason
		return spec
	}

	attrs, _, err := p.ExtractFargateAttributes("", resource)
	if err != nil {
		spec.Description = status.Convert(err).Message()
		return spec
	}
	hourly, err := p.fargateHourlyRate(resource, attrs)
	if err != nil {
		spec.Description = fmt.Sprintf(PricingNotFoundTemplate, "ECS Fargate rates", resource.GetRegion())
		spec.Assumptions = []string{"Fargate rates not found in embedded pricing data"}
		return spec
	}
	spec.RatePerUnit = hourly
	spec.Description = "ECS Fargate cost = per-task hourly rate x desired_count x 730 hrs/month"
	spec.Assumptions = []string{
		"Per-task rate = vCPU and memory rates, plus ephemeral storage above 20 GiB",
		"Windows tasks add the per-vCPU OS license fee",
		"On-demand pricing only; Fargate Spot and Savings Plans are not priced",
		"Cost = per-task hourly rate x desired_count x 730",
	}
	return spec
}
