package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// parseLastLogEntry parses the last non-empty JSON line from a log buffer.
// This handles cases where the logger emits multiple entries during a test.
// Returns the parsed map and any error encountered.
func parseLastLogEntry(buf *bytes.Buffer) (map[string]interface{}, error) {
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 || lines[len(lines)-1] == "" {
		return nil, nil
	}
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &logEntry); err != nil {
		return nil, err
	}
	return logEntry, nil
}

// TestGetRecommendations_NilRequest verifies FR-009: Return ERROR_CODE_INVALID_RESOURCE
// when request is nil.
func TestGetRecommendations_NilRequest(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	_, err := plugin.GetRecommendations(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for nil request")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Code = %v, want %v", st.Code(), codes.InvalidArgument)
	}

	// Check for ErrorCode in details
	var foundErrorCode bool
	for _, detail := range st.Details() {
		if errDetail, ok := detail.(*pbc.ErrorDetail); ok {
			if errDetail.Code == pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE {
				foundErrorCode = true
			}
		}
	}

	if !foundErrorCode {
		t.Error("Expected ERROR_CODE_INVALID_RESOURCE in error details")
	}
}

// TestGetRecommendations_EmptyRequest verifies FR-008: Return empty recommendations
// list (not error) when request is valid but has no filter context.
func TestGetRecommendations_EmptyRequest(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{}
	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if len(resp.Recommendations) != 0 {
		t.Errorf("Expected empty recommendations list, got %d", len(resp.Recommendations))
	}

	if resp.Summary == nil {
		t.Error("Expected non-nil summary")
	}
}

