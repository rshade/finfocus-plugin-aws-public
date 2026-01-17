package plugin

import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

// ServiceClassification defines metadata for cost estimation enrichment
type ServiceClassification struct {
	// GrowthType specifies the cost growth pattern for forecasting
	GrowthType pbc.GrowthType

	// AffectedByDevMode indicates if dev mode multipliers affect this service
	AffectedByDevMode bool

	// ParentTagKeys defines priority order for extracting parent resource identifiers
	ParentTagKeys []string

	// ParentType specifies the expected parent resource type for relationships
	ParentType string

	// Relationship defines the cost allocation relationship type
	Relationship string
}

// serviceClassifications is a read-only map of AWS service types to their metadata
var serviceClassifications = map[string]ServiceClassification{
	"aws:ec2:instance": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: true, // Instance hours
		ParentTagKeys:     nil,
	},
	"aws:ebs:volume": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: false, // Storage is not time-based
		ParentTagKeys:     []string{"instance_id"},
		ParentType:        "aws:ec2:instance:Instance",
		Relationship:      RelationshipAttachedTo,
	},
	"aws:eks:cluster": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: true, // Cluster hours
		ParentTagKeys:     nil,
	},
	"aws:s3:bucket": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_LINEAR,
		AffectedByDevMode: false, // Storage is not time-based
		ParentTagKeys:     nil,
	},
	"aws:lambda:function": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: false, // Usage-based
		ParentTagKeys:     nil,
	},
	"aws:dynamodb:table": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_LINEAR,
		AffectedByDevMode: false, // Usage-based
		ParentTagKeys:     nil,
	},
	"aws:elasticloadbalancing:loadbalancer": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: true, // Load balancer hours
		ParentTagKeys:     []string{"vpc_id"},
		ParentType:        "aws:ec2:vpc:Vpc",
		Relationship:      RelationshipWithin,
	},
	"aws:ec2:nat-gateway": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: true, // Gateway hours
		ParentTagKeys:     []string{"vpc_id", "subnet_id"},
		ParentType:        "aws:ec2:vpc:Vpc",
		Relationship:      RelationshipWithin,
	},
	"aws:cloudwatch:metric": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: false, // Ingestion is throughput
		ParentTagKeys:     nil,
	},
	"aws:elasticache:cluster": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: true, // Node hours
		ParentTagKeys:     []string{"vpc_id"},
		ParentType:        "aws:ec2:vpc:Vpc",
		Relationship:      RelationshipWithin,
	},
	"aws:rds:instance": {
		GrowthType:        pbc.GrowthType_GROWTH_TYPE_NONE,
		AffectedByDevMode: true, // Instance hours
		ParentTagKeys:     []string{"vpc_id"},
		ParentType:        "aws:ec2:vpc:Vpc",
		Relationship:      RelationshipWithin,
	},
}

// GetServiceClassification retrieves the classification metadata for a service type.
// Returns the ServiceClassification and a boolean indicating if the service was found.
func GetServiceClassification(serviceType string) (ServiceClassification, bool) {
	classification, ok := serviceClassifications[serviceType]
	return classification, ok
}
