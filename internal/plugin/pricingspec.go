package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// GetPricingSpec returns detailed pricing specification for a resource type.
// This provides information about how a resource is billed without calculating the actual cost.
func (p *AWSPublicPlugin) GetPricingSpec(ctx context.Context, req *pbc.GetPricingSpecRequest) (*pbc.GetPricingSpecResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	// FR-009, FR-010: Use SDK validation + custom region check (US2)
	// GetPricingSpecRequest wraps GetProjectedCostRequest internally
	projReq := &pbc.GetProjectedCostRequest{Resource: nil}
	if req != nil {
		projReq.Resource = req.Resource
	}
	if _, err := p.ValidateProjectedCostRequest(ctx, projReq); err != nil {
		p.traceLogger(traceID, "GetPricingSpec").Error().
			Err(err).
			Msg("validation failed")
		return nil, err
	}

	resource := req.Resource

	// Normalize resource type (handles Pulumi formats like aws:ec2/instance:Instance)
	normalizedResourceType := normalizeResourceType(resource.ResourceType)
	serviceType := detectService(normalizedResourceType)

	var spec *pbc.PricingSpec

	switch serviceType {
	case "ec2":
		spec = p.ec2PricingSpec(resource)
	case "ebs":
		spec = p.ebsPricingSpec(resource)
	case "s3":
		spec = p.s3PricingSpec(resource)
	case "lambda":
		spec = p.lambdaPricingSpec(resource)
	case "rds":
		spec = p.rdsPricingSpec(resource)
	case "dynamodb":
		spec = p.dynamoDBPricingSpec(resource)
	case "eks":
		spec = p.eksPricingSpec(resource)
	case "elb", "alb", "nlb":
		spec = p.elbPricingSpec(resource)
	case "natgw", "nat_gateway", "nat-gateway":
		spec = p.natGatewayPricingSpec(resource)
	case "cloudwatch":
		spec = p.cloudWatchPricingSpec(resource)
	default:
		spec = &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          resource.Sku,
			Region:       resource.Region,
			BillingMode:  "unknown",
			RatePerUnit:  0,
			Currency:     "USD",
			Description:  fmt.Sprintf("Resource type %q not supported for pricing specification", resource.ResourceType),
			Source:       "aws-public",
		}
	}

	p.traceLogger(traceID, "GetPricingSpec").Info().
		Str(pluginsdk.FieldResourceType, resource.ResourceType).
		Str("aws_region", resource.Region).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("pricing spec retrieved")

	return &pbc.GetPricingSpecResponse{
		Spec: spec,
	}, nil
}

// ec2PricingSpec returns the pricing specification for an EC2 instance.
func (p *AWSPublicPlugin) ec2PricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	instanceType := resource.Sku
	os := "Linux"
	tenancy := "Shared"

	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, os, tenancy)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          resource.Sku,
			Region:       resource.Region,
			BillingMode:  "per_hour",
			RatePerUnit:  0,
			Currency:     "USD",
			Unit:         "hour",
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", instanceType),
			Source:       "aws-public",
			Assumptions:  []string{"Instance type not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          resource.Sku,
		Region:       resource.Region,
		BillingMode:  "per_hour",
		RatePerUnit:  hourlyRate,
		Currency:     "USD",
		Unit:         "hour",
		Description:  fmt.Sprintf("On-demand %s EC2 instance with %s tenancy", os, tenancy),
		Source:       "aws-public",
		Assumptions: []string{
			fmt.Sprintf("Operating System: %s", os),
			fmt.Sprintf("Tenancy: %s", tenancy),
			"Pre-installed software: None",
			"Capacity Status: Used",
		},
	}
}