// TestGetRecommendations_TraceIDInLogs verifies FR-010: Include trace_id in all log entries.
func TestGetRecommendations_TraceIDInLogs(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	expectedTraceID := "test-recommendations-trace-12345"
	md := metadata.New(map[string]string{
		pluginsdk.TraceIDMetadataKey: expectedTraceID,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := &pbc.GetRecommendationsRequest{}
	_, err := plugin.GetRecommendations(ctx, req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Verify trace_id in log output
	logEntry, err := parseLastLogEntry(&logBuf)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nRaw: %s", err, logBuf.String())
	}

	traceID, ok := logEntry["trace_id"].(string)
	if !ok {
		t.Fatal("trace_id not found in log entry")
	}

	if traceID != expectedTraceID {
		t.Errorf("trace_id = %q, want %q", traceID, expectedTraceID)
	}

	// Verify operation field
	operation, ok := logEntry["operation"].(string)
	if !ok || operation != "GetRecommendations" {
		t.Errorf("operation = %q, want %q", operation, "GetRecommendations")
	}
}

// TestGetRecommendations_EC2WithFilter verifies that GetRecommendations returns
// EC2 recommendations when provided with a proper filter containing SKU and resource type.
func TestGetRecommendations_EC2WithFilter(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// t2.medium is older, t3.medium is newer and cheaper
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		Filter: &pbc.RecommendationFilter{
			ResourceType: "ec2",
			Sku:          "t2.medium",
			Region:       "us-east-1",
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	// Should have at least one recommendation (generation upgrade t2->t3)
	if len(resp.Recommendations) == 0 {
		t.Error("Expected at least one recommendation for t2.medium")
	}

	// Verify a generation upgrade recommendation exists
	var foundGenUpgrade bool
	for _, rec := range resp.Recommendations {
		if rec.GetModify() != nil && rec.GetModify().ModificationType == modTypeGenUpgrade {
			foundGenUpgrade = true
			// Verify the recommended config
			if rec.GetModify().RecommendedConfig["instance_type"] != "t3.medium" {
				t.Errorf("Expected recommended instance_type t3.medium, got %s",
					rec.GetModify().RecommendedConfig["instance_type"])
			}
		}
	}

	if !foundGenUpgrade {
		t.Error("Expected generation upgrade recommendation")
	}

	// Summary should reflect the recommendations
	if resp.Summary == nil {
		t.Error("Expected non-nil summary")
	}
}

// TestGetRecommendations_EBSWithFilter verifies that GetRecommendations returns
// EBS recommendations when provided with a proper filter for gp2 volumes.
func TestGetRecommendations_EBSWithFilter(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		Filter: &pbc.RecommendationFilter{
			ResourceType: "ebs",
			Sku:          "gp2",
			Region:       "us-east-1",
			Tags:         map[string]string{"size": "100"},
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	// Should have exactly one recommendation (gp2->gp3)
	if len(resp.Recommendations) != 1 {
		t.Errorf("Expected 1 recommendation for gp2 volume, got %d", len(resp.Recommendations))
	}

	if len(resp.Recommendations) > 0 {
		rec := resp.Recommendations[0]
		if rec.GetModify() == nil {
			t.Fatal("Expected modify action")
		}
		if rec.GetModify().ModificationType != modTypeVolumeUpgrade {
			t.Errorf("Expected modification type %s, got %s", modTypeVolumeUpgrade, rec.GetModify().ModificationType)
		}
		if rec.GetModify().RecommendedConfig["volume_type"] != "gp3" {
			t.Errorf("Expected recommended volume_type gp3, got %s",
				rec.GetModify().RecommendedConfig["volume_type"])
		}
	}
}

// TestGetRecommendations_DefaultRegion verifies that when filter.Region is empty,
// the plugin uses its own region for recommendations.
func TestGetRecommendations_DefaultRegion(t *testing.T) {
	mock := newMockPricingClient("us-west-2", "USD")
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6i.large/Linux/Shared"] = 0.096
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-west-2", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		Filter: &pbc.RecommendationFilter{
			ResourceType: "ec2",
			Sku:          "m5.large",
			// Region intentionally empty - should use plugin's region
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	// Should have recommendations (m5->m6i generation upgrade)
	if len(resp.Recommendations) == 0 {
		t.Error("Expected recommendations when region defaults to plugin region")
	}

	// Verify recommendations use the plugin's region
	for _, rec := range resp.Recommendations {
		if rec.Resource != nil && rec.Resource.Region != "us-west-2" {
			t.Errorf("Expected region us-west-2, got %s", rec.Resource.Region)
		}
	}
}

// TestGetRecommendations_PulumiResourceType verifies that Pulumi-format resource types
// (e.g., "aws:ec2/instance:Instance") are correctly handled.
func TestGetRecommendations_PulumiResourceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["c5.xlarge/Linux/Shared"] = 0.17
	mock.ec2Prices["c6i.xlarge/Linux/Shared"] = 0.17
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		Filter: &pbc.RecommendationFilter{
			ResourceType: "aws:ec2/instance:Instance", // Pulumi format
			Sku:          "c5.xlarge",
			Region:       "us-east-1",
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Should have at least one recommendation
	if len(resp.Recommendations) == 0 {
		t.Error("Expected recommendations for Pulumi-format resource type")
	}
}

// TestGenerateEC2Recommendations_GenerationUpgrade verifies FR-002: Return generation
// upgrade recommendations for EC2 instances when newer generations offer same or lower price.
func TestGenerateEC2Recommendations_GenerationUpgrade(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// t2.medium is older, t3.medium is newer and cheaper
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	recs := plugin.generateEC2Recommendations("t2.medium", "us-east-1")

	// Should have at least one recommendation (generation upgrade)
	var genUpgradeRec *pbc.Recommendation
	for _, rec := range recs {
		if rec.GetModify() != nil && rec.GetModify().ModificationType == modTypeGenUpgrade {
			genUpgradeRec = rec
			break
		}
	}

	if genUpgradeRec == nil {
		t.Fatal("Expected generation upgrade recommendation")
	}

	// Verify recommendation details
	if genUpgradeRec.Category != pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST {
		t.Errorf("Category = %v, want COST", genUpgradeRec.Category)
	}

	if genUpgradeRec.ActionType != pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY {
		t.Errorf("ActionType = %v, want MODIFY", genUpgradeRec.ActionType)
	}

	// Verify resource info
	if genUpgradeRec.Resource == nil {
		t.Fatal("Expected Resource to be set")
	}
	if genUpgradeRec.Resource.Provider != "aws" {
		t.Errorf("Provider = %q, want %q", genUpgradeRec.Resource.Provider, "aws")
	}
	if genUpgradeRec.Resource.ResourceType != "ec2" {
		t.Errorf("ResourceType = %q, want %q", genUpgradeRec.Resource.ResourceType, "ec2")
	}

	// Verify modify action
	modify := genUpgradeRec.GetModify()
	if modify == nil {
		t.Fatal("Expected Modify action")
	}
	if modify.CurrentConfig["instance_type"] != "t2.medium" {
		t.Errorf("CurrentConfig[instance_type] = %q, want %q", modify.CurrentConfig["instance_type"], "t2.medium")
	}
	if modify.RecommendedConfig["instance_type"] != "t3.medium" {
		t.Errorf("RecommendedConfig[instance_type] = %q, want %q", modify.RecommendedConfig["instance_type"], "t3.medium")
	}

	// FR-006: Verify confidence level is 0.9 (high) for generation upgrades
	if genUpgradeRec.ConfidenceScore == nil || *genUpgradeRec.ConfidenceScore != confidenceHigh {
		t.Errorf("ConfidenceScore = %v, want %v", genUpgradeRec.ConfidenceScore, confidenceHigh)
	}

	// Verify impact calculations (FR-005: 730 hours/month)
	if genUpgradeRec.Impact == nil {
		t.Fatal("Expected Impact to be set")
	}
	expectedCurrentMonthly := 0.0464 * hoursPerMonth  // ~33.87
	expectedNewMonthly := 0.0416 * hoursPerMonth      // ~30.37
	expectedSavings := expectedCurrentMonthly - expectedNewMonthly

	if genUpgradeRec.Impact.CurrentCost != expectedCurrentMonthly {
		t.Errorf("CurrentCost = %v, want %v", genUpgradeRec.Impact.CurrentCost, expectedCurrentMonthly)
	}
	if genUpgradeRec.Impact.ProjectedCost != expectedNewMonthly {
		t.Errorf("ProjectedCost = %v, want %v", genUpgradeRec.Impact.ProjectedCost, expectedNewMonthly)
	}
	if genUpgradeRec.Impact.EstimatedSavings != expectedSavings {
		t.Errorf("EstimatedSavings = %v, want %v", genUpgradeRec.Impact.EstimatedSavings, expectedSavings)
	}

	// Verify source
	if genUpgradeRec.Source != sourceAWSPublic {
		t.Errorf("Source = %q, want %q", genUpgradeRec.Source, sourceAWSPublic)
	}
}

// TestGenerateEC2Recommendations_NoUpgradeWhenNewIsExpensive verifies FR-011:
// Only recommend when new price <= current price.
func TestGenerateEC2Recommendations_NoUpgradeWhenNewIsExpensive(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// Make the newer generation MORE expensive (edge case)
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0400
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0500 // More expensive
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	recs := plugin.generateEC2Recommendations("t2.medium", "us-east-1")

	// Should NOT have generation upgrade recommendation
	for _, rec := range recs {
		if rec.GetModify() != nil && rec.GetModify().ModificationType == modTypeGenUpgrade {
			t.Error("Should not recommend upgrade when new generation is more expensive")
		}
	}
}

// TestGenerateEC2Recommendations_NoUpgradeWhenPricingMissing verifies that we don't
// recommend when pricing data is unavailable.
func TestGenerateEC2Recommendations_NoUpgradeWhenPricingMissing(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// Only set current price, not the new generation price
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	// t3.medium price NOT set
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	recs := plugin.generateEC2Recommendations("t2.medium", "us-east-1")

	// Should NOT have generation upgrade recommendation
	for _, rec := range recs {
		if rec.GetModify() != nil && rec.GetModify().ModificationType == modTypeGenUpgrade {
			t.Error("Should not recommend upgrade when new generation pricing is missing")
		}
	}
}

// TestGenerateEC2Recommendations_LatestGeneration verifies no recommendation when
// instance is already the latest generation.
func TestGenerateEC2Recommendations_LatestGeneration(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// t3a is end of upgrade chain, no recommendation expected
	mock.ec2Prices["t3a.micro/Linux/Shared"] = 0.0094
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	rec := plugin.getGenerationUpgradeRecommendation("t3a.micro", "us-east-1")

	if rec != nil {
		t.Error("Expected no recommendation for latest generation instance")
	}
}

// TestGenerateEC2Recommendations_GravitonMigration verifies FR-003: Return Graviton/ARM
// migration recommendations for compatible x86 instance families.
func TestGenerateEC2Recommendations_GravitonMigration(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6g.large/Linux/Shared"] = 0.077 // ~20% cheaper
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	recs := plugin.generateEC2Recommendations("m5.large", "us-east-1")

	// Should have Graviton recommendation
	var gravitonRec *pbc.Recommendation
	for _, rec := range recs {
		if rec.GetModify() != nil && rec.GetModify().ModificationType == modTypeGraviton {
			gravitonRec = rec
			break
		}
	}

	if gravitonRec == nil {
		t.Fatal("Expected Graviton migration recommendation")
	}

	// FR-007: Verify confidence level is 0.7 (medium) for Graviton recommendations
	if gravitonRec.ConfidenceScore == nil || *gravitonRec.ConfidenceScore != confidenceMedium {
		t.Errorf("ConfidenceScore = %v, want %v", gravitonRec.ConfidenceScore, confidenceMedium)
	}

	// Verify architecture change in config
	modify := gravitonRec.GetModify()
	if modify.CurrentConfig["architecture"] != "x86_64" {
		t.Errorf("CurrentConfig[architecture] = %q, want %q", modify.CurrentConfig["architecture"], "x86_64")
	}
	if modify.RecommendedConfig["architecture"] != "arm64" {
		t.Errorf("RecommendedConfig[architecture] = %q, want %q", modify.RecommendedConfig["architecture"], "arm64")
	}

	// FR-012: Verify metadata includes architecture warning
	if gravitonRec.Metadata == nil {
		t.Fatal("Expected Metadata to be set")
	}
	if _, ok := gravitonRec.Metadata["architecture_change"]; !ok {
		t.Error("Expected architecture_change in metadata")
	}
	if _, ok := gravitonRec.Metadata["requires_validation"]; !ok {
		t.Error("Expected requires_validation in metadata")
	}

	// Verify priority is LOW (Graviton requires validation)
	if gravitonRec.Priority != pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_LOW {
		t.Errorf("Priority = %v, want LOW", gravitonRec.Priority)
	}
}

// TestGenerateEC2Recommendations_BothUpgrades verifies that both generation upgrade
// AND Graviton recommendations can be returned when applicable.
func TestGenerateEC2Recommendations_BothUpgrades(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// m5.large can upgrade to m6i and also migrate to m6g
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6i.large/Linux/Shared"] = 0.096 // Same price
	mock.ec2Prices["m6g.large/Linux/Shared"] = 0.077  // Cheaper (Graviton)
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	recs := plugin.generateEC2Recommendations("m5.large", "us-east-1")

	var hasGenUpgrade, hasGraviton bool
	for _, rec := range recs {
		if rec.GetModify() == nil {
			continue
		}
		switch rec.GetModify().ModificationType {
		case modTypeGenUpgrade:
			hasGenUpgrade = true
		case modTypeGraviton:
			hasGraviton = true
		}
	}

	if !hasGenUpgrade {
		t.Error("Expected generation upgrade recommendation")
	}
	if !hasGraviton {
		t.Error("Expected Graviton recommendation")
	}
}

// TestGetEBSRecommendations_Gp2ToGp3 verifies FR-004: Return gp2→gp3 volume type
// upgrade recommendations for EBS volumes.
func TestGetEBSRecommendations_Gp2ToGp3(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp2"] = 0.10 // per GB-month
	mock.ebsPrices["gp3"] = 0.08 // 20% cheaper
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	tags := map[string]string{"size": "500"}
	recs := plugin.getEBSRecommendations("gp2", "us-east-1", tags)

	if len(recs) == 0 {
		t.Fatal("Expected EBS recommendation")
	}

	rec := recs[0]

	// Verify category
	if rec.Category != pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST {
		t.Errorf("Category = %v, want COST", rec.Category)
	}

	// Verify resource type
	if rec.Resource == nil || rec.Resource.ResourceType != "ebs" {
		t.Error("Expected ResourceType = 'ebs'")
	}

	// FR-006: Verify confidence level is 0.9 (high) for EBS volume changes
	if rec.ConfidenceScore == nil || *rec.ConfidenceScore != confidenceHigh {
		t.Errorf("ConfidenceScore = %v, want %v", rec.ConfidenceScore, confidenceHigh)
	}

	// Verify modify action
	modify := rec.GetModify()
	if modify == nil {
		t.Fatal("Expected Modify action")
	}
	if modify.ModificationType != modTypeVolumeUpgrade {
		t.Errorf("ModificationType = %q, want %q", modify.ModificationType, modTypeVolumeUpgrade)
	}
	if modify.CurrentConfig["volume_type"] != "gp2" {
		t.Errorf("CurrentConfig[volume_type] = %q, want %q", modify.CurrentConfig["volume_type"], "gp2")
	}
	if modify.RecommendedConfig["volume_type"] != "gp3" {
		t.Errorf("RecommendedConfig[volume_type] = %q, want %q", modify.RecommendedConfig["volume_type"], "gp3")
	}

	// Verify impact calculations
	expectedCurrentMonthly := 0.10 * 500 // $50
	expectedGp3Monthly := 0.08 * 500     // $40
	expectedSavings := 10.0              // $10

	if rec.Impact.CurrentCost != expectedCurrentMonthly {
		t.Errorf("CurrentCost = %v, want %v", rec.Impact.CurrentCost, expectedCurrentMonthly)
	}
	if rec.Impact.ProjectedCost != expectedGp3Monthly {
		t.Errorf("ProjectedCost = %v, want %v", rec.Impact.ProjectedCost, expectedGp3Monthly)
	}
	if rec.Impact.EstimatedSavings != expectedSavings {
		t.Errorf("EstimatedSavings = %v, want %v", rec.Impact.EstimatedSavings, expectedSavings)
	}

	// FR-012: Verify metadata includes performance info
	if rec.Metadata == nil {
		t.Fatal("Expected Metadata to be set")
	}
	if _, ok := rec.Metadata["baseline_iops"]; !ok {
		t.Error("Expected baseline_iops in metadata")
	}
	if _, ok := rec.Metadata["baseline_throughput"]; !ok {
		t.Error("Expected baseline_throughput in metadata")
	}
}

// TestGetEBSRecommendations_DefaultSize verifies default size of 100GB is used
// when size is not specified in tags (edge case from spec.md).
func TestGetEBSRecommendations_DefaultSize(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// No size tag
	tags := map[string]string{}
	recs := plugin.getEBSRecommendations("gp2", "us-east-1", tags)

	if len(recs) == 0 {
		t.Fatal("Expected EBS recommendation")
	}

	rec := recs[0]
	modify := rec.GetModify()
	if modify.CurrentConfig["size_gb"] != "100" {
		t.Errorf("CurrentConfig[size_gb] = %q, want %q (default)", modify.CurrentConfig["size_gb"], "100")
	}

	// Verify impact calculations use default 100GB
	expectedCurrentMonthly := 0.10 * 100 // $10
	if rec.Impact.CurrentCost != expectedCurrentMonthly {
		t.Errorf("CurrentCost = %v, want %v", rec.Impact.CurrentCost, expectedCurrentMonthly)
	}
}

// TestGetEBSRecommendations_VolumeSizeTag verifies alternative "volume_size" tag is supported.
func TestGetEBSRecommendations_VolumeSizeTag(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Use "volume_size" instead of "size"
	tags := map[string]string{"volume_size": "200"}
	recs := plugin.getEBSRecommendations("gp2", "us-east-1", tags)

	if len(recs) == 0 {
		t.Fatal("Expected EBS recommendation")
	}

	rec := recs[0]
	modify := rec.GetModify()
	if modify.CurrentConfig["size_gb"] != "200" {
		t.Errorf("CurrentConfig[size_gb] = %q, want %q", modify.CurrentConfig["size_gb"], "200")
	}
}

// TestGetEBSRecommendations_NoRecommendationForGp3 verifies no recommendation
// when volume is already gp3.
func TestGetEBSRecommendations_NoRecommendationForGp3(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	recs := plugin.getEBSRecommendations("gp3", "us-east-1", nil)

	if len(recs) != 0 {
		t.Errorf("Expected no recommendations for gp3 volume, got %d", len(recs))
	}
}

// TestGetEBSRecommendations_NoRecommendationForIo1 verifies no recommendation
// for io1/io2 volumes (out of scope per spec.md).
func TestGetEBSRecommendations_NoRecommendationForIo1(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	for _, volumeType := range []string{"io1", "io2", "st1", "sc1"} {
		recs := plugin.getEBSRecommendations(volumeType, "us-east-1", nil)
		if len(recs) != 0 {
			t.Errorf("Expected no recommendations for %s volume, got %d", volumeType, len(recs))
		}
	}
}

// TestGenerateEC2Recommendations_InvalidInstanceType verifies no recommendations
// for invalid instance type formats.
func TestGenerateEC2Recommendations_InvalidInstanceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	invalidTypes := []string{"", "invalid", "t2", ".medium", "t2.", "..."}

	for _, instanceType := range invalidTypes {
		recs := plugin.generateEC2Recommendations(instanceType, "us-east-1")
		if len(recs) != 0 {
			t.Errorf("Expected no recommendations for invalid instance type %q, got %d", instanceType, len(recs))
		}
	}
}

// TestRecommendationHasUniqueID verifies each recommendation has a unique UUID.
func TestRecommendationHasUniqueID(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6i.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6g.large/Linux/Shared"] = 0.077
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	recs := plugin.generateEC2Recommendations("m5.large", "us-east-1")

	if len(recs) < 2 {
		t.Skip("Need at least 2 recommendations to test uniqueness")
	}

	ids := make(map[string]bool)
	for _, rec := range recs {
		if rec.Id == "" {
			t.Error("Recommendation ID should not be empty")
		}
		if ids[rec.Id] {
			t.Errorf("Duplicate recommendation ID: %s", rec.Id)
		}
		ids[rec.Id] = true

		// Verify UUID format (36 chars with hyphens)
		if len(rec.Id) != 36 {
			t.Errorf("Recommendation ID %q should be UUID format (36 chars)", rec.Id)
		}
	}
}

// TestGetRecommendations_LogsContainDurationMs verifies log entries include duration_ms.
func TestGetRecommendations_LogsContainDurationMs(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{}
	_, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	logEntry, err := parseLastLogEntry(&logBuf)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nRaw: %s", err, logBuf.String())
	}

	durationMs, ok := logEntry["duration_ms"].(float64)
	if !ok {
		t.Fatal("duration_ms not found in log entry")
	}

	if durationMs < 0 {
		t.Errorf("duration_ms = %v, should be non-negative", durationMs)
	}
}

// TestGetRecommendations_LogsContainRecommendationCount verifies log entries include
// the count of recommendations generated.
func TestGetRecommendations_LogsContainRecommendationCount(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{}
	_, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	logEntry, err := parseLastLogEntry(&logBuf)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nRaw: %s", err, logBuf.String())
	}

	_, ok := logEntry["recommendation_count"].(float64)
	if !ok {
		t.Fatal("recommendation_count not found in log entry")
	}
}

// TestGetRecommendations_ErrorLogsContainErrorCode verifies FR-009 error logging.
func TestGetRecommendations_ErrorLogsContainErrorCode(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(&logBuf).Level(zerolog.ErrorLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	_, err := plugin.GetRecommendations(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for nil request")
	}

	logEntry, parseErr := parseLastLogEntry(&logBuf)
	if parseErr != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nRaw: %s", parseErr, logBuf.String())
	}

	errorCode, ok := logEntry["error_code"].(string)
	if !ok || errorCode == "" {
		t.Error("error_code field should be present in error log")
	}

	if !strings.Contains(errorCode, "INVALID_RESOURCE") {
		t.Errorf("error_code = %q, should contain INVALID_RESOURCE", errorCode)
	}
}

// TestGetRecommendations_Batch verifies US1: Batch resource analysis.
// It sends a request with multiple resources in TargetResources and verifies
// that recommendations are generated for each supported resource.
func TestGetRecommendations_Batch(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// Setup pricing for test resources
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6i.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6g.large/Linux/Shared"] = 0.077

	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "us-east-1", Provider: "aws"}, // Upgrade
			{ResourceType: "aws:ebs:Volume", Sku: "gp2", Region: "us-east-1", Provider: "aws", Tags: map[string]string{"size": "100"}}, // Upgrade
			{ResourceType: "aws:ec2:Instance", Sku: "m5.large", Region: "us-east-1", Provider: "aws"}, // Upgrade + Graviton
			{ResourceType: "aws:ec2:Instance", Sku: "t3.medium", Region: "us-east-1", Provider: "aws"}, // No upgrade (already new)
			{ResourceType: "aws:ebs:Volume", Sku: "gp3", Region: "us-east-1", Provider: "aws"}, // No upgrade
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	// Expected recommendations:
	// 1. t2.medium -> t3.medium (1 rec)
	// 2. gp2 -> gp3 (1 rec)
	// 3. m5.large -> m6i.large (1 rec) + m6g.large (1 rec)
	// 4. t3.medium -> None
	// 5. gp3 -> None
	// Total: 4 recommendations
	expectedCount := 4
	if len(resp.Recommendations) != expectedCount {
		t.Errorf("Expected %d recommendations, got %d", expectedCount, len(resp.Recommendations))
	}
}

