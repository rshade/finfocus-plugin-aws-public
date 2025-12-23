package pricing

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// PricingClient provides pricing data lookups
type PricingClient interface {
	// Region returns the AWS region for this pricing data
	Region() string

	// Currency returns the currency code (always "USD" for v1)
	Currency() string

	// EC2OnDemandPricePerHour returns hourly rate for an EC2 instance
	// Returns (price, true) if found, (0, false) if not found
	EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool)

	// EBSPricePerGBMonth returns monthly rate per GB for an EBS volume
	// Returns (price, true) if found, (0, false) if not found
	EBSPricePerGBMonth(volumeType string) (float64, bool)

	// S3PricePerGBMonth returns monthly rate per GB for S3 storage
	// Returns (price, true) if found, (0, false) if not found
	S3PricePerGBMonth(storageClass string) (float64, bool)

	// RDSOnDemandPricePerHour returns hourly rate for an RDS instance
	// instanceType: e.g., "db.t3.medium"
	// engine: normalized engine name, e.g., "MySQL", "PostgreSQL"
	// Returns (price, true) if found, (0, false) if not found
	RDSOnDemandPricePerHour(instanceType, engine string) (float64, bool)

	// RDSStoragePricePerGBMonth returns monthly rate per GB for RDS storage
	// volumeType: e.g., "gp2", "gp3", "io1"
	// Returns (price, true) if found, (0, false) if not found
	RDSStoragePricePerGBMonth(volumeType string) (float64, bool)

	// EKSClusterPricePerHour returns hourly rate for EKS cluster control plane.
	// extendedSupport: true for extended support pricing, false for standard support.
	// Returns (price, true) if found, (0, false) if not found.
	EKSClusterPricePerHour(extendedSupport bool) (float64, bool)

	// LambdaPricePerRequest returns the cost per request (same for all architectures)
	// Returns (price, true) if found, (0, false) if not found
	LambdaPricePerRequest() (float64, bool)

	// LambdaPricePerGBSecond returns the cost per GB-second of compute duration.
	// arch: "x86_64" or "arm64" (defaults to x86_64 if unrecognized)
	// Returns (price, true) if found, (0, false) if not found
	LambdaPricePerGBSecond(arch string) (float64, bool)

	// DynamoDBOnDemandReadPrice returns the cost per read request unit.
	// Returns (price, true) if found, (0, false) if not found
	DynamoDBOnDemandReadPrice() (float64, bool)

	// DynamoDBOnDemandWritePrice returns the cost per write request unit.
	// Returns (price, true) if found, (0, false) if not found
	DynamoDBOnDemandWritePrice() (float64, bool)

	// DynamoDBStoragePricePerGBMonth returns the monthly rate per GB for table storage.
	// Returns (price, true) if found, (0, false) if not found
	DynamoDBStoragePricePerGBMonth() (float64, bool)

	// DynamoDBProvisionedRCUPrice returns the cost per RCU-hour.
	// Returns (price, true) if found, (0, false) if not found
	DynamoDBProvisionedRCUPrice() (float64, bool)

	// DynamoDBProvisionedWCUPrice returns the cost per WCU-hour.
	// Returns (price, true) if found, (0, false) if not found
	DynamoDBProvisionedWCUPrice() (float64, bool)

	// ALBPricePerHour returns the fixed hourly rate for an Application Load Balancer.
	// Returns (price, true) if found, (0, false) if not found.
	ALBPricePerHour() (float64, bool)

	// ALBPricePerLCU returns the cost per LCU-hour for an Application Load Balancer.
	// Returns (price, true) if found, (0, false) if not found.
	ALBPricePerLCU() (float64, bool)

	// NLBPricePerHour returns the fixed hourly rate for a Network Load Balancer.
	// Returns (price, true) if found, (0, false) if not found.
	NLBPricePerHour() (float64, bool)

	// NLBPricePerNLCU returns the cost per NLCU-hour for a Network Load Balancer.
	// Returns (price, true) if found, (0, false) if not found.
	NLBPricePerNLCU() (float64, bool)

	// NATGatewayPrice returns the pricing for a NAT Gateway (hourly and data processing).
	// Returns (price, true) if found, (nil, false) if not found.
	NATGatewayPrice() (*NATGatewayPrice, bool)
}

// Client implements PricingClient with embedded JSON data
type Client struct {
	region   string
	currency string
	logger   zerolog.Logger // Add zerolog logger

	// Thread-safe initialization
	once sync.Once
	err  error

	// In-memory pricing indexes (built on first access)
	ec2Index map[string]ec2Price
	ebsIndex map[string]ebsPrice
	s3Index  map[string]s3Price

	// RDS pricing indexes (key: "instanceType/engine" for instances, "volumeType" for storage)
	rdsInstanceIndex map[string]rdsInstancePrice
	rdsStorageIndex  map[string]rdsStoragePrice

	// EKS pricing (single cluster rate)
	eksPricing *eksPrice

	// Lambda pricing (single rate per region)
	lambdaPricing *lambdaPrice

	// DynamoDB pricing (single rate per region)
	dynamoDBPricing *dynamoDBPrice

	// ELB pricing (single rate per region)
	elbPricing *elbPrice

	// NAT Gateway pricing (single rate per region)
	natGatewayPricing *NATGatewayPrice
}

