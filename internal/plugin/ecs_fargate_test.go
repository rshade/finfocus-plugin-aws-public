package plugin

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ecsServiceType = "aws:ecs/service:Service"

func ecsService(tags map[string]string) *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: ecsServiceType,
		Region:       "us-east-1",
		Tags:         tags,
	}
}

func projectedECS(t *testing.T, res *pbc.ResourceDescriptor) *pbc.GetProjectedCostResponse {
	t.Helper()
	resp, err := newFargatePlugin().GetProjectedCost(context.Background(),
		&pbc.GetProjectedCostRequest{Resource: res})
	if err != nil {
		t.Fatalf("GetProjectedCost() error: %v", err)
	}
	return resp
}

func assertMonthly(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("CostPerMonth = %v, want %v", got, want)
	}
}

// TestGetProjectedCost_ECSFargate verifies the per-task hourly formula, desired
// count, ephemeral storage above 20 GiB, the Windows license fee, and arm64
// rates for each supported input form.
func TestGetProjectedCost_ECSFargate(t *testing.T) {
	x, a, w := testFargateX86, testFargateARM, testFargateWindows
	tests := []struct {
		name   string
		tags   map[string]string
		hourly float64
		count  int
	}{
		{
			name:   "issue example, 2 tasks of 0.5 vCPU and 1 GiB",
			tags:   map[string]string{"cpu": "512", "memory": "1024", "desired_count": "2"},
			hourly: 0.5*x.VCPUPerHour + 1*x.GiBPerHour, count: 2,
		},
		{
			name: "arm64 via runtime_platform go map",
			tags: map[string]string{
				"cpu":             "0.5 vCPU",
				"memory":          "1GB",
				"desiredCount":    "2",
				"runtimePlatform": "map[cpuArchitecture:ARM64 operatingSystemFamily:LINUX]",
			},
			hourly: 0.5*a.VCPUPerHour + 1*a.GiBPerHour,
			count:  2,
		},
		{
			name:   "arm64 via dotted tag",
			tags:   map[string]string{"cpu": "1", "memory": "2048", "runtime_platform.cpu_architecture": "ARM64"},
			hourly: 1*a.VCPUPerHour + 2*a.GiBPerHour, count: 1,
		},
		{
			name: "windows adds license fee",
			tags: map[string]string{
				"cpu": "1024", "memory": "2048",
				"operating_system_family": "WINDOWS_SERVER_2022_CORE",
			},
			hourly: 1*(w.VCPUPerHour+w.OSPerVCPUHour) + 2*w.GiBPerHour, count: 1,
		},
		{
			name:   "ephemeral storage above 20 GiB",
			tags:   map[string]string{"cpu": "256", "memory": "512", "ephemeral_storage_gib": "50"},
			hourly: 0.25*x.VCPUPerHour + 0.5*x.GiBPerHour + 30*testFargateEphemeral, count: 1,
		},
		{
			name:   "ephemeral storage nested go map",
			tags:   map[string]string{"cpu": "256", "memory": "512", "ephemeralStorage": "map[sizeInGib:100]"},
			hourly: 0.25*x.VCPUPerHour + 0.5*x.GiBPerHour + 80*testFargateEphemeral, count: 1,
		},
		{
			name:   "ephemeral storage under 20 is clamped",
			tags:   map[string]string{"cpu": "256", "memory": "512", "ephemeral_storage_gib": "10"},
			hourly: 0.25*x.VCPUPerHour + 0.5*x.GiBPerHour, count: 1,
		},
		{
			name:   "defaults when nothing is set",
			tags:   map[string]string{},
			hourly: 0.25*x.VCPUPerHour + 0.5*x.GiBPerHour, count: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := projectedECS(t, ecsService(tt.tags))
			assertMonthly(t, resp.GetCostPerMonth(), tt.hourly*float64(tt.count)*HoursPerMonthProd)
			if math.Abs(resp.GetUnitPrice()-tt.hourly) > 1e-12 {
				t.Errorf("UnitPrice = %v, want %v", resp.GetUnitPrice(), tt.hourly)
			}
			if resp.GetGrowthType() != pbc.GrowthType_GROWTH_TYPE_NONE {
				t.Errorf("GrowthType = %v, want NONE", resp.GetGrowthType())
			}
			if !strings.Contains(resp.GetBillingDetail(), "carbon estimate not available") {
				t.Errorf("BillingDetail = %q, want carbon note", resp.GetBillingDetail())
			}
		})
	}
}

