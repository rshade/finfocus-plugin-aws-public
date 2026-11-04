package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestZeroCostNetworking_PulumiTypes verifies that internet gateways, route
// tables, route table associations, and routes are priced at $0 without a SKU.
//
// These types used to collapse to the "ec2" service, so Supports said yes and
// GetProjectedCost then failed with "resource.sku is required". Each type must
// now resolve to its own zero-cost service and carry its own billing detail.
func TestZeroCostNetworking_PulumiTypes(t *testing.T) {
	tests := []struct {
		resourceType string
		wantService  string
		wantDetail   string
	}{
		{"aws:ec2/internetGateway:InternetGateway", serviceInternetGateway, "Internet Gateways have no direct charge"},
		{"aws:ec2/routeTable:RouteTable", serviceRouteTable, "Route Tables have no direct charge"},
		{
			"aws:ec2/routeTableAssociation:RouteTableAssociation",
			serviceRouteTableAssoc,
			"Route Table associations have no direct charge",
		},
		{"aws:ec2/route:Route", serviceRoute, "Routes have no direct charge"},
	}

	plugin := newTestPluginForActual()
	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			assert.Equal(t, tt.wantService, detectService(normalizeResourceType(tt.resourceType)))

			res := &pbc.ResourceDescriptor{Provider: "aws", ResourceType: tt.resourceType, Region: "us-east-1"}

			sup, err := plugin.Supports(context.Background(), &pbc.SupportsRequest{Resource: res})
			require.NoError(t, err)
			assert.True(t, sup.GetSupported())

			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{Resource: res})
			require.NoError(t, err)
			assert.InDelta(t, 0.0, resp.GetCostPerMonth(), 0)
			assert.Contains(t, resp.GetBillingDetail(), tt.wantDetail)

			spec, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{Resource: res})
			require.NoError(t, err)
			assert.Equal(t, billingModeZeroCost, spec.GetSpec().GetBillingMode())
		})
	}
}

// TestZeroCostNetworking_RouteDoesNotCaptureRouteTable pins the token-aware
// pattern match. "ec2/route" is a prefix of "ec2/routetable" and
// "ec2/routetableassociation", and must not claim them.
func TestZeroCostNetworking_RouteDoesNotCaptureRouteTable(t *testing.T) {
	assert.Equal(t, serviceRoute, normalizeResourceType("aws:ec2/route:Route"))
	assert.Equal(t, serviceRouteTable, normalizeResourceType("aws:ec2/routeTable:RouteTable"))
	assert.Equal(t, serviceRouteTableAssoc,
		normalizeResourceType("aws:ec2/routeTableAssociation:RouteTableAssociation"))
}

// TestZeroCostPulumiPatterns_AllCostRPCs walks every entry in
// ZeroCostPulumiPatterns through every cost RPC.
//
// Each RPC keeps its own list of zero-cost services in a switch, so adding a
// pattern without updating all of them leaves one RPC answering "unknown
// resource". Driving the test from the map means a future entry that misses a
// switch fails here.
func TestZeroCostPulumiPatterns_AllCostRPCs(t *testing.T) {
	patterns := make([]string, 0, len(ZeroCostPulumiPatterns))
	for pattern := range ZeroCostPulumiPatterns {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)

	plugin := newTestPluginForActual()
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, pattern := range patterns {
		resourceType := "aws:" + pattern + ":Resource"
		t.Run(pattern, func(t *testing.T) {
			require.True(t, IsZeroCostService(ZeroCostPulumiPatterns[pattern]))

			res := &pbc.ResourceDescriptor{Provider: "aws", ResourceType: resourceType, Region: "us-east-1"}

			resp, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{Resource: res})
			require.NoError(t, err)
			assert.InDelta(t, 0.0, resp.GetCostPerMonth(), 0)
			assert.NotContains(t, resp.GetBillingDetail(), "not supported")

			spec, err := plugin.GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{Resource: res})
			require.NoError(t, err)
			assert.Equal(t, billingModeZeroCost, spec.GetSpec().GetBillingMode())

			actual, err := plugin.GetActualCost(context.Background(), &pbc.GetActualCostRequest{
				ResourceId: makeResourceJSON("aws", resourceType, "", "us-east-1", nil),
				Start:      timestamppb.New(from),
				End:        timestamppb.New(from.Add(24 * time.Hour)),
			})
			require.NoError(t, err)
			require.NotEmpty(t, actual.GetResults())
			assert.InDelta(t, 0.0, actual.GetResults()[0].GetCost(), 0)
		})
	}
}

// TestGetProjectedCost_FailureLogNamesResource verifies that a failed request
// logs the resource type and the service it resolved to.
//
// The log line used to carry only the trace id and the error, so a failure
// could not be traced to a resource without joining against the host's output.
func TestGetProjectedCost_FailureLogNamesResource(t *testing.T) {
	var buf bytes.Buffer
	plugin := NewAWSPublicPlugin("us-east-1", "test-version", newMockPricingClient("us-east-1", "USD"),
		zerolog.New(&buf))

	_, err := plugin.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2", Region: "us-east-1"},
	})
	require.Error(t, err)

	var failed map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["message"] == "request failed" {
			failed = entry
		}
	}
	require.NotNil(t, failed, "expected a 'request failed' log line, got:\n%s", buf.String())
	assert.Equal(t, "ec2", failed["resource_type"])
	assert.Equal(t, serviceEC2, failed["service_type"])
}