// NewClient creates a Client from embedded rawPricingJSON.
// NewClient creates and returns a new Client that provides pricing lookups.
// The provided logger is attached to the client and used for performance
// warnings during pricing lookups and other client-level diagnostics.
// It returns an initialized *Client or a non-nil error if initialization fails.
func NewClient(logger zerolog.Logger) (*Client, error) {
	c := &Client{
		logger: logger, // Initialize the logger
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

// init parses embedded pricing data exactly once.
// Parsing is parallelized across services for faster initialization.
func (c *Client) init() error {
	c.once.Do(func() {
		// Initialize indexes
		c.currency = "USD"
		c.region = "unknown"
		c.ec2Index = make(map[string]ec2Price)
		c.ebsIndex = make(map[string]ebsPrice)
		c.s3Index = make(map[string]s3Price)
		c.rdsInstanceIndex = make(map[string]rdsInstancePrice)
		c.rdsStorageIndex = make(map[string]rdsStoragePrice)

		// Parse each service file in parallel for faster initialization.
		// Each parser writes to its own dedicated index(es), so no locking needed.
		// Region is captured from EC2 (largest/most reliable) after all parsing completes.
		//
		// Thread safety: zerolog.Logger is safe for concurrent use per
		// https://github.com/rs/zerolog#thread-safety ("zerolog's Logger is thread-safe")
		// so Error() and Warn() calls from multiple goroutines are safe.
		var wg sync.WaitGroup
		var ec2Region string
		var ec2Metadata *pricingMetadata
		start := time.Now()

		// Error collection for critical services.
		//
		// CRITICAL services (fail initialization on error):
		//   - EC2/EBS: Primary cost drivers, most commonly estimated. EBS is parsed
		//     inside parseEC2Pricing() since both are in the AmazonEC2 offer file.
		//
		// NON-CRITICAL services (log error, continue initialization):
		//   - S3, RDS, EKS, Lambda, DynamoDB, ELB: Currently stub or partial implementations.
		//     Failures are logged but don't block plugin startup, allowing EC2/EBS
		//     estimation to work even if other services have parsing issues.
		var parseErrMu sync.Mutex
		var parseErrs []error

		// 1. Parse EC2 pricing (includes EBS volumes) - largest file, start first
		// EC2 is CRITICAL - failure to parse means $0 for all compute estimates
		wg.Add(1)
		go func() {
			defer wg.Done()
			if region, meta, err := c.parseEC2Pricing(rawEC2JSON); err != nil {
				parseErrMu.Lock()
				parseErrs = append(parseErrs, fmt.Errorf("EC2: %w", err))
				parseErrMu.Unlock()
				c.logger.Error().Err(err).Msg("failed to parse EC2 pricing")
			} else {
				ec2Region = region
				ec2Metadata = meta
			}
		}()

		// 2. Parse S3 pricing
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.parseS3Pricing(rawS3JSON); err != nil {
				c.logger.Error().Err(err).Msg("failed to parse S3 pricing")
			}
		}()

		// 3. Parse RDS pricing
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.parseRDSPricing(rawRDSJSON); err != nil {
				c.logger.Error().Err(err).Msg("failed to parse RDS pricing")
			}
		}()

		// 4. Parse EKS pricing
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.parseEKSPricing(rawEKSJSON); err != nil {
				c.logger.Error().Err(err).Msg("failed to parse EKS pricing")
			}
		}()

		// 5. Parse Lambda pricing
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.parseLambdaPricing(rawLambdaJSON); err != nil {
				c.logger.Error().Err(err).Msg("failed to parse Lambda pricing")
			}
		}()

		// 6. Parse DynamoDB pricing
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.parseDynamoDBPricing(rawDynamoDBJSON); err != nil {
				c.logger.Error().Err(err).Msg("failed to parse DynamoDB pricing")
			}
		}()

		// 7. Parse ELB pricing
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.parseELBPricing(rawELBJSON); err != nil {
				c.logger.Error().Err(err).Msg("failed to parse ELB pricing")
			}
		}()

		// 8. Parse NAT Gateway pricing
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.parseNATGatewayPricing(rawVPCJSON); err != nil {
				c.logger.Error().Err(err).Msg("failed to parse NAT Gateway pricing")
			}
		}()

		// Wait for all parsing to complete
		wg.Wait()

		// Log initialization duration for performance monitoring
		c.logger.Debug().
			Dur("init_duration_ms", time.Since(start)).
			Int("ec2_products", len(c.ec2Index)).
			Int("ebs_products", len(c.ebsIndex)).
			Bool("natgw_found", c.natGatewayPricing != nil).
			Msg("Pricing data parsed")

		// Fail initialization if critical service parsing failed
		if len(parseErrs) > 0 {
			c.err = fmt.Errorf("pricing initialization failed: %v", parseErrs)
			return
		}

		// Set region from EC2 data (all services have the same region in a regional binary)
		if ec2Region != "" {
			c.region = ec2Region
		}

		// Validate critical EC2/EBS indexes are populated (prevents v0.0.10 regression)
		// For non-fallback builds (real regional binaries), empty indexes are fatal errors.
		// For fallback builds (region == "unknown"), empty indexes are expected for some services.
		isFallbackBuild := c.region == "unknown"

		if len(c.ec2Index) == 0 {
			if isFallbackBuild {
				c.logger.Debug().Msg("EC2 pricing index empty (expected for fallback build)")
			} else {
				c.err = fmt.Errorf("EC2 pricing index is empty - data corruption or filtering issue")
				c.logger.Error().Msg("EC2 pricing index is empty - failing initialization")
				return
			}
		}
		if len(c.ebsIndex) == 0 {
			if isFallbackBuild {
				c.logger.Debug().Msg("EBS pricing index empty (expected for fallback build)")
			} else {
				c.err = fmt.Errorf("EBS pricing index is empty - data corruption or filtering issue")
				c.logger.Error().Msg("EBS pricing index is empty - failing initialization")
				return
			}
		}

		// Log embedded pricing metadata for debugging and traceability (T034)
		// This helps identify which AWS pricing version is embedded in the binary.
		if ec2Metadata != nil {
			c.logger.Debug().
				Str("region", c.region).
				Str("version", ec2Metadata.Version).
				Str("publicationDate", ec2Metadata.PublicationDate).
				Str("offerCode", ec2Metadata.OfferCode).
				Msg("Embedded pricing metadata loaded")
		}

		// Validate non-critical service pricing was loaded.
		// Missing pricing is logged as a warning but doesn't fail initialization.
		// Uses helper to reduce boilerplate while maintaining detailed logging.
		warnMissing := func(service, field string, value float64) {
			if value == 0 {
				c.logger.Warn().
					Str("region", c.region).
					Str("service", service).
					Str("field", field).
					Msg("pricing field not found in embedded data")
			}
		}

		// EKS pricing validation
		if c.eksPricing != nil {
			warnMissing("EKS", "StandardHourlyRate", c.eksPricing.StandardHourlyRate)
			warnMissing("EKS", "ExtendedHourlyRate", c.eksPricing.ExtendedHourlyRate)
		} else {
			c.logger.Warn().Str("region", c.region).Msg("EKS pricing not loaded")
		}

		// Lambda pricing validation
		if c.lambdaPricing != nil {
			warnMissing("Lambda", "RequestPrice", c.lambdaPricing.RequestPrice)
			warnMissing("Lambda", "X86GBSecondPrice", c.lambdaPricing.X86GBSecondPrice)
			warnMissing("Lambda", "ARMGBSecondPrice", c.lambdaPricing.ARMGBSecondPrice)
		} else {
			c.logger.Warn().Str("region", c.region).Msg("Lambda pricing not loaded")
		}

		// DynamoDB pricing validation
		if c.dynamoDBPricing != nil {
			warnMissing("DynamoDB", "OnDemandReadPrice", c.dynamoDBPricing.OnDemandReadPrice)
			warnMissing("DynamoDB", "OnDemandWritePrice", c.dynamoDBPricing.OnDemandWritePrice)
			warnMissing("DynamoDB", "StoragePrice", c.dynamoDBPricing.StoragePrice)
			warnMissing("DynamoDB", "ProvisionedRCUPrice", c.dynamoDBPricing.ProvisionedRCUPrice)
			warnMissing("DynamoDB", "ProvisionedWCUPrice", c.dynamoDBPricing.ProvisionedWCUPrice)
		} else {
			c.logger.Warn().Str("region", c.region).Msg("DynamoDB pricing not loaded")
		}

		// ELB pricing validation
		if c.elbPricing != nil {
			warnMissing("ELB", "ALBHourlyRate", c.elbPricing.ALBHourlyRate)
			warnMissing("ELB", "ALBLCURate", c.elbPricing.ALBLCURate)
			warnMissing("ELB", "NLBHourlyRate", c.elbPricing.NLBHourlyRate)
			warnMissing("ELB", "NLBNLCURate", c.elbPricing.NLBNLCURate)
		} else {
			c.logger.Warn().Str("region", c.region).Msg("ELB pricing not loaded")
		}
	})
	return c.err
}

