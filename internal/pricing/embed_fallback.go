//go:build !region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1 && !region_cac1 && !region_sae1

package pricing

// rawPricingJSON contains fallback dummy pricing data for development
// This is used when no region-specific build tag is provided
var rawPricingJSON = []byte(`{
  "region": "unknown",
  "currency": "USD",
  "ec2": {
    "t3.micro": {
      "instance_type": "t3.micro",
      "operating_system": "Linux",
      "tenancy": "Shared",
      "hourly_rate": 0.0104
    },
    "t3.small": {
      "instance_type": "t3.small",
      "operating_system": "Linux",
      "tenancy": "Shared",
      "hourly_rate": 0.0208
    }
  },
  "ebs": {
    "gp3": {
      "volume_type": "gp3",
      "rate_per_gb_month": 0.08
    },
    "gp2": {
      "volume_type": "gp2",
      "rate_per_gb_month": 0.10
    }
  }
}`)