// TestGetRecommendations_FilteredBatch verifies US2: Filtered batch analysis.
// It sends a batch request with a filter and verifies that only matching resources
// are included in the recommendations.
func TestGetRecommendations_FilteredBatch(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416

	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "us-east-1", Provider: "aws"},
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "us-west-2", Provider: "aws"},
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "eu-west-1", Provider: "aws"},
		},
		Filter: &pbc.RecommendationFilter{
			Region: "us-east-1", // Only match the first resource
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Only the us-east-1 resource should match the filter
	// t2.medium -> t3.medium = 1 recommendation
	expectedCount := 1
	if len(resp.Recommendations) != expectedCount {
		t.Errorf("Expected %d recommendations for filtered batch, got %d", expectedCount, len(resp.Recommendations))
	}

	// Verify the recommendation is for the correct region
	if len(resp.Recommendations) > 0 {
		rec := resp.Recommendations[0]
		if rec.Resource == nil || rec.Resource.Region != "us-east-1" {
			t.Errorf("Expected recommendation for us-east-1, got %v", rec.Resource)
		}
	}
}

// TestGetRecommendations_FilteredBatch_ByResourceType verifies filtering by resource type.
func TestGetRecommendations_FilteredBatch_ByResourceType(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08

	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "us-east-1", Provider: "aws"},
			{ResourceType: "aws:ebs:Volume", Sku: "gp2", Region: "us-east-1", Provider: "aws", Tags: map[string]string{"size": "100"}},
		},
		Filter: &pbc.RecommendationFilter{
			ResourceType: "aws:ebs:Volume", // Only match EBS volumes
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Only the EBS volume should match - gp2 -> gp3 = 1 recommendation
	expectedCount := 1
	if len(resp.Recommendations) != expectedCount {
		t.Errorf("Expected %d recommendations for filtered batch, got %d", expectedCount, len(resp.Recommendations))
	}

	// Verify the recommendation is for EBS
	if len(resp.Recommendations) > 0 {
		rec := resp.Recommendations[0]
		if rec.Resource == nil || rec.Resource.ResourceType != "ebs" {
			t.Errorf("Expected recommendation for ebs, got %v", rec.Resource)
		}
	}
}