// getOnDemandPrice extracts the OnDemand price for a SKU from parsed AWS pricing data.
//
// AWS Price List API returns a nested structure for pricing:
//
//	Terms["OnDemand"][SKU][OfferTermCode] -> term
//	  └── term.PriceDimensions[RateCode] -> priceDimension
//	        └── priceDimension.PricePerUnit["USD"] -> price string
//
// This function navigates this structure to find the USD price. The iteration
// through multiple terms and dimensions handles cases where a SKU has multiple
// offer term codes (e.g., different effective dates) or multiple price dimensions
// (e.g., hourly rate + data transfer). We return the first valid USD price found.
//
// Parameters:
//   - data: Parsed AWS pricing JSON containing Products and Terms
//   - sku: The product SKU to look up (unique identifier in AWS pricing)
//
// Returns:
//   - price: The USD price as a float64 (0 if not found or parse error)
//   - unit: The billing unit (e.g., "Hrs", "GB-Mo", "LCU-Hrs")
//   - found: True if a valid USD price was found and parsed successfully
//
// Used by all service parsers (EC2, S3, RDS, EKS, Lambda, DynamoDB, ELB) to extract
// prices from the raw AWS pricing data during initialization.
func getOnDemandPrice(data *awsPricing, sku string) (float64, string, bool) {
	termMap, ok := data.Terms["OnDemand"][sku]
	if !ok {
		return 0, "", false
	}
	for _, term := range termMap {
		for _, dim := range term.PriceDimensions {
			if amountStr, ok := dim.PricePerUnit["USD"]; ok {
				amount, err := strconv.ParseFloat(amountStr, 64)
				if err == nil {
					return amount, dim.Unit, true
				}
			}
		}
	}
	return 0, "", false
}

