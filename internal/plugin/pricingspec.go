package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// PricingSpec billing modes describing how each service is charged.
const (
	billingModePerHour             = "per_hour"
	billingModePerHourPlusLCU      = "per_hour_plus_lcu"
	billingModePerHourPlusNLCU     = "per_hour_plus_nlcu"
	billingModePerHourPlusData     = "per_hour_plus_data"
	billingModePerGBMonth          = "per_gb_month"
	billingModePerRequestAndGBSec  = "per_request_and_gb_second"
	billingModeProvisionedCapacity = "provisioned_capacity"
	billingModeOnDemand            = "on_demand"
	billingModeTieredPerMetric     = "tiered_per_metric"
	billingModeTieredIngestStorage = "tiered_ingestion_plus_storage"
	billingModeZeroCost            = "zero_cost"
	billingModeUnknown             = "unknown"
)

// PricingSpec rate units.
const (
	unitHour            = "hour"
	unitGBMonth         = "GB-month"
	unitGBSecond        = "GB-second"
	unitGBIngested      = "GB-ingested"
	unitRCUHour         = "RCU-hour"
	unitMetricMonth     = "metric-month"
	unitPerInstanceHour = "per-instance-hour"
)

// assumptionDataTransferExcluded is the shared assumption note for services whose
// data transfer charges are out of scope.
const assumptionDataTransferExcluded = "Data transfer costs not included"

// GetPricingSpec returns detailed pricing specification for a resource type.
// This provides information about how a resource is billed without calculating the actual cost.
func (p *AWSPublicPlugin) GetPricingSpec(
	ctx context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	start := time.Now()
	traceID := p.getTraceID(ctx)

	// FR-009, FR-010: Use SDK validation + custom region check (US2)
	// GetPricingSpecRequest wraps GetProjectedCostRequest internally
	projReq := &pbc.GetProjectedCostRequest{Resource: nil}
	if req != nil {
		projReq.Resource = req.GetResource()
	}
	if _, err := p.ValidateProjectedCostRequest(ctx, projReq); err != nil {
		p.traceLogger(traceID, "GetPricingSpec").Error().
			Err(err).
			Msg("validation failed")
		return nil, err
	}

	resource := req.GetResource()

	// Use serviceResolver for consistent normalization (optimization: compute once per request)
	resolver := newServiceResolver(resource.GetResourceType())
	serviceType := resolver.ServiceType()

	var spec *pbc.PricingSpec

	switch serviceType {
	case serviceEC2:
		spec = p.ec2PricingSpec(resource)
	case serviceEBS:
		spec = p.ebsPricingSpec(resource)
	case serviceS3:
		spec = p.s3PricingSpec(resource)
	case serviceLambda:
		spec = p.lambdaPricingSpec(resource)
	case serviceRDS:
		spec = p.rdsPricingSpec(resource)
	case serviceDynamoDB:
		spec = p.dynamoDBPricingSpec(resource)
	case serviceEKS:
		spec = p.eksPricingSpec(resource)
	case serviceELB, serviceALB, serviceNLB:
		spec = p.elbPricingSpec(resource)
	case serviceNATGW:
		spec = p.natGatewayPricingSpec(resource)
	case serviceCloudWatch:
		spec = p.cloudWatchPricingSpec(resource)
	case serviceASG:
		spec = p.asgPricingSpec(resource)
	case serviceVPC, serviceSecurityGroup, serviceSubnet, serviceIAM, serviceLaunchTmpl, serviceLaunchConfig:
		spec = p.zeroCostPricingSpec(resource, serviceType)
	default:
		spec = &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          resource.GetSku(),
			Region:       resource.GetRegion(),
			BillingMode:  billingModeUnknown,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Description: fmt.Sprintf(
				"Resource type %q not supported for pricing specification",
				resource.GetResourceType(),
			),
			Source: sourceAWSPublic,
		}
	}

	p.traceLogger(traceID, "GetPricingSpec").Info().
		Str(pluginsdk.FieldResourceType, resource.GetResourceType()).
		Str("aws_region", resource.GetRegion()).
		Int64(pluginsdk.FieldDurationMs, time.Since(start).Milliseconds()).
		Msg("pricing spec retrieved")

	return &pbc.GetPricingSpecResponse{
		Spec: spec,
	}, nil
}