// TestGetProjectedCost_ECSFargate_IssueFigures checks the figures in issue 393.
func TestGetProjectedCost_ECSFargate_IssueFigures(t *testing.T) {
	x86 := projectedECS(t, ecsService(map[string]string{"cpu": "0.5 vCPU", "memory": "1GB", "desired_count": "2"}))
	if got := x86.GetCostPerMonth(); math.Abs(got-36.04) > 0.01 {
		t.Errorf("x86_64 cost = %v, want about 36.04", got)
	}
	arm := projectedECS(t, ecsService(map[string]string{
		"cpu": "0.5 vCPU", "memory": "1GB", "desired_count": "2", "cpu_architecture": "arm64",
	}))
	if got := arm.GetCostPerMonth(); math.Abs(got-28.84) > 0.01 {
		t.Errorf("arm64 cost = %v, want about 28.84", got)
	}
}

// TestGetProjectedCost_ECSFargate_Defaults verifies missing cpu, memory, and
// desired_count are recorded and an explicit set is not.
func TestGetProjectedCost_ECSFargate_Defaults(t *testing.T) {
	resp := projectedECS(t, ecsService(nil))
	for _, field := range []string{"cpu", "memory", "desired_count"} {
		if !strings.Contains(resp.GetMetadata()["defaults_applied"], field) {
			t.Errorf("defaults_applied = %q, missing %s", resp.GetMetadata()["defaults_applied"], field)
		}
	}
	if resp.GetMetadata()["estimate_quality"] != qualityMedium {
		t.Errorf("estimate_quality = %q, want %q", resp.GetMetadata()["estimate_quality"], qualityMedium)
	}

	explicit := projectedECS(t, ecsService(map[string]string{"cpu": "256", "memory": "512", "desired_count": "1"}))
	if len(explicit.GetMetadata()) != 0 {
		t.Errorf("explicit inputs recorded defaults: %v", explicit.GetMetadata())
	}
}

// TestGetProjectedCost_ECSFargate_NonFargateLaunchTypes verifies EC2 and
// EXTERNAL services return $0 with an explanation rather than an error.
func TestGetProjectedCost_ECSFargate_NonFargateLaunchTypes(t *testing.T) {
	for _, launch := range []string{"EC2", "external", "ec2"} {
		t.Run(launch, func(t *testing.T) {
			resp := projectedECS(
				t,
				ecsService(map[string]string{"launchType": launch, "cpu": "1024", "memory": "2048"}),
			)
			if resp.GetCostPerMonth() != 0 || resp.GetUnitPrice() != 0 {
				t.Errorf("cost = %v/%v, want 0", resp.GetCostPerMonth(), resp.GetUnitPrice())
			}
			if !strings.Contains(resp.GetBillingDetail(), "attributed to the underlying EC2/ASG") {
				t.Errorf("BillingDetail = %q", resp.GetBillingDetail())
			}
		})
	}
}