// parseEC2Pricing parses EC2 pricing data including EBS volumes.
// Returns the detected region, pricing metadata, and any parsing error.
func (c *Client) parseEC2Pricing(data []byte) (string, *pricingMetadata, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", nil, fmt.Errorf("failed to parse EC2 JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AmazonEC2" {
		c.logger.Warn().
			Str("expected", "AmazonEC2").
			Str("actual", pricing.OfferCode).
			Msg("EC2 pricing data has unexpected offerCode")
	}

	// Capture metadata for debugging (T034)
	meta := &pricingMetadata{
		Version:         pricing.Version,
		PublicationDate: pricing.PublicationDate,
		OfferCode:       pricing.OfferCode,
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		// Capture region from first product that has it
		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		// EC2 Instances
		if prod.ProductFamily == "Compute Instance" {
			instType := attrs["instanceType"]
			os := attrs["operatingSystem"]
			tenancy := attrs["tenancy"]
			capacityStatus := attrs["capacitystatus"]
			preInstalledSw := attrs["preInstalledSw"]

			if instType != "" && os != "" && tenancy != "" &&
				capacityStatus == "Used" &&
				(preInstalledSw == "NA" || preInstalledSw == "") {

				key := fmt.Sprintf("%s/%s/%s", instType, os, tenancy)
				rate, unit, found := getOnDemandPrice(&pricing, sku)
				if found {
					c.ec2Index[key] = ec2Price{
						Unit:       unit,
						HourlyRate: rate,
						Currency:   "USD",
					}
				}
			}
		}

		// EBS Volumes (included in EC2 pricing file)
		if prod.ProductFamily == "Storage" {
			volType := attrs["volumeApiName"]
			if volType == "" {
				continue
			}
			rate, unit, found := getOnDemandPrice(&pricing, sku)
			if found && unit == "GB-Mo" {
				c.ebsIndex[volType] = ebsPrice{
					Unit:           unit,
					RatePerGBMonth: rate,
					Currency:       "USD",
				}
			}
		}
	}
	return region, meta, nil
}

// parseS3Pricing parses S3 pricing data.
// Returns the detected region and any parsing error.
func (c *Client) parseS3Pricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse S3 JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AmazonS3" {
		c.logger.Warn().
			Str("expected", "AmazonS3").
			Str("actual", pricing.OfferCode).
			Msg("S3 pricing data has unexpected offerCode")
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		if prod.ProductFamily == "Storage" {
			storageClass := attrs["storageClass"]
			if storageClass == "" {
				continue
			}
			rate, unit, found := getOnDemandPrice(&pricing, sku)
			if found && unit == "GB-Mo" {
				c.s3Index[storageClass] = s3Price{
					Unit:           unit,
					RatePerGBMonth: rate,
					Currency:       "USD",
				}
			}
		}
	}
	return region, nil
}

