package plugin

import (
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestCanonicalizeResource_AliasPrecedence checks that the clone does not
// overwrite an explicit canonical tag or a non-empty SKU, and that an alias
// fills the canonical key only when that key is empty.
func TestCanonicalizeResource_AliasPrecedence(t *testing.T) {
	t.Parallel()

	original := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "rds",
		Sku:          "db.t3.micro",
		Region:       "us-east-1",
		Tags: map[string]string{
			"storage_type": "gp3",
			"storageType":  "io1",
			"multiAz":      "true",
		},
	}

	got := canonicalizeResource(serviceRDS, original)

	assert.NotSame(t, original, got)
	assert.Equal(t, "gp3", original.GetTags()["storage_type"], "caller tags must be unchanged")
	assert.Equal(t, "gp3", got.GetTags()["storage_type"])
	assert.Equal(t, "io1", got.GetTags()["storageType"])
	assert.Equal(t, "true", got.GetTags()["multi_az"])
	assert.Equal(t, "db.t3.micro", got.GetSku())
}

// TestCanonicalizeResource_DerivesEmptySKU covers SKU derivation and the
// rule that a gateway load balancer is not labeled as an ALB.
func TestCanonicalizeResource_DerivesEmptySKU(t *testing.T) {
	t.Parallel()

	t.Run("keeps a non-empty sku", func(t *testing.T) {
		t.Parallel()
		got := canonicalizeResource(serviceELB, &pbc.ResourceDescriptor{
			Sku:  "alb",
			Tags: map[string]string{"loadBalancerType": "network"},
		})
		assert.Equal(t, "alb", got.GetSku())
	})

	t.Run("elasticache node type", func(t *testing.T) {
		t.Parallel()
		got := canonicalizeResource(serviceElastiCache, &pbc.ResourceDescriptor{
			Tags: map[string]string{"nodeType": "cache.t3.micro"},
		})
		assert.Equal(t, "cache.t3.micro", got.GetSku())
	})

	t.Run("lambda memory from attributes", func(t *testing.T) {
		t.Parallel()
		attrs, err := structpb.NewStruct(map[string]any{"memorySize": 512.0})
		require.NoError(t, err)
		got := canonicalizeResource(serviceLambda, &pbc.ResourceDescriptor{Attributes: attrs})
		assert.Equal(t, "512", got.GetSku())
	})

	t.Run("dynamodb billing mode", func(t *testing.T) {
		t.Parallel()
		got := canonicalizeResource(serviceDynamoDB, &pbc.ResourceDescriptor{
			Tags: map[string]string{"billingMode": "PAY_PER_REQUEST"},
		})
		assert.Equal(t, "on-demand", got.GetSku())

		provisioned := canonicalizeResource(serviceDynamoDB, &pbc.ResourceDescriptor{
			Tags: map[string]string{"billingMode": "PROVISIONED"},
		})
		assert.Equal(t, "provisioned", provisioned.GetSku())
	})

	t.Run("gateway is not an alb", func(t *testing.T) {
		t.Parallel()
		got := canonicalizeResource(serviceELB, &pbc.ResourceDescriptor{
			Tags: map[string]string{"loadBalancerType": "gateway"},
		})
		assert.Equal(t, "gwlb", got.GetSku())
		assert.NotEqual(t, "alb", got.GetSku())
	})

	t.Run("s3 sku stays empty", func(t *testing.T) {
		t.Parallel()
		got := canonicalizeResource(serviceS3, &pbc.ResourceDescriptor{
			Tags: map[string]string{"bucket": "logs"},
		})
		assert.Empty(t, got.GetSku())
	})
}

// TestNormalizeAliasValue lowercases EKS support and keeps the first architecture.
func TestNormalizeAliasValue(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "extended", normalizeAliasValue("support_type", "EXTENDED"))
	assert.Equal(t, "arm64", normalizeAliasValue("arch", "arm64,x86_64"))
	assert.Equal(t, "100", normalizeAliasValue("storage_size", "100"))
}

// TestPricingTags_SourcePrecedence checks attribute-over-tag resolution and
// the RDS rule that a Pulumi tag wins over a legacy tag.
func TestPricingTags_SourcePrecedence(t *testing.T) {
	t.Parallel()

	t.Run("attribute alias beats canonical tag", func(t *testing.T) {
		t.Parallel()
		attrs, err := structpb.NewStruct(map[string]any{"readCapacity": 5.0})
		require.NoError(t, err)
		tags := pricingTags(&pbc.ResourceDescriptor{
			ResourceType: "dynamodb",
			Attributes:   attrs,
			Tags:         map[string]string{"read_capacity_units": "3"},
		})
		assert.Equal(t, "5", tags["read_capacity_units"])
	})

	t.Run("rds pulumi tag beats legacy tag", func(t *testing.T) {
		t.Parallel()
		tags := pricingTags(&pbc.ResourceDescriptor{
			ResourceType: "rds",
			Tags:         map[string]string{"storageType": "io1", "storage_type": "gp3"},
		})
		assert.Equal(t, "io1", tags["storage_type"])
		assert.Equal(t, "io1", tags["storageType"])
	})

	t.Run("rds attribute beats pulumi tag", func(t *testing.T) {
		t.Parallel()
		attrs, err := structpb.NewStruct(map[string]any{"storage_type": "gp3"})
		require.NoError(t, err)
		tags := pricingTags(&pbc.ResourceDescriptor{
			ResourceType: "rds",
			Attributes:   attrs,
			Tags:         map[string]string{"storageType": "io1"},
		})
		assert.Equal(t, "gp3", tags["storage_type"])
		assert.Equal(t, "gp3", tags["storageType"])
	})
}
