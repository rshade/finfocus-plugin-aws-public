# Prompt: Implement EC2 & EBS estimation logic for gRPC

You are implementing the `pulumicost-plugin-aws-public` Go plugin.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Implement the **v1 estimation logic** for:

- EC2 instances
- EBS volumes

using the embedded pricing client from `internal/pricing` and the gRPC `ResourceDescriptor` input format.

Other services (S3, Lambda, RDS, DynamoDB) can be stubbed, but the focus here is EC2/EBS.

## 1. Implement GetProjectedCost() for EC2

Update `internal/plugin/projected.go` to handle EC2 resources:

```go
package plugin

import (
	"context"
	"fmt"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetProjectedCost estimates cost for a single resource
func (p *AWSPublicPlugin) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if req == nil || req.Resource == nil {
		return nil, status.Error(codes.InvalidArgument, "missing resource descriptor")
	}

	rd := req.Resource

	// Validate required fields
	if rd.Provider == "" || rd.ResourceType == "" || rd.Region == "" {
		// Return ERROR_CODE_INVALID_RESOURCE via gRPC status
		return nil, status.Error(codes.InvalidArgument, "missing required fields in ResourceDescriptor")
	}

	// Check region match
	if rd.Region != p.region {
		// Return ERROR_CODE_UNSUPPORTED_REGION
		st := status.New(codes.FailedPrecondition, fmt.Sprintf(
			"Resource in region %s but plugin compiled for %s",
			rd.Region, p.region,
		))
		// TODO: Add ErrorDetail with pluginRegion and requiredRegion in details map
		return nil, st.Err()
	}

	// Route to appropriate estimator based on resource_type
	switch rd.ResourceType {
	case "ec2":
		return p.estimateEC2(ctx, rd)
	case "ebs":
		return p.estimateEBS(ctx, rd)
	case "s3", "lambda", "rds", "dynamodb":
		return p.estimateStub(ctx, rd)
	default:
		return nil, status.Error(codes.Unimplemented, fmt.Sprintf("resource type %s not supported", rd.ResourceType))
	}
}

// estimateEC2 calculates cost for EC2 instances
func (p *AWSPublicPlugin) estimateEC2(
	ctx context.Context,
	rd *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	// sku contains instance type (e.g., "t3.micro")
	instanceType := rd.Sku
	if instanceType == "" {
		return nil, status.Error(codes.InvalidArgument, "EC2 resource missing sku (instance type)")
	}

	// For v1, assume Linux/Shared tenancy
	const (
		operatingSystem = "Linux"
		tenancy         = "Shared"
		hoursPerMonth   = 730.0
	)

	// Lookup pricing
	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, operatingSystem, tenancy)
	if !found {
		// Return $0 with explanation
		return &pbc.GetProjectedCostResponse{
			UnitPrice:     0,
			Currency:      p.pricing.Currency(),
			CostPerMonth:  0,
			BillingDetail: fmt.Sprintf("EC2 %s pricing not found in region %s", instanceType, p.region),
		}, nil
	}

	monthlyC cost := hourlyRate * hoursPerMonth

	return &pbc.GetProjectedCostResponse{
		UnitPrice:    hourlyRate,
		Currency:     p.pricing.Currency(),
		CostPerMonth: monthlyCost,
		BillingDetail: fmt.Sprintf(
			"On-demand %s/%s instance, %s, %.0f hrs/month",
			operatingSystem, tenancy, instanceType, hoursPerMonth,
		),
	}, nil
}
```

## 2. Implement GetProjectedCost() for EBS

Add EBS estimation logic to `internal/plugin/projected.go`:

```go
// estimateEBS calculates cost for EBS volumes
func (p *AWSPublicPlugin) estimateEBS(
	ctx context.Context,
	rd *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	// sku contains volume type (e.g., "gp3", "gp2", "io1", "io2")
	volumeType := rd.Sku
	if volumeType == "" {
		return nil, status.Error(codes.InvalidArgument, "EBS resource missing sku (volume type)")
	}

	// Size from tags (look for "size" or "volume_size")
	var sizeGB float64
	var sizeAssigned bool

	if sizeStr, ok := rd.Tags["size"]; ok {
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			sizeGB = size
			sizeAssigned = true
		}
	} else if sizeStr, ok := rd.Tags["volume_size"]; ok {
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			sizeGB = size
			sizeAssigned = true
		}
	}

	// Default to 8 GB if not specified
	if !sizeAssigned {
		sizeGB = 8.0
	}

	// Lookup pricing
	ratePerGBMonth, found := p.pricing.EBSPricePerGBMonth(volumeType)
	if !found {
		return &pbc.GetProjectedCostResponse{
			UnitPrice:     0,
			Currency:      p.pricing.Currency(),
			CostPerMonth:  0,
			BillingDetail: fmt.Sprintf("EBS %s pricing not found in region %s", volumeType, p.region),
		}, nil
	}

	monthlyCost := sizeGB * ratePerGBMonth

	billingDetail := fmt.Sprintf("EBS %s storage, %.0fGB", volumeType, sizeGB)
	if !sizeAssigned {
		billingDetail += ", defaulted to 8GB"
	}

	return &pbc.GetProjectedCostResponse{
		UnitPrice:    ratePerGBMonth,
		Currency:     p.pricing.Currency(),
		CostPerMonth: monthlyCost,
		BillingDetail: billingDetail,
	}, nil
}
```

