package plugin

import (
	"fmt"
	"testing"
)

// TestPricingNotFoundTemplate verifies the template produces expected format.
func TestPricingNotFoundTemplate(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		value    string
		want     string
	}{
		{
			name:     "EC2 instance type",
			resource: "EC2 instance type",
			value:    "t3.micro",
			want:     `EC2 instance type "t3.micro" not found in pricing data`,
		},
		{
			name:     "EBS volume type",
			resource: "EBS volume type",
			value:    "gp3",
			want:     `EBS volume type "gp3" not found in pricing data`,
		},
		{
			name:     "S3 storage class",
			resource: "S3 storage class",
			value:    "GLACIER",
			want:     `S3 storage class "GLACIER" not found in pricing data`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf(PricingNotFoundTemplate, tt.resource, tt.value)
			if got != tt.want {
				t.Errorf("PricingNotFoundTemplate = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPricingUnavailableTemplate verifies the template produces expected format.
func TestPricingUnavailableTemplate(t *testing.T) {
	tests := []struct {
		name    string
		service string
		region  string
		want    string
	}{
		{
			name:    "CloudWatch in ap-northeast-3",
			service: "CloudWatch",
			region:  "ap-northeast-3",
			want:    "CloudWatch pricing data not available for region ap-northeast-3",
		},
		{
			name:    "EKS in us-east-1",
			service: "EKS",
			region:  "us-east-1",
			want:    "EKS pricing data not available for region us-east-1",
		},
		{
			name:    "Lambda in eu-central-1",
			service: "Lambda",
			region:  "eu-central-1",
			want:    "Lambda pricing data not available for region eu-central-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf(PricingUnavailableTemplate, tt.service, tt.region)
			if got != tt.want {
				t.Errorf("PricingUnavailableTemplate = %q, want %q", got, tt.want)
			}
		})
	}
}
