package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstanceSpec_KnownTypes(t *testing.T) {
	tests := []struct {
		name         string
		instanceType string
		wantFound    bool
		wantVCPU     int
	}{
		{
			name:         "t3.micro exists",
			instanceType: "t3.micro",
			wantFound:    true,
			wantVCPU:     2,
		},
		{
			name:         "m5.large exists",
			instanceType: "m5.large",
			wantFound:    true,
			wantVCPU:     2,
		},
		{
			name:         "c5.xlarge exists",
			instanceType: "c5.xlarge",
			wantFound:    true,
			wantVCPU:     4,
		},
		{
			name:         "r5.2xlarge exists",
			instanceType: "r5.2xlarge",
			wantFound:    true,
			wantVCPU:     8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, found := GetInstanceSpec(tt.instanceType)
			assert.Equal(t, tt.wantFound, found, "expected found=%v", tt.wantFound)
			if tt.wantFound {
				assert.Equal(t, tt.instanceType, spec.InstanceType)
				assert.Equal(t, tt.wantVCPU, spec.VCPUCount)
				assert.Greater(t, spec.MinWatts, 0.0, "MinWatts should be positive")
				assert.GreaterOrEqual(t, spec.MaxWatts, spec.MinWatts, "MaxWatts should be >= MinWatts")
			}
		})
	}
}

func TestGetInstanceSpec_UnknownType(t *testing.T) {
	spec, found := GetInstanceSpec("nonexistent.type")
	assert.False(t, found)
	assert.Equal(t, InstanceSpec{}, spec)
}

func TestGetInstanceSpec_EmptyType(t *testing.T) {
	spec, found := GetInstanceSpec("")
	assert.False(t, found)
	assert.Equal(t, InstanceSpec{}, spec)
}

func TestInstanceSpecCount_HasEntries(t *testing.T) {
	count := InstanceSpecCount()
	// CCF data has 500+ instance types
	require.Greater(t, count, 400, "expected at least 400 instance types in CCF data")
}

func TestGetInstanceSpec_PowerValues(t *testing.T) {
	// t3.micro should have specific power characteristics
	spec, found := GetInstanceSpec("t3.micro")
	require.True(t, found, "t3.micro should exist in CCF data")

	// Verify power values are reasonable
	assert.Greater(t, spec.MinWatts, 0.0, "MinWatts should be positive")
	assert.Less(t, spec.MinWatts, 50.0, "MinWatts should be reasonable")
	assert.Greater(t, spec.MaxWatts, spec.MinWatts, "MaxWatts should be > MinWatts")
	assert.Less(t, spec.MaxWatts, 100.0, "MaxWatts should be reasonable")
}
