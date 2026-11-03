package plugin

import "github.com/rshade/finfocus-plugin-aws-public/internal/pricing"

// testFargateX86 and testFargateARM are the us-east-1 Linux Fargate rates
// published in the AmazonECS Price List offer.
var (
	testFargateX86 = pricing.FargateRates{VCPUPerHour: 0.04048, GiBPerHour: 0.004445}
	testFargateARM = pricing.FargateRates{VCPUPerHour: 0.03238, GiBPerHour: 0.00356}
)

func fargateTestRates(architecture string) (pricing.FargateRates, bool) {
	switch architecture {
	case "", pricing.FargateArchX86:
		return testFargateX86, true
	case pricing.FargateArchARM:
		return testFargateARM, true
	default:
		return pricing.FargateRates{}, false
	}
}

func (m *mockPricingClient) FargateOnDemandRates(architecture string) (pricing.FargateRates, bool) {
	return fargateTestRates(architecture)
}

func (m *mockPricingClientActual) FargateOnDemandRates(architecture string) (pricing.FargateRates, bool) {
	return fargateTestRates(architecture)
}
