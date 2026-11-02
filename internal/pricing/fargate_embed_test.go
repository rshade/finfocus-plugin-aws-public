//go:build region_use1 || region_usw1 || region_usw2 || region_govw1 || region_gove1 || region_euw1 || region_apse1 || region_apse2 || region_apne1 || region_aps1 || region_cac1 || region_sae1

package pricing

import (
	"testing"

	"github.com/rs/zerolog"
)

// TestEmbeddedData_FargateRates verifies the embedded AmazonECS offer yields
// positive Linux x86_64 and arm64 Fargate rates in the region binary.
func TestEmbeddedData_FargateRates(t *testing.T) {
	c, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	for _, arch := range []string{FargateArchX86, FargateArchARM} {
		r, ok := c.FargateOnDemandRates(arch)
		if !ok || r.VCPUPerHour <= 0 || r.GiBPerHour <= 0 {
			t.Errorf("%s Fargate rates = %+v, found=%v; want positive vCPU and GiB rates", arch, r, ok)
		}
	}
}
