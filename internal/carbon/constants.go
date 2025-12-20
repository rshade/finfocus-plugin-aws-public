// Package carbon provides carbon emission estimation for AWS resources
// using Cloud Carbon Footprint (CCF) methodology.
package carbon

const (
	// AWSPUE is the Power Usage Effectiveness for AWS datacenters.
	// Source: Cloud Carbon Footprint methodology.
	AWSPUE = 1.135

	// DefaultUtilization is the default CPU utilization assumption (50%)
	// when no utilization_percentage is provided.
	// Source: CCF hyperscale datacenter assumption.
	DefaultUtilization = 0.50

	// HoursPerMonth is the standard hours per month for cost calculations.
	HoursPerMonth = 730.0
)
