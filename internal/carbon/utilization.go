package carbon

// GetUtilization determines the CPU utilization to use for carbon calculations.
// It selects a value by priority:
//  1. perResourceUtil: A non-nil positive override from ResourceDescriptor (clamped to [0.0, 1.0])
//  2. requestUtil: A positive value from GetProjectedCostRequest (clamped to [0.0, 1.0])
//  3. DefaultUtilization: A fallback constant if neither above provide a positive value.
//
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
