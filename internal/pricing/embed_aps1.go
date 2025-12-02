//go:build region_aps1

package pricing

import _ "embed"

//go:embed data/aws_pricing_ap-south-1.json
var rawPricingJSON []byte