//go:build region_cac1

package pricing

import _ "embed"

// Per-service pricing data for ca-central-1.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_ca-central-1.json
var rawEC2JSON []byte

//go:embed data/s3_ca-central-1.json
var rawS3JSON []byte

//go:embed data/rds_ca-central-1.json
var rawRDSJSON []byte

//go:embed data/eks_ca-central-1.json
var rawEKSJSON []byte

//go:embed data/lambda_ca-central-1.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_ca-central-1.json
var rawDynamoDBJSON []byte

//go:embed data/elb_ca-central-1.json
var rawELBJSON []byte

//go:embed data/vpc_ca-central-1.json
var rawVPCJSON []byte

//go:embed data/cloudwatch_ca-central-1.json
var rawCloudWatchJSON []byte
