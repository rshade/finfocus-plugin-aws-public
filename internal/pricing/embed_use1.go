//go:build region_use1

package pricing

import _ "embed"

// Per-service pricing data for us-east-1.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_us-east-1.json
var rawEC2JSON []byte

//go:embed data/s3_us-east-1.json
var rawS3JSON []byte

//go:embed data/rds_us-east-1.json
var rawRDSJSON []byte

//go:embed data/eks_us-east-1.json
var rawEKSJSON []byte

//go:embed data/lambda_us-east-1.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_us-east-1.json
var rawDynamoDBJSON []byte

//go:embed data/elb_us-east-1.json
var rawELBJSON []byte

//go:embed data/vpc_us-east-1.json
var rawVPCJSON []byte

//go:embed data/cloudwatch_us-east-1.json
var rawCloudWatchJSON []byte

//go:embed data/elasticache_us-east-1.json
var rawElastiCacheJSON []byte