// parseRDSPricing parses RDS pricing data.
// Returns the detected region and any parsing error.
func (c *Client) parseRDSPricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse RDS JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AmazonRDS" {
		c.logger.Warn().
			Str("expected", "AmazonRDS").
			Str("actual", pricing.OfferCode).
			Msg("RDS pricing data has unexpected offerCode")
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		// RDS Database Instances
		if prod.ProductFamily == "Database Instance" {
			instClass := attrs["instanceType"]
			engine := attrs["databaseEngine"]
			deployOption := attrs["deploymentOption"]

			if instClass != "" && engine != "" && deployOption == "Single-AZ" {
				key := fmt.Sprintf("%s/%s", instClass, engine)
				rate, unit, found := getOnDemandPrice(&pricing, sku)
				if found && unit == "Hrs" {
					c.rdsInstanceIndex[key] = rdsInstancePrice{
						Unit:       unit,
						HourlyRate: rate,
						Currency:   "USD",
					}
				}
			}
		}

		// RDS Database Storage
		if prod.ProductFamily == "Database Storage" {
			volType := attrs["volumeType"]
			usageType := attrs["usagetype"]

			var apiVolType string
			switch volType {
			case "General Purpose":
				if usageType != "" && strings.Contains(usageType, "gp3") {
					apiVolType = "gp3"
				} else {
					apiVolType = "gp2"
				}
			case "General Purpose (SSD)":
				apiVolType = "gp2"
			case "Provisioned IOPS", "Provisioned IOPS (SSD)":
				if usageType != "" && strings.Contains(usageType, "io2") {
					apiVolType = "io2"
				} else {
					apiVolType = "io1"
				}
			case "Magnetic":
				apiVolType = "standard"
			default:
				continue
			}

			rate, unit, found := getOnDemandPrice(&pricing, sku)
			if found && unit == "GB-Mo" {
				if _, exists := c.rdsStorageIndex[apiVolType]; !exists {
					c.rdsStorageIndex[apiVolType] = rdsStoragePrice{
						Unit:           unit,
						RatePerGBMonth: rate,
						Currency:       "USD",
					}
				}
			}
		}
	}
	return region, nil
}

// parseEKSPricing parses EKS pricing data.
// Returns the detected region and any parsing error.
func (c *Client) parseEKSPricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse EKS JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AmazonEKS" {
		c.logger.Warn().
			Str("expected", "AmazonEKS").
			Str("actual", pricing.OfferCode).
			Msg("EKS pricing data has unexpected offerCode")
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		if attrs["servicecode"] == "AmazonEKS" {
			operation := attrs["operation"]
			usageType := attrs["usagetype"]

			if c.eksPricing == nil {
				c.eksPricing = &eksPrice{
					Unit:     "Hrs",
					Currency: "USD",
				}
			}

			rate, unit, found := getOnDemandPrice(&pricing, sku)
			// AWS returns unit as "Hours", "Hrs", or "hours" depending on the product
			unitLower := strings.ToLower(unit)
			if found && (unitLower == "hrs" || unitLower == "hours") && rate > 0 {
				if operation == "ExtendedSupport" || strings.Contains(usageType, "extendedSupport") {
					c.eksPricing.ExtendedHourlyRate = rate
				} else if operation == "CreateOperation" || strings.Contains(usageType, "perCluster") {
					// Standard cluster pricing: operation=CreateOperation, usageType contains "perCluster"
					c.eksPricing.StandardHourlyRate = rate
				}
			}
		}
	}
	return region, nil
}

// parseLambdaPricing parses Lambda pricing data.
// Returns the detected region and any parsing error.
func (c *Client) parseLambdaPricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse Lambda JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AWSLambda" {
		c.logger.Warn().
			Str("expected", "AWSLambda").
			Str("actual", pricing.OfferCode).
			Msg("Lambda pricing data has unexpected offerCode")
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		if prod.ProductFamily == "AWS Lambda" || prod.ProductFamily == "Serverless" {
			group := attrs["group"]

			if c.lambdaPricing == nil {
				c.lambdaPricing = &lambdaPrice{
					Currency: "USD",
				}
			}

			rate, unit, found := getOnDemandPrice(&pricing, sku)
			if found {
				if group == "AWS-Lambda-Requests" && unit == "Requests" {
					c.lambdaPricing.RequestPrice = rate
				} else if group == "AWS-Lambda-Duration" && (unit == "Second" || unit == "Lambda-GB-Second") {
					c.lambdaPricing.X86GBSecondPrice = rate
				} else if group == "AWS-Lambda-Duration-ARM" && (unit == "Second" || unit == "Lambda-GB-Second") {
					c.lambdaPricing.ARMGBSecondPrice = rate
				}
			}
		}
	}
	return region, nil
}

