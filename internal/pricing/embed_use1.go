//go:build region_use1

package pricing

import _ "embed"

//go:embed data/aws_pricing_us-east-1.json
var rawPricingJSON []byte