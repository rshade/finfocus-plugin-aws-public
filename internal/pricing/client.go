package pricing

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
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

	// RDSOnDemandPricePerHour returns hourly rate for an RDS instance
	// instanceType: e.g., "db.t3.medium"
	// engine: normalized engine name, e.g., "MySQL", "PostgreSQL"
	// Returns (price, true) if found, (0, false) if not found
	RDSOnDemandPricePerHour(instanceType, engine string) (float64, bool)

	// RDSStoragePricePerGBMonth returns monthly rate per GB for RDS storage
	// volumeType: e.g., "gp2", "gp3", "io1"
	// Returns (price, true) if found, (0, false) if not found
	RDSStoragePricePerGBMonth(volumeType string) (float64, bool)
}

// Client implements PricingClient with embedded JSON data
type Client struct {
	region   string
	currency string

	// Thread-safe initialization
	once sync.Once
	err  error

	// In-memory pricing indexes (built on first access)
	ec2Index map[string]ec2Price
	ebsIndex map[string]ebsPrice

	// RDS pricing indexes (key: "instanceType/engine" for instances, "volumeType" for storage)
	rdsInstanceIndex map[string]rdsInstancePrice
	rdsStorageIndex  map[string]rdsStoragePrice
}

// NewClient creates a Client from embedded rawPricingJSON
func NewClient() (*Client, error) {
	c := &Client{}
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

			// --- RDS Database Instances ---
			// RDS uses productFamily="Database Instance" for compute pricing
			if prod.ProductFamily == "Database Instance" {
				instClass := attrs["instanceType"]     // e.g., "db.t3.medium"
				engine := attrs["databaseEngine"]      // e.g., "MySQL", "PostgreSQL"
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
			log.Printf("[pulumicost-plugin-aws-public] WARN: EC2 pricing lookup for %s/%s/%s took %v (>50ms)",
				instanceType, os, tenancy, elapsed)
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
			log.Printf("[pulumicost-plugin-aws-public] WARN: EBS pricing lookup for %s took %v (>50ms)",
				volumeType, elapsed)
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

// RDSOnDemandPricePerHour returns hourly rate for an RDS instance
// instanceType: e.g., "db.t3.medium"
// engine: normalized engine name, e.g., "MySQL", "PostgreSQL"
func (c *Client) RDSOnDemandPricePerHour(instanceType, engine string) (float64, bool) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			log.Printf("[pulumicost-plugin-aws-public] WARN: RDS pricing lookup for %s/%s took %v (>50ms)",
				instanceType, engine, elapsed)
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
			log.Printf("[pulumicost-plugin-aws-public] WARN: RDS storage pricing lookup for %s took %v (>50ms)",
				volumeType, elapsed)
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