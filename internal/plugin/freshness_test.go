package plugin

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
)

func (m *mockPricingClient) PricingMetadata(service string) (pricing.PricingMetadata, bool) {
	if m.pricingMeta == nil {
		return pricing.PricingMetadata{}, false
	}
	meta, ok := m.pricingMeta[service]
	return meta, ok
}

func (m *mockPricingClient) PricingAsOf() string {
	return m.pricingAsOf
}

func (m *mockPricingClientActual) PricingMetadata(string) (pricing.PricingMetadata, bool) {
	return pricing.PricingMetadata{}, false
}

func (m *mockPricingClientActual) PricingAsOf() string {
	return ""
}

func TestPricingFileForService(t *testing.T) {
	t.Parallel()
	cases := []struct {
		service string
		file    string
	}{
		{serviceEC2, pricing.ServiceEC2},
		{serviceEBS, pricing.ServiceEC2},
		{serviceASG, pricing.ServiceEC2},
		{serviceS3, pricing.ServiceS3},
		{serviceRDS, pricing.ServiceRDS},
		{serviceEKS, pricing.ServiceEKS},
		{serviceLambda, pricing.ServiceLambda},
		{serviceDynamoDB, pricing.ServiceDynamoDB},
		{serviceELB, pricing.ServiceELB},
		{serviceNATGW, pricing.ServiceVPC},
		{serviceEIP, pricing.ServiceVPC},
		{serviceCloudWatch, pricing.ServiceCloudWatch},
		{serviceElastiCache, pricing.ServiceElastiCache},
		{serviceEKSFargate, pricing.ServiceECS},
		{serviceFargate, pricing.ServiceECS},
		{serviceEFS, pricing.ServiceEFS},
		{serviceVPC, ""},
		{serviceSecurityGroup, ""},
		{serviceIAM, ""},
		{"unknown", ""},
	}
	for _, tc := range cases {
		t.Run(tc.service, func(t *testing.T) {
			t.Parallel()
			if got := pricingFileForService(tc.service); got != tc.file {
				t.Errorf("pricingFileForService(%q) = %q, want %q", tc.service, got, tc.file)
			}
		})
	}
}

func TestGetProjectedCost_StampsPricingFreshness(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	mock.ebsPrices["gp2"] = 0.10
	published := "2024-01-01T00:00:00Z"
	mock.pricingMeta = map[string]pricing.PricingMetadata{
		pricing.ServiceEC2: {
			Version:         pricing.PricingVersionFallback,
			PublicationDate: published,
			OfferCode:       "AmazonEC2",
		},
	}
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, zerolog.Nop())

	ec2, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost() error = %v", err)
	}
	if ec2.GetMetadata()[metadataKeyPricingAsOf] != published {
		t.Errorf("pricing_as_of = %q, want %q", ec2.GetMetadata()[metadataKeyPricingAsOf], published)
	}
	if ec2.GetMetadata()[metadataKeyPricingVersion] != pricing.PricingVersionFallback {
		t.Errorf("pricing_version = %q, want fallback", ec2.GetMetadata()[metadataKeyPricingVersion])
	}
	if ec2.GetMetadata()[metadataKeyPricingSource] != pricingSourceFallback {
		t.Errorf("pricing_source = %q, want fallback", ec2.GetMetadata()[metadataKeyPricingSource])
	}
	if _, ok := ec2.GetMetadata()[metadataKeyDefaultsApplied]; ok {
		t.Errorf("defaults_applied = %q, want absent", ec2.GetMetadata()[metadataKeyDefaultsApplied])
	}

	ebs, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ebs",
			Sku:          "gp2",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(ebs) error = %v", err)
	}
	if ebs.GetMetadata()[metadataKeyDefaultsApplied] == "" {
		t.Fatal("defaults_applied missing on defaulted EBS size")
	}
	if ebs.GetMetadata()[metadataKeyEstimateQuality] == "" {
		t.Fatal("estimate_quality missing on defaulted EBS size")
	}
	if ebs.GetMetadata()[metadataKeyPricingAsOf] != published {
		t.Errorf("EBS pricing_as_of = %q, want %q", ebs.GetMetadata()[metadataKeyPricingAsOf], published)
	}

	vpc, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "aws:ec2/vpc:Vpc",
			Region:       "us-east-1",
		},
	})
	if err != nil {
		t.Fatalf("GetProjectedCost(vpc) error = %v", err)
	}
	if _, ok := vpc.GetMetadata()[metadataKeyPricingAsOf]; ok {
		t.Errorf("zero-cost VPC stamped pricing_as_of = %q", vpc.GetMetadata()[metadataKeyPricingAsOf])
	}
}

func TestGetPluginInfo_PricingFreshness(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.pricingAsOf = "2024-01-01T00:00:00Z"
	mock.pricingMeta = map[string]pricing.PricingMetadata{
		pricing.ServiceEC2: {Version: pricing.PricingVersionFallback, PublicationDate: mock.pricingAsOf},
		pricing.ServiceS3:  {Version: "20260911124544", PublicationDate: "2026-09-11T12:45:44Z"},
	}
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", mock, zerolog.Nop())
	resp, err := plugin.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo() error = %v", err)
	}
	meta := resp.GetMetadata()
	if meta["pricing_as_of"] != mock.pricingAsOf {
		t.Errorf("pricing_as_of = %q", meta["pricing_as_of"])
	}
	if meta["pricing_version.ec2"] != pricing.PricingVersionFallback {
		t.Errorf("pricing_version.ec2 = %q", meta["pricing_version.ec2"])
	}
	if meta["pricing_version.s3"] != "20260911124544" {
		t.Errorf("pricing_version.s3 = %q", meta["pricing_version.s3"])
	}
	if meta[metadataKeyPricingSource] != pricingSourceFallback {
		t.Errorf("pricing_source = %q, want fallback", meta[metadataKeyPricingSource])
	}
	if meta["region"] != "us-east-1" || meta["type"] != "public-pricing-fallback" {
		t.Errorf("base metadata changed: %v", meta)
	}
	if len(meta) > 32 {
		t.Errorf("metadata has %d entries, limit is 32", len(meta))
	}
	for key := range meta {
		if len(key) > 64 {
			t.Errorf("metadata key %q is longer than 64 bytes", key)
		}
	}
}
