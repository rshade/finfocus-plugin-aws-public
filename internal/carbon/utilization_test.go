package carbon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetUtilization_Priority tests the priority order: perResource > request > default (T033 support)
func TestGetUtilization_Priority(t *testing.T) {
	tests := []struct {
		name           string
		requestUtil    float64
		perResourceVal *float64
		want           float64
	}{
		{
			name:           "default when both zero/nil",
			requestUtil:    0,
			perResourceVal: nil,
			want:           DefaultUtilization, // 0.50
		},
		{
			name:           "request-level used when perResource nil",
			requestUtil:    0.7,
			perResourceVal: nil,
			want:           0.7,
		},
		{
			name:           "perResource takes priority over request",
			requestUtil:    0.7,
			perResourceVal: ptr(0.9),
			want:           0.9,
		},
		{
			name:           "perResource zero falls back to request",
			requestUtil:    0.6,
			perResourceVal: ptr(0.0),
			want:           0.6,
		},
		{
			name:           "perResource zero, request zero uses default",
			requestUtil:    0,
			perResourceVal: ptr(0.0),
			want:           DefaultUtilization,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUtilization(tt.requestUtil, tt.perResourceVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestClamp tests value clamping to 0.0-1.0 range (T034)
func TestClamp(t *testing.T) {
	tests := []struct {
		name string
		v    float64
		min  float64
		max  float64
		want float64
	}{
		{"value within range", 0.5, 0.0, 1.0, 0.5},
		{"value below min", -0.5, 0.0, 1.0, 0.0},
		{"value above max", 1.5, 0.0, 1.0, 1.0},
		{"value at min boundary", 0.0, 0.0, 1.0, 0.0},
		{"value at max boundary", 1.0, 0.0, 1.0, 1.0},
		{"negative min range", -0.5, -1.0, 0.0, -0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Clamp(tt.v, tt.min, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestGetUtilization_Clamping tests that utilization values are clamped to 0.0-1.0 (T034)
func TestGetUtilization_Clamping(t *testing.T) {
	tests := []struct {
		name           string
		requestUtil    float64
		perResourceVal *float64
		want           float64
	}{
		{
			name:           "request clamped to max",
			requestUtil:    1.5,
			perResourceVal: nil,
			want:           1.0,
		},
		{
			name:           "request negative clamped to min",
			requestUtil:    -0.5,
			perResourceVal: nil,
			want:           DefaultUtilization, // -0.5 is < 0, so falls through to default
		},
		{
			name:           "perResource clamped to max",
			requestUtil:    0.5,
			perResourceVal: ptr(1.5),
			want:           1.0,
		},
		{
			name:           "perResource valid not clamped",
			requestUtil:    0.5,
			perResourceVal: ptr(0.8),
			want:           0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUtilization(tt.requestUtil, tt.perResourceVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ptr returns a pointer to the given float64 value.
func ptr(f float64) *float64 {
	return &f
}
