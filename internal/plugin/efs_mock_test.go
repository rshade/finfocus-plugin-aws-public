package plugin

import "github.com/rshade/finfocus-plugin-aws-public/internal/pricing"

// efsUSEast1Rates returns the us-east-1 EFS rates from the AWS Price List.
func efsUSEast1Rates() *pricing.EFSPrice {
	return &pricing.EFSPrice{
		StandardGBMonth:       0.30,
		IAGBMonth:             0.025,
		IAElasticGBMonth:      0.016,
		ArchiveGBMonth:        0.008,
		OneZoneGBMonth:        0.16,
		OneZoneIAGBMonth:      0.0133,
		ProvisionedMiBpsMonth: 6.00,
		ElasticReadGB:         0.03,
		ElasticWriteGB:        0.06,
		IAReadGB:              0.01,
		IAWriteGB:             0.01,
		ArchiveReadGB:         0.03,
		ArchiveWriteGB:        0.03,
		Currency:              "USD",
	}
}

// EFSPrice returns the configured EFS rates, or not found when none are set.
func (m *mockPricingClient) EFSPrice() (*pricing.EFSPrice, bool) {
	if m.efsPrices == nil {
		return nil, false
	}
	return m.efsPrices, true
}

// EFSPrice returns the us-east-1 EFS rates.
func (m *mockPricingClientActual) EFSPrice() (*pricing.EFSPrice, bool) {
	return efsUSEast1Rates(), true
}