// ebsPricingSpec returns the pricing specification for an EBS volume.
func (p *AWSPublicPlugin) ebsPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	volumeType := resource.Sku

	ratePerGBMonth, found := p.pricing.EBSPricePerGBMonth(volumeType)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          resource.Sku,
			Region:       resource.Region,
			BillingMode:  "per_gb_month",
			RatePerUnit:  0,
			Currency:     "USD",
			Unit:         "GB-month",
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "EBS volume type", volumeType),
			Source:       "aws-public",
			Assumptions:  []string{"Volume type not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          resource.Sku,
		Region:       resource.Region,
		BillingMode:  "per_gb_month",
		RatePerUnit:  ratePerGBMonth,
		Currency:     "USD",
		Unit:         "GB-month",
		Description:  fmt.Sprintf("EBS %s storage", volumeType),
		Source:       "aws-public",
		Assumptions: []string{
			"Storage only (IOPS/throughput not included)",
			"Standard provisioned capacity",
		},
	}
}

// s3PricingSpec returns the pricing specification for S3 storage.
func (p *AWSPublicPlugin) s3PricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	storageClass := resource.Sku
	if storageClass == "" {
		storageClass = "STANDARD"
	}

	ratePerGBMonth, found := p.pricing.S3PricePerGBMonth(storageClass)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          storageClass,
			Region:       resource.Region,
			BillingMode:  "per_gb_month",
			RatePerUnit:  0,
			Currency:     "USD",
			Unit:         "GB-month",
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "S3 storage class", storageClass),
			Source:       "aws-public",
			Assumptions:  []string{"Storage class not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          storageClass,
		Region:       resource.Region,
		BillingMode:  "per_gb_month",
		RatePerUnit:  ratePerGBMonth,
		Currency:     "USD",
		Unit:         "GB-month",
		Description:  fmt.Sprintf("S3 %s storage", storageClass),
		Source:       "aws-public",
		Assumptions: []string{
			"Storage cost only",
			"Requests and data transfer billed separately",
			"Lifecycle transitions not included",
		},
	}
}

// lambdaPricingSpec returns the pricing specification for Lambda functions.
func (p *AWSPublicPlugin) lambdaPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	arch := "x86_64"
	if resource.Sku != "" {
		arch = resource.Sku
	}
	if a, ok := resource.Tags["architecture"]; ok && a != "" {
		arch = a
	}

	requestRate, requestFound := p.pricing.LambdaPricePerRequest()
	gbSecRate, gbSecFound := p.pricing.LambdaPricePerGBSecond(arch)

	if !requestFound || !gbSecFound {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          arch,
			Region:       resource.Region,
			BillingMode:  "per_request_and_gb_second",
			RatePerUnit:  0,
			Currency:     "USD",
			Description:  "Lambda pricing not found in embedded data",
			Source:       "aws-public",
			Assumptions:  []string{"Lambda pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          arch,
		Region:       resource.Region,
		BillingMode:  "per_request_and_gb_second",
		RatePerUnit:  gbSecRate, // Primary rate is GB-second (compute)
		Currency:     "USD",
		Unit:         "GB-second",
		Description:  fmt.Sprintf("Lambda %s architecture", arch),
		Source:       "aws-public",
		Assumptions: []string{
			fmt.Sprintf("Request rate: $%.10f per request", requestRate),
			fmt.Sprintf("Compute rate: $%.10f per GB-second (%s)", gbSecRate, arch),
			"Provisioned concurrency not included",
			"Lambda@Edge pricing differs",
		},
	}
}

// rdsPricingSpec returns the pricing specification for RDS instances.
func (p *AWSPublicPlugin) rdsPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	instanceType := resource.Sku
	engine := "mysql"
	if e, ok := resource.Tags["engine"]; ok && e != "" {
		engine = e
	}

	hourlyRate, found := p.pricing.RDSOnDemandPricePerHour(instanceType, engine)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          instanceType,
			Region:       resource.Region,
			BillingMode:  "per_hour",
			RatePerUnit:  0,
			Currency:     "USD",
			Unit:         "hour",
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "RDS instance", instanceType),
			Source:       "aws-public",
			Assumptions:  []string{fmt.Sprintf("Instance type %s with engine %s not found", instanceType, engine)},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          instanceType,
		Region:       resource.Region,
		BillingMode:  "per_hour",
		RatePerUnit:  hourlyRate,
		Currency:     "USD",
		Unit:         "hour",
		Description:  fmt.Sprintf("RDS %s instance with %s engine", instanceType, engine),
		Source:       "aws-public",
		Assumptions: []string{
			fmt.Sprintf("Database engine: %s", engine),
			"Single-AZ deployment",
			"Storage costs billed separately",
			"Backup storage not included",
			"Read replicas billed separately",
		},
	}
}

