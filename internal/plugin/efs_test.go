package plugin

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
)

const efsPulumiType = "aws:efs/fileSystem:FileSystem"

func newEFSTestPlugin(withPricing bool) *AWSPublicPlugin {
	mock := newMockPricingClient("us-east-1", currencyUSD)
	if withPricing {
		mock.efsPrices = efsUSEast1Rates()
	}
	return NewAWSPublicPlugin("us-east-1", "test", mock, zerolog.Nop())
}

func efsDescriptor(resourceType, sku string, tags map[string]string) *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{
		Provider:     providerAWS,
		ResourceType: resourceType,
		Sku:          sku,
		Region:       "us-east-1",
		Tags:         tags,
	}
}

// TestEstimateEFS_Costs prices each storage class and throughput mode against the us-east-1 rates.
func TestEstimateEFS_Costs(t *testing.T) {
	plugin := newEFSTestPlugin(true)

	tests := []struct {
		name       string
		sku        string
		tags       map[string]string
		wantCost   float64
		wantDetail []string
	}{
		{
			name:       "standard plus IA bursting",
			tags:       map[string]string{"storage_gb": "100", "ia_storage_gb": "400"},
			wantCost:   100*0.30 + 400*0.025,
			wantDetail: []string{"EFS Regional", "100GB Standard", "400GB IA", "bursting throughput"},
		},
		{
			name: "elastic uses IA-ET rate and data access",
			tags: map[string]string{
				"storage_gb": "100", "ia_storage_gb": "400", "throughput_mode": "elastic",
				"elastic_read_gb": "10", "elastic_write_gb": "5",
			},
			wantCost:   100*0.30 + 400*0.016 + 10*0.03 + 5*0.06,
			wantDetail: []string{"elastic throughput (10GB read, 5GB write)"},
		},
		{
			name:       "elastic storage only is 30 + 6.40",
			tags:       map[string]string{"storage_gb": "100", "ia_storage_gb": "400", "throughput_mode": "elastic"},
			wantCost:   36.40,
			wantDetail: []string{"elastic throughput"},
		},
		{
			name: "provisioned bills only above the baseline",
			tags: map[string]string{
				"storage_gb":                      "100",
				"throughput_mode":                 "provisioned",
				"provisioned_throughput_in_mibps": "20",
			},
			wantCost:   100*0.30 + (20-100*50.0/1024)*6.00,
			wantDetail: []string{"provisioned throughput 20 MiB/s", "billable above 4.88 MiB/s baseline"},
		},
		{
			name: "provisioned below baseline is free",
			tags: map[string]string{
				"storage_gb":                      "1000",
				"throughput_mode":                 "provisioned",
				"provisioned_throughput_in_mibps": "10",
			},
			wantCost:   1000 * 0.30,
			wantDetail: []string{"0.00 MiB/s billable"},
		},
		{
			name:       "one zone storage",
			tags:       map[string]string{"storage_gb": "100", "availability_zone_name": "us-east-1a"},
			wantCost:   16.00,
			wantDetail: []string{"EFS One Zone"},
		},
		{
			name: "one zone ignores archive",
			tags: map[string]string{
				"storage_gb":             "100",
				"availability_zone_name": "us-east-1a",
				"archive_storage_gb":     "50",
			},
			wantCost:   16.00,
			wantDetail: []string{"50GB Archive ignored: One Zone has no Archive class"},
		},
		{
			name: "archive storage and access",
			tags: map[string]string{
				"storage_gb":         "10",
				"archive_storage_gb": "100",
				"archive_read_gb":    "2",
				"archive_write_gb":   "1",
			},
			wantCost:   10*0.30 + 100*0.008 + 2*0.03 + 1*0.03,
			wantDetail: []string{"100GB Archive", "Archive access 2GB read, 1GB write"},
		},
		{
			name: "IA access charges",
			tags: map[string]string{
				"storage_gb":    "10",
				"ia_storage_gb": "100",
				"ia_read_gb":    "5",
				"ia_write_gb":   "3",
			},
			wantCost:   10*0.30 + 100*0.025 + 5*0.01 + 3*0.01,
			wantDetail: []string{"IA access 5GB read, 3GB write"},
		},
		{
			name:       "SKU names the class of storage_gb",
			sku:        "Infrequent Access",
			tags:       map[string]string{"storage_gb": "200"},
			wantCost:   200 * 0.025,
			wantDetail: []string{"200GB IA"},
		},
		{
			name:       "one zone IA SKU",
			sku:        "one_zone_ia",
			tags:       map[string]string{"storage_gb": "300"},
			wantCost:   300 * 0.0133,
			wantDetail: []string{"EFS One Zone", "300GB IA"},
		},
		{
			name:       "storage_class tag selects one zone",
			tags:       map[string]string{"storage_class": "one_zone", "storage_gb": "100"},
			wantCost:   16.00,
			wantDetail: []string{"EFS One Zone"},
		},
		{
			name:       "size_in_bytes total without split is Standard",
			tags:       map[string]string{"sizeInBytes": "107374182400"},
			wantCost:   100 * 0.30,
			wantDetail: []string{"100GB Standard"},
		},
		{
			name:       "invalid negative value is treated as zero",
			tags:       map[string]string{"storage_gb": "100", "ia_storage_gb": "-5"},
			wantCost:   100 * 0.30,
			wantDetail: []string{"100GB Standard"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
				Resource: efsDescriptor(efsPulumiType, tt.sku, tt.tags),
			})
			require.NoError(t, err)
			assert.InDelta(t, tt.wantCost, resp.GetCostPerMonth(), 1e-9)
			for _, want := range tt.wantDetail {
				assert.Contains(t, resp.GetBillingDetail(), want)
			}
			assert.Equal(t, pbc.GrowthType_GROWTH_TYPE_LINEAR, resp.GetGrowthType())
		})
	}
}