// TestGetRecommendations_Legacy verifies US3: Legacy single-resource fallback.
// When TargetResources is empty but Filter has Sku and ResourceType,
// the plugin should construct a single-item scope from Filter fields.
func TestGetRecommendations_Legacy(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416

	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Legacy mode: no TargetResources, only Filter with Sku
	req := &pbc.GetRecommendationsRequest{
		// TargetResources intentionally empty
		Filter: &pbc.RecommendationFilter{
			ResourceType: "ec2",
			Sku:          "t2.medium",
			Region:       "us-east-1",
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Should generate recommendations from the filter-based single-item scope
	// t2.medium -> t3.medium = 1 recommendation
	expectedCount := 1
	if len(resp.Recommendations) != expectedCount {
		t.Errorf("Expected %d recommendations for legacy mode, got %d", expectedCount, len(resp.Recommendations))
	}
}

// TestGetRecommendations_LegacyWithDefaultRegion verifies legacy mode uses plugin region.
func TestGetRecommendations_LegacyWithDefaultRegion(t *testing.T) {
	mock := newMockPricingClient("us-west-2", "USD")
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6i.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6g.large/Linux/Shared"] = 0.077

	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-west-2", mock, logger)

	// Legacy mode without explicit region - should use plugin's region
	req := &pbc.GetRecommendationsRequest{
		Filter: &pbc.RecommendationFilter{
			ResourceType: "ec2",
			Sku:          "m5.large",
			// Region intentionally empty
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Should generate recommendations using plugin's region (us-west-2)
	// m5.large -> m6i.large (gen) + m6g.large (Graviton) = 2 recommendations
	expectedCount := 2
	if len(resp.Recommendations) != expectedCount {
		t.Errorf("Expected %d recommendations for legacy mode with default region, got %d", expectedCount, len(resp.Recommendations))
	}

	// Verify all recommendations use the plugin's region
	for _, rec := range resp.Recommendations {
		if rec.Resource == nil || rec.Resource.Region != "us-west-2" {
			t.Errorf("Expected recommendation for us-west-2, got %v", rec.Resource)
		}
	}
}

// TestGetRecommendations_SummaryLogging verifies summary logging per batch.
// Only one log entry should be generated per batch at INFO level.
func TestGetRecommendations_SummaryLogging(t *testing.T) {
	var logBuf bytes.Buffer
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08

	logger := zerolog.New(&logBuf).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "us-east-1", Provider: "aws"},
			{ResourceType: "aws:ebs:Volume", Sku: "gp2", Region: "us-east-1", Provider: "aws", Tags: map[string]string{"size": "100"}},
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "us-east-1", Provider: "aws"},
		},
	}

	_, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Parse all log entries
	lines := strings.Split(strings.TrimSpace(logBuf.String()), "\n")
	infoLogs := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Fatalf("Failed to parse log line as JSON: %v\nLine: %s", err, line)
		}
		if level, ok := logEntry["level"].(string); ok && level == "info" {
			infoLogs++
		}
	}

	// Should have exactly 1 INFO log (the summary)
	if infoLogs != 1 {
		t.Errorf("Expected 1 INFO log entry (summary), got %d", infoLogs)
	}

	// Verify the summary log contains expected fields
	logEntry, err := parseLastLogEntry(&logBuf)
	if err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Check for summary fields
	if _, ok := logEntry["total_resources"]; !ok {
		t.Error("Summary log should contain total_resources")
	}
	if _, ok := logEntry["matched_resources"]; !ok {
		t.Error("Summary log should contain matched_resources")
	}
	if _, ok := logEntry["total_savings"]; !ok {
		t.Error("Summary log should contain total_savings")
	}
}

