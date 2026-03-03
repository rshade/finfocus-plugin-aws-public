package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPricingUnavailableError verifies PricingUnavailableError message formatting
// with and without explicit billing detail text.
// Why: this error text is surfaced to clients and must remain stable for fallback logic.
// Workflow: subtests validate custom-detail precedence and default template fallback.
// Run: go test ./internal/plugin -run TestPricingUnavailableError -v.
func TestPricingUnavailableError(t *testing.T) {
	t.Run("with billing detail", func(t *testing.T) {
		err := &PricingUnavailableError{
			Service:       "EC2",
			SKU:           "XYZ-123",
			BillingDetail: "Custom detail message",
		}
		assert.Equal(t, "Custom detail message", err.Error())
	})

	t.Run("without billing detail", func(t *testing.T) {
		err := &PricingUnavailableError{
			Service: "S3",
			SKU:     "ABC-456",
		}
		assert.Equal(t, "pricing unavailable for S3 (SKU: ABC-456)", err.Error())
	})
}