// TestEstimateEFS_Attributes reads Pulumi attributes, including the nested size_in_bytes struct,
// and verifies attributes win over tags.
func TestEstimateEFS_Attributes(t *testing.T) {
	plugin := newEFSTestPlugin(true)
	attrs, err := structpb.NewStruct(map[string]any{
		"throughputMode":               "provisioned",
		"provisionedThroughputInMibps": 20,
		"sizeInBytes": map[string]any{
			"value":           float64(500) * bytesPerGB,
			"valueInStandard": float64(100) * bytesPerGB,
			"valueInIa":       float64(400) * bytesPerGB,
		},
	})
	require.NoError(t, err)

	resource := efsDescriptor(efsPulumiType, "", map[string]string{"throughput_mode": "elastic"})
	resource.Attributes = attrs

	resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{Resource: resource})
	require.NoError(t, err)

	want := 100*0.30 + 400*0.025 + (20-100*50.0/1024)*6.00
	assert.InDelta(t, want, resp.GetCostPerMonth(), 1e-9)
	assert.Contains(t, resp.GetBillingDetail(), "provisioned throughput")
	assert.Empty(t, resp.GetMetadata()["defaults_applied"])
}

// TestEstimateEFS_TerraformSizeList reads the one-element size_in_bytes list Terraform state carries.
func TestEstimateEFS_TerraformSizeList(t *testing.T) {
	plugin := newEFSTestPlugin(true)
	attrs, err := structpb.NewStruct(map[string]any{
		"size_in_bytes": []any{map[string]any{
			"value":             float64(500) * bytesPerGB,
			"value_in_standard": float64(100) * bytesPerGB,
			"value_in_ia":       float64(400) * bytesPerGB,
		}},
	})
	require.NoError(t, err)

	resource := efsDescriptor(efsPulumiType, "", nil)
	resource.Attributes = attrs

	resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{Resource: resource})
	require.NoError(t, err)
	assert.InDelta(t, 100*0.30+400*0.025, resp.GetCostPerMonth(), 1e-9)
}

