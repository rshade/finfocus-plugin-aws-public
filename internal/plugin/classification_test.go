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
	if classification.ParentType != "aws:ec2:vpc:Vpc" {
		t.Errorf("ParentType = %q, want %q", classification.ParentType, "aws:ec2:vpc:Vpc")
	}
	expectedTagKeys := []string{"vpc_id"}
	if len(classification.ParentTagKeys) != len(expectedTagKeys) {
		t.Fatalf("ParentTagKeys length = %d, want %d", len(classification.ParentTagKeys), len(expectedTagKeys))
	}
	for i, key := range expectedTagKeys {
		if classification.ParentTagKeys[i] != key {
			t.Errorf("ParentTagKeys[%d] = %q, want %q", i, classification.ParentTagKeys[i], key)
		}
	}
}

// TestClassification_ASG_PulumiFormat verifies the Pulumi-format alias resolves to the same classification.
func TestClassification_ASG_PulumiFormat(t *testing.T) {
	classification, ok := GetServiceClassification("aws:autoscaling/group:Group")
	if !ok {
		t.Fatal("ASG classification entry not found for Pulumi format key")
	}
	if classification.GrowthType != pbc.GrowthType_GROWTH_TYPE_NONE {
		t.Errorf("GrowthType = %v, want GROWTH_TYPE_NONE", classification.GrowthType)
	}
	if !classification.AffectedByDevMode {
		t.Error("AffectedByDevMode should be true for ASG (Pulumi format)")
	}
}
