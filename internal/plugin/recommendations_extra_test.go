package plugin

import (
	"context"
	"math"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func recommend(t *testing.T, mock *mockPricingClient, res *pbc.ResourceDescriptor) []*pbc.Recommendation {
	t.Helper()
	plugin := NewAWSPublicPlugin("us-east-1", "test", mock, zerolog.New(nil))
	resp, err := plugin.GetRecommendations(context.Background(),
		&pbc.GetRecommendationsRequest{TargetResources: []*pbc.ResourceDescriptor{res}})
	if err != nil {
		t.Fatalf("GetRecommendations() error: %v", err)
	}
	return resp.GetRecommendations()
}

func recByType(recs []*pbc.Recommendation, modType string) *pbc.Recommendation {
	for _, r := range recs {
		if r.GetModify().GetModificationType() == modType {
			return r
		}
	}
	return nil
}

func TestParseCacheInstanceType(t *testing.T) {
	tests := []struct{ in, family, size string }{
		{"cache.m5.large", "cache.m5", "large"},
		{"cache.r6g.2xlarge", "cache.r6g", "2xlarge"},
		{"db.m5.large", "", ""},
		{"cache.m5", "", ""},
		{"cache..large", "", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		family, size := parseCacheInstanceType(tt.in)
		if family != tt.family || size != tt.size {
			t.Errorf("parseCacheInstanceType(%q) = (%q, %q), want (%q, %q)", tt.in, family, size, tt.family, tt.size)
		}
	}
}

func TestElastiCacheRecommendations(t *testing.T) {
	newMock := func() *mockPricingClient {
		m := newMockPricingClient("us-east-1", "USD")
		m.elasticachePrices["cache.r4.large:Redis"] = 0.228
		m.elasticachePrices["cache.r5.large:Redis"] = 0.216
		m.elasticachePrices["cache.r6g.large:Redis"] = 0.206
		m.elasticachePrices["cache.r6g.large:Valkey"] = 0.1648
		m.elasticachePrices["cache.m4.large:Redis"] = 0.156
		m.elasticachePrices["cache.m5.large:Redis"] = 0.156
		m.elasticachePrices["cache.m6g.large:Redis"] = 0.149
		return m
	}

	t.Run("generation upgrade multiplies by node count", func(t *testing.T) {
		recs := recommend(t, newMock(), &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "elasticache", Sku: "cache.r4.large", Region: "us-east-1",
			Tags: map[string]string{"num_nodes": "3"},
		})
		rec := recByType(recs, modTypeGenUpgrade)
		if rec == nil {
			t.Fatal("expected generation upgrade recommendation")
		}
		want := (0.228 - 0.216) * 3 * 730
		if got := rec.GetImpact().GetEstimatedSavings(); math.Abs(got-want) > 1e-9 {
			t.Errorf("savings = %v, want %v", got, want)
		}
		if rec.GetModify().GetRecommendedConfig()[recFieldNodeType] != "cache.r5.large" {
			t.Errorf("unexpected recommended config %v", rec.GetModify().GetRecommendedConfig())
		}
		if rec.GetResource().GetResourceType() != serviceElastiCache {
			t.Errorf("resource type = %q", rec.GetResource().GetResourceType())
		}
	})

	t.Run("equal price upgrade still emitted with zero savings", func(t *testing.T) {
		rec := recByType(recommend(t, newMock(), &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "elasticache", Sku: "cache.m4.large", Region: "us-east-1",
		}), modTypeGenUpgrade)
		if rec == nil || rec.GetImpact().GetEstimatedSavings() != 0 {
			t.Fatalf("expected zero-savings upgrade, got %v", rec)
		}
	})

	t.Run("graviton", func(t *testing.T) {
		rec := recByType(recommend(t, newMock(), &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "elasticache", Sku: "cache.r5.large", Region: "us-east-1",
		}), modTypeGraviton)
		if rec == nil {
			t.Fatal("expected graviton recommendation")
		}
		want := (0.216 - 0.206) * 730
		if got := rec.GetImpact().GetEstimatedSavings(); math.Abs(got-want) > 1e-9 {
			t.Errorf("savings = %v, want %v", got, want)
		}
		if rec.GetConfidenceScore() != confidenceMedium {
			t.Errorf("confidence = %v", rec.GetConfidenceScore())
		}
	})

	t.Run("valkey switch", func(t *testing.T) {
		rec := recByType(recommend(t, newMock(), &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "elasticache", Sku: "cache.r6g.large", Region: "us-east-1",
		}), modTypeEngineMigration)
		if rec == nil {
			t.Fatal("expected engine migration recommendation")
		}
		want := (0.206 - 0.1648) * 730
		if got := rec.GetImpact().GetEstimatedSavings(); math.Abs(got-want) > 1e-9 {
			t.Errorf("savings = %v, want %v", got, want)
		}
	})

	t.Run("price guard rejects costlier target", func(t *testing.T) {
		m := newMock()
		m.elasticachePrices["cache.m6g.large:Redis"] = 0.200
		recs := recommend(t, m, &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "elasticache", Sku: "cache.m5.large", Region: "us-east-1",
		})
		if recByType(recs, modTypeGraviton) != nil {
			t.Error("graviton target costs more; expected no recommendation")
		}
	})

	t.Run("missing pricing, unknown family, memcached", func(t *testing.T) {
		for _, res := range []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "elasticache", Sku: "cache.t2.micro", Region: "us-east-1"},
			{Provider: "aws", ResourceType: "elasticache", Sku: "cache.x9.large", Region: "us-east-1"},
			{Provider: "aws", ResourceType: "elasticache", Sku: "bogus", Region: "us-east-1"},
			{
				Provider: "aws", ResourceType: "elasticache", Sku: "cache.r6g.large", Region: "us-east-1",
				Tags: map[string]string{"engine": "memcached"},
			},
		} {
			if recs := recommend(t, newMock(), res); len(recs) != 0 {
				t.Errorf("%s: expected none, got %d", res.GetSku(), len(recs))
			}
		}
	})

	t.Run("pulumi and terraform forms match", func(t *testing.T) {
		pulumi := recommend(t, newMock(), &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "aws:elasticache/cluster:Cluster", Region: "us-east-1",
			Tags: map[string]string{"nodeType": "cache.r4.large", "numCacheNodes": "2"},
		})
		short := recommend(t, newMock(), &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "elasticache", Sku: "cache.r4.large", Region: "us-east-1",
			Tags: map[string]string{"num_nodes": "2"},
		})
		p, s := recByType(pulumi, modTypeGenUpgrade), recByType(short, modTypeGenUpgrade)
		if p == nil || s == nil {
			t.Fatal("expected recommendations for both forms")
		}
		if p.GetImpact().GetEstimatedSavings() != s.GetImpact().GetEstimatedSavings() {
			t.Errorf("savings differ: %v vs %v",
				p.GetImpact().GetEstimatedSavings(), s.GetImpact().GetEstimatedSavings())
		}
	})
}

