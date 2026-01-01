//go:build region_euw1

package pricing

import _ "embed"

// Per-service pricing data for eu-west-1.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_eu-west-1.json
var rawEC2JSON []byte

//go:embed data/s3_eu-west-1.json
var rawS3JSON []byte

//go:embed data/rds_eu-west-1.json
var rawRDSJSON []byte

//go:embed data/eks_eu-west-1.json
var rawEKSJSON []byte

//go:embed data/lambda_eu-west-1.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_eu-west-1.json
var rawDynamoDBJSON []byte

//go:embed data/elb_eu-west-1.json
var rawELBJSON []byte

//go:embed data/vpc_eu-west-1.json
var rawVPCJSON []byte

//go:embed data/cloudwatch_eu-west-1.json
var rawCloudWatchJSON []byte

//go:embed data/elasticache_eu-west-1.json
var rawElastiCacheJSON []byte