// ec2PricingSpec returns the pricing specification for an EC2 instance.
func (p *AWSPublicPlugin) ec2PricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	instanceType := resource.GetSku()
	os := defaultOS
	tenancy := defaultTenancy

	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, os, tenancy)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          resource.GetSku(),
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerHour,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Unit:         unitHour,
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", instanceType),
			Source:       sourceAWSPublic,
			Assumptions:  []string{"Instance type not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerHour,
		RatePerUnit:  hourlyRate,
		Currency:     currencyUSD,
		Unit:         unitHour,
		Description:  fmt.Sprintf("On-demand %s EC2 instance with %s tenancy", os, tenancy),
		Source:       sourceAWSPublic,
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
	volumeType := resource.GetSku()

	ratePerGBMonth, found := p.pricing.EBSPricePerGBMonth(volumeType)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          resource.GetSku(),
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerGBMonth,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Unit:         unitGBMonth,
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "EBS volume type", volumeType),
			Source:       sourceAWSPublic,
			Assumptions:  []string{"Volume type not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerGBMonth,
		RatePerUnit:  ratePerGBMonth,
		Currency:     currencyUSD,
		Unit:         unitGBMonth,
		Description:  fmt.Sprintf("EBS %s storage", volumeType),
		Source:       sourceAWSPublic,
		Assumptions: []string{
			"Storage only (IOPS/throughput not included)",
			"Standard provisioned capacity",
		},
	}
}

// s3PricingSpec returns the pricing specification for S3 storage.
func (p *AWSPublicPlugin) s3PricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	storageClass := resource.GetSku()
	if storageClass == "" {
		storageClass = "STANDARD"
	}

	ratePerGBMonth, found := p.pricing.S3PricePerGBMonth(storageClass)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          storageClass,
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerGBMonth,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Unit:         unitGBMonth,
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "S3 storage class", storageClass),
			Source:       sourceAWSPublic,
			Assumptions:  []string{"Storage class not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          storageClass,
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerGBMonth,
		RatePerUnit:  ratePerGBMonth,
		Currency:     currencyUSD,
		Unit:         unitGBMonth,
		Description:  fmt.Sprintf("S3 %s storage", storageClass),
		Source:       sourceAWSPublic,
		Assumptions: []string{
			"Storage cost only",
			"Requests and data transfer billed separately",
			"Lifecycle transitions not included",
		},
	}
}

