//go:build region_usw2

package pricing

import _ "embed"

// Per-service pricing data for us-west-2.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_us-west-2.json
var rawEC2JSON []byte

//go:embed data/s3_us-west-2.json
var rawS3JSON []byte

//go:embed data/rds_us-west-2.json
var rawRDSJSON []byte

//go:embed data/eks_us-west-2.json
var rawEKSJSON []byte

//go:embed data/lambda_us-west-2.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_us-west-2.json
var rawDynamoDBJSON []byte

//go:embed data/elb_us-west-2.json
var rawELBJSON []byte
