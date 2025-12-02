package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

// EstimateCost returns an estimated monthly cost for a resource based on its
// type and configuration attributes. This is the preferred method for pre-deployment
// cost estimation as it works with Pulumi resource types directly.
func (p *AWSPublicPlugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (*pbc.EstimateCostResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	if req == nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "missing request", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "EstimateCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	if req.ResourceType == "" {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, "missing resource_type", pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "EstimateCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Parse Pulumi resource type (e.g., "aws:ec2/instance:Instance")
	resourceInfo, err := parsePulumiResourceType(req.ResourceType)
	if err != nil {
		err := p.newErrorWithID(traceID, codes.InvalidArgument, fmt.Sprintf("invalid resource_type format: %v", err), pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		p.logErrorWithID(traceID, "EstimateCost", err, pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE)
		return nil, err
	}

	// Only support AWS resources
	if resourceInfo.provider != "aws" {
		return &pbc.EstimateCostResponse{
			Currency:    "USD",
			CostMonthly: 0,
		}, nil
	}

	// Extract attributes
	attrs := req.Attributes
	if attrs == nil {
		attrs = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	}

	// Get region from attributes or use plugin's region
	region := p.region
	if regionVal, ok := getStringAttr(attrs, "region"); ok && regionVal != "" {
		region = regionVal
	} else if availZone, ok := getStringAttr(attrs, "availabilityZone"); ok && availZone != "" {
		// Extract region from AZ (e.g., "us-east-1a" -> "us-east-1")
		if len(availZone) > 1 {
			region = availZone[:len(availZone)-1]
		}
	}

	// Check region match
	if region != p.region {
		// Return $0 for wrong region (let the correct plugin handle it)
		return &pbc.EstimateCostResponse{
			Currency:    "USD",
			CostMonthly: 0,
		}, nil
	}

	var costMonthly float64

	switch resourceInfo.module {
	case "ec2":
		costMonthly = p.estimateEC2FromAttrs(traceID, resourceInfo.resource, attrs)
	case "ebs":
		costMonthly = p.estimateEBSFromAttrs(traceID, resourceInfo.resource, attrs)
	default:
		// Unsupported module - return $0
		costMonthly = 0
	}

	p.logger.Info().
		Str(pluginsdk.FieldTraceID, traceID).
		Str(pluginsdk.FieldOperation, "EstimateCost").
		Str("pulumi_type", req.ResourceType).
		Str("aws_region", region).
		Float64(pluginsdk.FieldCostMonthly, costMonthly).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("cost estimated")

	return &pbc.EstimateCostResponse{
		Currency:    "USD",
		CostMonthly: costMonthly,
	}, nil
}

// resourceTypeInfo holds parsed Pulumi resource type information.
type resourceTypeInfo struct {
	provider string // e.g., "aws"
	module   string // e.g., "ec2"
	resource string // e.g., "Instance"
}

// parsePulumiResourceType parses a Pulumi resource type string.
// Format: "provider:module/resource:Type" (e.g., "aws:ec2/instance:Instance")
func parsePulumiResourceType(resourceType string) (*resourceTypeInfo, error) {
	// Split by first ":"
	parts := strings.SplitN(resourceType, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format: expected 'provider:module/resource:Type'")
	}

	provider := parts[0]

	// Split rest by "/" and ":"
	rest := parts[1]
	moduleParts := strings.SplitN(rest, "/", 2)
	if len(moduleParts) != 2 {
		return nil, fmt.Errorf("invalid format: expected module/resource:Type")
	}

	module := moduleParts[0]

	// Split resource:Type
	resourceParts := strings.SplitN(moduleParts[1], ":", 2)
	if len(resourceParts) != 2 {
		return nil, fmt.Errorf("invalid format: expected resource:Type")
	}

	return &resourceTypeInfo{
		provider: provider,
		module:   module,
		resource: resourceParts[1],
	}, nil
}

// getStringAttr extracts a string attribute from a protobuf Struct.
func getStringAttr(attrs *structpb.Struct, key string) (string, bool) {
	if attrs == nil || attrs.Fields == nil {
		return "", false
	}
	if val, ok := attrs.Fields[key]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			return strVal, true
		}
	}
	return "", false
}

// getNumberAttr extracts a number attribute from a protobuf Struct.
func getNumberAttr(attrs *structpb.Struct, key string) (float64, bool) {
	if attrs == nil || attrs.Fields == nil {
		return 0, false
	}
	if val, ok := attrs.Fields[key]; ok {
		if numVal := val.GetNumberValue(); numVal != 0 {
			return numVal, true
		}
		// Also try string conversion
		if strVal := val.GetStringValue(); strVal != "" {
			if num, err := strconv.ParseFloat(strVal, 64); err == nil {
				return num, true
			}
		}
	}
	return 0, false
}

// estimateEC2FromAttrs calculates EC2 cost from Pulumi attributes.
func (p *AWSPublicPlugin) estimateEC2FromAttrs(traceID, resourceName string, attrs *structpb.Struct) float64 {
	// Only handle Instance resources
	if resourceName != "Instance" {
		return 0
	}

	// Get instance type from attributes
	instanceType, ok := getStringAttr(attrs, "instanceType")
	if !ok || instanceType == "" {
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "EstimateCost").
			Msg("EC2 instance missing instanceType attribute")
		return 0
	}

	// Extract OS and tenancy from attributes, with fallbacks
	os := "Linux"       // Default fallback
	tenancy := "Shared" // Default fallback

	// Check for platform information in attributes
	if platform, ok := getStringAttr(attrs, "platform"); ok && platform != "" {
		// AWS uses "windows" for Windows platforms, otherwise assume Linux-based
		if strings.ToLower(platform) == "windows" {
			os = "Windows"
		} else {
			os = "Linux" // Treat other platforms as Linux-based
		}
	}

	// Check for tenancy information in attributes
	if tenancyAttr, ok := getStringAttr(attrs, "tenancy"); ok && tenancyAttr != "" {
		// Validate tenancy values (AWS supports Shared, Dedicated, Host)
		switch strings.ToLower(tenancyAttr) {
		case "dedicated":
			tenancy = "Dedicated"
		case "host":
			tenancy = "Host"
		default:
			tenancy = "Shared" // Default to Shared for any other value
		}
	}

	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, os, tenancy)
	if !found {
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "EstimateCost").
			Str("instance_type", instanceType).
			Msg("EC2 instance type not found in pricing data")
		return 0
	}

	return hourlyRate * hoursPerMonth
}

// estimateEBSFromAttrs calculates EBS cost from Pulumi attributes.
func (p *AWSPublicPlugin) estimateEBSFromAttrs(traceID, resourceName string, attrs *structpb.Struct) float64 {
	// Handle Volume resources
	if resourceName != "Volume" {
		return 0
	}

	// Get volume type from attributes (default to gp2)
	volumeType, ok := getStringAttr(attrs, "type")
	if !ok || volumeType == "" {
		volumeType = "gp2"
	}

	// Get size from attributes (default to 8 GB)
	sizeGB := float64(defaultEBSGB)
	if size, ok := getNumberAttr(attrs, "size"); ok && size > 0 {
		sizeGB = size
	}

	ratePerGBMonth, found := p.pricing.EBSPricePerGBMonth(volumeType)
	if !found {
		p.logger.Debug().
			Str(pluginsdk.FieldTraceID, traceID).
			Str(pluginsdk.FieldOperation, "EstimateCost").
			Str("volume_type", volumeType).
			Msg("EBS volume type not found in pricing data")
		return 0
	}

	return ratePerGBMonth * sizeGB
}