// parseDynamoDBPricing parses DynamoDB pricing data.
// Returns the detected region and any parsing error.
func (c *Client) parseDynamoDBPricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse DynamoDB JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AmazonDynamoDB" {
		c.logger.Warn().
			Str("expected", "AmazonDynamoDB").
			Str("actual", pricing.OfferCode).
			Msg("DynamoDB pricing data has unexpected offerCode")
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		if attrs["servicecode"] == "AmazonDynamoDB" {
			if c.dynamoDBPricing == nil {
				c.dynamoDBPricing = &dynamoDBPrice{
					Currency: "USD",
				}
			}

			rate, unit, found := getOnDemandPrice(&pricing, sku)
			if found {
				if prod.ProductFamily == "Amazon DynamoDB PayPerRequest Throughput" {
					group := attrs["group"]
					switch group {
					case "DDB-ReadUnits":
						c.dynamoDBPricing.OnDemandReadPrice = rate
					case "DDB-WriteUnits":
						c.dynamoDBPricing.OnDemandWritePrice = rate
					}
				} else if prod.ProductFamily == "Provisioned IOPS" || strings.Contains(prod.ProductFamily, "Throughput") {
					usageType := attrs["usagetype"]
					if strings.Contains(usageType, "ReadCapacityUnit") && unit == "Hrs" {
						c.dynamoDBPricing.ProvisionedRCUPrice = rate
					} else if strings.Contains(usageType, "WriteCapacityUnit") && unit == "Hrs" {
						c.dynamoDBPricing.ProvisionedWCUPrice = rate
					}
				} else if prod.ProductFamily == "Database Storage" {
					usageType := attrs["usagetype"]
					if strings.Contains(usageType, "TimedStorage-ByteHrs") && unit == "GB-Mo" {
						c.dynamoDBPricing.StoragePrice = rate
					}
				}
			}
		}
	}
	return region, nil
}

// parseELBPricing parses ELB pricing data.
// Returns the detected region and any parsing error.
func (c *Client) parseELBPricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse ELB JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AWSELB" {
		c.logger.Warn().
			Str("expected", "AWSELB").
			Str("actual", pricing.OfferCode).
			Msg("ELB pricing data has unexpected offerCode")
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		if prod.ProductFamily == "Load Balancer-Application" || prod.ProductFamily == "Load Balancer-Network" {
			usageType := attrs["usagetype"]

			if c.elbPricing == nil {
				c.elbPricing = &elbPrice{
					Currency: "USD",
				}
			}

			rate, unit, found := getOnDemandPrice(&pricing, sku)
			if found {
				switch prod.ProductFamily {
				case "Load Balancer-Application":
					if strings.HasSuffix(usageType, "LoadBalancerUsage") && !strings.Contains(usageType, "TS-") && unit == "Hrs" {
						c.elbPricing.ALBHourlyRate = rate
					} else if strings.HasSuffix(usageType, "LCUUsage") && !strings.Contains(usageType, "Outposts-") && !strings.Contains(usageType, "Reserved") && unit == "LCU-Hrs" {
						c.elbPricing.ALBLCURate = rate
					}
				case "Load Balancer-Network":
					if strings.HasSuffix(usageType, "LoadBalancerUsage") && !strings.Contains(usageType, "TS-") && unit == "Hrs" {
						c.elbPricing.NLBHourlyRate = rate
					} else if strings.HasSuffix(usageType, "LCUUsage") && !strings.Contains(usageType, "Outposts-") && !strings.Contains(usageType, "Reserved") && unit == "LCU-Hrs" {
						// AWS uses "LCUUsage" with "LCU-Hrs" for NLB capacity units too
						// The description differentiates: "Network load balancer capacity unit-hour"
						c.elbPricing.NLBNLCURate = rate
					}
				}
			}
		}
	}
	return region, nil
}

// parseNATGatewayPricing parses VPC pricing data for NAT Gateways.
// Returns the detected region and any parsing error.
func (c *Client) parseNATGatewayPricing(data []byte) (string, error) {
	var pricing awsPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		return "", fmt.Errorf("failed to parse VPC JSON: %w", err)
	}

	// Validate offerCode matches expected service (T031)
	if pricing.OfferCode != "AmazonVPC" {
		c.logger.Warn().
			Str("expected", "AmazonVPC").
			Str("actual", pricing.OfferCode).
			Msg("VPC pricing data has unexpected offerCode")
	}

	var region string
	for sku, prod := range pricing.Products {
		attrs := prod.Attributes

		if region == "" && attrs["regionCode"] != "" {
			region = attrs["regionCode"]
		}

		if prod.ProductFamily == "NAT Gateway" {
			usageType := attrs["usagetype"]

			if c.natGatewayPricing == nil {
				c.natGatewayPricing = &NATGatewayPrice{
					Currency: "USD",
				}
			}

			rate, unit, found := getOnDemandPrice(&pricing, sku)
			if found {
				if strings.Contains(usageType, "NatGateway-Hours") && unit == "Hrs" {
					c.natGatewayPricing.HourlyRate = rate
				} else if strings.Contains(usageType, "NatGateway-Bytes") && (unit == "Quantity" || unit == "GB") {
					// AWS Pricing API returns "Quantity" as the unit for NatGateway-Bytes,
					// but the rate is actually per-GB (not per-byte). No conversion needed.
					// See: specs/001-nat-gateway-cost/research.md for verification.
					c.natGatewayPricing.DataProcessingRate = rate
				}
			}
		}
	}
	return region, nil
}

