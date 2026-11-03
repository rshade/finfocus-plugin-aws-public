package pricing

import (
	"testing"

	"github.com/rs/zerolog"
)

func fargateSKU(sku, usageType, unit, usd string) fixtureSKU {
	return fixtureSKU{
		sku:    sku,
		family: "Compute",
		attrs: map[string]string{
			"usagetype":    usageType,
			"locationType": "AWS Region",
			"regionCode":   "us-east-1",
			"servicecode":  "AmazonECS",
		},
		dims: []fixtureDim{{begin: "0", end: "Inf", unit: unit, usd: usd}},
	}
}

func newFargateTestClient() *Client {
	return &Client{logger: zerolog.Nop(), fargateRates: make(map[string]FargateRates)}
}

// TestParseFargatePricing_MatchesLinuxComputeUsageOnly verifies that only the
// Linux x86_64 and arm64 vCPU and memory usage types are priced, with or
// without a region prefix, and that Windows, ephemeral storage, OS license,
// and unrelated products never overwrite them.
func TestParseFargatePricing_MatchesLinuxComputeUsageOnly(t *testing.T) {
	raw := buildPriceList(t, "AmazonECS", []fixtureSKU{
		fargateSKU("X86V", "USE1-Fargate-vCPU-Hours:perCPU", "hours", "0.0404800000"),
		fargateSKU("X86M", "USE1-Fargate-GB-Hours", "hours", "0.0044450000"),
		fargateSKU("ARMV", "USE1-Fargate-ARM-vCPU-Hours:perCPU", "hours", "0.0323800000"),
		fargateSKU("ARMM", "USE1-Fargate-ARM-GB-Hours", "hours", "0.0035600000"),
		fargateSKU("WINV", "USE1-Fargate-Windows-vCPU-Hours:perCPU", "hours", "0.0465520000"),
		fargateSKU("WINM", "USE1-Fargate-Windows-GB-Hours", "hours", "0.0051117500"),
		fargateSKU("WINOS", "USE1-Fargate-Windows-OS-Hours:perCPU", "hours", "0.0460000000"),
		fargateSKU("EPH", "USE1-Fargate-EphemeralStorage-GB-Hours", "GB-Hours", "0.0001110000"),
		fargateSKU("SPOTV", "USE1-Fargate-Spot-vCPU-Hours:perCPU", "hours", "0.0121440000"),
		fargateSKU("ZERO", "USE1-Fargate-ARM-GB-Hours", "hours", "0.0000000000"),
	})

	for i := range parseRepetitions {
		c := newFargateTestClient()
		assertParsedRegion(t, c.parseFargatePricing, raw)
		if got := c.fargateRates[FargateArchX86]; got != (FargateRates{VCPUPerHour: 0.04048, GiBPerHour: 0.004445}) {
			t.Fatalf("run %d: x86_64 rates = %+v", i, got)
		}
		if got := c.fargateRates[FargateArchARM]; got != (FargateRates{VCPUPerHour: 0.03238, GiBPerHour: 0.00356}) {
			t.Fatalf("run %d: arm64 rates = %+v", i, got)
		}
	}
}

// TestParseFargatePricing_NoRegionPrefixAndIncompletePair verifies the
// unprefixed usage type form and that a vCPU rate without a memory rate is
// not served.
func TestParseFargatePricing_NoRegionPrefixAndIncompletePair(t *testing.T) {
	raw := buildPriceList(t, "AmazonECS", []fixtureSKU{
		fargateSKU("X86V", "Fargate-vCPU-Hours:perCPU", "hours", "0.04048"),
		fargateSKU("X86M", "Fargate-GB-Hours", "hours", "0.004445"),
		fargateSKU("ARMV", "EU-Fargate-ARM-vCPU-Hours:perCPU", "hours", "0.03238"),
	})

	c := newFargateTestClient()
	assertParsedRegion(t, c.parseFargatePricing, raw)
	if _, ok := c.fargateRates[FargateArchX86]; !ok {
		t.Fatal("x86_64 rates missing for unprefixed usage types")
	}
	if got, ok := c.fargateRates[FargateArchARM]; ok {
		t.Fatalf("arm64 rates served with only a vCPU rate: %+v", got)
	}
}

