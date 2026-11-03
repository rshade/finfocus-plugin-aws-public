package plugin

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultFargateVCPU      = 0.25
	defaultFargateMemoryGiB = 0.5
	fargateFreeEphemeralGiB = 20.0
	fargateMaxEphemeralGiB  = 200.0
	fargateMaxVCPU          = 16.0
	mibPerGiB               = 1024.0

	fargateLaunchType = "FARGATE"
	ecsSpotProvider   = "FARGATE_SPOT"
)

// FargateAttributes holds the task configuration used to price an ECS
// service that runs on Fargate.
type FargateAttributes struct {
	VCPU          float64
	MemoryGiB     float64
	DesiredCount  int
	Windows       bool
	ARM           bool
	EphemeralGiB  float64
	SpotRequested bool
}

// fargateInvalid reports a bad input as InvalidArgument.
func (p *AWSPublicPlugin) fargateInvalid(traceID, msg string) error {
	return p.newErrorWithID(traceID, codes.InvalidArgument, msg, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
}

// fargateNested reads one field of a nested block. It checks a struct on attributes,
// then dotted tags, then a Go map-string tag, which is how
// finfocus core flattens a multi-field Terraform block.
func fargateNested(tags map[string]string, in resourceInputs, parents, fields []string) string {
	for _, parent := range parents {
		if st, ok := in.Struct(parent); ok {
			if v := structString(st, fields...); v != "" {
				return v
			}
		}
	}
	for _, parent := range parents {
		for _, field := range fields {
			if v := strings.TrimSpace(tags[parent+"."+field]); v != "" {
				return v
			}
		}
	}
	for _, parent := range parents {
		if v := goMapField(tags[parent], fields); v != "" {
			return v
		}
	}
	return ""
}

func goMapField(raw string, fields []string) string {
	if !strings.HasPrefix(strings.TrimSpace(raw), "map[") {
		return ""
	}
	m := parseGoMapString(raw)
	for _, field := range fields {
		if v := m[field]; v != "" {
			return v
		}
	}
	return ""
}

// fargateProviders returns the upper-cased capacity provider strategy text
// from tags or attributes, for substring checks.
func fargateProviders(tags map[string]string, in resourceInputs) string {
	keys := []string{"capacity_provider_strategy", "capacityProviderStrategy", "capacityProviderStrategies"}
	if v := firstTag(tags, keys...); v != "" {
		return strings.ToUpper(v)
	}
	for _, key := range keys {
		if v, ok := pluginsdk.AttributeValue(in.attrs, key); ok && v != nil {
			return strings.ToUpper(protojson.Format(v))
		}
	}
	return ""
}

// parseFargateCPU converts a task CPU setting to vCPU. "256" and "1024" are
// CPU units, "0.25 vCPU" is vCPU, and a bare value of 16 or less is vCPU.
func parseFargateCPU(raw string) (float64, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if rest, ok := strings.CutSuffix(s, "vcpu"); ok {
		return parseFargateQuantity(rest)
	}
	v, ok := parseFargateQuantity(s)
	if !ok {
		return 0, false
	}
	if v > fargateMaxVCPU {
		return v / mibPerGiB, true
	}
	return v, true
}

// parseFargateMemory converts a task memory setting to GiB. A bare value is
// MiB, and a "GB" or "GiB" suffix is GiB.
func parseFargateMemory(raw string) (float64, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	for _, suffix := range []string{"gib", "gb"} {
		if rest, ok := strings.CutSuffix(s, suffix); ok {
			return parseFargateQuantity(rest)
		}
	}
	v, ok := parseFargateQuantity(s)
	if !ok {
		return 0, false
	}
	return v / mibPerGiB, true
}

// ExtractFargateAttributes reads the task configuration of an ECS service
// from attributes, then tags. Missing cpu, memory, and desired count use
// documented defaults that are recorded in the returned DefaultsTracker. The
// task definition holds cpu and memory in real state, so callers normally
// supply them.
//
// cpu accepts CPU units ("256") or vCPU ("0.25 vCPU"); a bare value above 16
// is CPU units. memory accepts MiB ("512") or GiB ("2GB").
//
// Returns InvalidArgument for a non-numeric or non-positive value, ephemeral
// storage above 200 GiB, or Windows on arm64.
func (p *AWSPublicPlugin) ExtractFargateAttributes(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (FargateAttributes, *DefaultsTracker, error) {
	var dt DefaultsTracker
	tags := pricingTags(resource)
	in := newResourceInputs(resource)
	attrs := FargateAttributes{
		VCPU:          defaultFargateVCPU,
		MemoryGiB:     defaultFargateMemoryGiB,
		DesiredCount:  1,
		Windows:       false,
		ARM:           false,
		EphemeralGiB:  fargateFreeEphemeralGiB,
		SpotRequested: false,
	}

	if err := p.fargateSizing(traceID, tags, &attrs, &dt); err != nil {
		return attrs, &dt, err
	}
	if err := p.fargateRuntime(traceID, tags, in, &attrs); err != nil {
		return attrs, &dt, err
	}
	attrs.SpotRequested = strings.Contains(fargateProviders(tags, in), ecsSpotProvider)
	return attrs, &dt, nil
}

// fargateSizing reads cpu, memory, and desired_count, recording defaults.
func (p *AWSPublicPlugin) fargateSizing(
	traceID string,
	tags map[string]string,
	attrs *FargateAttributes,
	dt *DefaultsTracker,
) error {
	if raw := firstTag(tags, "cpu"); raw != "" {
		v, ok := parseFargateCPU(raw)
		if !ok {
			return p.fargateInvalid(traceID,
				fmt.Sprintf("ECS Fargate requires a positive \"cpu\" (CPU units or vCPU), got %q", raw))
		}
		attrs.VCPU = v
	} else {
		dt.Add("cpu", "0.25 vCPU", KindConfig)
	}

	if raw := firstTag(tags, "memory"); raw != "" {
		v, ok := parseFargateMemory(raw)
		if !ok {
			return p.fargateInvalid(traceID,
				fmt.Sprintf("ECS Fargate requires a positive \"memory\" (MiB or GiB), got %q", raw))
		}
		attrs.MemoryGiB = v
	} else {
		dt.Add("memory", "0.5 GiB", KindConfig)
	}

	if raw := firstTag(tags, "desired_count", "desiredCount"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return p.fargateInvalid(traceID,
				fmt.Sprintf("ECS Fargate requires a non-negative \"desired_count\", got %q", raw))
		}
		attrs.DesiredCount = n
	} else {
		dt.Add("desired_count", "1", KindConfig)
	}
	return nil
}

// fargateRuntime reads the architecture, operating system, and ephemeral
// storage settings.
func (p *AWSPublicPlugin) fargateRuntime(
	traceID string,
	tags map[string]string,
	in resourceInputs,
	attrs *FargateAttributes,
) error {
	platform := []string{"runtime_platform", "runtimePlatform"}
	arch := firstTag(tags, "cpu_architecture", "cpuArchitecture")
	if arch == "" {
		arch = fargateNested(tags, in, platform, []string{"cpu_architecture", "cpuArchitecture"})
	}
	attrs.ARM = strings.EqualFold(arch, "ARM64")
	osFamily := firstTag(tags, "operating_system_family", "operatingSystemFamily")
	if osFamily == "" {
		osFamily = fargateNested(tags, in, platform, []string{"operating_system_family", "operatingSystemFamily"})
	}
	attrs.Windows = strings.HasPrefix(strings.ToUpper(osFamily), "WINDOWS")
	if attrs.Windows && attrs.ARM {
		return p.fargateInvalid(traceID, "Fargate does not offer Windows tasks on ARM64")
	}

	raw := firstTag(tags, "ephemeral_storage_gib", "ephemeralStorageGib")
	if raw == "" {
		raw = fargateNested(tags, in,
			[]string{"ephemeral_storage", "ephemeralStorage"}, []string{"size_in_gib", "sizeInGib"})
	}
	if raw != "" {
		v, ok := parseFargateQuantity(raw)
		if !ok || v > fargateMaxEphemeralGiB {
			return p.fargateInvalid(traceID,
				fmt.Sprintf("ECS Fargate ephemeral storage must be between 20 and 200 GiB, got %q", raw))
		}
		attrs.EphemeralGiB = math.Max(v, fargateFreeEphemeralGiB)
	}
	return nil
}

// ecsLaunchRouting classifies how an ECS service is billed from its launch
// type. It returns a non-empty reason when the
// service runs on EC2 or EXTERNAL capacity, and an error for an unknown launch
// type.
func (p *AWSPublicPlugin) ecsLaunchRouting(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (string, error) {
	tags := pricingTags(resource)
	launch := strings.ToUpper(firstTag(tags, "launch_type", "launchType"))
	switch launch {
	case "", fargateLaunchType:
		return "", nil
	case "EC2", "EXTERNAL":
		return "ECS on " + launch + ": cost is attributed to the underlying EC2/ASG resources", nil
	default:
		return "", p.fargateInvalid(traceID,
			fmt.Sprintf("unsupported ECS launch_type %q (expected FARGATE, EC2, or EXTERNAL)", launch))
	}
}
