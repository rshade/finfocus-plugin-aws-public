package plugin

import "fmt"

// PricingUnavailableError is returned when a cost calculation cannot be performed
// because the required pricing data is missing from the embedded database.
// This is used to signal GetActualCost to return empty results for fallback.
type PricingUnavailableError struct {
	Service       string // e.g., "EC2", "S3"
	SKU           string // The unique identifier used for lookup
	BillingDetail string // Human-readable explanation (for GetProjectedCost backward compat)
}

func (e *PricingUnavailableError) Error() string {
	if e.BillingDetail != "" {
		return e.BillingDetail
	}
	return fmt.Sprintf("pricing unavailable for %s (SKU: %s)", e.Service, e.SKU)
}
