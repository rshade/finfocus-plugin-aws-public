//go:build region_sae1

package pricing

import _ "embed"

//go:embed data/aws_pricing_sa-east-1.json
var rawPricingJSON []byte
