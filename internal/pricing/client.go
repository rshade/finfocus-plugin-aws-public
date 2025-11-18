package pricing

import (
	"encoding/json"
	"fmt"
	"sync"
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
}

// Client implements PricingClient with embedded JSON data
type Client struct {
	region   string
	currency string

	// Thread-safe initialization
	once sync.Once
	err  error

	// In-memory pricing indexes (built on first access)
	ec2Index map[string]ec2OnDemandPrice
	ebsIndex map[string]ebsVolumePrice
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
		var data pricingData
		if err := json.Unmarshal(rawPricingJSON, &data); err != nil {
			c.err = fmt.Errorf("failed to parse pricing data: %w", err)
			return
		}

		c.region = data.Region
		c.currency = data.Currency
		c.ec2Index = buildEC2Index(data.EC2)
		c.ebsIndex = buildEBSIndex(data.EBS)
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
	if err := c.init(); err != nil {
		return 0, false
	}

	price, found := c.ebsIndex[volumeType]
	if !found {
		return 0, false
	}
	return price.RatePerGBMonth, true
}

// buildEC2Index creates O(1) lookup map from pricing data
func buildEC2Index(ec2Data map[string]ec2OnDemandPrice) map[string]ec2OnDemandPrice {
	index := make(map[string]ec2OnDemandPrice, len(ec2Data))
	for _, price := range ec2Data {
		key := fmt.Sprintf("%s/%s/%s", price.InstanceType, price.OperatingSystem, price.Tenancy)
		index[key] = price
	}
	return index
}

// buildEBSIndex creates O(1) lookup map from pricing data
func buildEBSIndex(ebsData map[string]ebsVolumePrice) map[string]ebsVolumePrice {
	// Already keyed by volume type, just return as-is
	return ebsData
}
