package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	testEIPInUseRate = 0.005
	testEIPIdleRate  = 0.007
	eipType          = "aws:ec2/eip:Eip"
)

// newEIPTestPlugin returns a plugin whose in-use and idle public IPv4 rates
// differ, so a test can tell which rate an estimate used.
func newEIPTestPlugin(inUse, idle float64) *AWSPublicPlugin {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.eipInUsePrice = inUse
	mock.eipIdlePrice = idle
	return NewAWSPublicPlugin("us-east-1", "test-version", mock, zerolog.New(nil).Level(zerolog.InfoLevel))
}

func eipResource(tags map[string]string, attrs map[string]any) *pbc.ResourceDescriptor {
	res := &pbc.ResourceDescriptor{Provider: "aws", ResourceType: eipType, Region: "us-east-1", Tags: tags}
	if attrs != nil {
		s, err := structpb.NewStruct(attrs)
		if err != nil {
			panic(err)
		}
		res.Attributes = s
	}
	return res
}

// TestEstimateEIP verifies that an Elastic IP is billed hourly and that the
// rate follows whether the address is associated.
//
// An Elastic IP has no SKU. The in-use rate applies when the descriptor names an
// instance, a network interface, or an association, on tags or on attributes
// in either snake_case or the Pulumi camelCase name. An empty value does not
// count. The two rates are set apart, so choosing the wrong one fails.
func TestEstimateEIP(t *testing.T) {
	tests := []struct {
		name      string
		tags      map[string]string
		attrs     map[string]any
		wantRate  float64
		wantState string
	}{
		{name: "no association is idle", wantRate: testEIPIdleRate, wantState: "idle"},
		{
			name:      "instance tag is in-use",
			tags:      map[string]string{"instance": "i-0123456789abcdef0"},
			wantRate:  testEIPInUseRate,
			wantState: "in-use",
		},
		{
			name:      "association_id tag is in-use",
			tags:      map[string]string{"association_id": "eipassoc-0123456789abcdef0"},
			wantRate:  testEIPInUseRate,
			wantState: "in-use",
		},
		{
			name:      "Pulumi networkInterface attribute is in-use",
			attrs:     map[string]any{"networkInterface": "eni-0123456789abcdef0"},
			wantRate:  testEIPInUseRate,
			wantState: "in-use",
		},
		{
			name:      "Pulumi associationId attribute is in-use",
			attrs:     map[string]any{"associationId": "eipassoc-0123456789abcdef0"},
			wantRate:  testEIPInUseRate,
			wantState: "in-use",
		},
		{
			name:      "empty association values stay idle",
			tags:      map[string]string{"instance": "", "association_id": ""},
			attrs:     map[string]any{"networkInterface": ""},
			wantRate:  testEIPIdleRate,
			wantState: "idle",
		},
	}

	plugin := newEIPTestPlugin(testEIPInUseRate, testEIPIdleRate)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: eipResource(tt.tags, tt.attrs),
			})

			require.NoError(t, err)
			assert.InDelta(t, tt.wantRate*730, resp.GetCostPerMonth(), 1e-9)
			assert.InDelta(t, tt.wantRate, resp.GetUnitPrice(), 1e-12)
			assert.Contains(t, resp.GetBillingDetail(), "Elastic IP (public IPv4, "+tt.wantState+")")
			assert.Contains(t, resp.GetBillingDetail(), "730 hrs/month")
			assert.Contains(t, resp.GetBillingDetail(), "free tier and data transfer not included")
		})
	}
}

// TestEstimateEIP_AllRPCs verifies Supports, GetPricingSpec, and GetActualCost
// handle an Elastic IP, including one named only by an ARN.
func TestEstimateEIP_AllRPCs(t *testing.T) {
	plugin := newEIPTestPlugin(testEIPInUseRate, testEIPIdleRate)
	res := eipResource(map[string]string{"instance": "i-0123456789abcdef0"}, nil)

	sup, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{Resource: res})
	require.NoError(t, err)
	assert.True(t, sup.GetSupported())

	spec, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{Resource: res})
	require.NoError(t, err)
	assert.Equal(t, billingModePerHour, spec.GetSpec().GetBillingMode())
	assert.InDelta(t, testEIPInUseRate, spec.GetSpec().GetRatePerUnit(), 1e-12)
	assert.Equal(t, unitHour, spec.GetSpec().GetUnit())

	idleSpec, err := plugin.GetPricingSpec(
		context.Background(),
		&pbc.GetPricingSpecRequest{Resource: eipResource(nil, nil)},
	)
	require.NoError(t, err)
	assert.InDelta(t, testEIPIdleRate, idleSpec.GetSpec().GetRatePerUnit(), 1e-12)

	actualPlugin := NewAWSPublicPlugin("us-east-1", "test-version", &mockPricingClientActual{
		region:        "us-east-1",
		eipInUsePrice: testEIPInUseRate,
		eipIdlePrice:  testEIPIdleRate,
	}, zerolog.New(nil).Level(zerolog.InfoLevel))
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for name, req := range map[string]*pbc.GetActualCostRequest{
		"resource id": {
			ResourceId: makeResourceJSON("aws", eipType, "", "us-east-1", nil),
			Start:      timestamppb.New(from),
			End:        timestamppb.New(from.Add(24 * time.Hour)),
		},
		"arn": {
			Arn:   "arn:aws:ec2:us-east-1:123456789012:elastic-ip/eipalloc-0123456789abcdef0",
			Start: timestamppb.New(from),
			End:   timestamppb.New(from.Add(24 * time.Hour)),
		},
	} {
		t.Run("actual cost by "+name, func(t *testing.T) {
			resp, err := actualPlugin.GetActualCost(context.Background(), req)
			require.NoError(t, err)
			require.NotEmpty(t, resp.GetResults())
			assert.InDelta(t, testEIPIdleRate*24, resp.GetResults()[0].GetCost(), 1e-9)
		})
	}
}

// TestEstimateEIP_PricingUnavailable verifies a region with no public IPv4
// data reports that, instead of pricing the address at a silent zero.
func TestEstimateEIP_PricingUnavailable(t *testing.T) {
	plugin := newEIPTestPlugin(0, 0)

	resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: eipResource(nil, nil),
	})

	require.NoError(t, err)
	assert.InDelta(t, 0.0, resp.GetCostPerMonth(), 0)
	assert.Contains(t, resp.GetBillingDetail(), "Elastic IP pricing data not available")
}

// TestElasticIPRouting pins how Elastic IP types resolve. "ec2/eip" is a prefix
// of "ec2/eipassociation", and an association is free while the address is not.
func TestElasticIPRouting(t *testing.T) {
	tests := []struct {
		resourceType string
		want         string
	}{
		{"aws:ec2/eip:Eip", serviceEIP},
		{"AWS:EC2/EIP:EIP", serviceEIP},
		{"aws:ec2/eipAssociation:EipAssociation", serviceAddressAssoc},
		{"eip", serviceEIP},
		{"elastic_ip", serviceEIP},
		{"elastic-ip", serviceEIP},
	}
	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			assert.Equal(t, tt.want, detectService(normalizeResourceType(tt.resourceType)))
		})
	}
	assert.True(t, IsZeroCostService(serviceAddressAssoc))
	assert.False(t, IsZeroCostService(serviceEIP), "an Elastic IP is billed and must never be zero-cost")
}
