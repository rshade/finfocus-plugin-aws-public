//go:build region_usw1

package pricing

import _ "embed"

//go:embed data/aws_pricing_us-west-1.json
var rawPricingJSON []byte