// TestEstimateEFS_Defaults records defaulted inputs and lowers quality for defaulted usage.
func TestEstimateEFS_Defaults(t *testing.T) {
	plugin := newEFSTestPlugin(true)

	t.Run("empty descriptor defaults to 1GB Standard bursting", func(t *testing.T) {
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: efsDescriptor("efs", "", nil),
		})
		require.NoError(t, err)
		assert.InDelta(t, 0.30, resp.GetCostPerMonth(), 1e-9)
		assert.Equal(t, "storage_gb=1,throughput_mode=bursting", resp.GetMetadata()["defaults_applied"])
		assert.Equal(t, qualityMedium, resp.GetMetadata()["estimate_quality"])
	})

	t.Run("elastic without usage is low quality", func(t *testing.T) {
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: efsDescriptor("efs", "", map[string]string{"storage_gb": "10", "throughput_mode": "elastic"}),
		})
		require.NoError(t, err)
		assert.Contains(t, resp.GetMetadata()["defaults_applied"], "elastic_read_gb=0")
		assert.Contains(t, resp.GetMetadata()["defaults_applied"], "elastic_write_gb=0")
		assert.Equal(t, qualityLow, resp.GetMetadata()["estimate_quality"])
	})

	t.Run("explicit inputs have no metadata", func(t *testing.T) {
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: efsDescriptor("efs", "", map[string]string{"storage_gb": "10", "throughput_mode": "bursting"}),
		})
		require.NoError(t, err)
		assert.Nil(t, resp.GetMetadata())
	})
}

// TestEstimateEFS_Errors covers invalid inputs and unavailable pricing.
func TestEstimateEFS_Errors(t *testing.T) {
	plugin := newEFSTestPlugin(true)

	for name, resource := range map[string]*pbc.ResourceDescriptor{
		"unknown throughput mode": efsDescriptor("efs", "", map[string]string{"throughput_mode": "turbo"}),
		"unknown storage class":   efsDescriptor("efs", "glacier", nil),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{Resource: resource})
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}

	t.Run("pricing unavailable returns zero with detail", func(t *testing.T) {
		resp, err := newEFSTestPlugin(false).GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: efsDescriptor("efs", "", map[string]string{"storage_gb": "10"}),
		})
		require.NoError(t, err)
		assert.Zero(t, resp.GetCostPerMonth())
		assert.Contains(t, resp.GetBillingDetail(), "EFS pricing data not available")
	})

	t.Run("missing class rate returns zero with detail", func(t *testing.T) {
		mock := newMockPricingClient("us-east-1", currencyUSD)
		rates := efsUSEast1Rates()
		rates.ArchiveGBMonth = 0
		mock.efsPrices = rates
		resp, err := NewAWSPublicPlugin("us-east-1", "test", mock, zerolog.Nop()).GetProjectedCost(
			t.Context(),
			&pbc.GetProjectedCostRequest{
				Resource: efsDescriptor("efs", "", map[string]string{"archive_storage_gb": "5"}),
			},
		)
		require.NoError(t, err)
		assert.Zero(t, resp.GetCostPerMonth())
		assert.Contains(t, resp.GetBillingDetail(), "Archive")
	})

	for name, tc := range map[string]struct {
		tags   map[string]string
		mutate func(*pricing.EFSPrice)
		want   string
	}{
		"elastic read": {
			tags:   map[string]string{"throughput_mode": "elastic", "elastic_read_gb": "10"},
			mutate: func(r *pricing.EFSPrice) { r.ElasticReadGB = 0 },
			want:   "Elastic read",
		},
		"ia write": {
			tags:   map[string]string{"ia_write_gb": "10"},
			mutate: func(r *pricing.EFSPrice) { r.IAWriteGB = 0 },
			want:   "IA write",
		},
		"archive read": {
			tags:   map[string]string{"archive_read_gb": "10"},
			mutate: func(r *pricing.EFSPrice) { r.ArchiveReadGB = 0 },
			want:   "Archive read",
		},
	} {
		t.Run("missing access rate "+name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", currencyUSD)
			rates := efsUSEast1Rates()
			tc.mutate(rates)
			mock.efsPrices = rates
			resp, err := NewAWSPublicPlugin("us-east-1", "test", mock, zerolog.Nop()).GetProjectedCost(
				t.Context(),
				&pbc.GetProjectedCostRequest{Resource: efsDescriptor("efs", "", tc.tags)},
			)
			require.NoError(t, err)
			assert.Zero(t, resp.GetCostPerMonth())
			assert.Contains(t, resp.GetBillingDetail(), tc.want)
		})
	}
}

