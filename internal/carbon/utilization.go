package carbon

// GetUtilization determines the CPU utilization to use for carbon calculations.
// Priority order: perResourceUtil > requestUtil > DefaultUtilization.
//
// Parameters:
//   - requestUtil: Utilization from GetProjectedCostRequest.UtilizationPercentage
//   - perResourceUtil: Utilization from ResourceDescriptor.UtilizationPercentage (nil if not set)
//
// GetUtilization determines the CPU utilization value to use for carbon calculations.
// It selects a value by priority: a non-nil perResourceUtil greater than 0, then
// requestUtil if greater than 0, and finally DefaultUtilization. Any selected value
// is clamped to the range [0.0, 1.0]; if perResourceUtil is nil it is ignored.
//
 // GetUtilization selects the CPU utilization to use for carbon calculations.
// If perResourceUtil is non-nil and greater than 0, its value (clamped to [0.0, 1.0]) is used.
// Otherwise, if requestUtil is greater than 0, requestUtil (clamped to [0.0, 1.0]) is used.
// If neither provides a positive value, DefaultUtilization is returned.
// perResourceUtil is an optional per-resource override; requestUtil is the request-level value.
// The returned value is a float64 in the range [0.0, 1.0].
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

// Clamp returns v constrained to the inclusive range [min, max]. If v is less than min, Clamp returns min; if v is greater than max, Clamp returns max; otherwise it returns v.
func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}