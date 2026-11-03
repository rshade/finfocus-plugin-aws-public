package plugin

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
)

// serviceEKSFargate identifies a pod scheduled on an EKS Fargate virtual node
// (aws:eks/fargate:Pod). It is distinct from serviceEKS, the cluster fee.
const serviceEKSFargate = "eksfargate"

const (
	eksFargateTagCPU    = "cpu"
	eksFargateTagMemory = "memory_gib"

	eksFargatePodType = "aws:eks/fargate:pod"
)

// isEKSFargatePodType reports whether a lowercased resource type is the
// Pulumi EKS Fargate pod token. The full token match keeps other
// aws:eks/fargate:* types and aws:eks/fargateProfile:FargateProfile out.
func isEKSFargatePodType(lowerResourceType string) bool {
	return lowerResourceType == eksFargatePodType
}

// parseFargateQuantity parses a positive, finite decimal such as "0.25".
func parseFargateQuantity(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0, false
	}
	return v, true
}

// estimateEKSFargate calculates the projected monthly cost of one EKS Fargate
// pod from its requested vCPU and memory.
//
// Cost formula: (vCPU rate * cpu + GiB rate * memory_gib) * 730 hours/month,
// at on-demand Linux x86_64 rates.
//
// Required inputs, read from attributes first and then tags:
//   - "cpu": requested vCPU as a decimal, in cores (for example "0.25")
//   - "memory_gib": requested memory in GiB (for example "0.5")
//
// Fargate rounds a pod up to a supported size and adds ephemeral storage
// above 20 GiB; neither is modeled.
func (p *AWSPublicPlugin) estimateEKSFargate(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	tags := pricingTags(resource)

	cpu, ok := parseFargateQuantity(tags[eksFargateTagCPU])
	if !ok {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
			fmt.Sprintf("EKS Fargate pod requires a positive %q (vCPU cores), got %q",
				eksFargateTagCPU, tags[eksFargateTagCPU]),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}
	memoryGiB, ok := parseFargateQuantity(tags[eksFargateTagMemory])
	if !ok {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
			fmt.Sprintf("EKS Fargate pod requires a positive %q (memory GiB), got %q",
				eksFargateTagMemory, tags[eksFargateTagMemory]),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	rates, found := p.pricing.FargateOnDemandRates("")
	if !found {
		return nil, &PricingUnavailableError{
			Service:       "EKS Fargate",
			SKU:           resource.GetSku(),
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "EKS Fargate rates", resource.GetRegion()),
		}
	}

	hourly := rates.VCPUPerHour*cpu + rates.GiBPerHour*memoryGiB
	monthly := hourly * HoursPerMonthProd
	if math.IsNaN(hourly) || math.IsInf(hourly, 0) || math.IsNaN(monthly) || math.IsInf(monthly, 0) {
		return nil, p.newErrorWithID(traceID, codes.InvalidArgument,
			fmt.Sprintf("EKS Fargate pod cost is not finite for cpu %q and memory_gib %q",
				tags[eksFargateTagCPU], tags[eksFargateTagMemory]),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
	}

	p.traceLogger(traceID, "GetProjectedCost").Debug().
		Float64("cpu", cpu).
		Float64("memory_gib", memoryGiB).
		Float64("hourly_rate", hourly).
		Float64("monthly_cost", monthly).
		Msg("EKS Fargate pod cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth: monthly,
		UnitPrice:    hourly,
		Currency:     currencyUSD,
		BillingDetail: fmt.Sprintf(
			"EKS Fargate pod, %s vCPU + %s GiB, 730 hrs/month "+
				"(on-demand Linux x86_64; excludes Fargate size rounding and ephemeral storage)",
			strconv.FormatFloat(cpu, 'f', -1, 64), strconv.FormatFloat(memoryGiB, 'f', -1, 64)),
	}

	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:eks:fargate-pod", resp)

	return resp, nil
}
