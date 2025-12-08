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
