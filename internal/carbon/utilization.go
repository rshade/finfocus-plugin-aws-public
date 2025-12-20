package carbon

// GetUtilization determines the CPU utilization to use for carbon calculations.
// Priority order: perResourceUtil > requestUtil > DefaultUtilization.
//
// Parameters:
//   - requestUtil: Utilization from GetProjectedCostRequest.UtilizationPercentage
//   - perResourceUtil: Utilization from ResourceDescriptor.UtilizationPercentage (nil if not set)
//
// Returns: Clamped utilization value between 0.0 and 1.0
func GetUtilization(requestUtil float64, perResourceUtil *float64) float64 {
	// Priority 1: Per-resource override
	if perResourceUtil != nil && *perResourceUtil > 0 {
		return Clamp(*perResourceUtil, 0.0, 1.0)
	}

	// Priority 2: Request-level value
	if requestUtil > 0 {
		return Clamp(requestUtil, 0.0, 1.0)
	}

	// Priority 3: Default
	return DefaultUtilization
}

// Clamp restricts a value to the range [min, max].
func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
