//go:build region_gove1

package pricing

import _ "embed"

//go:embed data/aws_pricing_us-gov-east-1.json
var rawPricingJSON []byte
