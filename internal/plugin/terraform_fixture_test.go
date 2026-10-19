package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
	"github.com/rshade/finfocus-plugin-aws-public/internal/typeregistry"
)

// TestTerraformStateFixtures prices one Terraform state v4 fixture for every
// mapped resource type.
//
// The helper copies finfocus core v0.3.7: SnakeToCamel and convertKeysRecursive
// from finfocus/internal/ingest/terraform_map.go, and ConvertValueToString from
// finfocus/internal/engine/engine.go. It then applies TerraformMappings and
// calls GetProjectedCost. Each case must return no gRPC error, route to the
// mapped service, and show the Terraform attribute in the billing detail or
// defaults_applied. The test uses a mock pricing client so it passes on the
// fallback build, where RDS, Lambda, and DynamoDB embeds are empty.
//
// Run with: go test ./internal/plugin -run TestTerraformStateFixtures.
func TestTerraformStateFixtures(t *testing.T) {
	plugin := newTerraformFixturePlugin(t)
	dir := terraformFixtureDir(t)

	for tfType, pulumiType := range typeregistry.TerraformMappings {
		t.Run(tfType, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, tfType+".tfstate"))
			require.NoError(t, err)
			resource := resourceFromTerraformState(t, raw, pulumiType)

			resp, err := plugin.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
				Resource: resource,
			})
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, currencyUSD, resp.GetCurrency())

			gotService := detectService(normalizeResourceType(pulumiType))
			assert.Equal(t, gotService, detectService(normalizeResourceType(resource.GetResourceType())))

			evidence := resp.GetBillingDetail() + " " + resp.GetMetadata()["defaults_applied"]
			for _, want := range terraformEvidence[tfType] {
				assert.Contains(t, evidence, want)
			}
		})
	}
}

// TestTerraformMappingsHaveFixtures fails when a TerraformMappings entry has
// no state file, or a state file has no mapping.
func TestTerraformMappingsHaveFixtures(t *testing.T) {
	dir := terraformFixtureDir(t)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tfstate") {
			continue
		}
		tfType := strings.TrimSuffix(entry.Name(), ".tfstate")
		seen[tfType] = true
		if _, ok := typeregistry.TerraformMappings[tfType]; !ok {
			t.Errorf("fixture %s has no TerraformMappings entry", entry.Name())
		}
	}
	for tfType := range typeregistry.TerraformMappings {
		if !seen[tfType] {
			t.Errorf("TerraformMappings %s has no fixture", tfType)
		}
	}
}

// terraformEvidence is the billing or defaults text that proves the fixture
// attribute was used. Zero-cost types prove routing by their billing detail.
var terraformEvidence = map[string][]string{
	"aws_instance":                {"20GB gp3 root volume"},
	"aws_launch_template":         {"Launch Templates are configuration-only"},
	"aws_launch_configuration":    {"Launch Configurations are configuration-only"},
	"aws_autoscaling_group":       {"desired capacity 3", "mixed instance policies"},
	"aws_ebs_volume":              {"100 GB"},
	"aws_s3_bucket":               {"storage_class=STANDARD"},
	"aws_db_instance":             {"100GB gp3"},
	"aws_dynamodb_table":          {"5 RCUs", "5 WCUs"},
	"aws_elasticache_cluster":     {"cache.t3.micro", "2 nodes"},
	"aws_lb":                      {"NLB"},
	"aws_alb":                     {"ALB"},
	"aws_nat_gateway":             {"NAT Gateway"},
	"aws_vpc":                     {"VPC has no direct"},
	"aws_subnet":                  {"Subnets have no direct"},
	"aws_security_group":          {"Security Groups have no direct"},
	"aws_lambda_function":         {"256MB", "arm64"},
	"aws_eks_cluster":             {"extended support"},
	"aws_cloudwatch_log_group":    {"sku=logs"},
	"aws_cloudwatch_metric_alarm": {"alarm pricing is not supported"},
	"aws_iam_role":                {"IAM resources"},
	"aws_iam_policy":              {"IAM resources"},
	"aws_iam_instance_profile":    {"IAM resources"},
	"aws_iam_user":                {"IAM resources"},
}

func terraformFixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "testdata", "terraform")
}