// TestGetRecommendations_EmptyBothModes verifies empty response for no context.
func TestGetRecommendations_EmptyBothModes(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// No TargetResources and no valid Filter
	req := &pbc.GetRecommendationsRequest{
		// TargetResources: empty
		// Filter: nil
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Should return empty recommendations
	if len(resp.Recommendations) != 0 {
		t.Errorf("Expected 0 recommendations for empty request, got %d", len(resp.Recommendations))
	}
}

// TestGetRecommendations_ProviderFilter verifies that non-AWS resources are skipped.
func TestGetRecommendations_ProviderFilter(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416

	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{ResourceType: "aws:ec2:Instance", Sku: "t2.medium", Region: "us-east-1", Provider: "aws"},
			{ResourceType: "gcp:compute:Instance", Sku: "n1-standard-1", Region: "us-central1", Provider: "gcp"}, // Should be skipped
			{ResourceType: "azure:vm:Instance", Sku: "Standard_B1s", Region: "eastus", Provider: "azure"},       // Should be skipped
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Only the AWS resource should be processed
	expectedCount := 1 // t2.medium -> t3.medium
	if len(resp.Recommendations) != expectedCount {
		t.Errorf("Expected %d recommendations (AWS only), got %d", expectedCount, len(resp.Recommendations))
	}
}

// TestGetRecommendations_BatchSizeLimit verifies that batch requests exceeding
// the maximum size of 100 resources are rejected with an appropriate error.
func TestGetRecommendations_BatchSizeLimit(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Create 101 resources (exceeds limit of 100)
	resources := make([]*pbc.ResourceDescriptor, 101)
	for i := range resources {
		resources[i] = &pbc.ResourceDescriptor{
			ResourceType: "aws:ec2:Instance",
			Sku:          "t3.micro",
			Region:       "us-east-1",
			Provider:     "aws",
		}
	}

	req := &pbc.GetRecommendationsRequest{
		TargetResources: resources,
	}

	_, err := plugin.GetRecommendations(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for batch size exceeding limit")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Code = %v, want %v", st.Code(), codes.InvalidArgument)
	}

	if !strings.Contains(st.Message(), "exceeds maximum") {
		t.Errorf("Message should mention exceeds maximum, got: %s", st.Message())
	}
}

