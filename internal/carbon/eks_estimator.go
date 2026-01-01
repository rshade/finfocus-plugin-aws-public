package carbon

// EKSEstimator estimates carbon footprint for EKS clusters.
type EKSEstimator struct{}

// NewEKSEstimator creates a new EKS carbon estimator.
func NewEKSEstimator() *EKSEstimator {
	return &EKSEstimator{}
}

// EstimateCarbonGrams returns zero carbon for EKS control plane.
//
// EKS control plane carbon is not allocated to customers because:
//  1. The control plane is shared across customers (multi-tenant)
//  2. AWS Customer Carbon Footprint Tool excludes control plane from allocations
//  3. Worker nodes should be estimated as EC2 instances separately
//
// Parameters:
//   - config: EKS cluster configuration
//
// Returns (0, true) always - the billing detail explains worker node estimation.
func (e *EKSEstimator) EstimateCarbonGrams(config EKSClusterConfig) (float64, bool) {
	// Control plane carbon is shared and not allocated to customers
	return 0, true
}

// GetBillingDetail returns a human-readable description explaining EKS carbon.
// The guidance for worker node estimation is included directly in the billing detail
// since that's what users see in cost responses.
func (e *EKSEstimator) GetBillingDetail(config EKSClusterConfig) string {
	return "EKS control plane carbon is shared and not allocated to customers. " +
		"Estimate worker nodes as EC2 instances for cluster carbon footprint."
}