// dynamoDBPricingSpec returns the pricing specification for DynamoDB tables.
func (p *AWSPublicPlugin) dynamoDBPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	mode := resource.Sku
	if mode == "" {
		mode = "on-demand"
	}

	isProvisioned := mode == "provisioned" || mode == "PROVISIONED"

	if isProvisioned {
		rcuPrice, rcuFound := p.pricing.DynamoDBProvisionedRCUPrice()
		wcuPrice, wcuFound := p.pricing.DynamoDBProvisionedWCUPrice()
		storagePrice, storageFound := p.pricing.DynamoDBStoragePricePerGBMonth()

		if !rcuFound || !wcuFound || !storageFound {
			return &pbc.PricingSpec{
				Provider:     resource.Provider,
				ResourceType: resource.ResourceType,
				Sku:          mode,
				Region:       resource.Region,
				BillingMode:  "provisioned_capacity",
				RatePerUnit:  0,
				Currency:     "USD",
				Description:  "DynamoDB provisioned pricing not found",
				Source:       "aws-public",
				Assumptions:  []string{"Provisioned capacity pricing data not available"},
			}
		}

		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          mode,
			Region:       resource.Region,
			BillingMode:  "provisioned_capacity",
			RatePerUnit:  rcuPrice, // Primary rate is RCU
			Currency:     "USD",
			Unit:         "RCU-hour",
			Description:  "DynamoDB provisioned capacity mode",
			Source:       "aws-public",
			Assumptions: []string{
				fmt.Sprintf("Read Capacity Unit: $%.6f per hour", rcuPrice),
				fmt.Sprintf("Write Capacity Unit: $%.6f per hour", wcuPrice),
				fmt.Sprintf("Storage: $%.4f per GB-month", storagePrice),
				"Auto-scaling adjustments not included",
				"Reserved capacity discounts not applied",
			},
		}
	}

	// On-demand mode
	readPrice, readFound := p.pricing.DynamoDBOnDemandReadPrice()
	writePrice, writeFound := p.pricing.DynamoDBOnDemandWritePrice()
	storagePrice, storageFound := p.pricing.DynamoDBStoragePricePerGBMonth()

	if !readFound || !writeFound || !storageFound {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          mode,
			Region:       resource.Region,
			BillingMode:  "on_demand",
			RatePerUnit:  0,
			Currency:     "USD",
			Description:  "DynamoDB on-demand pricing not found",
			Source:       "aws-public",
			Assumptions:  []string{"On-demand pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          mode,
		Region:       resource.Region,
		BillingMode:  "on_demand",
		RatePerUnit:  storagePrice, // Primary rate for on-demand is storage
		Currency:     "USD",
		Unit:         "GB-month",
		Description:  "DynamoDB on-demand capacity mode",
		Source:       "aws-public",
		Assumptions: []string{
			fmt.Sprintf("Read request units: $%.6f per million", readPrice*1_000_000),
			fmt.Sprintf("Write request units: $%.6f per million", writePrice*1_000_000),
			fmt.Sprintf("Storage: $%.4f per GB-month", storagePrice),
			"Global tables replication costs not included",
			"DynamoDB Streams not included",
		},
	}
}

// eksPricingSpec returns the pricing specification for EKS clusters.
func (p *AWSPublicPlugin) eksPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	supportType := "standard"
	if s, ok := resource.Tags["support_type"]; ok && s != "" {
		supportType = s
	}

	extendedSupport := supportType == "extended"
	hourlyRate, found := p.pricing.EKSClusterPricePerHour(extendedSupport)

	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          supportType,
			Region:       resource.Region,
			BillingMode:  "per_hour",
			RatePerUnit:  0,
			Currency:     "USD",
			Unit:         "hour",
			Description:  "EKS pricing not found in embedded data",
			Source:       "aws-public",
			Assumptions:  []string{"EKS pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          supportType,
		Region:       resource.Region,
		BillingMode:  "per_hour",
		RatePerUnit:  hourlyRate,
		Currency:     "USD",
		Unit:         "hour",
		Description:  fmt.Sprintf("EKS cluster with %s support", supportType),
		Source:       "aws-public",
		Assumptions: []string{
			"Control plane costs only",
			"Worker node EC2 instances billed separately",
			"EKS add-ons may incur additional costs",
			"Data transfer costs not included",
		},
	}
}

