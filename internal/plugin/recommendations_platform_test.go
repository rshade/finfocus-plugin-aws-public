package plugin

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// platformRecsPlugin returns a plugin whose mock prices differ by OS so a
// recommendation priced with the wrong OS is detectable.
func platformRecsPlugin() *AWSPublicPlugin {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6i.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6g.large/Linux/Shared"] = 0.077
	mock.ec2Prices["m5.large/Windows/Shared"] = 0.188
	mock.ec2Prices["m6i.large/Windows/Shared"] = 0.180
	mock.ec2Prices["m6g.large/Windows/Shared"] = 0.150
	mock.ec2Prices["m5.large/RHEL/Shared"] = 0.126
	mock.ec2Prices["m6i.large/RHEL/Shared"] = 0.126
	mock.ec2Prices["m6g.large/RHEL/Shared"] = 0.107
	mock.ec2Prices["m5.large/Linux/Dedicated"] = 0.110
	mock.ec2Prices["m6i.large/Linux/Dedicated"] = 0.100
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	return NewAWSPublicPlugin("us-east-1", "test-version", mock, logger)
}

// recsByModType indexes recommendations by their modification type.
func recsByModType(recs []*pbc.Recommendation) map[string]*pbc.Recommendation {
	out := map[string]*pbc.Recommendation{}
	for _, rec := range recs {
		out[rec.GetModify().GetModificationType()] = rec
	}
	return out
}

// TestGenerateEC2Recommendations_PlatformAware verifies that Graviton
// migration is skipped for Windows, kept for Linux and RHEL, and that every
// recommendation is priced with the instance's own OS and tenancy.
//
// Run with: go test ./internal/plugin/... -run TestGenerateEC2Recommendations_PlatformAware.
func TestGenerateEC2Recommendations_PlatformAware(t *testing.T) {
	tests := []struct {
		name         string
		attrs        EC2Attributes
		wantGraviton bool
		wantAltNote  bool
		wantUpgrade  bool
		wantCurrent  float64
	}{
		{"linux", EC2Attributes{OS: "Linux", Tenancy: "Shared"}, true, true, true, 0.096 * 730},
		{"default attributes", DefaultEC2Attributes(), true, true, true, 0.096 * 730},
		{"windows", EC2Attributes{OS: "Windows", Tenancy: "Shared"}, false, false, true, 0.188 * 730},
		{"rhel", EC2Attributes{OS: "RHEL", Tenancy: "Shared"}, true, true, true, 0.126 * 730},
		{"suse has no price", EC2Attributes{OS: "SUSE", Tenancy: "Shared"}, false, false, false, 0},
		{"dedicated tenancy", EC2Attributes{OS: "Linux", Tenancy: "Dedicated"}, false, true, true, 0.110 * 730},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := platformRecsPlugin().generateEC2Recommendations("m5.large", "us-east-1", tt.attrs)
			byType := recsByModType(recs)

			if _, ok := byType[modTypeGraviton]; ok != tt.wantGraviton {
				t.Errorf("graviton recommendation present = %v, want %v", ok, tt.wantGraviton)
			}

			upgrade, ok := byType[modTypeGenUpgrade]
			if ok != tt.wantUpgrade {
				t.Fatalf("generation upgrade present = %v, want %v", ok, tt.wantUpgrade)
			}
			if !ok {
				return
			}
			if got := upgrade.GetImpact().GetCurrentCost(); got < tt.wantCurrent-0.01 || got > tt.wantCurrent+0.01 {
				t.Errorf("upgrade current cost = %.2f, want %.2f", got, tt.wantCurrent)
			}
			hasNote := strings.Contains(strings.Join(upgrade.GetReasoning(), "\n"), "Alternative: consider")
			if hasNote != tt.wantAltNote {
				t.Errorf("Graviton alternative note present = %v, want %v", hasNote, tt.wantAltNote)
			}
		})
	}
}

// TestGetRecommendations_WindowsPlatformSources verifies the Windows platform
// is honoured from every input GetRecommendations reads: the platform tag, the
// platform attribute, and a full "Windows Server 2022" value.
//
// Run with: go test ./internal/plugin/... -run TestGetRecommendations_WindowsPlatformSources.
func TestGetRecommendations_WindowsPlatformSources(t *testing.T) {
	attrs, err := structpb.NewStruct(map[string]any{"platform": "Windows Server 2022"})
	if err != nil {
		t.Fatalf("NewStruct() error: %v", err)
	}

	tests := []struct {
		name         string
		resource     *pbc.ResourceDescriptor
		wantGraviton bool
	}{
		{"windows tag", &pbc.ResourceDescriptor{Tags: map[string]string{"platform": "windows"}}, false},
		{
			"windows server 2022 tag",
			&pbc.ResourceDescriptor{Tags: map[string]string{"platform": "Windows Server 2022"}},
			false,
		},
		{"windows attribute", &pbc.ResourceDescriptor{Attributes: attrs}, false},
		{"linux tag", &pbc.ResourceDescriptor{Tags: map[string]string{"platform": "linux"}}, true},
		{"missing platform", &pbc.ResourceDescriptor{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.resource.Provider = "aws"
			tt.resource.ResourceType = "aws:ec2/instance:Instance"
			tt.resource.Sku = "m5.large"
			tt.resource.Region = "us-east-1"

			resp, err := platformRecsPlugin().GetRecommendations(context.Background(),
				&pbc.GetRecommendationsRequest{TargetResources: []*pbc.ResourceDescriptor{tt.resource}})
			if err != nil {
				t.Fatalf("GetRecommendations() error: %v", err)
			}
			if _, ok := recsByModType(resp.GetRecommendations())[modTypeGraviton]; ok != tt.wantGraviton {
				t.Errorf("graviton recommendation present = %v, want %v", ok, tt.wantGraviton)
			}
		})
	}
}

// TestPlatformSupportsGraviton verifies which normalized platforms may be
// offered an ARM migration, and that both attribute extractors agree on the
// Windows normalization that drives it.
func TestPlatformSupportsGraviton(t *testing.T) {
	for os, want := range map[string]bool{"Linux": true, "RHEL": true, "SUSE": true, "Windows": false} {
		if got := platformSupportsGraviton(os); got != want {
			t.Errorf("platformSupportsGraviton(%q) = %v, want %v", os, got, want)
		}
	}

	const value = "Windows Server 2022"
	fromTags := ExtractEC2AttributesFromTags(map[string]string{"platform": value})
	st, err := structpb.NewStruct(map[string]any{"platform": value})
	if err != nil {
		t.Fatalf("NewStruct() error: %v", err)
	}
	fromStruct := ExtractEC2AttributesFromStruct(st)
	if fromTags.OS != platformWindows || fromStruct.OS != platformWindows {
		t.Errorf("tags OS = %q, struct OS = %q, want both %q", fromTags.OS, fromStruct.OS, platformWindows)
	}
}