func TestLambdaArm64Recommendation(t *testing.T) {
	newMock := func() *mockPricingClient {
		m := newMockPricingClient("us-east-1", "USD")
		m.lambdaPrices["request"] = 0.0000002
		m.lambdaPrices["gb-second"] = 0.0000166667
		m.lambdaPrices["gb-second-arm64"] = 0.0000133334
		return m
	}
	res := func(tags map[string]string) *pbc.ResourceDescriptor {
		return &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "lambda", Sku: "512", Region: "us-east-1", Tags: tags,
		}
	}

	t.Run("savings match formula", func(t *testing.T) {
		recs := recommend(t, newMock(), res(map[string]string{
			"requests_per_month": "1000000", "avg_duration_ms": "200",
		}))
		rec := recByType(recs, modTypeGraviton)
		if rec == nil {
			t.Fatal("expected arm64 recommendation")
		}
		gbSec := 0.5 * 0.2 * 1000000
		want := gbSec * (0.0000166667 - 0.0000133334)
		if got := rec.GetImpact().GetEstimatedSavings(); math.Abs(got-want) > 1e-9 {
			t.Errorf("savings = %v, want %v", got, want)
		}
		if rec.GetModify().GetRecommendedConfig()[recFieldArchitecture] != archARM64 {
			t.Errorf("unexpected config %v", rec.GetModify().GetRecommendedConfig())
		}
	})

	t.Run("skipped cases", func(t *testing.T) {
		noARM := newMock()
		delete(noARM.lambdaPrices, "gb-second-arm64")
		cases := map[string]struct {
			mock *mockPricingClient
			tags map[string]string
		}{
			"no requests":      {newMock(), map[string]string{"avg_duration_ms": "200"}},
			"zero requests":    {newMock(), map[string]string{"requests_per_month": "0"}},
			"already arm64":    {newMock(), map[string]string{"requests_per_month": "1000", "arch": "arm64"}},
			"already arm":      {newMock(), map[string]string{"requests_per_month": "1000", "architecture": "arm"}},
			"arm price absent": {noARM, map[string]string{"requests_per_month": "1000"}},
		}
		for name, tc := range cases {
			if recs := recommend(t, tc.mock, res(tc.tags)); len(recs) != 0 {
				t.Errorf("%s: expected none, got %d", name, len(recs))
			}
		}
	})

	t.Run("pulumi attribute names", func(t *testing.T) {
		recs := recommend(t, newMock(), &pbc.ResourceDescriptor{
			Provider: "aws", ResourceType: "aws:lambda/function:Function", Region: "us-east-1",
			Tags: map[string]string{"memorySize": "512", "requestsPerMonth": "1000000", "avgDurationMs": "200"},
		})
		if recByType(recs, modTypeGraviton) == nil {
			t.Fatal("expected arm64 recommendation for Pulumi form")
		}
	})
}

