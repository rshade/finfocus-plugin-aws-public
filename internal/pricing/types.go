package pricing

// awsPricing represents the structure of the official AWS Price List API JSON
type awsPricing struct {
	FormatVersion   string                                `json:"formatVersion"`
	Disclaimer      string                                `json:"disclaimer"`
	OfferCode       string                                `json:"offerCode"`
	Version         string                                `json:"version"`
	PublicationDate string                                `json:"publicationDate"`
	Products        map[string]product                    `json:"products"`
	Terms           map[string]map[string]map[string]term `json:"terms"` // Type -> SKU -> OfferTermCode -> Term
}

type product struct {
	Sku           string            `json:"sku"`
	ProductFamily string            `json:"productFamily"`
	Attributes    map[string]string `json:"attributes"`
}

type term struct {
	OfferTermCode   string                    `json:"offerTermCode"`
	Sku             string                    `json:"sku"`
	EffectiveDate   string                    `json:"effectiveDate"`
	PriceDimensions map[string]priceDimension `json:"priceDimensions"`
}

type priceDimension struct {
	RateCode     string            `json:"rateCode"`
	Description  string            `json:"description"`
	BeginRange   string            `json:"beginRange"`
	EndRange     string            `json:"endRange"`
	Unit         string            `json:"unit"`
	PricePerUnit map[string]string `json:"pricePerUnit"` // Currency -> Amount (string)
	AppliesTo    []string          `json:"appliesTo"`
}

// Internal lookup structures (distilled from raw JSON)
type ec2Price struct {
	Unit       string
	HourlyRate float64
	Currency   string
}

type ebsPrice struct {
	Unit           string
	RatePerGBMonth float64
	Currency       string
}