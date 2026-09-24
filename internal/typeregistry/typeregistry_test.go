package typeregistry_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

	"github.com/rshade/finfocus-plugin-aws-public/internal/typeregistry"
)

// TestNew_RegistersAllTerraformMappings verifies that New registers every entry
// in TerraformMappings, so no mapping is silently dropped at construction.
func TestNew_RegistersAllTerraformMappings(t *testing.T) {
	registry := typeregistry.New()
	assert.Equal(t, len(typeregistry.TerraformMappings), registry.Len())
}

// TestResolve_KnownTypes verifies that each Terraform type in TerraformMappings
// resolves to its expected Pulumi token and is reported as supported.
func TestResolve_KnownTypes(t *testing.T) {
	registry := typeregistry.New()

	for tfType, expectedToken := range typeregistry.TerraformMappings {
		t.Run(tfType, func(t *testing.T) {
			resp := registry.Resolve(&pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
				SourceTypes:  []string{tfType},
			})

			require.Len(t, resp.GetMappings(), 1)
			mapping := resp.GetMappings()[tfType]
			require.NotNil(t, mapping)
			assert.Equal(t, expectedToken, mapping.GetPulumiToken())
			assert.True(t, mapping.GetSupported())
		})
	}
}

// TestResolve_UnknownTypeOmitted verifies that unrecognized Terraform types are
// left out of the response while known types in the same request still resolve.
func TestResolve_UnknownTypeOmitted(t *testing.T) {
	registry := typeregistry.New()

	resp := registry.Resolve(&pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_definitely_not_real"},
	})

	require.Len(t, resp.GetMappings(), 1)
	assert.Contains(t, resp.GetMappings(), "aws_instance")
	assert.NotContains(t, resp.GetMappings(), "aws_definitely_not_real")
}

// TestResolve_UnspecifiedFormatReturnsEmpty verifies that a request with
// SOURCE_FORMAT_UNSPECIFIED yields no mappings, even for known type names.
func TestResolve_UnspecifiedFormatReturnsEmpty(t *testing.T) {
	registry := typeregistry.New()

	resp := registry.Resolve(&pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_UNSPECIFIED,
		SourceTypes:  []string{"aws_instance"},
	})

	assert.Empty(t, resp.GetMappings())
}

// TestResolve_EmptySourceTypesReturnsEmpty verifies that a request with no
// source types yields an empty mapping set.
func TestResolve_EmptySourceTypesReturnsEmpty(t *testing.T) {
	registry := typeregistry.New()

	resp := registry.Resolve(&pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{},
	})

	assert.Empty(t, resp.GetMappings())
}

// TestResolve_SetsExpiresAt verifies that responses carry an expires_at
// timestamp 24 hours after the time of resolution.
func TestResolve_SetsExpiresAt(t *testing.T) {
	registry := typeregistry.New()

	before := time.Now()
	resp := registry.Resolve(&pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance"},
	})
	after := time.Now()

	expiresAt := resp.GetExpiresAt()
	require.NotNil(t, expiresAt)
	assert.False(t, expiresAt.AsTime().Before(before.Add(24*time.Hour)))
	assert.False(t, expiresAt.AsTime().After(after.Add(24*time.Hour)))
}