// lambdaPricingSpec returns the pricing specification for Lambda functions.
func (p *AWSPublicPlugin) lambdaPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	arch := archX86
	if resource.GetSku() != "" {
		arch = resource.GetSku()
	}
	if a, ok := resource.GetTags()["architecture"]; ok && a != "" {
		arch = a
	}

	requestRate, requestFound := p.pricing.LambdaPricePerRequest()
	gbSecRate, gbSecFound := p.pricing.LambdaPricePerGBSecond(arch)

	if !requestFound || !gbSecFound {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          arch,
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerRequestAndGBSec,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Description:  "Lambda pricing not found in embedded data",
			Source:       sourceAWSPublic,
			Assumptions:  []string{"Lambda pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          arch,
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerRequestAndGBSec,
		RatePerUnit:  gbSecRate, // Primary rate is GB-second (compute)
		Currency:     currencyUSD,
		Unit:         unitGBSecond,
		Description:  fmt.Sprintf("Lambda %s architecture", arch),
		Source:       sourceAWSPublic,
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
	instanceType := resource.GetSku()
	engine := defaultRDSEngine
	if e, ok := resource.GetTags()["engine"]; ok && e != "" {
		engine = e
	}

	hourlyRate, found := p.pricing.RDSOnDemandPricePerHour(instanceType, engine)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          instanceType,
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerHour,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Unit:         unitHour,
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "RDS instance", instanceType),
			Source:       sourceAWSPublic,
			Assumptions:  []string{fmt.Sprintf("Instance type %s with engine %s not found", instanceType, engine)},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          instanceType,
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerHour,
		RatePerUnit:  hourlyRate,
		Currency:     currencyUSD,
		Unit:         unitHour,
		Description:  fmt.Sprintf("RDS %s instance with %s engine", instanceType, engine),
		Source:       sourceAWSPublic,
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
	mode := resource.GetSku()
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
				Provider:     resource.GetProvider(),
				ResourceType: resource.GetResourceType(),
				Sku:          mode,
				Region:       resource.GetRegion(),
				BillingMode:  billingModeProvisionedCapacity,
				RatePerUnit:  0,
				Currency:     currencyUSD,
				Description:  "DynamoDB provisioned pricing not found",
				Source:       sourceAWSPublic,
				Assumptions:  []string{"Provisioned capacity pricing data not available"},
			}
		}

		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          mode,
			Region:       resource.GetRegion(),
			BillingMode:  billingModeProvisionedCapacity,
			RatePerUnit:  rcuPrice, // Primary rate is RCU
			Currency:     currencyUSD,
			Unit:         unitRCUHour,
			Description:  "DynamoDB provisioned capacity mode",
			Source:       sourceAWSPublic,
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
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          mode,
			Region:       resource.GetRegion(),
			BillingMode:  billingModeOnDemand,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Description:  "DynamoDB on-demand pricing not found",
			Source:       sourceAWSPublic,
			Assumptions:  []string{"On-demand pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          mode,
		Region:       resource.GetRegion(),
		BillingMode:  billingModeOnDemand,
		RatePerUnit:  storagePrice, // Primary rate for on-demand is storage
		Currency:     currencyUSD,
		Unit:         unitGBMonth,
		Description:  "DynamoDB on-demand capacity mode",
		Source:       sourceAWSPublic,
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
	if s, ok := resource.GetTags()["support_type"]; ok && s != "" {
		supportType = s
	}

	extendedSupport := supportType == "extended"
	hourlyRate, found := p.pricing.EKSClusterPricePerHour(extendedSupport)

	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          supportType,
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerHour,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Unit:         unitHour,
			Description:  "EKS pricing not found in embedded data",
			Source:       sourceAWSPublic,
			Assumptions:  []string{"EKS pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          supportType,
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerHour,
		RatePerUnit:  hourlyRate,
		Currency:     currencyUSD,
		Unit:         unitHour,
		Description:  fmt.Sprintf("EKS cluster with %s support", supportType),
		Source:       sourceAWSPublic,
		Assumptions: []string{
			"Control plane costs only",
			"Worker node EC2 instances billed separately",
			"EKS add-ons may incur additional costs",
			assumptionDataTransferExcluded,
		},
	}
}

// elbPricingSpec returns the pricing specification for Elastic Load Balancers.
func (p *AWSPublicPlugin) elbPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	lbType := resource.GetSku()
	if lbType == "" {
		lbType = serviceALB
	}

	isNLB := lbType == serviceNLB || lbType == "NLB" || lbType == "network"

	if isNLB {
		hourlyRate, hourlyFound := p.pricing.NLBPricePerHour()
		nlcuRate, nlcuFound := p.pricing.NLBPricePerNLCU()

		if !hourlyFound || !nlcuFound {
			return &pbc.PricingSpec{
				Provider:     resource.GetProvider(),
				ResourceType: resource.GetResourceType(),
				Sku:          serviceNLB,
				Region:       resource.GetRegion(),
				BillingMode:  billingModePerHourPlusNLCU,
				RatePerUnit:  0,
				Currency:     currencyUSD,
				Description:  "NLB pricing not found in embedded data",
				Source:       sourceAWSPublic,
				Assumptions:  []string{"NLB pricing data not available"},
			}
		}

		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          serviceNLB,
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerHourPlusNLCU,
			RatePerUnit:  hourlyRate,
			Currency:     currencyUSD,
			Unit:         unitHour,
			Description:  "Network Load Balancer",
			Source:       sourceAWSPublic,
			Assumptions: []string{
				fmt.Sprintf("Fixed hourly rate: $%.4f", hourlyRate),
				fmt.Sprintf("NLCU rate: $%.4f per NLCU-hour", nlcuRate),
				assumptionDataTransferExcluded,
				"Cross-zone data transfer may incur additional costs",
			},
		}
	}

	// ALB (default)
	hourlyRate, hourlyFound := p.pricing.ALBPricePerHour()
	lcuRate, lcuFound := p.pricing.ALBPricePerLCU()

	if !hourlyFound || !lcuFound {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          serviceALB,
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerHourPlusLCU,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Description:  "ALB pricing not found in embedded data",
			Source:       sourceAWSPublic,
			Assumptions:  []string{"ALB pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          serviceALB,
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerHourPlusLCU,
		RatePerUnit:  hourlyRate,
		Currency:     currencyUSD,
		Unit:         unitHour,
		Description:  "Application Load Balancer",
		Source:       sourceAWSPublic,
		Assumptions: []string{
			fmt.Sprintf("Fixed hourly rate: $%.4f", hourlyRate),
			fmt.Sprintf("LCU rate: $%.4f per LCU-hour", lcuRate),
			assumptionDataTransferExcluded,
			"SSL/TLS termination included",
		},
	}
}

