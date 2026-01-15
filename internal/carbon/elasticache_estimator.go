package carbon

import "strings"

// ElastiCacheEstimator estimates carbon footprint for ElastiCache clusters.
type ElastiCacheEstimator struct{}

// NewElastiCacheEstimator returns a pointer to a new ElastiCacheEstimator.
func NewElastiCacheEstimator() *ElastiCacheEstimator {
	return &ElastiCacheEstimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for an ElastiCache cluster.
//
// ElastiCache carbon is calculated as:
//   Compute carbon: EC2-equivalent carbon for the node type * number of nodes
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
	nodes := config.Nodes
	if nodes <= 0 {
		nodes = 1
	}

	totalCarbon := nodeCarbon * float64(nodes)

	return totalCarbon, true
}

// GetBillingDetail returns a human-readable description of the carbon estimation.
func (e *ElastiCacheEstimator) GetBillingDetail(config ElastiCacheConfig) string {
	nodes := config.Nodes
	if nodes <= 0 {
		nodes = 1
	}

	return "ElastiCache " + config.NodeType + " (" + config.Engine + "), " +
		formatInt(nodes) + " nodes, " +
		formatFloat(config.Hours) + " hrs, " +
		formatInt(int(config.Utilization*100)) + "% utilization"
}

// elasticacheToEC2InstanceType converts an ElastiCache node type to its EC2 equivalent.
// elasticacheToEC2InstanceType converts an ElastiCache node type to the equivalent EC2 instance type.
// If the input begins with the "cache." prefix (for example, "cache.m5.large"), the prefix is removed;
// otherwise the input is returned unchanged.
func elasticacheToEC2InstanceType(nodeType string) string {
	// Remove "cache." prefix if present
	if strings.HasPrefix(nodeType, "cache.") {
		return nodeType[6:]
	}
	return nodeType
}