package carbon

import "testing"

// TestGetGPUSpec_KnownTypes verifies that known GPU instance types are found
// in the embedded GPU specifications data.
func TestGetGPUSpec_KnownTypes(t *testing.T) {
	tests := []struct {
		name         string
		instanceType string
		wantGPUModel string
		wantGPUCount int
		wantTDP      float64
	}{
		{
			name:         "p4d.24xlarge has 8x A100 GPUs",
			instanceType: "p4d.24xlarge",
			wantGPUModel: "A100",
			wantGPUCount: 8,
			wantTDP:      400,
		},
		{
			name:         "g5.xlarge has 1x A10G GPU",
			instanceType: "g5.xlarge",
			wantGPUModel: "A10G",
			wantGPUCount: 1,
			wantTDP:      150,
		},
		{
			name:         "g5.12xlarge has 4x A10G GPUs",
			instanceType: "g5.12xlarge",
			wantGPUModel: "A10G",
			wantGPUCount: 4,
			wantTDP:      150,
		},
		{
			name:         "inf2.24xlarge has 6x Inferentia2 chips",
			instanceType: "inf2.24xlarge",
			wantGPUModel: "Inferentia2",
			wantGPUCount: 6,
			wantTDP:      175,
		},
		{
			name:         "trn1.32xlarge has 16x Trainium chips",
			instanceType: "trn1.32xlarge",
			wantGPUModel: "Trainium",
			wantGPUCount: 16,
			wantTDP:      175,
		},
		{
			name:         "g4dn.xlarge has 1x T4 GPU",
			instanceType: "g4dn.xlarge",
			wantGPUModel: "T4",
			wantGPUCount: 1,
			wantTDP:      70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, ok := GetGPUSpec(tt.instanceType)
			if !ok {
				t.Fatalf("GetGPUSpec(%q) returned false, want true", tt.instanceType)
			}

			if spec.GPUModel != tt.wantGPUModel {
				t.Errorf("GPUModel = %q, want %q", spec.GPUModel, tt.wantGPUModel)
			}
			if spec.GPUCount != tt.wantGPUCount {
				t.Errorf("GPUCount = %d, want %d", spec.GPUCount, tt.wantGPUCount)
			}
			if spec.TDPPerGPU != tt.wantTDP {
				t.Errorf("TDPPerGPU = %f, want %f", spec.TDPPerGPU, tt.wantTDP)
			}
		})
	}
}

// TestGetGPUSpec_NonGPUInstance verifies that non-GPU instances return false.
func TestGetGPUSpec_NonGPUInstance(t *testing.T) {
	tests := []string{
		"t3.micro",
		"m5.large",
		"c5.xlarge",
		"r5.2xlarge",
		"unknown-type",
	}

	for _, instanceType := range tests {
		t.Run(instanceType, func(t *testing.T) {
			_, ok := GetGPUSpec(instanceType)
			if ok {
				t.Errorf("GetGPUSpec(%q) returned true, want false for non-GPU instance", instanceType)
			}
		})
	}
}

// TestHasGPU verifies the HasGPU convenience function.
func TestHasGPU(t *testing.T) {
	tests := []struct {
		instanceType string
		want         bool
	}{
		{"p4d.24xlarge", true},
		{"g5.xlarge", true},
		{"t3.micro", false},
		{"m5.large", false},
		{"inf2.xlarge", true},
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			got := HasGPU(tt.instanceType)
			if got != tt.want {
				t.Errorf("HasGPU(%q) = %v, want %v", tt.instanceType, got, tt.want)
			}
		})
	}
}

// TestGPUSpecCount verifies that GPU specs are loaded.
func TestGPUSpecCount(t *testing.T) {
	count := GPUSpecCount()
	// We should have at least the core GPU instance types
	if count < 10 {
		t.Errorf("GPUSpecCount() = %d, want at least 10", count)
	}
}

// TestCalculateGPUPowerWatts verifies GPU power calculation.
func TestCalculateGPUPowerWatts(t *testing.T) {
	tests := []struct {
		name         string
		instanceType string
		utilization  float64
		wantPower    float64
	}{
		{
			name:         "p4d.24xlarge at 100% utilization",
			instanceType: "p4d.24xlarge",
			utilization:  1.0,
			// 8 GPUs × 400W TDP × 1.0 utilization = 3200W
			wantPower: 3200,
		},
		{
			name:         "p4d.24xlarge at 50% utilization",
			instanceType: "p4d.24xlarge",
			utilization:  0.5,
			// 8 GPUs × 400W TDP × 0.5 utilization = 1600W
			wantPower: 1600,
		},
		{
			name:         "g5.xlarge at 100% utilization",
			instanceType: "g5.xlarge",
			utilization:  1.0,
			// 1 GPU × 150W TDP × 1.0 utilization = 150W
			wantPower: 150,
		},
		{
			name:         "non-GPU instance",
			instanceType: "t3.micro",
			utilization:  1.0,
			wantPower:    0,
		},
		{
			name:         "zero utilization",
			instanceType: "p4d.24xlarge",
			utilization:  0.0,
			wantPower:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateGPUPowerWatts(tt.instanceType, tt.utilization)
			if got != tt.wantPower {
				t.Errorf("CalculateGPUPowerWatts(%q, %f) = %f, want %f",
					tt.instanceType, tt.utilization, got, tt.wantPower)
			}
		})
	}
}