// Region returns the AWS region for this pricing data
func (c *Client) Region() string {
	_ = c.init() // Ensure initialization
	return c.region
}

// Currency returns the currency code
func (c *Client) Currency() string {
	_ = c.init() // Ensure initialization
	return c.currency
}

// EC2OnDemandPricePerHour returns hourly rate for an EC2 instance
func (c *Client) EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "EC2").
				Str("instance_type", instanceType).
				Str("os", os).
				Str("tenancy", tenancy).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	key := fmt.Sprintf("%s/%s/%s", instanceType, os, tenancy)
	price, found := c.ec2Index[key]
	if !found {
		return 0, false
	}
	return price.HourlyRate, true
}

// EBSPricePerGBMonth returns monthly rate per GB for an EBS volume
func (c *Client) EBSPricePerGBMonth(volumeType string) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "EBS").
				Str("volume_type", volumeType).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	price, found := c.ebsIndex[volumeType]
	if !found {
		return 0, false
	}
	return price.RatePerGBMonth, true
}

// S3PricePerGBMonth returns monthly rate per GB for S3 storage
func (c *Client) S3PricePerGBMonth(storageClass string) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "S3").
				Str("storage_class", storageClass).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	price, found := c.s3Index[storageClass]
	if !found {
		return 0, false
	}
	return price.RatePerGBMonth, true
}

// RDSOnDemandPricePerHour returns hourly rate for an RDS instance
// instanceType: e.g., "db.t3.medium"
// engine: normalized engine name, e.g., "MySQL", "PostgreSQL"
func (c *Client) RDSOnDemandPricePerHour(instanceType, engine string) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "RDS").
				Str("instance_type", instanceType).
				Str("engine", engine).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	key := fmt.Sprintf("%s/%s", instanceType, engine)
	price, found := c.rdsInstanceIndex[key]
	if !found {
		return 0, false
	}
	return price.HourlyRate, true
}

// RDSStoragePricePerGBMonth returns monthly rate per GB for RDS storage
// volumeType: e.g., "gp2", "gp3", "io1", "standard"
func (c *Client) RDSStoragePricePerGBMonth(volumeType string) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "RDS_Storage").
				Str("volume_type", volumeType).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	price, found := c.rdsStorageIndex[volumeType]
	if !found {
		return 0, false
	}
	return price.RatePerGBMonth, true
}

// EKSClusterPricePerHour returns hourly rate for EKS cluster control plane.
// extendedSupport: true for extended support pricing, false for standard support.
func (c *Client) EKSClusterPricePerHour(extendedSupport bool) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "EKS").
				Bool("extended_support", extendedSupport).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	if c.eksPricing == nil {
		return 0, false
	}

	if extendedSupport {
		if c.eksPricing.ExtendedHourlyRate > 0 {
			return c.eksPricing.ExtendedHourlyRate, true
		}
		return 0, false
	}

	if c.eksPricing.StandardHourlyRate > 0 {
		return c.eksPricing.StandardHourlyRate, true
	}
	return 0, false
}

// LambdaPricePerRequest returns the cost per request for AWS Lambda invocations.
// The rate is sourced from AWS Price List API product family "AWS Lambda" with
// group "AWS-Lambda-Requests". Standard pricing is $0.20 per 1 million requests
// ($0.0000002 per request) as of December 2025.
// Returns (price, true) if found, (0, false) if not found.
func (c *Client) LambdaPricePerRequest() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "Lambda").
				Str("metric", "Requests").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	if c.lambdaPricing == nil || c.lambdaPricing.RequestPrice == 0 {
		return 0, false
	}
	return c.lambdaPricing.RequestPrice, true
}

// LambdaPricePerGBSecond returns the cost per GB-second of compute duration.
// The rate is sourced from AWS Price List API product family "Serverless" with
// group "AWS-Lambda-Duration" (x86) or "AWS-Lambda-Duration-ARM" (arm64).
// This represents the compute cost based on allocated memory and execution time.
//
// Architecture pricing (as of December 2025):
//   - x86_64: ~$0.0000166667 per GB-second
//   - arm64:  ~$0.0000133334 per GB-second (~20% cheaper)
//
// arch parameter accepts: "x86_64", "arm64", "x86", "arm" (defaults to x86_64)
// Returns (price, true) if found, (0, false) if not found.
func (c *Client) LambdaPricePerGBSecond(arch string) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "Lambda").
				Str("metric", "GB-Second").
				Str("architecture", arch).
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}

	if c.lambdaPricing == nil {
		return 0, false
	}

	// Normalize architecture string and select appropriate price
	switch strings.ToLower(arch) {
	case "arm64", "arm":
		if c.lambdaPricing.ARMGBSecondPrice > 0 {
			return c.lambdaPricing.ARMGBSecondPrice, true
		}
		// Fall back to x86 if ARM pricing not available
		if c.lambdaPricing.X86GBSecondPrice > 0 {
			return c.lambdaPricing.X86GBSecondPrice, true
		}
		return 0, false
	default:
		// x86_64, x86, or any unrecognized value defaults to x86
		if c.lambdaPricing.X86GBSecondPrice > 0 {
			return c.lambdaPricing.X86GBSecondPrice, true
		}
		return 0, false
	}
}