// TestGetRecommendations_NativeIDPassthrough verifies the native ResourceDescriptor.Id
// field is passed through to Recommendation.Resource.Id.
//
// This test validates the ID passthrough feature (001-resourceid-passthrough):
//   - FR-001: Native Id field populated → use as Resource.Id
//   - FR-002: Multiple recommendations from same resource → identical Resource.Id
//   - FR-003: Empty native Id → fall back to tags["resource_id"]
//   - FR-004: tags["name"] correlation unchanged
//
// Test cases cover:
//   - Native ID populated (uses native ID)
//   - Empty native ID with tag fallback (uses tag)
//   - Whitespace-only native ID (treats as empty, uses tag)
//   - Native ID takes priority when both present
//   - Neither present (Resource.Id remains empty)
//   - Name tag correlation preserved (unchanged behavior)
func TestGetRecommendations_NativeIDPassthrough(t *testing.T) {
	tests := []struct {
		name           string
		nativeID       string
		tagResourceID  string
		tagName        string
		expectedID     string
		expectedName   string
	}{
		{
			name:          "native ID populated",
			nativeID:      "urn:pulumi:dev::myproject::aws:ec2/instance:Instance::web-server",
			tagResourceID: "",
			tagName:       "",
			expectedID:    "urn:pulumi:dev::myproject::aws:ec2/instance:Instance::web-server",
			expectedName:  "",
		},
		{
			name:          "empty native ID falls back to tag",
			nativeID:      "",
			tagResourceID: "legacy-resource-123",
			tagName:       "",
			expectedID:    "legacy-resource-123",
			expectedName:  "",
		},
		{
			name:          "whitespace native ID treated as empty",
			nativeID:      "   ",
			tagResourceID: "fallback-id",
			tagName:       "",
			expectedID:    "fallback-id",
			expectedName:  "",
		},
		{
			name:          "native ID takes priority over tag",
			nativeID:      "native-id",
			tagResourceID: "tag-id",
			tagName:       "",
			expectedID:    "native-id",
			expectedName:  "",
		},
		{
			name:          "neither present leaves ID empty",
			nativeID:      "",
			tagResourceID: "",
			tagName:       "",
			expectedID:    "",
			expectedName:  "",
		},
		{
			name:          "name tag correlation preserved",
			nativeID:      "resource-123",
			tagResourceID: "",
			tagName:       "MyWebServer",
			expectedID:    "resource-123",
			expectedName:  "MyWebServer",
		},
		{
			name:          "all fields populated",
			nativeID:      "native-urn",
			tagResourceID: "tag-resource-id",
			tagName:       "ServerName",
			expectedID:    "native-urn",
			expectedName:  "ServerName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockPricingClient("us-east-1", "USD")
			// Setup pricing so recommendations are generated
			mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
			mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416
			logger := zerolog.New(nil).Level(zerolog.InfoLevel)
			plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

			// Build tags map
			tags := make(map[string]string)
			if tt.tagResourceID != "" {
				tags["resource_id"] = tt.tagResourceID
			}
			if tt.tagName != "" {
				tags["name"] = tt.tagName
			}

			req := &pbc.GetRecommendationsRequest{
				TargetResources: []*pbc.ResourceDescriptor{
					{
						Id:           tt.nativeID,
						ResourceType: "aws:ec2:Instance",
						Sku:          "t2.medium",
						Region:       "us-east-1",
						Provider:     "aws",
						Tags:         tags,
					},
				},
			}

			resp, err := plugin.GetRecommendations(context.Background(), req)
			if err != nil {
				t.Fatalf("GetRecommendations() error: %v", err)
			}

			if len(resp.Recommendations) == 0 {
				t.Fatal("Expected at least one recommendation")
			}

			// Verify all recommendations have the expected Resource.Id and Resource.Name
			for i, rec := range resp.Recommendations {
				if rec.Resource == nil {
					t.Errorf("Recommendation[%d]: Resource is nil", i)
					continue
				}
				if rec.Resource.Id != tt.expectedID {
					t.Errorf("Recommendation[%d]: Resource.Id = %q, want %q",
						i, rec.Resource.Id, tt.expectedID)
				}
				if rec.Resource.Name != tt.expectedName {
					t.Errorf("Recommendation[%d]: Resource.Name = %q, want %q",
						i, rec.Resource.Name, tt.expectedName)
				}
			}
		})
	}
}

