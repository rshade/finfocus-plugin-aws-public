package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetPricingSpec_FailureLogNamesResource verifies that a rejected
// GetPricingSpec request logs the resource type and the service it resolved to.
//
// The line used to name only the resource type, under "validation failed".
// It now uses the same helper as GetProjectedCost, so an aws:ec2/* type that
// collapses to ec2 shows service_type=ec2. A missing descriptor writes both
// fields empty and does not panic. Attributes and tags are not logged.
func TestGetPricingSpec_FailureLogNamesResource(t *testing.T) {
	cases := []struct {
		name        string
		req         *pbc.GetPricingSpecRequest
		wantType    string
		wantService string
	}{
		{
			name: "resolvable ec2 type missing sku",
			req: &pbc.GetPricingSpecRequest{Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "aws:ec2/instance:Instance",
				Region:       "us-east-1",
				Tags:         map[string]string{"Name": "do-not-log"},
			}},
			wantType:    "aws:ec2/instance:Instance",
			wantService: serviceEC2,
		},
		{
			name: "unresolvable type",
			req: &pbc.GetPricingSpecRequest{Resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "madeup",
				Tags:         map[string]string{"Name": "do-not-log"},
			}},
			wantType:    "madeup",
			wantService: "madeup",
		},
		{
			name:        "nil request",
			req:         nil,
			wantType:    "",
			wantService: "",
		},
		{
			name:        "nil resource",
			req:         &pbc.GetPricingSpecRequest{},
			wantType:    "",
			wantService: "",
		},
		{
			name: "empty resource type",
			req: &pbc.GetPricingSpecRequest{Resource: &pbc.ResourceDescriptor{
				Provider: "aws",
				Region:   "us-east-1",
			}},
			wantType:    "",
			wantService: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			plugin := NewAWSPublicPlugin("us-east-1", "test-version", newMockPricingClient("us-east-1", "USD"),
				zerolog.New(&buf))

			_, err := plugin.GetPricingSpec(context.Background(), tc.req)
			require.Error(t, err)

			var failed map[string]any
			for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
				if line == "" {
					continue
				}
				var entry map[string]any
				require.NoError(t, json.Unmarshal([]byte(line), &entry))
				if entry["message"] == "request failed" {
					failed = entry
				}
				assert.NotContains(t, line, "do-not-log")
			}
			require.NotNil(t, failed, "expected a 'request failed' log line, got:\n%s", buf.String())
			assert.Equal(t, tc.wantType, failed["resource_type"])
			assert.Equal(t, tc.wantService, failed["service_type"])
			_, hasTags := failed["tags"]
			assert.False(t, hasTags)
		})
	}
}
