//go:build region_cac1

package pricing

import _ "embed"

//go:embed data/aws_pricing_ca-central-1.json
var rawPricingJSON []byte