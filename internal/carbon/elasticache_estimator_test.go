package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElastiCacheEstimator_EstimateCarbonGrams(t *testing.T) {
	estimator := NewElastiCacheEstimator()

	tests := []struct {
		name    string
		config  ElastiCacheConfig
		wantOk  bool
		wantMin float64 // Minimum expected carbon (sanity check)
	}{
		{
			name: "cache.m5.large (2 nodes) in us-east-1",
			config: ElastiCacheConfig{
				NodeType:    "cache.m5.large",
				Engine:      "redis",
				Nodes:       2,
				Region:      "us-east-1",
				Utilization: 0.5,
				Hours:       730,
			},
			wantOk:  true,
			wantMin: 1000, // Roughly > 1kg CO2e for m5.large for 730h
		},
		{
			name: "cache.t3.micro (1 node) in us-east-1",
			config: ElastiCacheConfig{
				NodeType:    "cache.t3.micro",
				Engine:      "memcached",
				Nodes:       1,
				Region:      "us-east-1",
				Utilization: 0.1,
				Hours:       730,
			},
			wantOk:  true,
			wantMin: 10, // Small but non-zero
		},
		{
			name: "unknown node type",
			config: ElastiCacheConfig{
				NodeType: "cache.unknown.ultra",
				Region:   "us-east-1",
			},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := estimator.EstimateCarbonGrams(tt.config)
			if tt.wantOk {
				require.True(t, ok)
				assert.GreaterOrEqual(t, got, tt.wantMin)
			} else {
				assert.False(t, ok)
				assert.Equal(t, 0.0, got)
			}
		})
	}
}

func TestElastiCacheEstimator_GetBillingDetail(t *testing.T) {
	estimator := NewElastiCacheEstimator()
	config := ElastiCacheConfig{
		NodeType:    "cache.m5.large",
		Engine:      "redis",
		Nodes:       3,
		Region:      "us-east-1",
		Utilization: 0.5,
		Hours:       730,
	}

	detail := estimator.GetBillingDetail(config)
	expected := "ElastiCache cache.m5.large (redis), 3 nodes, 730 hrs, 50% utilization"
	assert.Equal(t, expected, detail)
}

func TestElasticacheToEC2InstanceType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cache.m5.large", "m5.large"},
		{"cache.t3.micro", "t3.micro"},
		{"m5.xlarge", "m5.xlarge"}, // No prefix
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := elasticacheToEC2InstanceType(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
