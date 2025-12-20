package carbon

import (
	_ "embed"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"sync"
)

// CSV column indices from CCF aws-instances.csv
// Source: https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients/blob/main/data/aws-instances.csv
const (
	colInstanceType = 0  // Instance type (e.g., "t3.micro")
	colVCPUCount    = 2  // Instance vCPU
	colMinWatts     = 14 // PkgWatt @ Idle
	colMaxWatts     = 17 // PkgWatt @ 100%
)

//go:embed data/ccf_instance_specs.csv
var instanceSpecsCSV string

// InstanceSpec contains power consumption characteristics for an EC2 instance type.
type InstanceSpec struct {
	InstanceType string
	VCPUCount    int
	MinWatts     float64 // Power consumption at idle (watts per vCPU)
	MaxWatts     float64 // Power consumption at 100% utilization (watts per vCPU)
}

var (
	instanceSpecs     map[string]InstanceSpec
	instanceSpecsOnce sync.Once
)

// parseInstanceSpecs parses the embedded CSV data into the instanceSpecs map.
// This function is called exactly once via sync.Once.
func parseInstanceSpecs() {
	instanceSpecs = make(map[string]InstanceSpec)

	reader := csv.NewReader(strings.NewReader(instanceSpecsCSV))

	// Skip header row
	_, err := reader.Read()
	if err != nil {
		return
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows
			continue
		}

		// Ensure we have enough columns
		if len(record) <= colMaxWatts {
			continue
		}

		instanceType := strings.TrimSpace(record[colInstanceType])
		if instanceType == "" {
			continue
		}

		// Parse vCPU count
		vcpuCount, err := strconv.Atoi(strings.TrimSpace(record[colVCPUCount]))
		if err != nil || vcpuCount < 1 {
			continue
		}

		// Parse min/max watts (CSV uses comma as decimal separator)
		minWatts := parseEuropeanFloat(record[colMinWatts])
		maxWatts := parseEuropeanFloat(record[colMaxWatts])

		// Skip invalid power values
		if minWatts < 0 || maxWatts < minWatts {
			continue
		}

		instanceSpecs[instanceType] = InstanceSpec{
			InstanceType: instanceType,
			VCPUCount:    vcpuCount,
			MinWatts:     minWatts,
			MaxWatts:     maxWatts,
		}
	}
}

// parseEuropeanFloat parses a float from European format (comma as decimal separator).
func parseEuropeanFloat(s string) float64 {
	s = strings.TrimSpace(s)
	// Replace comma with period for European format
	s = strings.ReplaceAll(s, ",", ".")
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

// GetInstanceSpec returns the power consumption specification for an instance type.
// Returns false if the instance type is not found in the CCF data.
func GetInstanceSpec(instanceType string) (InstanceSpec, bool) {
	instanceSpecsOnce.Do(parseInstanceSpecs)
	spec, ok := instanceSpecs[instanceType]
	return spec, ok
}

// InstanceSpecCount returns the number of loaded instance specifications.
// Useful for testing and validation.
func InstanceSpecCount() int {
	instanceSpecsOnce.Do(parseInstanceSpecs)
	return len(instanceSpecs)
}
