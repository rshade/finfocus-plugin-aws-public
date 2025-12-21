//go:build region_govw1

package pricing

import _ "embed"

// Per-service pricing data for us-gov-west-1.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_us-gov-west-1.json
var rawEC2JSON []byte

//go:embed data/s3_us-gov-west-1.json
var rawS3JSON []byte

//go:embed data/rds_us-gov-west-1.json
var rawRDSJSON []byte

//go:embed data/eks_us-gov-west-1.json
var rawEKSJSON []byte

//go:embed data/lambda_us-gov-west-1.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_us-gov-west-1.json
var rawDynamoDBJSON []byte

//go:embed data/elb_us-gov-west-1.json
var rawELBJSON []byte