// TestGetRecommendations_MultipleRecsFromSameResource verifies FR-002:
// Multiple recommendations from the same resource have identical Resource.Id.
// This tests the invariant that all recommendations generated from a single
// resource descriptor share the same correlation ID.
func TestGetRecommendations_MultipleRecsFromSameResource(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	// m5.large generates both generation upgrade AND Graviton recommendation
	mock.ec2Prices["m5.large/Linux/Shared"] = 0.096
	mock.ec2Prices["m6i.large/Linux/Shared"] = 0.096 // Generation upgrade
	mock.ec2Prices["m6g.large/Linux/Shared"] = 0.077 // Graviton
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	expectedID := "urn:pulumi:stack::project::aws:ec2/instance:Instance::production-api"

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{
				Id:           expectedID,
				ResourceType: "aws:ec2:Instance",
				Sku:          "m5.large",
				Region:       "us-east-1",
				Provider:     "aws",
			},
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Should have exactly 2 recommendations (gen upgrade + Graviton)
	if len(resp.Recommendations) != 2 {
		t.Fatalf("Expected 2 recommendations (gen upgrade + Graviton), got %d", len(resp.Recommendations))
	}

	// Verify both recommendations have the same Resource.Id
	for i, rec := range resp.Recommendations {
		if rec.Resource == nil {
			t.Errorf("Recommendation[%d]: Resource is nil", i)
			continue
		}
		if rec.Resource.Id != expectedID {
			t.Errorf("Recommendation[%d]: Resource.Id = %q, want %q",
				i, rec.Resource.Id, expectedID)
		}
	}
}

