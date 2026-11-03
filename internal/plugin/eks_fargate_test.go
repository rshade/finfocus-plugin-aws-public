package plugin

import (
	"context"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func fargatePod(cpu, memory string) *pbc.ResourceDescriptor {
	tags := map[string]string{"kind": "fargate", "cluster": "c1", "namespace": "default", "pod": "web-0"}
	if cpu != "" {
		tags["cpu"] = cpu
	}
	if memory != "" {
		tags["memory_gib"] = memory
	}
	return &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "aws:eks/fargate:Pod",
		Sku:          "fargate",
		Region:       "us-east-1",
		Tags:         tags,
	}
}

func newFargatePlugin() *AWSPublicPlugin {
	return NewAWSPublicPlugin("us-east-1", "test-version",
		newMockPricingClient("us-east-1", "USD"), zerolog.New(nil).Level(zerolog.InfoLevel))
}

// TestGetProjectedCost_EKSFargatePod verifies the monthly cost is
// (vCPU rate * cpu + GiB rate * memory) * 730 and is positive for fractional
// requests.
func TestGetProjectedCost_EKSFargatePod(t *testing.T) {
	tests := []struct {
		name, cpu, memory string
	}{
		{"smallest fractional pod", "0.25", "0.5"},
		{"whole cores", "2", "4"},
		{"decimal memory", "1", "2.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := newFargatePlugin().GetProjectedCost(context.Background(),
				&pbc.GetProjectedCostRequest{Resource: fargatePod(tt.cpu, tt.memory)})
			if err != nil {
				t.Fatalf("GetProjectedCost() error: %v", err)
			}
			cpu, mem := mustFloat(t, tt.cpu), mustFloat(t, tt.memory)
			hourly := testFargateX86.VCPUPerHour*cpu + testFargateX86.GiBPerHour*mem
			if got, want := resp.GetCostPerMonth(), hourly*730; math.Abs(got-want) > 1e-9 || got <= 0 {
				t.Errorf("CostPerMonth = %v, want %v (> 0)", got, want)
			}
			if got := resp.GetUnitPrice(); math.Abs(got-hourly) > 1e-12 {
				t.Errorf("UnitPrice = %v, want hourly %v", got, hourly)
			}
			if resp.GetCurrency() != "USD" {
				t.Errorf("Currency = %q, want USD", resp.GetCurrency())
			}
			if d := resp.GetBillingDetail(); !strings.Contains(d, "EKS Fargate pod") ||
				!strings.Contains(d, "730 hrs/month") {
				t.Errorf("BillingDetail = %q", d)
			}
		})
	}
}

// TestGetProjectedCost_EKSFargatePod_IsNotEKSCluster guards against the
// generic aws:eks/... normalization pricing a Fargate pod as a cluster fee.
func TestGetProjectedCost_EKSFargatePod_IsNotEKSCluster(t *testing.T) {
	if got := normalizeResourceType("aws:eks/fargate:Pod"); got != serviceEKSFargate {
		t.Errorf("normalizeResourceType = %q, want %q", got, serviceEKSFargate)
	}
	if got := detectService(normalizeResourceType("aws:eks/fargate:Pod")); got != serviceEKSFargate {
		t.Errorf("detectService = %q, want %q", got, serviceEKSFargate)
	}
	if got := detectService(normalizeResourceType("aws:eks/cluster:Cluster")); got != serviceEKS {
		t.Errorf("cluster detectService = %q, want %q", got, serviceEKS)
	}
}

// TestIsEKSFargatePodType verifies only the exact lowercased pod token matches.
func TestIsEKSFargatePodType(t *testing.T) {
	tests := []struct {
		resourceType string
		want         bool
	}{
		{"aws:eks/fargate:pod", true},
		{"aws:eks/fargate:podx", false},
		{"aws:eks/fargate:other", false},
		{"aws:eks/fargate:", false},
		{"aws:eks/fargateprofile:fargateprofile", false},
		{"aws:eks/cluster:cluster", false},
	}
	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			if got := isEKSFargatePodType(tt.resourceType); got != tt.want {
				t.Errorf("isEKSFargatePodType(%q) = %v, want %v", tt.resourceType, got, tt.want)
			}
		})
	}
}