type terraformStateFile struct {
	Version   int `json:"version"`
	Resources []struct {
		Type      string `json:"type"`
		Instances []struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"instances"`
	} `json:"resources"`
}

func resourceFromTerraformState(t *testing.T, raw []byte, pulumiType string) *pbc.ResourceDescriptor {
	t.Helper()
	var state terraformStateFile
	require.NoError(t, json.Unmarshal(raw, &state))
	require.Equal(t, 4, state.Version)
	require.NotEmpty(t, state.Resources)
	require.NotEmpty(t, state.Resources[0].Instances)

	tags := terraformTags(state.Resources[0].Instances[0].Attributes)
	return &pbc.ResourceDescriptor{
		Provider:     providerAWS,
		ResourceType: pulumiType,
		Sku:          terraformSKU(pulumiType, tags),
		Region:       terraformRegion(tags),
		Tags:         tags,
	}
}

// terraformTags converts one attribute map the way finfocus core v0.3.7 does
// before it fills ResourceDescriptor.Tags.
//
// Keys copy SnakeToCamel and convertKeysRecursive
// (finfocus/internal/ingest/terraform_map.go). Values copy
// ConvertValueToString (finfocus/internal/engine/engine.go). These fixtures
// have no secrets, so credential stripping is not copied.
func terraformTags(attrs map[string]any) map[string]string {
	camel := terraformConvertKeys(attrs)
	out := make(map[string]string, len(camel))
	for key, value := range camel {
		out[key] = terraformValueToString(value)
	}
	return out
}

func terraformConvertKeys(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for key, value := range m {
		out[terraformSnakeToCamel(key)] = terraformConvertValue(value)
	}
	return out
}

func terraformConvertValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return terraformConvertKeys(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = terraformConvertValue(item)
		}
		return out
	default:
		return val
	}
}

func terraformSnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func terraformValueToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case map[string]any:
		if value, ok := val["value"]; ok {
			return terraformValueToString(value)
		}
		if id, ok := val["id"]; ok {
			return terraformValueToString(id)
		}
		if name, ok := val["name"]; ok {
			return terraformValueToString(name)
		}
		if len(val) == 1 {
			for _, inner := range val {
				return terraformValueToString(inner)
			}
		}
		return fmt.Sprintf("%v", val)
	case []any:
		if len(val) == 1 {
			return terraformValueToString(val[0])
		}
		if len(val) > 0 {
			parts := make([]string, len(val))
			for i, elem := range val {
				parts[i] = terraformValueToString(elem)
			}
			return strings.Join(parts, ",")
		}
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// terraformSKU copies the SKU keys finfocus core tries before this plugin
// sees the request. eks/cluster is the only skus.ResolveSKU fallback this
// plugin's mappings need.
func terraformSKU(pulumiType string, tags map[string]string) string {
	for _, key := range []string{
		"instanceType", "instanceClass", "type", "volumeType",
		"dbInstanceClass", "sku", "tier",
	} {
		if tags[key] != "" {
			return tags[key]
		}
	}
	if strings.Contains(pulumiType, "eks/cluster") {
		return "cluster"
	}
	return ""
}

// terraformRegion uses region, then the availability zone with its trailing
// letter removed. An empty result uses us-east-1, which is what core sends
// when AWS_REGION is set and the state has no region attribute.
func terraformRegion(tags map[string]string) string {
	if tags["region"] != "" {
		return tags["region"]
	}
	if az := tags["availabilityZone"]; len(az) > 1 {
		return az[:len(az)-1]
	}
	return "us-east-1"
}

func newTerraformFixturePlugin(t *testing.T) *AWSPublicPlugin {
	t.Helper()
	mock := newMockPricingClient("us-east-1", currencyUSD)
	mock.ec2Prices["t3.micro/Linux/Shared"] = 0.0104
	mock.ebsPrices["gp3"] = 0.08
	mock.ebsPrices["gp2"] = 0.10
	mock.s3Prices["STANDARD"] = 0.023
	mock.s3Prices["GLACIER"] = 0.004
	mock.rdsStoragePrices["gp2"] = 0.115
	mock.rdsInstancePrices["db.t3.micro/PostgreSQL"] = 0.018
	mock.rdsInstancePrices["db.t3.micro/MySQL"] = 0.017
	mock.rdsStoragePrices["gp3"] = 0.115
	mock.eksStandardPrice = 0.10
	mock.eksExtendedPrice = 0.60
	mock.albHourlyPrice = 0.0225
	mock.albLCUPrice = 0.008
	mock.nlbHourlyPrice = 0.0225
	mock.nlbNLCUPrice = 0.006
	mock.natgwHourlyPrice = 0.045
	mock.natgwDataPrice = 0.045
	mock.lambdaPrices["request"] = 0.0000002
	mock.lambdaPrices["gb-second"] = 0.0000166667
	mock.lambdaPrices["gb-second-arm64"] = 0.0000133334
	mock.dynamoDBPrices["storage"] = 0.25
	mock.dynamoDBPrices["provisioned-rcu"] = 0.00013
	mock.dynamoDBPrices["provisioned-wcu"] = 0.00065
	mock.dynamoDBPrices["on-demand-read"] = 0.00000025
	mock.dynamoDBPrices["on-demand-write"] = 0.00000125
	mock.elasticachePrices["cache.t3.micro:Redis"] = 0.017
	mock.cwLogsIngestionTiers = []pricing.TierRate{{UpTo: 1e12, Rate: 0.50}}
	mock.cwLogsStorageRate = 0.03
	logger := zerolog.New(os.Stderr).Level(zerolog.ErrorLevel)
	plugin := NewAWSPublicPlugin("us-east-1", "test", mock, logger)
	require.NotNil(t, plugin)
	return plugin
}
