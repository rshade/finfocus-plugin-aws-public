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

// init parses embedded pricing data exactly once
func (c *Client) init() error {
	c.once.Do(func() {
		// 1. Parse the raw AWS Price List JSON
		var data awsPricing
		if err := json.Unmarshal(rawPricingJSON, &data); err != nil {
			c.err = fmt.Errorf("failed to parse pricing data: %w", err)
			return
		}

		// 2. Extract metadata (infer region/currency from content if possible, or default)
		// The raw JSON doesn't strictly have a top-level "region" field in the same way simple JSON did.
		// It has "products" where each product has "attributes.location" and "attributes.regionCode".
		// We'll scan the first product to find the region, or assume it matches the build tag.
		c.currency = "USD" // Default for AWS public pricing API
		c.region = "unknown"

		// 3. Build Lookup Indexes
		c.ec2Index = make(map[string]ec2Price)
		c.ebsIndex = make(map[string]ebsPrice)
		c.s3Index = make(map[string]s3Price)
		c.rdsInstanceIndex = make(map[string]rdsInstancePrice)
		c.rdsStorageIndex = make(map[string]rdsStoragePrice)

		// Helper to find OnDemand price for a SKU
		getOnDemandPrice := func(sku string) (float64, string, bool) {
			termMap, ok := data.Terms["OnDemand"][sku]
			if !ok {
				return 0, "", false
			}
			// There might be multiple offerTermCodes; usually just one for OnDemand generic.
			// We pick the first valid one.
			for _, term := range termMap {
				for _, dim := range term.PriceDimensions {
					// We want the price per unit.
					// AWS Price List API returns map["USD"] = "0.123"
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

		for sku, prod := range data.Products {
			attrs := prod.Attributes

			// Set region from first product if not set
			if c.region == "unknown" && attrs["regionCode"] != "" {
				c.region = attrs["regionCode"]
			}

			// --- EC2 Instances ---
			if prod.ProductFamily == "Compute Instance" {
				// We need instanceType, operatingSystem, tenancy
				instType := attrs["instanceType"]
				os := attrs["operatingSystem"]
				tenancy := attrs["tenancy"]
				capacityStatus := attrs["capacitystatus"] // "Used", "AllocatedCapacityReservation", etc.
				preInstalledSw := attrs["preInstalledSw"] // "NA", "SQL Std", etc.

				// Filter for base On-Demand instances:
				// 1. Must have valid basic attributes
				// 2. capacitystatus should be "Used" (standard on-demand usage)
				// 3. preInstalledSw should be "NA" (no extra software license fees)
				if instType != "" && os != "" && tenancy != "" &&
					capacityStatus == "Used" &&
					(preInstalledSw == "NA" || preInstalledSw == "") {

					// Composite key: "t3.micro/Linux/Shared"
					key := fmt.Sprintf("%s/%s/%s", instType, os, tenancy)

					// Only index if we don't have it (or overwrite? duplicates shouldn't exist for same keys)
					rate, unit, found := getOnDemandPrice(sku)
					if found {
						c.ec2Index[key] = ec2Price{
							Unit:       unit,
							HourlyRate: rate,
							Currency:   "USD",
						}
					}
				}
			}

			// --- EBS Volumes ---
			// EBS often has productFamily="Storage" or "System Operation" (IOPS)
			// We look for volumeApiName (e.g. "gp3")
			if prod.ProductFamily == "Storage" {
				volType := attrs["volumeApiName"] // e.g., "gp3", "io1"
				if volType == "" {
					// Fallback for older/mapped names if necessary, but volumeApiName is standard for modern API
					continue
				}

				// We want "Storage" usage type, not IOPS fees or throughput fees
				// usageType often contains "EBS:VolumeUsage.gp3"
				// attributes["usagetype"] might be useful if strict filtering needed.

				rate, unit, found := getOnDemandPrice(sku)
				if found {
					// We only want the "per GB-Mo" price, not IOPS charges.
					// Check unit.
					if unit == "GB-Mo" {
						c.ebsIndex[volType] = ebsPrice{
							Unit:           unit,
							RatePerGBMonth: rate,
							Currency:       "USD",
						}
					}
				}
			}

			// --- S3 Storage ---
			// S3 uses productFamily="Storage" and servicecode="AmazonS3"
			// Index by storageClass (e.g., "Standard", "Standard - Infrequent Access")
			if prod.ProductFamily == "Storage" && attrs["servicecode"] == "AmazonS3" {
				storageClass := attrs["storageClass"]
				if storageClass == "" {
					continue
				}

				rate, unit, found := getOnDemandPrice(sku)
				if found && unit == "GB-Mo" {
					c.s3Index[storageClass] = s3Price{
						Unit:           unit,
						RatePerGBMonth: rate,
						Currency:       "USD",
					}
				}
			}

			// --- RDS Database Instances ---
			// RDS uses productFamily="Database Instance" for compute pricing
			if prod.ProductFamily == "Database Instance" {
				instClass := attrs["instanceType"] // e.g., "db.t3.medium"
				engine := attrs["databaseEngine"]  // e.g., "MySQL", "PostgreSQL"
				deployOption := attrs["deploymentOption"]

				// Filter for Single-AZ On-Demand instances only
				// - Must have valid instance class and engine
				// - deploymentOption must be "Single-AZ" (excludes Multi-AZ pricing)
				if instClass != "" && engine != "" && deployOption == "Single-AZ" {
					// Composite key: "db.t3.medium/MySQL"
					key := fmt.Sprintf("%s/%s", instClass, engine)

					rate, unit, found := getOnDemandPrice(sku)
					if found && unit == "Hrs" {
						c.rdsInstanceIndex[key] = rdsInstancePrice{
							Unit:       unit,
							HourlyRate: rate,
							Currency:   "USD",
						}
					}
				}
			}

			// --- RDS Database Storage ---
			// RDS storage uses productFamily="Database Storage"
			if prod.ProductFamily == "Database Storage" {
				volType := attrs["volumeType"] // e.g., "General Purpose", "Provisioned IOPS"
				usageType := attrs["usagetype"]

				// Map volume type names to API names
				var apiVolType string
				switch volType {
				case "General Purpose":
					// Check usagetype to distinguish gp2 vs gp3
					if usageType != "" {
						if strings.Contains(usageType, "gp3") {
							apiVolType = "gp3"
						} else {
							apiVolType = "gp2" // Default GP to gp2
						}
					} else {
						apiVolType = "gp2"
					}
				case "General Purpose (SSD)":
					apiVolType = "gp2"
				case "Provisioned IOPS", "Provisioned IOPS (SSD)":
					// Check usagetype to distinguish io1 vs io2
					if usageType != "" && strings.Contains(usageType, "io2") {
						apiVolType = "io2"
					} else {
						apiVolType = "io1"
					}
				case "Magnetic":
					apiVolType = "standard"
				default:
					continue // Unknown storage type
				}

				rate, unit, found := getOnDemandPrice(sku)
				if found && unit == "GB-Mo" {
					// Only store if we don't have it yet or this is a better match
					if _, exists := c.rdsStorageIndex[apiVolType]; !exists {
						c.rdsStorageIndex[apiVolType] = rdsStoragePrice{
							Unit:           unit,
							RatePerGBMonth: rate,
							Currency:       "USD",
						}
					}
				}
			}

			// --- EKS Cluster Control Plane ---
			// EKS uses servicecode="AmazonEKS" with two support tiers:
			// - Standard support: operation="CreateOperation", usagetype contains "perCluster"
			// - Extended support: operation="ExtendedSupport", usagetype contains "extendedSupport"
			if attrs["servicecode"] == "AmazonEKS" {
				operation := attrs["operation"]
				usageType := attrs["usagetype"]

				// Initialize eksPrice struct if nil
				if c.eksPricing == nil {
					c.eksPricing = &eksPrice{
						Unit:     "Hrs",
						Currency: "USD",
					}
				}

				rate, unit, found := getOnDemandPrice(sku)
				if found && unit == "Hrs" && rate > 0 {
					// Extended support: ExtendedSupport operation or extendedSupport in usagetype
					// Always update with valid non-zero rates. This handles cases where AWS pricing
					// data may contain multiple entries or change order in future API responses.
					if operation == "ExtendedSupport" || strings.Contains(usageType, "extendedSupport") {
						c.eksPricing.ExtendedHourlyRate = rate
					} else {
						// Standard support: CreateOperation with perCluster, or any non-extended EKS pricing
						// This includes legacy data that doesn't have specific operation/usagetype
						c.eksPricing.StandardHourlyRate = rate
					}
				}
			}

			// --- Lambda Functions ---
			// Lambda uses two product families:
			// 1. "AWS Lambda" (Requests): group="AWS-Lambda-Requests"
			// 2. "Serverless" (Duration): group="AWS-Lambda-Duration" (x86) or
			//    "AWS-Lambda-Duration-ARM" (arm64/Graviton2)
			if prod.ProductFamily == "AWS Lambda" || prod.ProductFamily == "Serverless" {
				group := attrs["group"]

				// Initialize lambdaPricing struct if nil
				if c.lambdaPricing == nil {
					c.lambdaPricing = &lambdaPrice{
						Currency: "USD",
					}
				}

				rate, unit, found := getOnDemandPrice(sku)
				if found {
					if group == "AWS-Lambda-Requests" && unit == "Requests" {
						c.lambdaPricing.RequestPrice = rate
					} else if group == "AWS-Lambda-Duration" && (unit == "Second" || unit == "Lambda-GB-Second") {
						// x86_64 duration pricing (per GB-second)
						c.lambdaPricing.X86GBSecondPrice = rate
					} else if group == "AWS-Lambda-Duration-ARM" && (unit == "Second" || unit == "Lambda-GB-Second") {
						// arm64/Graviton2 duration pricing (per GB-second)
						c.lambdaPricing.ARMGBSecondPrice = rate
					}
				}
			}
		}

		// Validate EKS pricing data was loaded successfully
		if c.eksPricing == nil || c.eksPricing.StandardHourlyRate == 0 {
			c.logger.Warn().
				Str("region", c.region).
				Msg("EKS standard pricing not found in embedded data")
		}
		if c.eksPricing != nil && c.eksPricing.ExtendedHourlyRate == 0 {
			c.logger.Warn().
				Str("region", c.region).
				Msg("EKS extended support pricing not found in embedded data")
		}
	})
	return c.err
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

