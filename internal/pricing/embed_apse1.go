//go:build region_apse1

package pricing

import _ "embed"

//go:embed data/aws_pricing_ap-southeast-1.json
var rawPricingJSON []byte