// natGatewayPricingSpec returns the pricing specification for NAT Gateways.
func (p *AWSPublicPlugin) natGatewayPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	pricing, found := p.pricing.NATGatewayPrice()

	if !found || pricing == nil {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          resource.GetSku(),
			Region:       resource.GetRegion(),
			BillingMode:  billingModePerHourPlusData,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Description:  "NAT Gateway pricing not found in embedded data",
			Source:       sourceAWSPublic,
			Assumptions:  []string{"NAT Gateway pricing data not available"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerHourPlusData,
		RatePerUnit:  pricing.HourlyRate,
		Currency:     currencyUSD,
		Unit:         unitHour,
		Description:  "NAT Gateway",
		Source:       sourceAWSPublic,
		Assumptions: []string{
			fmt.Sprintf("Hourly rate: $%.4f", pricing.HourlyRate),
			fmt.Sprintf("Data processing: $%.4f per GB", pricing.DataProcessingRate),
			"Data transfer OUT to internet billed separately",
			"Cross-AZ data transfer costs not included",
		},
	}
}

// cloudWatchPricingSpec returns the pricing specification for CloudWatch.
func (p *AWSPublicPlugin) cloudWatchPricingSpec( //nolint:gocognit,funlen
	resource *pbc.ResourceDescriptor,
) *pbc.PricingSpec {
	sku := resource.GetSku()
	if sku == "" {
		sku = skuLogs
	}

	switch sku {
	case skuMetrics:
		tiers, found := p.pricing.CloudWatchMetricsTiers()
		if !found || len(tiers) == 0 {
			return &pbc.PricingSpec{
				Provider:     resource.GetProvider(),
				ResourceType: resource.GetResourceType(),
				Sku:          sku,
				Region:       resource.GetRegion(),
				BillingMode:  billingModeTieredPerMetric,
				RatePerUnit:  0,
				Currency:     currencyUSD,
				Description:  "CloudWatch metrics pricing not found",
				Source:       sourceAWSPublic,
				Assumptions:  []string{"Metrics pricing data not available"},
			}
		}

		assumptions := []string{"Tiered pricing based on metric count:"}
		prevBound := 0.0
		for _, tier := range tiers {
			if tier.UpTo < 1e15 { // Has an upper bound
				assumptions = append(
					assumptions,
					fmt.Sprintf("  %.0f-%.0f metrics: $%.4f/metric", prevBound, tier.UpTo, tier.Rate),
				)
				prevBound = tier.UpTo
			} else { // No upper bound (final tier)
				assumptions = append(
					assumptions,
					fmt.Sprintf("  Above %.0f metrics: $%.4f/metric", prevBound, tier.Rate),
				)
			}
		}

		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          sku,
			Region:       resource.GetRegion(),
			BillingMode:  billingModeTieredPerMetric,
			RatePerUnit:  tiers[0].Rate, // First tier rate
			Currency:     currencyUSD,
			Unit:         unitMetricMonth,
			Description:  "CloudWatch custom metrics",
			Source:       sourceAWSPublic,
			Assumptions:  assumptions,
		}

	default: // logs
		ingestionTiers, ingestionFound := p.pricing.CloudWatchLogsIngestionTiers()
		storagePrice, storageFound := p.pricing.CloudWatchLogsStoragePrice()

		if !ingestionFound || !storageFound {
			return &pbc.PricingSpec{
				Provider:     resource.GetProvider(),
				ResourceType: resource.GetResourceType(),
				Sku:          skuLogs,
				Region:       resource.GetRegion(),
				BillingMode:  billingModeTieredIngestStorage,
				RatePerUnit:  0,
				Currency:     currencyUSD,
				Description:  "CloudWatch logs pricing not found",
				Source:       sourceAWSPublic,
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
				assumptions = append(
					assumptions,
					fmt.Sprintf("  %.0f-%.0f GB: $%.4f/GB", prevBound, tier.UpTo, tier.Rate),
				)
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
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          skuLogs,
			Region:       resource.GetRegion(),
			BillingMode:  billingModeTieredIngestStorage,
			RatePerUnit:  firstTierRate,
			Currency:     currencyUSD,
			Unit:         unitGBIngested,
			Description:  "CloudWatch Logs",
			Source:       sourceAWSPublic,
			Assumptions:  assumptions,
		}
	}
}