// elbPricingSpec returns the pricing specification for Elastic Load Balancers.
func (p *AWSPublicPlugin) elbPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	lbType := resource.Sku
	if lbType == "" {
		lbType = "alb"
	}

	isNLB := lbType == "nlb" || lbType == "NLB" || lbType == "network"

	if isNLB {
		hourlyRate, hourlyFound := p.pricing.NLBPricePerHour()
		nlcuRate, nlcuFound := p.pricing.NLBPricePerNLCU()

		if !hourlyFound || !nlcuFound {
			return &pbc.PricingSpec{
				Provider:     resource.Provider,
				ResourceType: resource.ResourceType,
				Sku:          "nlb",
				Region:       resource.Region,
				BillingMode:  "per_hour_plus_nlcu",
				RatePerUnit:  0,
				Currency:     "USD",
				Description:  "NLB pricing not found in embedded data",
				Source:       "aws-public",
				Assumptions:  []string{"NLB pricing data not available"},
			}
		}

		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          "nlb",
			Region:       resource.Region,
			BillingMode:  "per_hour_plus_nlcu",
			RatePerUnit:  hourlyRate,
			Currency:     "USD",
			Unit:         "hour",
			Description:  "Network Load Balancer",
			Source:       "aws-public",
			Assumptions: []string{
				fmt.Sprintf("Fixed hourly rate: $%.4f", hourlyRate),
				fmt.Sprintf("NLCU rate: $%.4f per NLCU-hour", nlcuRate),
				"Data transfer costs not included",
				"Cross-zone data transfer may incur additional costs",
			},
		}
	}

	// ALB (default)
	hourlyRate, hourlyFound := p.pricing.ALBPricePerHour()
	lcuRate, lcuFound := p.pricing.ALBPricePerLCU()

	if !hourlyFound || !lcuFound {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          "alb",
			Region:       resource.Region,
			BillingMode:  "per_hour_plus_lcu",
			RatePerUnit:  0,
			Currency:     "USD",
			Description:  "ALB pricing not found in embedded data",
			Source:       "aws-public",
			Assumptions:  []string{"ALB pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          "alb",
		Region:       resource.Region,
		BillingMode:  "per_hour_plus_lcu",
		RatePerUnit:  hourlyRate,
		Currency:     "USD",
		Unit:         "hour",
		Description:  "Application Load Balancer",
		Source:       "aws-public",
		Assumptions: []string{
			fmt.Sprintf("Fixed hourly rate: $%.4f", hourlyRate),
			fmt.Sprintf("LCU rate: $%.4f per LCU-hour", lcuRate),
			"Data transfer costs not included",
			"SSL/TLS termination included",
		},
	}
}

// natGatewayPricingSpec returns the pricing specification for NAT Gateways.
func (p *AWSPublicPlugin) natGatewayPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	pricing, found := p.pricing.NATGatewayPrice()

	if !found || pricing == nil {
		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          resource.Sku,
			Region:       resource.Region,
			BillingMode:  "per_hour_plus_data",
			RatePerUnit:  0,
			Currency:     "USD",
			Description:  "NAT Gateway pricing not found in embedded data",
			Source:       "aws-public",
			Assumptions:  []string{"NAT Gateway pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.Provider,
		ResourceType: resource.ResourceType,
		Sku:          resource.Sku,
		Region:       resource.Region,
		BillingMode:  "per_hour_plus_data",
		RatePerUnit:  pricing.HourlyRate,
		Currency:     "USD",
		Unit:         "hour",
		Description:  "NAT Gateway",
		Source:       "aws-public",
		Assumptions: []string{
			fmt.Sprintf("Hourly rate: $%.4f", pricing.HourlyRate),
			fmt.Sprintf("Data processing: $%.4f per GB", pricing.DataProcessingRate),
			"Data transfer OUT to internet billed separately",
			"Cross-AZ data transfer costs not included",
		},
	}
}