func TestNATGatewayEndpointRecommendation(t *testing.T) {
	newMock := func() *mockPricingClient {
		m := newMockPricingClient("us-east-1", "USD")
		m.natgwHourlyPrice = 0.045
		m.natgwDataPrice = 0.045
		return m
	}
	res := func(rt string, tags map[string]string) *pbc.ResourceDescriptor {
		return &pbc.ResourceDescriptor{Provider: "aws", ResourceType: rt, Region: "us-east-1", Tags: tags}
	}

	t.Run("savings equal tag times data rate", func(t *testing.T) {
		recs := recommend(t, newMock(), res("natgw", map[string]string{
			"s3_dynamodb_data_gb": "500", "data_processed_gb": "800",
		}))
		if len(recs) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(recs))
		}
		if recs[0].GetActionType() != pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_REFACTOR {
			t.Errorf("action type = %v", recs[0].GetActionType())
		}
		if got, want := recs[0].GetImpact().GetEstimatedSavings(), 500*0.045; math.Abs(got-want) > 1e-9 {
			t.Errorf("savings = %v, want %v", got, want)
		}
	})

	t.Run("capped at data processed", func(t *testing.T) {
		recs := recommend(t, newMock(), res("natgw", map[string]string{
			"s3_dynamodb_data_gb": "500", "data_processed_gb": "100",
		}))
		if len(recs) != 1 || math.Abs(recs[0].GetImpact().GetEstimatedSavings()-100*0.045) > 1e-9 {
			t.Fatalf("expected savings capped at 100 GB, got %v", recs)
		}
	})

	t.Run("no recommendation without a positive tag", func(t *testing.T) {
		for name, tags := range map[string]map[string]string{
			"no tag":   {"data_processed_gb": "800"},
			"zero":     {"s3_dynamodb_data_gb": "0"},
			"negative": {"s3_dynamodb_data_gb": "-5"},
			"garbage":  {"s3_dynamodb_data_gb": "lots"},
		} {
			if recs := recommend(t, newMock(), res("natgw", tags)); len(recs) != 0 {
				t.Errorf("%s: expected none, got %d", name, len(recs))
			}
		}
	})

	t.Run("pulumi form matches", func(t *testing.T) {
		recs := recommend(t, newMock(), res("aws:ec2/natGateway:NatGateway", map[string]string{
			"s3_dynamodb_data_gb": "10",
		}))
		if len(recs) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(recs))
		}
	})

	t.Run("missing pricing", func(t *testing.T) {
		m := newMock()
		m.natgwHourlyPrice = 0
		if recs := recommend(t, m, res("natgw", map[string]string{"s3_dynamodb_data_gb": "10"})); len(recs) != 0 {
			t.Errorf("expected none, got %d", len(recs))
		}
	})
}