// DynamoDBOnDemandReadPrice returns the cost per read request unit.
// Returns (price, true) if found, (0, false) if not found.
func (c *Client) DynamoDBOnDemandReadPrice() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "DynamoDB").
				Str("metric", "OnDemandRead").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.dynamoDBPricing == nil || c.dynamoDBPricing.OnDemandReadPrice == 0 {
		return 0, false
	}
	return c.dynamoDBPricing.OnDemandReadPrice, true
}

// DynamoDBOnDemandWritePrice returns the cost per write request unit.
// Returns (price, true) if found, (0, false) if not found.
func (c *Client) DynamoDBOnDemandWritePrice() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "DynamoDB").
				Str("metric", "OnDemandWrite").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.dynamoDBPricing == nil || c.dynamoDBPricing.OnDemandWritePrice == 0 {
		return 0, false
	}
	return c.dynamoDBPricing.OnDemandWritePrice, true
}

// DynamoDBStoragePricePerGBMonth returns the monthly rate per GB for table storage.
// Returns (price, true) if found, (0, false) if not found.
func (c *Client) DynamoDBStoragePricePerGBMonth() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "DynamoDB").
				Str("metric", "Storage").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.dynamoDBPricing == nil || c.dynamoDBPricing.StoragePrice == 0 {
		return 0, false
	}
	return c.dynamoDBPricing.StoragePrice, true
}

// DynamoDBProvisionedRCUPrice returns the cost per RCU-hour.
// Returns (price, true) if found, (0, false) if not found.
func (c *Client) DynamoDBProvisionedRCUPrice() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "DynamoDB").
				Str("metric", "ProvisionedRCU").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.dynamoDBPricing == nil || c.dynamoDBPricing.ProvisionedRCUPrice == 0 {
		return 0, false
	}
	return c.dynamoDBPricing.ProvisionedRCUPrice, true
}

// DynamoDBProvisionedWCUPrice returns the cost per WCU-hour.
// Returns (price, true) if found, (0, false) if not found.
func (c *Client) DynamoDBProvisionedWCUPrice() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "DynamoDB").
				Str("metric", "ProvisionedWCU").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.dynamoDBPricing == nil || c.dynamoDBPricing.ProvisionedWCUPrice == 0 {
		return 0, false
	}
	return c.dynamoDBPricing.ProvisionedWCUPrice, true
}

// ALBPricePerHour returns the fixed hourly rate for an Application Load Balancer.
func (c *Client) ALBPricePerHour() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "ELB").
				Str("lb_type", "ALB").
				Str("metric", "FixedHourly").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.elbPricing == nil || c.elbPricing.ALBHourlyRate == 0 {
		return 0, false
	}
	return c.elbPricing.ALBHourlyRate, true
}

// ALBPricePerLCU returns the cost per LCU-hour for an Application Load Balancer.
func (c *Client) ALBPricePerLCU() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "ELB").
				Str("lb_type", "ALB").
				Str("metric", "LCU").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.elbPricing == nil || c.elbPricing.ALBLCURate == 0 {
		return 0, false
	}
	return c.elbPricing.ALBLCURate, true
}

// NLBPricePerHour returns the fixed hourly rate for a Network Load Balancer.
func (c *Client) NLBPricePerHour() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "ELB").
				Str("lb_type", "NLB").
				Str("metric", "FixedHourly").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.elbPricing == nil || c.elbPricing.NLBHourlyRate == 0 {
		return 0, false
	}
	return c.elbPricing.NLBHourlyRate, true
}

// NLBPricePerNLCU returns the cost per NLCU-hour for a Network Load Balancer.
func (c *Client) NLBPricePerNLCU() (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "ELB").
				Str("lb_type", "NLB").
				Str("metric", "NLCU").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return 0, false
	}
	if c.elbPricing == nil || c.elbPricing.NLBNLCURate == 0 {
		return 0, false
	}
	return c.elbPricing.NLBNLCURate, true
}

// NATGatewayPrice returns the pricing for a NAT Gateway.
func (c *Client) NATGatewayPrice() (*NATGatewayPrice, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			c.logger.Warn().
				Str("resource_type", "NATGateway").
				Dur("elapsed", elapsed).
				Msg("pricing lookup took too long")
		}
	}()

	if err := c.init(); err != nil {
		return nil, false
	}
	if c.natGatewayPricing == nil || c.natGatewayPricing.HourlyRate == 0 {
		return nil, false
	}
	return c.natGatewayPricing, true
}


