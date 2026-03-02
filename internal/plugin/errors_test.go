package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
