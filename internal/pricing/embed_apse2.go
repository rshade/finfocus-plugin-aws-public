//go:build region_apse2

package pricing

import _ "embed"

// Per-service pricing data for ap-southeast-2.
// Each file contains raw AWS Price List API response with preserved metadata.

//go:embed data/ec2_ap-southeast-2.json
var rawEC2JSON []byte

//go:embed data/s3_ap-southeast-2.json
var rawS3JSON []byte

//go:embed data/rds_ap-southeast-2.json
var rawRDSJSON []byte

//go:embed data/eks_ap-southeast-2.json
var rawEKSJSON []byte

//go:embed data/lambda_ap-southeast-2.json
var rawLambdaJSON []byte

//go:embed data/dynamodb_ap-southeast-2.json
var rawDynamoDBJSON []byte

//go:embed data/elb_ap-southeast-2.json
var rawELBJSON []byte

//go:embed data/vpc_ap-southeast-2.json
var rawVPCJSON []byte

//go:embed data/cloudwatch_ap-southeast-2.json
var rawCloudWatchJSON []byte

//go:embed data/elasticache_ap-southeast-2.json
var rawElastiCacheJSON []byte
