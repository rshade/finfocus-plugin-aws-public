//go:build region_apne1

package pricing

import _ "embed"

// Per-service pricing data for ap-northeast-1.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_ap-northeast-1.json
var rawEC2JSON []byte

//go:embed data/s3_ap-northeast-1.json
var rawS3JSON []byte

//go:embed data/rds_ap-northeast-1.json
var rawRDSJSON []byte

//go:embed data/eks_ap-northeast-1.json
var rawEKSJSON []byte

//go:embed data/lambda_ap-northeast-1.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_ap-northeast-1.json
var rawDynamoDBJSON []byte

//go:embed data/elb_ap-northeast-1.json
var rawELBJSON []byte