// TestEFSPricingSpec_StorageClassTag verifies the spec resolves storage_class
// like the estimator and reports an unrecognized class instead of pricing Standard.
func TestEFSPricingSpec_StorageClassTag(t *testing.T) {
	plugin := newEFSTestPlugin(true)

	ia := plugin.efsPricingSpec(efsDescriptor("efs", "", map[string]string{"storage_class": "ia"}))
	assert.InDelta(t, efsUSEast1Rates().IAGBMonth, ia.GetRatePerUnit(), 1e-9)

	bad := plugin.efsPricingSpec(efsDescriptor("efs", "", map[string]string{"storage_class": "glacier"}))
	assert.Contains(t, bad.GetDescription(), "glacier")
	assert.Zero(t, bad.GetRatePerUnit())
}

// TestEstimateEFS_Carbon verifies the carbon metric is attached and One Zone emits less.
func TestEstimateEFS_Carbon(t *testing.T) {
	plugin := newEFSTestPlugin(true)
	carbonFor := func(tags map[string]string) float64 {
		resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: efsDescriptor(efsPulumiType, "", tags),
		})
		require.NoError(t, err)
		require.Len(t, resp.GetImpactMetrics(), 1)
		assert.Equal(t, pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT, resp.GetImpactMetrics()[0].GetKind())
		return resp.GetImpactMetrics()[0].GetValue()
	}

	regional := carbonFor(map[string]string{"storage_gb": "1000"})
	oneZone := carbonFor(map[string]string{"storage_gb": "1000", "availability_zone_name": "us-east-1a"})
	assert.Positive(t, regional)
	assert.InDelta(t, regional*2/3, oneZone, 1e-9)
}