// TestGetProjectedCost_ECSFargate_Spot verifies a Spot capacity provider
// returns the on-demand estimate plus a note, from tags and attributes.
func TestGetProjectedCost_ECSFargate_Spot(t *testing.T) {
	base := 0.25*testFargateX86.VCPUPerHour + 0.5*testFargateX86.GiBPerHour

	resp := projectedECS(t, ecsService(map[string]string{
		"capacityProviderStrategy": "map[base:0 capacityProvider:FARGATE_SPOT weight:1]",
	}))
	assertMonthly(t, resp.GetCostPerMonth(), base*HoursPerMonthProd)
	if !strings.Contains(resp.GetBillingDetail(), "Fargate Spot not priced; on-demand rate shown") {
		t.Errorf("BillingDetail = %q", resp.GetBillingDetail())
	}

	attrs, err := structpb.NewStruct(map[string]any{
		"capacityProviderStrategies": []any{map[string]any{"capacityProvider": "FARGATE_SPOT", "weight": 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	res := ecsService(nil)
	res.Attributes = attrs
	resp = projectedECS(t, res)
	if !strings.Contains(resp.GetBillingDetail(), "Fargate Spot not priced") {
		t.Errorf("attribute list BillingDetail = %q", resp.GetBillingDetail())
	}

	plain := projectedECS(
		t,
		ecsService(map[string]string{"capacityProviderStrategy": "map[capacityProvider:FARGATE weight:1]"}),
	)
	if strings.Contains(plain.GetBillingDetail(), "Spot") {
		t.Errorf("FARGATE provider produced Spot note: %q", plain.GetBillingDetail())
	}
}

// TestGetProjectedCost_ECSFargate_ZeroDesiredCount verifies zero tasks cost $0.
func TestGetProjectedCost_ECSFargate_ZeroDesiredCount(t *testing.T) {
	resp := projectedECS(t, ecsService(map[string]string{"desired_count": "0"}))
	if resp.GetCostPerMonth() != 0 || !strings.Contains(resp.GetBillingDetail(), "zero tasks") {
		t.Errorf("cost %v detail %q", resp.GetCostPerMonth(), resp.GetBillingDetail())
	}
}

// TestGetProjectedCost_ECSFargate_Attributes verifies attributes win over tags,
// including nested structs.
func TestGetProjectedCost_ECSFargate_Attributes(t *testing.T) {
	attrs, err := structpb.NewStruct(map[string]any{
		"cpu":          "512",
		"memory":       "1024",
		"desiredCount": 3,
		"runtimePlatform": map[string]any{
			"cpuArchitecture": "ARM64",
		},
		"ephemeralStorage": map[string]any{"sizeInGib": 40},
	})
	if err != nil {
		t.Fatal(err)
	}
	res := ecsService(map[string]string{"cpu": "4096", "memory": "8192", "desired_count": "9"})
	res.Attributes = attrs

	a := testFargateARM
	hourly := 0.5*a.VCPUPerHour + 1*a.GiBPerHour + 20*testFargateEphemeral
	assertMonthly(t, projectedECS(t, res).GetCostPerMonth(), hourly*3*HoursPerMonthProd)
}

// TestGetProjectedCost_ECSFargate_InvalidInputs verifies bad inputs fail with
// InvalidArgument instead of returning an unpriced $0.
func TestGetProjectedCost_ECSFargate_InvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
	}{
		{"non numeric cpu", map[string]string{"cpu": "lots"}},
		{"zero cpu", map[string]string{"cpu": "0"}},
		{"negative memory", map[string]string{"memory": "-512"}},
		{"non numeric desired count", map[string]string{"desired_count": "two"}},
		{"negative desired count", map[string]string{"desired_count": "-1"}},
		{"ephemeral above 200", map[string]string{"ephemeral_storage_gib": "201"}},
		{"ephemeral non numeric", map[string]string{"ephemeral_storage_gib": "big"}},
		{"unknown launch type", map[string]string{"launch_type": "LAMBDA"}},
		{
			"windows on arm64",
			map[string]string{"cpu_architecture": "ARM64", "operating_system_family": "WINDOWS_SERVER_2019_FULL"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newFargatePlugin().GetProjectedCost(context.Background(),
				&pbc.GetProjectedCostRequest{Resource: ecsService(tt.tags)})
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("code = %v (err %v), want InvalidArgument", status.Code(err), err)
			}
		})
	}
}

// TestParseFargateCPUAndMemory verifies the unit heuristics.
func TestParseFargateCPUAndMemory(t *testing.T) {
	cpuTests := map[string]float64{
		"256":      0.25,
		"1024":     1,
		"4096":     4,
		"0.25":     0.25,
		"1":        1,
		"16":       16,
		"2 vCPU":   2,
		"0.5 VCPU": 0.5,
	}
	for in, want := range cpuTests {
		if got, ok := parseFargateCPU(in); !ok || math.Abs(got-want) > 1e-12 {
			t.Errorf("parseFargateCPU(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	memTests := map[string]float64{"512": 0.5, "2048": 2, "0.5 GB": 0.5, "2GB": 2, "4 GiB": 4}
	for in, want := range memTests {
		if got, ok := parseFargateMemory(in); !ok || math.Abs(got-want) > 1e-12 {
			t.Errorf("parseFargateMemory(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
}

// TestECSFargate_Normalization verifies service routing for every accepted
// resource type form and that the sibling ECS types stay zero-cost.
func TestECSFargate_Normalization(t *testing.T) {
	for _, rt := range []string{ecsServiceType, "ecs", "fargate", "ecs_service", "ECS"} {
		if got := detectService(normalizeResourceType(rt)); got != serviceFargate {
			t.Errorf("detectService(%q) = %q, want %q", rt, got, serviceFargate)
		}
	}
	zero := map[string]string{
		"aws:ecs/taskDefinition:TaskDefinition": serviceECSTaskDef,
		"aws:ecs/cluster:Cluster":               serviceECSCluster,
	}
	for rt, want := range zero {
		if got := detectService(normalizeResourceType(rt)); got != want {
			t.Errorf("detectService(%q) = %q, want %q", rt, got, want)
		}
		if !IsZeroCostService(want) {
			t.Errorf("%q is not zero-cost", want)
		}
	}
}

// TestECSZeroCostTypes verifies task definitions and clusters are supported,
// priced at $0, and described as zero_cost by GetPricingSpec.
func TestECSZeroCostTypes(t *testing.T) {
	plugin := newFargatePlugin()
	for _, rt := range []string{"aws:ecs/taskDefinition:TaskDefinition", "aws:ecs/cluster:Cluster"} {
		t.Run(rt, func(t *testing.T) {
			res := &pbc.ResourceDescriptor{Provider: "aws", ResourceType: rt, Region: "us-east-1"}

			sup, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{Resource: res})
			if err != nil || !sup.GetSupported() {
				t.Fatalf("Supports() = %v, %v", sup, err)
			}
			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{Resource: res})
			if err != nil || resp.GetCostPerMonth() != 0 {
				t.Fatalf("GetProjectedCost() = %v, %v", resp, err)
			}
			spec, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{Resource: res})
			if err != nil || spec.GetSpec().GetBillingMode() != billingModeZeroCost {
				t.Fatalf("GetPricingSpec() = %v, %v", spec, err)
			}
		})
	}
}

// TestSupportsAndPricingSpec_ECSFargate verifies Supports, GetPricingSpec for
// priced and non-Fargate services.
func TestSupportsAndPricingSpec_ECSFargate(t *testing.T) {
	plugin := newFargatePlugin()
	res := ecsService(map[string]string{"cpu": "512", "memory": "1024"})

	sup, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{Resource: res})
	if err != nil || !sup.GetSupported() {
		t.Fatalf("Supports() = %v, %v", sup, err)
	}

	spec, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{Resource: res})
	if err != nil {
		t.Fatalf("GetPricingSpec() error: %v", err)
	}
	want := 0.5*testFargateX86.VCPUPerHour + 1*testFargateX86.GiBPerHour
	if got := spec.GetSpec().GetRatePerUnit(); math.Abs(got-want) > 1e-12 {
		t.Errorf("RatePerUnit = %v, want %v", got, want)
	}
	if spec.GetSpec().GetUnit() != unitPerTaskHour || spec.GetSpec().GetBillingMode() != billingModeOnDemand {
		t.Errorf("spec = %v", spec.GetSpec())
	}

	ec2Res := ecsService(map[string]string{"launch_type": "EC2"})
	spec, err = plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{Resource: ec2Res})
	if err != nil || spec.GetSpec().GetBillingMode() != billingModeZeroCost || spec.GetSpec().GetRatePerUnit() != 0 {
		t.Errorf("EC2 launch spec = %v, %v", spec.GetSpec(), err)
	}
}

// TestGetActualCost_ECSFargate_MatchesProjected verifies GetActualCost and
// GetProjectedCost agree for a 730-hour month on every resource type form.
func TestGetActualCost_ECSFargate_MatchesProjected(t *testing.T) {
	tags := map[string]string{"cpu": "512", "memory": "1024", "desired_count": "2"}
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, rt := range []string{ecsServiceType, "ecs", "fargate"} {
		t.Run(rt, func(t *testing.T) {
			projected := projectedECS(t, &pbc.ResourceDescriptor{
				Provider: "aws", ResourceType: rt, Region: "us-east-1", Tags: tags,
			})
			resp, err := newTestPluginForActual().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", rt, "", "us-east-1", tags),
				Start:      timestamppb.New(from),
				End:        timestamppb.New(from.Add(HoursPerMonthProd * time.Hour)),
			})
			if err != nil {
				t.Fatalf("GetActualCost() error: %v", err)
			}
			if len(resp.GetResults()) == 0 {
				t.Fatal("no results")
			}
			if got := resp.GetResults()[0].GetCost(); math.Abs(got-projected.GetCostPerMonth()) > 1e-6 || got <= 0 {
				t.Errorf("actual %v != projected %v", got, projected.GetCostPerMonth())
			}
		})
	}
}

// TestECSARNRouting verifies ECS ARNs map to the ECS service types.
func TestECSARNRouting(t *testing.T) {
	tests := map[string]string{
		"arn:aws:ecs:us-east-1:123456789012:service/main/web":      serviceFargate,
		"arn:aws:ecs:us-east-1:123456789012:task-definition/web:1": serviceECSTaskDef,
		"arn:aws:ecs:us-east-1:123456789012:cluster/main":          serviceECSCluster,
	}
	for arn, want := range tests {
		parsed, err := ParseARN(arn)
		if err != nil {
			t.Fatalf("ParseARN(%q): %v", arn, err)
		}
		if got := parsed.ToPulumiResourceType(); got != want {
			t.Errorf("ToPulumiResourceType(%q) = %q, want %q", arn, got, want)
		}
	}
}
