package carbon

import "strings"

// ElastiCacheEstimator estimates carbon footprint for ElastiCache clusters.
type ElastiCacheEstimator struct{}

// NewElastiCacheEstimator creates a new ElastiCache carbon estimator.
func NewElastiCacheEstimator() *ElastiCacheEstimator {
	return &ElastiCacheEstimator{}
}

// resolveNodeCount returns the node count, defaulting to 1 if nodes <= 0.
func (e *ElastiCacheEstimator) resolveNodeCount(nodes int) int {
	if nodes <= 0 {
		return 1
	}
	return nodes
}

// EstimateCarbonGrams calculates the carbon footprint for an ElastiCache cluster.
//
// ElastiCache carbon is calculated as:
//
//	Compute carbon: EC2-equivalent carbon for the node type * number of nodes
//
// Parameters:
//   - config: ElastiCache cluster configuration
//
// Returns the carbon footprint in grams CO2e and whether the calculation succeeded.
// Returns (0, false) if the node type is unknown.
//
// This method is thread-safe and can be called concurrently.
func (e *ElastiCacheEstimator) EstimateCarbonGrams(config ElastiCacheConfig) (float64, bool) {
	// Convert ElastiCache node type to EC2 equivalent (cache.m5.large -> m5.large)
	ec2InstanceType := elasticacheToEC2InstanceType(config.NodeType)

	// Calculate compute carbon using a fresh EC2 estimator with GPU disabled.
	// ElastiCache nodes don't have GPUs.
	ec2Estimator := NewEstimator()
	ec2Estimator.IncludeGPU = false
	nodeCarbon, ok := ec2Estimator.EstimateCarbonGrams(
		ec2InstanceType,
		config.Region,
		config.Utilization,
		config.Hours,
	)
	if !ok {
		return 0, false
	}

	// Total carbon is node carbon multiplied by the number of nodes.
	// Defaults to 1 node if config.Nodes is 0 or less.
	nodes := e.resolveNodeCount(config.Nodes)
	totalCarbon := nodeCarbon * float64(nodes)

	return totalCarbon, true
}

// GetBillingDetail returns a human-readable description of the carbon estimation.
func (e *ElastiCacheEstimator) GetBillingDetail(config ElastiCacheConfig) string {
	nodes := e.resolveNodeCount(config.Nodes)

	return "ElastiCache " + config.NodeType + " (" + config.Engine + "), " +
		formatInt(nodes) + " nodes, " +
		formatFloat(config.Hours) + " hrs, " +
		formatInt(int(config.Utilization*100)) + "% utilization"
}

// elasticacheToEC2InstanceType converts an ElastiCache node type to its EC2 equivalent.
// elasticacheToEC2InstanceType returns the EC2-equivalent instance type for an ElastiCache node type by removing the leading "cache." prefix if present.
// If the input does not have that prefix, it is returned unchanged.
func elasticacheToEC2InstanceType(nodeType string) string {
	return strings.TrimPrefix(nodeType, "cache.")
}