// TestFargateOnDemandRates_ArchitectureAliases verifies architecture
// normalization and that an unknown architecture is not priced.
func TestFargateOnDemandRates_ArchitectureAliases(t *testing.T) {
	c, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	x86, ok := c.FargateOnDemandRates("")
	if !ok || x86.VCPUPerHour <= 0 || x86.GiBPerHour <= 0 {
		t.Fatalf("default architecture rates = %+v, %v", x86, ok)
	}
	for _, alias := range []string{"x86_64", "X86_64", "amd64", " x86 "} {
		if got, found := c.FargateOnDemandRates(alias); !found || got != x86 {
			t.Errorf("FargateOnDemandRates(%q) = %+v, %v; want %+v", alias, got, found, x86)
		}
	}
	arm, ok := c.FargateOnDemandRates("arm64")
	if !ok || arm.VCPUPerHour <= 0 || arm.GiBPerHour <= 0 {
		t.Fatalf("arm64 rates = %+v, %v", arm, ok)
	}
	if got, found := c.FargateOnDemandRates("ARM"); !found || got != arm {
		t.Errorf("FargateOnDemandRates(ARM) = %+v, %v; want %+v", got, found, arm)
	}
	if got, found := c.FargateOnDemandRates("riscv"); found {
		t.Errorf("unknown architecture priced: %+v", got)
	}
}

// TestParseFargatePricing_WindowsAndEphemeralStorage verifies the Windows
// compute, Windows license, and ephemeral storage rates are parsed from exact
// usage types, that the ephemeral rate needs the GB-Hours unit, and that an
// incomplete Windows set is not served.
func TestParseFargatePricing_WindowsAndEphemeralStorage(t *testing.T) {
	raw := buildPriceList(t, "AmazonECS", []fixtureSKU{
		fargateSKU("WINV", "USE1-Fargate-Windows-vCPU-Hours:perCPU", "hours", "0.0465520000"),
		fargateSKU("WINM", "USE1-Fargate-Windows-GB-Hours", "hours", "0.0051117500"),
		fargateSKU("WINOS", "USE1-Fargate-Windows-OS-Hours:perCPU", "hours", "0.0460000000"),
		fargateSKU("EPH", "USE1-Fargate-EphemeralStorage-GB-Hours", "GB-Hours", "0.0001110000"),
		fargateSKU("X86V", "USE1-Fargate-vCPU-Hours:perCPU", "hours", "0.04048"),
	})

	c := newFargateTestClient()
	assertParsedRegion(t, c.parseFargatePricing, raw)
	want := FargateWindowsRates{VCPUPerHour: 0.046552, GiBPerHour: 0.00511175, OSPerVCPUHour: 0.046}
	if !c.fargateWindowsFound || c.fargateWindows != want {
		t.Errorf("windows rates = %+v (found=%v), want %+v", c.fargateWindows, c.fargateWindowsFound, want)
	}
	if c.fargateEphemeralPerGiBHour != 0.000111 {
		t.Errorf("ephemeral rate = %v, want 0.000111", c.fargateEphemeralPerGiBHour)
	}

	partial := buildPriceList(t, "AmazonECS", []fixtureSKU{
		fargateSKU("WINV", "USE1-Fargate-Windows-vCPU-Hours:perCPU", "hours", "0.046552"),
		fargateSKU("WINM", "USE1-Fargate-Windows-GB-Hours", "hours", "0.00511175"),
		fargateSKU("EPH", "USE1-Fargate-EphemeralStorage-GB-Hours", "hours", "0.000111"),
	})
	c = newFargateTestClient()
	assertParsedRegion(t, c.parseFargatePricing, partial)
	if c.fargateWindowsFound {
		t.Errorf("windows rates served without the OS license rate: %+v", c.fargateWindows)
	}
	if c.fargateEphemeralPerGiBHour != 0 {
		t.Errorf("ephemeral rate accepted with unit hours: %v", c.fargateEphemeralPerGiBHour)
	}
}

// TestFargateWindowsAndEphemeralLookups verifies the public lookups return
// positive rates from the embedded data.
func TestFargateWindowsAndEphemeralLookups(t *testing.T) {
	c, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	win, ok := c.FargateWindowsOnDemandRates()
	if !ok || win.VCPUPerHour <= 0 || win.GiBPerHour <= 0 || win.OSPerVCPUHour <= 0 {
		t.Errorf("windows rates = %+v, found=%v", win, ok)
	}
	eph, ok := c.FargateEphemeralStoragePerGiBHour()
	if !ok || eph <= 0 {
		t.Errorf("ephemeral rate = %v, found=%v", eph, ok)
	}
}
