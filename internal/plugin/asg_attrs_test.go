package plugin

import (
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// TestExtractASGAttributes_InstanceType verifies the priority-based instance type resolution
// for ASG resources. The resolution order is: sku → instance_type tag → launch_template →
// launch_configuration. An error is returned when no instance type is found.
func TestExtractASGAttributes_InstanceType(t *testing.T) {
	tests := []struct {
		name             string
		sku              string
		tags             map[string]string
		wantInstanceType string
		wantErr          bool
	}{
		{
			name:             "sku takes priority over tags",
			sku:              "m5.large",
			tags:             map[string]string{"instance_type": "t3.micro"},
			wantInstanceType: "m5.large",
		},
		{
			name:             "instance_type tag fallback",
			tags:             map[string]string{"instance_type": "t3.medium"},
			wantInstanceType: "t3.medium",
		},
		{
			name:             "launch_template.instance_type fallback",
			tags:             map[string]string{"launch_template.instance_type": "c5.xlarge"},
			wantInstanceType: "c5.xlarge",
		},
		{
			name:             "launch_configuration.instance_type fallback",
			tags:             map[string]string{"launch_configuration.instance_type": "r5.2xlarge"},
			wantInstanceType: "r5.2xlarge",
		},
		{
			name:    "no instance type returns error",
			tags:    map[string]string{"desired_capacity": "3"},
			wantErr: true,
		},
		{
			name:    "nil tags and empty sku returns error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &pbc.ResourceDescriptor{
				Sku:  tt.sku,
				Tags: tt.tags,
			}
			attrs, _, err := ExtractASGAttributes(resource)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if attrs.InstanceType != tt.wantInstanceType {
				t.Errorf("InstanceType = %q, want %q", attrs.InstanceType, tt.wantInstanceType)
			}
		})
	}
}

// TestExtractASGAttributes_DesiredCapacity verifies capacity resolution from tags.
// Priority: desired_capacity → desiredCapacity → min_size → minSize → default 1.
func TestExtractASGAttributes_DesiredCapacity(t *testing.T) {
	tests := []struct {
		name         string
		tags         map[string]string
		wantCapacity int
		wantDefault  bool
	}{
		{
			name:         "desired_capacity tag",
			tags:         map[string]string{"instance_type": "t3.micro", "desired_capacity": "5"},
			wantCapacity: 5,
		},
		{
			name:         "desiredCapacity camelCase",
			tags:         map[string]string{"instance_type": "t3.micro", "desiredCapacity": "3"},
			wantCapacity: 3,
		},
		{
			name:         "min_size fallback",
			tags:         map[string]string{"instance_type": "t3.micro", "min_size": "2"},
			wantCapacity: 2,
		},
		{
			name:         "minSize camelCase fallback",
			tags:         map[string]string{"instance_type": "t3.micro", "minSize": "4"},
			wantCapacity: 4,
		},
		{
			name:         "default capacity of 1",
			tags:         map[string]string{"instance_type": "t3.micro"},
			wantCapacity: 1,
			wantDefault:  true,
		},
		{
			name:         "zero capacity",
			tags:         map[string]string{"instance_type": "t3.micro", "desired_capacity": "0"},
			wantCapacity: 0,
		},
		{
			name:         "negative capacity treated as zero",
			tags:         map[string]string{"instance_type": "t3.micro", "desired_capacity": "-1"},
			wantCapacity: 0,
		},
		{
			name:         "non-numeric capacity falls through to default",
			tags:         map[string]string{"instance_type": "t3.micro", "desired_capacity": "abc"},
			wantCapacity: 1,
			wantDefault:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &pbc.ResourceDescriptor{
				Tags: tt.tags,
			}
			attrs, dt, err := ExtractASGAttributes(resource)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if attrs.DesiredCapacity != tt.wantCapacity {
				t.Errorf("DesiredCapacity = %d, want %d", attrs.DesiredCapacity, tt.wantCapacity)
			}
			if tt.wantDefault {
				if dt.Quality() == qualityHigh {
					t.Error("expected defaults to be tracked when using default capacity")
				}
			}
		})
	}
}

// TestExtractASGAttributes_OS verifies OS resolution from tags.
// Checks operating_system tag, platform tag, and default Linux behavior.
func TestExtractASGAttributes_OS(t *testing.T) {
	tests := []struct {
		name   string
		tags   map[string]string
		wantOS string
	}{
		{
			name:   "operating_system tag",
			tags:   map[string]string{"instance_type": "t3.micro", "operating_system": "Windows"},
			wantOS: "Windows",
		},
		{
			name:   "platform tag fallback",
			tags:   map[string]string{"instance_type": "t3.micro", "platform": "rhel"},
			wantOS: "RHEL",
		},
		{
			name:   "default OS is Linux",
			tags:   map[string]string{"instance_type": "t3.micro"},
			wantOS: "Linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &pbc.ResourceDescriptor{
				Tags: tt.tags,
			}
			attrs, _, err := ExtractASGAttributes(resource)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if attrs.OS != tt.wantOS {
				t.Errorf("OS = %q, want %q", attrs.OS, tt.wantOS)
			}
		})
	}
}
