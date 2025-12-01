//go:build region_apne1

package pricing

import _ "embed"

//go:embed data/aws_pricing_ap-northeast-1.json
var rawPricingJSON []byte