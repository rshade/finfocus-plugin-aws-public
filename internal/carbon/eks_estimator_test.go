package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEKSEstimator_EstimateCarbonGrams verifies EKS returns zero carbon.
func TestEKSEstimator_EstimateCarbonGrams(t *testing.T) {
	e := NewEKSEstimator()

	tests := []struct {
		region string
	}{
		{"us-east-1"},
		{"eu-west-1"},
		{"ap-southeast-1"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			carbon, ok := e.EstimateCarbonGrams(EKSClusterConfig{
				Region: tt.region,
			})

			require.True(t, ok, "EstimateCarbonGrams should succeed")
			assert.Equal(t, 0.0, carbon, "EKS control plane should return zero carbon")
		})
	}
}

// TestEKSEstimator_GetBillingDetail verifies billing detail explains worker node estimation.
func TestEKSEstimator_GetBillingDetail(t *testing.T) {
	e := NewEKSEstimator()

	detail := e.GetBillingDetail(EKSClusterConfig{
		Region: "us-east-1",
	})

	assert.Contains(t, detail, "control plane")
	assert.Contains(t, detail, "shared")
	assert.Contains(t, detail, "worker nodes")
	assert.Contains(t, detail, "EC2")
}