// cloudWatchPricingSpec returns the pricing specification for CloudWatch.
func (p *AWSPublicPlugin) cloudWatchPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	sku := resource.Sku
	if sku == "" {
		sku = "logs"
	}

	switch sku {
	case "metrics":
		tiers, found := p.pricing.CloudWatchMetricsTiers()
		if !found || len(tiers) == 0 {
			return &pbc.PricingSpec{
				Provider:     resource.Provider,
				ResourceType: resource.ResourceType,
				Sku:          sku,
				Region:       resource.Region,
				BillingMode:  "tiered_per_metric",
				RatePerUnit:  0,
				Currency:     "USD",
				Description:  "CloudWatch metrics pricing not found",
				Source:       "aws-public",
				Assumptions:  []string{"Metrics pricing data not available"},
			}
		}

		assumptions := []string{"Tiered pricing based on metric count:"}
		prevBound := 0.0
		for _, tier := range tiers {
			if tier.UpTo < 1e15 { // Has an upper bound
				assumptions = append(assumptions, fmt.Sprintf("  %.0f-%.0f metrics: $%.4f/metric", prevBound, tier.UpTo, tier.Rate))
				prevBound = tier.UpTo
			} else { // No upper bound (final tier)
				assumptions = append(assumptions, fmt.Sprintf("  Above %.0f metrics: $%.4f/metric", prevBound, tier.Rate))
			}
		}

		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          sku,
			Region:       resource.Region,
			BillingMode:  "tiered_per_metric",
			RatePerUnit:  tiers[0].Rate, // First tier rate
			Currency:     "USD",
			Unit:         "metric-month",
			Description:  "CloudWatch custom metrics",
			Source:       "aws-public",
			Assumptions:  assumptions,
		}

	default: // logs
		ingestionTiers, ingestionFound := p.pricing.CloudWatchLogsIngestionTiers()
		storagePrice, storageFound := p.pricing.CloudWatchLogsStoragePrice()

		if !ingestionFound || !storageFound {
			return &pbc.PricingSpec{
				Provider:     resource.Provider,
				ResourceType: resource.ResourceType,
				Sku:          "logs",
				Region:       resource.Region,
				BillingMode:  "tiered_ingestion_plus_storage",
				RatePerUnit:  0,
				Currency:     "USD",
				Description:  "CloudWatch logs pricing not found",
				Source:       "aws-public",
				Assumptions:  []string{"Logs pricing data not available"},
			}
		}

		assumptions := []string{
			fmt.Sprintf("Storage: $%.4f per GB-month", storagePrice),
			"Ingestion tiered pricing:",
		}
		prevBound := 0.0
		for _, tier := range ingestionTiers {
			if tier.UpTo < 1e15 { // Has an upper bound
				assumptions = append(assumptions, fmt.Sprintf("  %.0f-%.0f GB: $%.4f/GB", prevBound, tier.UpTo, tier.Rate))
				prevBound = tier.UpTo
			} else { // No upper bound (final tier)
				assumptions = append(assumptions, fmt.Sprintf("  Above %.0f GB: $%.4f/GB", prevBound, tier.Rate))
			}
		}
		assumptions = append(assumptions, "Logs Insights queries billed separately")

		firstTierRate := 0.0
		if len(ingestionTiers) > 0 {
			firstTierRate = ingestionTiers[0].Rate
		}

		return &pbc.PricingSpec{
			Provider:     resource.Provider,
			ResourceType: resource.ResourceType,
			Sku:          "logs",
			Region:       resource.Region,
			BillingMode:  "tiered_ingestion_plus_storage",
			RatePerUnit:  firstTierRate,
			Currency:     "USD",
			Unit:         "GB-ingested",
			Description:  "CloudWatch Logs",
			Source:       "aws-public",
			Assumptions:  assumptions,
		}
	}
}
