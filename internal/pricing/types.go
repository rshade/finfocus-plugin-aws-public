package pricing

// awsPricing represents the structure of the official AWS Price List API JSON response.
// It contains metadata, products catalog, and pricing terms.
type awsPricing struct {
	FormatVersion   string                                `json:"formatVersion"`
	Disclaimer      string                                `json:"disclaimer"`
	OfferCode       string                                `json:"offerCode"`
	Version         string                                `json:"version"`
	PublicationDate string                                `json:"publicationDate"`
	Products        map[string]product                    `json:"products"`
	Terms           map[string]map[string]map[string]term `json:"terms"` // Type -> SKU -> OfferTermCode -> Term
}

// product represents an AWS product entry in the pricing data.
// Each product has a SKU, family classification, and attributes.
type product struct {
	Sku           string            `json:"sku"`
	ProductFamily string            `json:"productFamily"`
	Attributes    map[string]string `json:"attributes"`
}

// term represents a pricing term offer (e.g., OnDemand, Reserved).
// Contains offer details and associated price dimensions.
type term struct {
	OfferTermCode   string                    `json:"offerTermCode"`
	Sku             string                    `json:"sku"`
	EffectiveDate   string                    `json:"effectiveDate"`
	PriceDimensions map[string]priceDimension `json:"priceDimensions"`
}

// priceDimension represents a specific pricing dimension within a term.
// Contains rate information, unit of measure, and price per unit by currency.
type priceDimension struct {
	RateCode     string            `json:"rateCode"`
	Description  string            `json:"description"`
	BeginRange   string            `json:"beginRange"`
	EndRange     string            `json:"endRange"`
	Unit         string            `json:"unit"`
	PricePerUnit map[string]string `json:"pricePerUnit"` // Currency -> Amount (string)
	AppliesTo    []string          `json:"appliesTo"`
}

// ec2Price represents the hourly compute cost for EC2 instances.
// Distilled from raw AWS pricing JSON for fast lookups.
type ec2Price struct {
	Unit       string
	HourlyRate float64
	Currency   string
}

// ebsPrice represents the per-GB-month storage cost for EBS volumes.
// Distilled from raw AWS pricing JSON for fast lookups.
type ebsPrice struct {
	Unit           string
	RatePerGBMonth float64
	Currency       string
}

// s3Price represents the per-GB-month storage cost for S3 buckets.
// Distilled from raw AWS pricing JSON for fast lookups.
type s3Price struct {
	Unit           string
	RatePerGBMonth float64
	Currency       string
}

// rdsInstancePrice represents the hourly compute cost for RDS instances
type rdsInstancePrice struct {
	Unit       string
	HourlyRate float64
	Currency   string
}

// rdsStoragePrice represents the per-GB-month cost for RDS storage
type rdsStoragePrice struct {
	Unit           string
	RatePerGBMonth float64
	Currency       string
}

// eksPrice represents the hourly cost for EKS cluster control plane.
// EKS offers two support tiers with different pricing:
//   - Standard support: ~$0.10/cluster-hour
//   - Extended support: ~$0.50/cluster-hour (for clusters on older Kubernetes versions)
type eksPrice struct {
	Unit               string
	StandardHourlyRate float64 // Standard support hourly rate
	ExtendedHourlyRate float64 // Extended support hourly rate
	Currency           string
}

// lambdaPrice holds the regional pricing configuration for AWS Lambda.
// Derived from AWS Pricing API product families "Serverless" and "AWS Lambda".
// Lambda pricing varies by architecture (x86 vs ARM/Graviton2), with ARM being
// approximately 20% cheaper for compute duration.
type lambdaPrice struct {
	// RequestPrice is the cost per request (same for both architectures).
	// Source: Product Family "AWS Lambda", Group "AWS-Lambda-Requests"
	// Typical rate: $0.20 per 1M requests ($0.0000002 per request)
	RequestPrice float64

	// X86GBSecondPrice is the cost per GB-second for x86_64 architecture.
	// Source: Product Family "Serverless", Group "AWS-Lambda-Duration"
	// Typical rate: ~$0.0000166667 per GB-second
	X86GBSecondPrice float64

	// ARMGBSecondPrice is the cost per GB-second for arm64 (Graviton2) architecture.
	// Source: Product Family "Serverless", Group "AWS-Lambda-Duration-ARM"
	// Typical rate: ~$0.0000133334 per GB-second (~20% cheaper than x86)
	ARMGBSecondPrice float64

	// Currency code (e.g., "USD")
	Currency string
}