// TestEFS_ServiceRouting verifies type normalization, zero-cost siblings, Supports, and the pricing spec.
func TestEFS_ServiceRouting(t *testing.T) {
	cases := map[string]string{
		efsPulumiType:                     serviceEFS,
		"efs":                             serviceEFS,
		"aws:efs/mountTarget:MountTarget": serviceEFSMountTarget,
		"aws:efs/accessPoint:AccessPoint": serviceEFSAccessPoint,
		"aws:efs/fileSystemPolicy:FileSystemPolicy": "aws:efs/fileSystemPolicy:FileSystemPolicy",
		"aws:efs/backupPolicy:BackupPolicy":         "aws:efs/backupPolicy:BackupPolicy",
	}
	for resourceType, want := range cases {
		assert.Equal(t, want, detectService(normalizeResourceType(resourceType)), resourceType)
	}

	plugin := newEFSTestPlugin(true)

	t.Run("supports file system with carbon", func(t *testing.T) {
		resp, err := plugin.Supports(t.Context(), &pbc.SupportsRequest{Resource: efsDescriptor(efsPulumiType, "", nil)})
		require.NoError(t, err)
		assert.True(t, resp.GetSupported())
		assert.Equal(t, []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT}, resp.GetSupportedMetrics())
	})

	for _, resourceType := range []string{"aws:efs/mountTarget:MountTarget", "aws:efs/accessPoint:AccessPoint"} {
		t.Run("zero cost "+resourceType, func(t *testing.T) {
			sup, err := plugin.Supports(
				t.Context(),
				&pbc.SupportsRequest{Resource: efsDescriptor(resourceType, "", nil)},
			)
			require.NoError(t, err)
			assert.True(t, sup.GetSupported())

			resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
				Resource: efsDescriptor(resourceType, "", nil),
			})
			require.NoError(t, err)
			assert.Zero(t, resp.GetCostPerMonth())
			assert.Contains(t, resp.GetBillingDetail(), "no direct charge")

			spec, err := plugin.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{
				Resource: efsDescriptor(resourceType, "", nil),
			})
			require.NoError(t, err)
			assert.Zero(t, spec.GetSpec().GetRatePerUnit())
		})
	}

	t.Run("pricing spec", func(t *testing.T) {
		resp, err := plugin.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{
			Resource: efsDescriptor(efsPulumiType, "", nil),
		})
		require.NoError(t, err)
		assert.InDelta(t, 0.30, resp.GetSpec().GetRatePerUnit(), 1e-9)
		assert.Equal(t, unitGBMonth, resp.GetSpec().GetUnit())
		assert.Contains(t, resp.GetSpec().GetDescription(), "EFS Standard storage")

		oneZone, err := plugin.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{
			Resource: efsDescriptor(efsPulumiType, "", map[string]string{"availability_zone_name": "us-east-1a"}),
		})
		require.NoError(t, err)
		assert.InDelta(t, 0.16, oneZone.GetSpec().GetRatePerUnit(), 1e-9)

		missing, err := newEFSTestPlugin(false).GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{
			Resource: efsDescriptor(efsPulumiType, "", nil),
		})
		require.NoError(t, err)
		assert.Zero(t, missing.GetSpec().GetRatePerUnit())
	})
}

// TestEFS_ActualCostMatchesProjected verifies GetActualCost routes EFS like GetProjectedCost for
// both the short and Pulumi resource types, and routes the ARN service name.
func TestEFS_ActualCostMatchesProjected(t *testing.T) {
	plugin := newTestPluginForActual()
	tags := map[string]string{"storage_gb": "100", "ia_storage_gb": "400"}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(730 * time.Hour)

	projected, err := newEFSTestPlugin(true).GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: efsDescriptor(efsPulumiType, "", tags),
	})
	require.NoError(t, err)

	for _, resourceType := range []string{"efs", efsPulumiType} {
		t.Run(resourceType, func(t *testing.T) {
			resp, err := plugin.GetActualCost(t.Context(), &pbc.GetActualCostRequest{
				Resource:   efsDescriptor(resourceType, "", tags),
				ResourceId: "fs-0123456789abcdef0",
				Start:      timestamppb.New(start),
				End:        timestamppb.New(end),
			})
			require.NoError(t, err)
			require.NotEmpty(t, resp.GetResults())
			assert.InDelta(t, projected.GetCostPerMonth(), resp.GetResults()[0].GetCost(), 1e-6)
		})
	}

	t.Run("ARN", func(t *testing.T) {
		resp, err := plugin.GetActualCost(t.Context(), &pbc.GetActualCostRequest{
			Arn:   "arn:aws:elasticfilesystem:us-east-1:123456789012:file-system/fs-0123456789abcdef0",
			Tags:  tags,
			Start: timestamppb.New(start),
			End:   timestamppb.New(end),
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResults())
		assert.InDelta(t, projected.GetCostPerMonth(), resp.GetResults()[0].GetCost(), 1e-6)
	})

	t.Run("access point ARN is zero cost", func(t *testing.T) {
		resp, err := plugin.GetActualCost(t.Context(), &pbc.GetActualCostRequest{
			Arn:   "arn:aws:elasticfilesystem:us-east-1:123456789012:access-point/fsap-0123456789abcdef0",
			Start: timestamppb.New(start),
			End:   timestamppb.New(end),
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResults())
		assert.Zero(t, resp.GetResults()[0].GetCost())
	})
}
