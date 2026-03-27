package plugin

import (
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// TestClassification_ASG verifies the ASG classification entry exists with correct growth type.
func TestClassification_ASG(t *testing.T) {
	classification, ok := GetServiceClassification("aws:autoscaling:autoScalingGroup")
	if !ok {
		t.Fatal("ASG classification entry not found")
	}
	if classification.GrowthType != pbc.GrowthType_GROWTH_TYPE_NONE {
		t.Errorf("GrowthType = %v, want GROWTH_TYPE_NONE", classification.GrowthType)
	}
	if !classification.AffectedByDevMode {
		t.Error("AffectedByDevMode should be true for ASG")
	}
	if classification.Relationship != RelationshipWithin {
		t.Errorf("Relationship = %q, want %q", classification.Relationship, RelationshipWithin)
	}
}
