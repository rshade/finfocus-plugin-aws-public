package plugin

import (
	"context"
	"fmt"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// Supports checks if this plugin can estimate costs for the given resource.
func (p *AWSPublicPlugin) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	if req == nil || req.Resource == nil {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    "Invalid request: missing resource descriptor",
		}, nil
	}

	resource := req.Resource

	// Check provider
	if resource.Provider != "aws" {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Provider %q not supported (only 'aws' is supported)", resource.Provider),
		}, nil
	}

	// Check region match
	if resource.Region != p.region {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Region not supported by this binary (plugin region: %s, resource region: %s)", p.region, resource.Region),
		}, nil
	}

	// Check resource type
	switch resource.ResourceType {
	case "ec2", "ebs":
		// Fully supported
		return &pbc.SupportsResponse{
			Supported: true,
			Reason:    "",
		}, nil

	case "s3", "lambda", "rds", "dynamodb":
		// Stub support - returns $0 estimates
		return &pbc.SupportsResponse{
			Supported: true,
			Reason:    fmt.Sprintf("Limited support - %s cost estimation not fully implemented, returns $0 estimate", resource.ResourceType),
		}, nil

	default:
		// Unknown resource type
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Resource type %q not supported", resource.ResourceType),
		}, nil
	}
}
