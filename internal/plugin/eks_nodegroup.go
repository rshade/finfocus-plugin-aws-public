package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
)

// estimateEKSNodeGroup prices a managed node group as N on-demand EC2
// instances. The group has no fee of its own. Root EBS is excluded.
func (p *AWSPublicPlugin) estimateEKSNodeGroup(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	inst, count, osName, note, dt, err := p.eksNodeGroupShape(traceID, resource)
	if err != nil {
		return nil, err
	}
	hourly, found := p.pricing.EC2OnDemandPricePerHour(inst, osName, defaultTenancy)
	if !found {
		return nil, &PricingUnavailableError{
			Service:       eksNodeGroupServiceName,
			SKU:           inst,
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", inst),
		}
	}
	detail := fmt.Sprintf(
		"EKS managed node group: %d× %s On-demand %s, 730 hrs/month (EBS root volumes excluded)",
		count, inst, osName,
	)
	if note != "" {
		detail += " " + note
	}
	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:     hourly,
		CostPerMonth:  hourly * float64(count) * carbon.HoursPerMonth,
		Currency:      currencyUSD,
		BillingDetail: detail,
		Metadata:      dt.Metadata(),
	}
	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:eks:nodegroup", resp)
	return resp, nil
}

// eksNodeGroupPricingSpec returns the per-instance hourly rate for a node group.
func (p *AWSPublicPlugin) eksNodeGroupPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	spec := &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerHour,
		Currency:     currencyUSD,
		Unit:         unitPerInstanceHour,
		Source:       sourceAWSPublic,
	}
	inst, count, osName, note, _, err := p.eksNodeGroupShape("", resource)
	if err != nil {
		spec.Description = pricingUnavailableDetail(err)
		spec.RatePerUnit = 0
		return spec
	}
	hourly, found := p.pricing.EC2OnDemandPricePerHour(inst, osName, defaultTenancy)
	if !found {
		spec.Description = fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", inst)
		return spec
	}
	spec.RatePerUnit = hourly
	spec.Sku = inst
	spec.Description = fmt.Sprintf("EKS managed node group: %d× %s On-demand %s", count, inst, osName)
	if note != "" {
		spec.Description += " " + note
	}
	return spec
}

const eksNodeGroupServiceName = "EKS node group"

func pricingUnavailableDetail(err error) string {
	if pue, ok := errors.AsType[*PricingUnavailableError](err); ok && pue.BillingDetail != "" {
		return pue.BillingDetail
	}
	return err.Error()
}

// eksNodeGroupShape reads the instance type, desired size, and OS.
// The first instanceTypes entry is the on-demand type. With no list and no
// launch template, the AWS default is t3.medium. Spot is not estimated.
func (p *AWSPublicPlugin) eksNodeGroupShape(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (string, int, string, string, *DefaultsTracker, error) {
	in := newResourceInputs(resource)
	var dt DefaultsTracker
	if capType, ok := in.String("capacityType", "capacity_type"); ok && strings.EqualFold(capType, "SPOT") {
		return "", 0, "", "", &dt, &PricingUnavailableError{
			Service:       eksNodeGroupServiceName,
			SKU:           "",
			BillingDetail: "Spot node groups are not estimated",
		}
	}
	inst := strings.TrimSpace(resource.GetSku())
	if inst == "" {
		if v, ok := in.String("instance_type", "instanceType"); ok {
			inst = v
		}
	}
	types := in.Strings("instanceTypes", "instance_types")
	fromList := false
	if inst == "" && len(types) > 0 {
		inst = types[0]
		fromList = true
	}
	hasTemplate := in.Present("launchTemplate", "launch_template")
	if inst == "" && hasTemplate {
		return "", 0, "", "", &dt, &PricingUnavailableError{
			Service:       eksNodeGroupServiceName,
			SKU:           "",
			BillingDetail: "instance type is on the launch template; set instanceTypes or instance_type",
		}
	}
	if inst == "" {
		inst = "t3.medium"
		dt.Add("instance_type", inst, KindConfig)
	}
	count := 1
	if n, ok := in.Int(
		"scalingConfig.desiredSize", "scaling_config.desired_size", "desiredSize", "desired_size",
	); ok && n >= 0 {
		count = n
	} else {
		dt.Add("desired_size", "1", KindConfig)
	}
	osName := defaultOS
	if ami, ok := in.String("amiType", "ami_type"); ok && strings.HasPrefix(strings.ToUpper(ami), "WINDOWS_") {
		osName = "Windows"
	}
	note := ""
	if fromList && len(types) > 1 {
		note = fmt.Sprintf("(%d instance types listed, using %s)", len(types), inst)
	}
	_ = traceID
	return inst, count, osName, note, &dt, nil
}