## 3. Implement stub services

Add stub support for S3, Lambda, RDS, DynamoDB:

```go
// estimateStub returns $0 cost for unimplemented services
func (p *AWSPublicPlugin) estimateStub(
	ctx context.Context,
	rd *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{
		UnitPrice:     0,
		Currency:      p.pricing.Currency(),
		CostPerMonth:  0,
		BillingDetail: fmt.Sprintf("%s cost estimation not implemented - returning $0", rd.ResourceType),
	}, nil
}
```

## 4. Implement Supports() logic

Update `internal/plugin/supports.go`:

```go
package plugin

import (
	"context"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
)

// Supports checks if the plugin can estimate costs for a given resource
func (p *AWSPublicPlugin) Supports(
	ctx context.Context,
	req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
	if req == nil || req.Resource == nil {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    "Missing resource descriptor",
		}, nil
	}

	rd := req.Resource

	// Check provider
	if rd.Provider != "aws" {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    "Plugin only supports AWS resources",
		}, nil
	}

	// Check region
	if rd.Region != p.region {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Region %s not supported by this binary (compiled for %s)", rd.Region, p.region),
		}, nil
	}

	// Check resource type
	switch rd.ResourceType {
	case "ec2", "ebs":
		return &pbc.SupportsResponse{
			Supported: true,
			Reason:    "Fully supported",
		}, nil
	case "s3", "lambda", "rds", "dynamodb":
		return &pbc.SupportsResponse{
			Supported: true,
			Reason:    "Limited support - returns $0 estimate",
		}, nil
	default:
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("Resource type %s not recognized", rd.ResourceType),
		}, nil
	}
}
```

## 5. Testing

Add tests for the estimation logic in `internal/plugin/projected_test.go`:

```go
package plugin

import (
	"context"
	"testing"

	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPricingClient implements PricingClient for testing
type mockPricingClient struct {
	region   string
	currency string
	ec2Prices map[string]float64  // key: instanceType/os/tenancy
	ebsPrices map[string]float64  // key: volumeType
}

func (m *mockPricingClient) Region() string {
	return m.region
}

func (m *mockPricingClient) Currency() string {
	return m.currency
}

func (m *mockPricingClient) EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool) {
	key := instanceType + "/" + os + "/" + tenancy
	price, found := m.ec2Prices[key]
	return price, found
}

func (m *mockPricingClient) EBSPricePerGBMonth(volumeType string) (float64, bool) {
	price, found := m.ebsPrices[volumeType]
	return price, found
}

func TestGetProjectedCost_EC2(t *testing.T) {
	pricing := &mockPricingClient{
		region:   "us-east-1",
		currency: "USD",
		ec2Prices: map[string]float64{
			"t3.micro/Linux/Shared": 0.0104,
		},
		ebsPrices: map[string]float64{},
	}

	p := NewAWSPublicPlugin("us-east-1", pricing)

	resp, err := p.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, 0.0104, resp.UnitPrice)
	assert.Equal(t, "USD", resp.Currency)
	assert.InDelta(t, 7.592, resp.CostPerMonth, 0.01) // 0.0104 * 730
	assert.Contains(t, resp.BillingDetail, "On-demand")
	assert.Contains(t, resp.BillingDetail, "t3.micro")
}

func TestGetProjectedCost_EBS(t *testing.T) {
	pricing := &mockPricingClient{
		region:   "us-east-1",
		currency: "USD",
		ec2Prices: map[string]float64{},
		ebsPrices: map[string]float64{
			"gp3": 0.08,
		},
	}

	p := NewAWSPublicPlugin("us-east-1", pricing)

	resp, err := p.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ebs",
			Sku:          "gp3",
			Region:       "us-east-1",
			Tags: map[string]string{
				"size": "100",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, 0.08, resp.UnitPrice)
	assert.Equal(t, "USD", resp.Currency)
	assert.Equal(t, 8.0, resp.CostPerMonth) // 100 * 0.08
	assert.Contains(t, resp.BillingDetail, "gp3")
	assert.Contains(t, resp.BillingDetail, "100GB")
}

func TestSupports_EC2(t *testing.T) {
	pricing := &mockPricingClient{region: "us-east-1", currency: "USD"}
	p := NewAWSPublicPlugin("us-east-1", pricing)

	resp, err := p.Supports(context.Background(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	})

	require.NoError(t, err)
	assert.True(t, resp.Supported)
	assert.Contains(t, resp.Reason, "Fully supported")
}

func TestSupports_WrongRegion(t *testing.T) {
	pricing := &mockPricingClient{region: "us-east-1", currency: "USD"}
	p := NewAWSPublicPlugin("us-east-1", pricing)

	resp, err := p.Supports(context.Background(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-west-2",
		},
	})

	require.NoError(t, err)
	assert.False(t, resp.Supported)
	assert.Contains(t, resp.Reason, "not supported by this binary")
}
```

## Acceptance criteria

- `go test ./...` passes with estimation tests included
- `go build ./...` succeeds
- GetProjectedCost() correctly returns costs for EC2 and EBS when pricing data is available
- GetProjectedCost() returns $0 for stub services with appropriate billing_detail
- Supports() correctly identifies supported/unsupported resources and regions

Implement these changes now.
