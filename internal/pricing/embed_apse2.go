//go:build region_apse2

package pricing

import _ "embed"

//go:embed data/aws_pricing_ap-southeast-2.json
var rawPricingJSON []byte