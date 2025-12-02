//go:build region_usw2

package pricing

import _ "embed"

//go:embed data/aws_pricing_us-west-2.json
var rawPricingJSON []byte