// dynamoDBPrice holds the regional pricing configuration for Amazon DynamoDB.
// Derived from AWS Pricing API for service AmazonDynamoDB.
type dynamoDBPrice struct {
	// OnDemandReadPrice is the cost per read request unit.
	// Source: Product Family "Amazon DynamoDB PayPerRequest Throughput", Group "DDB-ReadUnits"
	OnDemandReadPrice float64

	// OnDemandWritePrice is the cost per write request unit.
	// Source: Product Family "Amazon DynamoDB PayPerRequest Throughput", Group "DDB-WriteUnits"
	OnDemandWritePrice float64

	// ProvisionedRCUPrice is the cost per RCU-hour.
	// Source: Product Family "Provisioned IOPS", UsageType containing "ReadCapacityUnit"
	ProvisionedRCUPrice float64

	// ProvisionedWCUPrice is the cost per WCU-hour.
	// Source: Product Family "Provisioned IOPS", UsageType containing "WriteCapacityUnit"
	ProvisionedWCUPrice float64

	// StoragePrice is the cost per GB-month of table storage.
	// Source: Product Family "Database Storage", UsageType containing "TimedStorage-ByteHrs"
	StoragePrice float64

	// Currency code (e.g., "USD")
	Currency string
}

// elbPrice represents the regional pricing for Elastic Load Balancers (ALB and NLB).
// Derived from AWS Pricing API for service AWSELB.
type elbPrice struct {
	// ALBHourlyRate is the fixed hourly cost for an Application Load Balancer.
	ALBHourlyRate float64
	// ALBLCURate is the cost per LCU-hour for an Application Load Balancer.
	ALBLCURate float64
	// NLBHourlyRate is the fixed hourly cost for a Network Load Balancer.
	NLBHourlyRate float64
	// NLBNLCURate is the cost per NLCU-hour for a Network Load Balancer.
	NLBNLCURate float64
	// Currency is the pricing currency (e.g., "USD").
	Currency string
}

// NATGatewayPrice represents the regional pricing for VPC NAT Gateways.
// Derived from AWS Pricing API for service AmazonVPC.
type NATGatewayPrice struct {
	// HourlyRate is the fixed hourly cost for a NAT Gateway.
	// Source: Product Family "NAT Gateway", usageType containing "NatGateway-Hours"
	HourlyRate float64

	// DataProcessingRate is the cost per GB of data processed.
	// Source: Product Family "NAT Gateway", usageType containing "NatGateway-Bytes"
	DataProcessingRate float64

	// Currency code (e.g., "USD")
	Currency string
}

// pricingMetadata holds AWS pricing data metadata for debugging and traceability (T034).
// Captured from the embedded pricing JSON during initialization.
type pricingMetadata struct {
	// Version is the AWS pricing data version (timestamp-based, e.g., "20251218235654").
	Version string
	// PublicationDate is the ISO timestamp when AWS published this pricing data.
	PublicationDate string
	// OfferCode identifies the AWS service (e.g., "AmazonEC2", "AWSELB").
	OfferCode string
}

// TierRate represents a single tier in AWS's tiered pricing structure.
// Used for services with volume-based pricing like CloudWatch logs and metrics.
type TierRate struct {
	// UpTo is the upper bound of this tier in GB (for logs) or count (for metrics).
	// Use math.MaxFloat64 for the final tier with no upper bound.
	UpTo float64
	// Rate is the price per unit ($/GB for logs, $/metric for metrics).
	Rate float64
}

// cloudWatchPrice holds the regional pricing configuration for Amazon CloudWatch.
// Derived from AWS Pricing API for service AmazonCloudWatch.
type cloudWatchPrice struct {
	// LogsIngestionTiers contains tiered pricing for log ingestion.
	// AWS uses volume-based tiers: first 10TB @ $0.50, next 20TB @ $0.25, etc.
	// Source: Product Family "Data Payload", usageType containing "DataProcessing-Bytes"
	LogsIngestionTiers []TierRate

	// LogsStorageRate is the flat rate per GB-month for stored logs.
	// Source: Product Family "Storage Snapshot", usageType containing "TimedStorage-ByteHrs"
	LogsStorageRate float64

	// MetricsTiers contains tiered pricing for custom metrics.
	// AWS uses volume-based tiers: first 10k @ $0.30, next 240k @ $0.10, etc.
	// Source: Product Family "Metric", usageType containing "MetricMonitorUsage"
	MetricsTiers []TierRate

	// Currency code (e.g., "USD")
	Currency string
}
