package plugin

import "github.com/rshade/finfocus-plugin-aws-public/internal/pricing"

// testFargateX86 and testFargateARM are the us-east-1 Linux Fargate rates
// published in the AmazonECS Price List offer. testFargateWindows and
// testFargateEphemeral are the matching Windows and ephemeral storage rates.
var (
	testFargateX86     = pricing.FargateRates{VCPUPerHour: 0.04048, GiBPerHour: 0.004445}
	testFargateARM     = pricing.FargateRates{VCPUPerHour: 0.03238, GiBPerHour: 0.00356}
	testFargateWindows = pricing.FargateWindowsRates{
		VCPUPerHour:   0.046552,
		GiBPerHour:    0.00511175,
		OSPerVCPUHour: 0.046,
	}
	testFargateEphemeral = 0.000111
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

func (m *mockPricingClient) FargateWindowsOnDemandRates() (pricing.FargateWindowsRates, bool) {
	return testFargateWindows, true
}

func (m *mockPricingClientActual) FargateWindowsOnDemandRates() (pricing.FargateWindowsRates, bool) {
	return testFargateWindows, true
}

func (m *mockPricingClient) FargateEphemeralStoragePerGiBHour() (float64, bool) {
	return testFargateEphemeral, true
}

func (m *mockPricingClientActual) FargateEphemeralStoragePerGiBHour() (float64, bool) {
	return testFargateEphemeral, true
}
