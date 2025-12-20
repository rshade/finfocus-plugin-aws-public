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

func init() {
	if instanceSpecsCSV == "" {
		panic("CCF instance specs not embedded. Run: make generate-carbon-data")
	}
}

// parseInstanceSpecs initializes the package-level instanceSpecs map by parsing
// the embedded CSV of EC2 instance power specifications.
//
// The function reads the CSV, skips the header, and loads rows with:
//   - Non-empty instance type
//   - vCPU count >= 1
//   - Valid power values (MinWatts >= 0, MaxWatts >= MinWatts)
//
// European-formatted decimals (comma as decimal separator) are accepted.
// Malformed or incomplete rows are skipped. This function should be invoked
// once via sync.Once to populate the lookup map.
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

// parseEuropeanFloat parses a decimal number that may use a comma as the decimal
// separator and returns it as a float64. It trims surrounding whitespace and
// accepts either '.' or ',' as the decimal point. Returns 0 if the string
// cannot be parsed as a floating-point number.
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

// GetInstanceSpec retrieves the InstanceSpec for the given EC2 instance type
// from the embedded CCF data. It ensures the embedded CSV data is parsed once
// on first use (lazy initialization) and looks up the instanceType in the
// internal registry. Returns the InstanceSpec and true if found, or an empty
// InstanceSpec and false otherwise.
func GetInstanceSpec(instanceType string) (InstanceSpec, bool) {
	instanceSpecsOnce.Do(parseInstanceSpecs)
	spec, ok := instanceSpecs[instanceType]
	return spec, ok
}

// InstanceSpecCount reports the number of loaded instance specifications.
// It performs lazy initialization by parsing the embedded CSV on the first call.
// Returns the count of valid instance types parsed from the embedded data.
func InstanceSpecCount() int {
	instanceSpecsOnce.Do(parseInstanceSpecs)
	return len(instanceSpecs)
}