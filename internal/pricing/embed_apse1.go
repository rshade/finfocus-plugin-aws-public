//go:build region_apse1

package pricing

import _ "embed"

// Per-service pricing data for ap-southeast-1.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_ap-southeast-1.json
var rawEC2JSON []byte

//go:embed data/s3_ap-southeast-1.json
var rawS3JSON []byte

//go:embed data/rds_ap-southeast-1.json
var rawRDSJSON []byte

//go:embed data/eks_ap-southeast-1.json
var rawEKSJSON []byte

//go:embed data/lambda_ap-southeast-1.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_ap-southeast-1.json
var rawDynamoDBJSON []byte

//go:embed data/elb_ap-southeast-1.json
var rawELBJSON []byte