// zeroCostPricingSpec returns a pricing specification for AWS resources with no direct charges.
// These include VPC, Security Groups, Subnets, IAM resources, Launch Templates, and Launch Configurations.
// The description is sourced from the shared zeroCostResourceDescriptions map for consistency
// with GetProjectedCost and GetActualCost responses.
func (p *AWSPublicPlugin) zeroCostPricingSpec(
	resource *pbc.ResourceDescriptor,
	serviceType string,
) *pbc.PricingSpec {
	description, ok := zeroCostResourceDescriptions[serviceType]
	if !ok {
		description = fmt.Sprintf("%s has no direct AWS charge", serviceType)
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModeZeroCost,
		RatePerUnit:  0,
		Currency:     currencyUSD,
		Description:  description,
		Source:       sourceAWSPublic,
		Assumptions:  []string{"No direct AWS charges for this resource type"},
	}
}

// asgPricingSpec returns the pricing specification for Auto Scaling Groups.
// ASGs delegate to EC2 instance pricing; the total cost is the per-instance rate × desired capacity.
func (p *AWSPublicPlugin) asgPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	instanceType := resource.GetSku()
	if instanceType == "" {
		if tags := resource.GetTags(); tags != nil {
			instanceType = resolveInstanceType("", tags)
		}
	}

	if instanceType == "" {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Region:       resource.GetRegion(),
			BillingMode:  billingModeOnDemand,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Unit:         unitPerInstanceHour,
			Description:  "ASG instance type not specified: set 'sku' field or 'instance_type' tag",
			Source:       sourceAWSPublic,
		}
	}

	os := resolveASGOS(resource.GetTags())

	hourlyRate, found := p.pricing.EC2OnDemandPricePerHour(instanceType, os, defaultTenancy)
	if !found {
		return &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          instanceType,
			Region:       resource.GetRegion(),
			BillingMode:  billingModeOnDemand,
			RatePerUnit:  0,
			Currency:     currencyUSD,
			Unit:         unitPerInstanceHour,
			Description:  fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", instanceType),
			Source:       sourceAWSPublic,
			Assumptions:  []string{"Instance type not found in embedded pricing data"},
		}
	}

	return &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          instanceType,
		Region:       resource.GetRegion(),
		BillingMode:  billingModeOnDemand,
		RatePerUnit:  hourlyRate,
		Currency:     currencyUSD,
		Unit:         unitPerInstanceHour,
		Description:  fmt.Sprintf("ASG cost = %s hourly rate × desired_capacity × 730 hrs/month", instanceType),
		Source:       sourceAWSPublic,
		Assumptions: []string{
			fmt.Sprintf("Instance type: %s", instanceType),
			fmt.Sprintf("On-demand %s, shared tenancy", os),
			"Cost = per-instance hourly rate × desired_capacity × 730",
			"Worker instance costs only (ASG has no separate charge)",
		},
	}
}
