//go:build region_euw1

package pricing

import _ "embed"

//go:embed data/aws_pricing_eu-west-1.json
var rawPricingJSON []byte
