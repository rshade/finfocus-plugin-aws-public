package carbon

// ASGEstimator estimates carbon footprint for Auto Scaling Groups.
type ASGEstimator struct{}

// NewASGEstimator creates a new ASG carbon estimator.
func NewASGEstimator() *ASGEstimator {
	return &ASGEstimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for an ASG.
//
// ASG carbon is calculated as:
//
//	EC2 single-instance carbon × desired_capacity
//
// Returns the carbon footprint in grams CO2e and whether the calculation succeeded.
// Returns (0, false) if the instance type is unknown in the CCF data.
// Returns (0, true) if desired capacity is zero (valid but no carbon).
//
// This method is thread-safe and can be called concurrently.
func (e *ASGEstimator) EstimateCarbonGrams(config ASGConfig) (float64, bool) {
	if config.DesiredCapacity <= 0 {
		return 0, true
	}

	ec2Estimator := NewEstimator()
	singleCarbon, ok := ec2Estimator.EstimateCarbonGrams(
		config.InstanceType,
		config.Region,
		config.Utilization,
		config.Hours,
	)
	if !ok {
		return 0, false
	}

	return singleCarbon * float64(config.DesiredCapacity), true
}
