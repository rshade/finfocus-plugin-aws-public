package plugin

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
)

// TestSKUOptionalServicesAcceptEmptySKU checks that the services fed by
// Terraform state are not rejected with the SDK SKU error.
//
// Run with: go test ./internal/plugin -run TestSKUOptionalServicesAcceptEmptySKU.
func TestSKUOptionalServicesAcceptEmptySKU(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)
	types := []string{
		"s3", "lambda", "dynamodb", "elasticache", "elb", "natgw",
		"cloudwatch", "aws:cloudwatch/metricAlarm:MetricAlarm", "asg",
	}
	for _, resourceType := range types {
		t.Run(resourceType, func(t *testing.T) {
			_, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     providerAWS,
					ResourceType: resourceType,
					Region:       "us-east-1",
				},
			})
			if err != nil {
				assert.NotContains(t, err.Error(), "resource.sku is required")
			}
		})
	}
}

// TestRegionAndUtilizationRules checks the SKU-optional region rules and
// that an out-of-range utilization is still rejected.
func TestRegionAndUtilizationRules(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)

	t.Run("s3 empty region uses the plugin region", func(t *testing.T) {
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     providerAWS,
				ResourceType: "s3",
			},
		})
		require.NoError(t, err)
		assert.Contains(t, resp.GetMetadata()["defaults_applied"], "storage_class=STANDARD")
	})

	t.Run("asg empty region is rejected", func(t *testing.T) {
		_, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     providerAWS,
				ResourceType: "asg",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "region is required")
		assert.NotContains(t, err.Error(), "tag-resolved")
	})

	t.Run("utilization out of range", func(t *testing.T) {
		_, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			UtilizationPercentage: 1.5,
			Resource: &pbc.ResourceDescriptor{
				Provider:     providerAWS,
				ResourceType: "s3",
				Region:       "us-east-1",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "utilization_percentage")
	})
}

// TestLambdaDefaultsAndGraviton checks the recorded Lambda defaults and that
// an arm64 architecture attribute selects the Graviton rate.
func TestLambdaDefaultsAndGraviton(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)

	t.Run("defaults", func(t *testing.T) {
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     providerAWS,
				ResourceType: "lambda",
				Region:       "us-east-1",
			},
		})
		require.NoError(t, err)
		applied := resp.GetMetadata()["defaults_applied"]
		assert.Contains(t, applied, "memory_mb=128")
		assert.Contains(t, applied, "arch=x86_64")
	})

	t.Run("arm64 attribute", func(t *testing.T) {
		attrs, err := structpb.NewStruct(map[string]any{
			"memorySize":       512.0,
			"architectures":    "arm64",
			"requestsPerMonth": "1000",
		})
		require.NoError(t, err)
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     providerAWS,
				ResourceType: "aws:lambda/function:Function",
				Region:       "us-east-1",
				Attributes:   attrs,
			},
		})
		require.NoError(t, err)
		assert.Contains(t, resp.GetBillingDetail(), "512MB")
		assert.Contains(t, resp.GetBillingDetail(), "arm64")
		assert.NotContains(t, resp.GetMetadata()["defaults_applied"], "memory_mb=128")
	})
}

// TestGatewayLoadBalancerIsNotALB checks that a gateway load balancer returns
// $0 and is not described as an ALB.
func TestGatewayLoadBalancerIsNotALB(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)
	for _, sku := range []string{"", "gateway", "gwlb"} {
		resource := &pbc.ResourceDescriptor{
			Provider:     providerAWS,
			ResourceType: "aws:lb/loadBalancer:LoadBalancer",
			Sku:          sku,
			Region:       "us-east-1",
			Tags:         map[string]string{"loadBalancerType": "gateway"},
		}
		if sku == "" {
			resource.Tags = map[string]string{"loadBalancerType": "gateway"}
		}
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{Resource: resource})
		require.NoError(t, err)
		assert.Zero(t, resp.GetCostPerMonth())
		assert.Contains(t, resp.GetBillingDetail(), "not supported")
		assert.NotContains(t, resp.GetBillingDetail(), "LCU")

		spec, err := plugin.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{Resource: resource})
		require.NoError(t, err)
		assert.Zero(t, spec.GetSpec().GetRatePerUnit())
		assert.Contains(t, spec.GetSpec().GetDescription(), detailGWLBUnsupported)
		assert.NotContains(t, spec.GetSpec().GetDescription(), "LCU")
	}
}

// TestCloudWatchAlarmIsNotLogs checks that a metric alarm is not priced as logs.
func TestCloudWatchAlarmIsNotLogs(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)
	resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     providerAWS,
			ResourceType: "aws:cloudwatch/metricAlarm:MetricAlarm",
			Region:       "us-east-1",
		},
	})
	require.NoError(t, err)
	assert.Zero(t, resp.GetCostPerMonth())
	assert.Contains(t, resp.GetBillingDetail(), "alarm pricing is not supported")
	assert.NotContains(t, strings.ToLower(resp.GetBillingDetail()), "log")

	spec, err := plugin.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     providerAWS,
			ResourceType: "aws:cloudwatch/metricAlarm:MetricAlarm",
			Region:       "us-east-1",
		},
	})
	require.NoError(t, err)
	assert.Zero(t, spec.GetSpec().GetRatePerUnit())
	assert.Contains(t, spec.GetSpec().GetDescription(), detailAlarmUnsupported)
	assert.NotContains(t, strings.ToLower(spec.GetSpec().GetDescription()), "log")
}