// TestGetRecommendations_BatchIDCorrelation verifies that in a batch request,
// each resource's native ID is correctly passed through to its recommendations.
// This tests the end-to-end batch correlation workflow.
func TestGetRecommendations_BatchIDCorrelation(t *testing.T) {
	mock := newMockPricingClient("us-east-1", "USD")
	mock.ec2Prices["t2.medium/Linux/Shared"] = 0.0464
	mock.ec2Prices["t3.medium/Linux/Shared"] = 0.0416
	mock.ebsPrices["gp2"] = 0.10
	mock.ebsPrices["gp3"] = 0.08
	logger := zerolog.New(nil).Level(zerolog.InfoLevel)
	plugin := NewAWSPublicPlugin("us-east-1", mock, logger)

	// Define resources with unique IDs
	resource1ID := "urn:pulumi:prod::app::aws:ec2/instance:Instance::web-1"
	resource2ID := "urn:pulumi:prod::app::aws:ebs/volume:Volume::data-vol"

	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{
				Id:           resource1ID,
				ResourceType: "aws:ec2:Instance",
				Sku:          "t2.medium",
				Region:       "us-east-1",
				Provider:     "aws",
			},
			{
				Id:           resource2ID,
				ResourceType: "aws:ebs:Volume",
				Sku:          "gp2",
				Region:       "us-east-1",
				Provider:     "aws",
				Tags:         map[string]string{"size": "100"},
			},
		},
	}

	resp, err := plugin.GetRecommendations(context.Background(), req)
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}

	// Should have 2 recommendations (1 EC2 gen upgrade + 1 EBS gp2->gp3)
	if len(resp.Recommendations) != 2 {
		t.Fatalf("Expected 2 recommendations, got %d", len(resp.Recommendations))
	}

	// Build a map of expected IDs by resource type
	expectedIDs := map[string]string{
		"ec2": resource1ID,
		"ebs": resource2ID,
	}

	// Verify each recommendation has the correct Resource.Id
	for _, rec := range resp.Recommendations {
		if rec.Resource == nil {
			t.Error("Recommendation has nil Resource")
			continue
		}
		expectedID, ok := expectedIDs[rec.Resource.ResourceType]
		if !ok {
			t.Errorf("Unexpected resource type: %s", rec.Resource.ResourceType)
			continue
		}
		if rec.Resource.Id != expectedID {
			t.Errorf("Resource type %s: Resource.Id = %q, want %q",
				rec.Resource.ResourceType, rec.Resource.Id, expectedID)
		}
	}
}