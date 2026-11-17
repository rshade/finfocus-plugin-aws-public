package router

import (
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// TestChildProjectedCost_PreservesMetadata verifies the router returns the
// child GetProjectedCost response without copying or stripping metadata.
// pricing_as_of stamped by a region binary therefore reaches the host.
func TestChildProjectedCost_PreservesMetadata(t *testing.T) {
	t.Parallel()
	in := &pbc.GetProjectedCostResponse{
		Currency: "USD",
		Metadata: map[string]string{
			"pricing_as_of":    "2026-09-11T12:45:44Z",
			"pricing_version":  "20260911124544",
			"defaults_applied": "size=8",
			"estimate_quality": "medium",
		},
	}
	out, err := childProjectedCost(in, nil)
	if err != nil {
		t.Fatalf("childProjectedCost() error = %v", err)
	}
	if out != in {
		t.Fatal("child response was copied")
	}
	if out.GetMetadata()["pricing_as_of"] != "2026-09-11T12:45:44Z" {
		t.Errorf("pricing_as_of = %q", out.GetMetadata()["pricing_as_of"])
	}
	if out.GetMetadata()["defaults_applied"] != "size=8" {
		t.Errorf("defaults_applied = %q", out.GetMetadata()["defaults_applied"])
	}
}