// TestAttributeSecretIsNotLogged checks that a credential-like attribute value
// never appears in the plugin log. Attributes are pricing inputs, not log fields.
func TestAttributeSecretIsNotLogged(t *testing.T) {
	const secret = "leak-marker-9f3a"
	var logs bytes.Buffer
	mock := newMockPricingClient("us-east-1", currencyUSD)
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	logger := zerolog.New(&logs)
	plugin := NewAWSPublicPlugin("us-east-1", "test", mock, logger)
	attrs, err := structpb.NewStruct(map[string]any{"description": secret})
	require.NoError(t, err)

	_, err = plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     providerAWS,
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
			Attributes:   attrs,
		},
	})
	require.NoError(t, err)
	assert.NotContains(t, logs.String(), secret)
}

// TestEC2CostParityAcrossRPCs checks that projected cost, pricing spec,
// actual cost, and EstimateCost agree for one EC2 instance.
func TestEC2CostParityAcrossRPCs(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)
	resource := &pbc.ResourceDescriptor{
		Provider:     providerAWS,
		ResourceType: "ec2",
		Sku:          "t3.micro",
		Region:       "us-east-1",
	}
	projected, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{Resource: resource})
	require.NoError(t, err)

	spec, err := plugin.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{Resource: resource})
	require.NoError(t, err)
	assert.InDelta(t, projected.GetUnitPrice(), spec.GetSpec().GetRatePerUnit(), 0.0000001)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Duration(carbon.HoursPerMonth) * time.Hour)
	actual, err := plugin.GetActualCost(t.Context(), &pbc.GetActualCostRequest{
		ResourceId: "i-parity",
		Resource:   resource,
		Start:      timestamppb.New(start),
		End:        timestamppb.New(end),
	})
	require.NoError(t, err)
	require.Len(t, actual.GetResults(), 1)
	assert.InDelta(t, projected.GetCostPerMonth(), actual.GetResults()[0].GetCost(), 0.0001)

	attrs, err := structpb.NewStruct(map[string]any{
		"instanceType": "t3.micro",
		"region":       "us-east-1",
	})
	require.NoError(t, err)
	estimated, err := plugin.EstimateCost(t.Context(), &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})
	require.NoError(t, err)
	assert.InDelta(t, projected.GetCostPerMonth(), estimated.GetCostMonthly(), 0.0001)
}

// TestProductionGetTagsStayInInputs fails when pricing code reads tags itself.
// Logging through sanitizeTagsForLogging is the allowed exception.
func TestProductionGetTagsStayInInputs(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == "inputs.go" {
			continue
		}
		body, err := os.ReadFile(name)
		require.NoError(t, err)
		for i, line := range strings.Split(string(body), "\n") {
			if !strings.Contains(line, "GetTags()") {
				continue
			}
			if strings.Contains(line, "sanitizeTagsForLogging") {
				continue
			}
			t.Errorf("%s:%d calls GetTags outside inputs.go: %s", name, i+1, strings.TrimSpace(line))
		}
	}
}

// TestAttributeBeatsDisagreeingTag checks that an attribute wins when a tag
// names the same property with a different key.
func TestAttributeBeatsDisagreeingTag(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)
	cases := []struct {
		name    string
		resType string
		sku     string
		attr    map[string]any
		tags    map[string]string
		want    string
		notWant string
	}{
		{
			name:    "s3 storage class",
			resType: "s3",
			attr:    map[string]any{"storageClass": "GLACIER"},
			tags:    map[string]string{"storage_class": "STANDARD"},
			want:    "GLACIER",
			notWant: "storage_class=STANDARD",
		},
		{
			name:    "elasticache node count",
			resType: "elasticache",
			sku:     "cache.t3.micro",
			attr:    map[string]any{"numCacheNodes": "2"},
			tags:    map[string]string{"num_nodes": "9"},
			want:    "2 nodes",
			notWant: "9 nodes",
		},
		{
			name:    "rds storage type",
			resType: "rds",
			sku:     "db.t3.micro",
			attr:    map[string]any{"storageType": "gp3", "engine": "mysql"},
			tags:    map[string]string{"storage_type": "gp2", "engine": "mysql"},
			want:    "gp3",
			notWant: "gp2",
		},
		{
			name:    "lambda memory",
			resType: "lambda",
			attr:    map[string]any{"memorySize": "512"},
			tags:    map[string]string{"memory_mb": "128"},
			want:    "512",
			notWant: "memory_mb=128",
		},
		{
			name:    "ebs size",
			resType: "ebs",
			sku:     "gp3",
			attr:    map[string]any{"volumeSize": "100"},
			tags:    map[string]string{"size": "8"},
			want:    "100",
			notWant: "8GB",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			attrs, err := structpb.NewStruct(tt.attr)
			require.NoError(t, err)
			resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
				Resource: &pbc.ResourceDescriptor{
					Provider:     providerAWS,
					ResourceType: tt.resType,
					Sku:          tt.sku,
					Region:       "us-east-1",
					Tags:         tt.tags,
					Attributes:   attrs,
				},
			})
			require.NoError(t, err)
			assert.Contains(t, resp.GetBillingDetail(), tt.want)
			combined := resp.GetBillingDetail() + resp.GetMetadata()["defaults_applied"]
			assert.NotContains(t, combined, tt.notWant)
		})
	}
}