// TestGetProjectedCost_EKSFargatePod_Attributes verifies attributes are read
// before tags, including numeric values.
func TestGetProjectedCost_EKSFargatePod_Attributes(t *testing.T) {
	attrs, err := structpb.NewStruct(map[string]any{"cpu": 0.5, "memory_gib": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	res := fargatePod("4", "8")
	res.Attributes = attrs

	resp, err := newFargatePlugin().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{Resource: res})
	if err != nil {
		t.Fatalf("GetProjectedCost() error: %v", err)
	}
	want := (testFargateX86.VCPUPerHour*0.5 + testFargateX86.GiBPerHour*1) * 730
	if got := resp.GetCostPerMonth(); math.Abs(got-want) > 1e-9 {
		t.Errorf("CostPerMonth = %v, want %v from attributes", got, want)
	}
}

// TestGetProjectedCost_EKSFargatePod_InvalidInputs verifies missing or
// unparseable requests fail instead of returning an unpriced $0.
func TestGetProjectedCost_EKSFargatePod_InvalidInputs(t *testing.T) {
	tests := []struct{ name, cpu, memory string }{
		{"missing cpu", "", "1"},
		{"missing memory", "1", ""},
		{"non numeric cpu", "abc", "1"},
		{"non numeric memory", "1", "lots"},
		{"zero cpu", "0", "1"},
		{"negative memory", "1", "-2"},
		{"NaN cpu", "NaN", "1"},
		{"infinite memory", "1", "Inf"},
		{"cost overflows from cpu", "1.7e308", "1"},
		{"cost overflows from memory", "1", "1.7e308"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newFargatePlugin().GetProjectedCost(context.Background(),
				&pbc.GetProjectedCostRequest{Resource: fargatePod(tt.cpu, tt.memory)})
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("code = %v (err %v), want InvalidArgument", status.Code(err), err)
			}
		})
	}
}

// TestGetProjectedCost_EKSFargatePod_RegionMismatch verifies a pod in
// another region is rejected by this region binary.
func TestGetProjectedCost_EKSFargatePod_RegionMismatch(t *testing.T) {
	res := fargatePod("1", "2")
	res.Region = "eu-west-1"
	_, err := newFargatePlugin().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{Resource: res})
	if err == nil {
		t.Fatal("expected region mismatch error")
	}
}

// TestSupports_EKSFargatePod verifies the pod type is advertised.
func TestSupports_EKSFargatePod(t *testing.T) {
	resp, err := newFargatePlugin().Supports(context.Background(), &pbc.SupportsRequest{Resource: fargatePod("1", "2")})
	if err != nil {
		t.Fatalf("Supports() error: %v", err)
	}
	if !resp.GetSupported() {
		t.Errorf("Supported = false, reason %q", resp.GetReason())
	}
}

// TestGetActualCost_EKSFargatePod verifies GetActualCost prices a pod the
// same way as GetProjectedCost for a full 730-hour month.
func TestGetActualCost_EKSFargatePod(t *testing.T) {
	plugin := newTestPluginForActual()
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	resp, err := plugin.GetActualCost(context.Background(), &pbc.GetActualCostRequest{
		ResourceId: makeResourceJSON("aws", "aws:eks/fargate:Pod", "fargate", "us-east-1",
			map[string]string{"cpu": "0.25", "memory_gib": "0.5"}),
		Start: timestamppb.New(from),
		End:   timestamppb.New(from.Add(HoursPerMonthProd * time.Hour)),
	})
	if err != nil {
		t.Fatalf("GetActualCost() error: %v", err)
	}
	if len(resp.GetResults()) == 0 {
		t.Fatal("GetActualCost() returned no results")
	}
	want := (testFargateX86.VCPUPerHour*0.25 + testFargateX86.GiBPerHour*0.5) * HoursPerMonthProd
	if got := resp.GetResults()[0].GetCost(); math.Abs(got-want) > 1e-6 || got <= 0 {
		t.Errorf("Cost = %v, want %v (> 0)", got, want)
	}
}

func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
