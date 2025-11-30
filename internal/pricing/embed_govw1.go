//go:build region_govw1

package pricing

import _ "embed"

//go:embed data/aws_pricing_us-gov-west-1.json
var rawPricingJSON []byte
