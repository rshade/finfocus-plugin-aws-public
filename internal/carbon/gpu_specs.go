package carbon

import (
	_ "embed"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"sync"
)

// CSV column indices for GPU specs.
const (
	colGPUInstanceType = 0 // instance_type
	colGPUModel        = 1 // gpu_model
	colGPUCount        = 2 // gpu_count
	colTDPPerGPU       = 3 // tdp_per_gpu_watts
)

//go:embed data/gpu_specs.csv
var gpuSpecsCSV string

// GPUSpec contains GPU specifications for an EC2 instance type.
type GPUSpec struct {
	// InstanceType is the GPU instance type (e.g., "p4d.24xlarge").
	InstanceType string

	// GPUModel is the GPU model name (e.g., "A100").
	GPUModel string

	// GPUCount is the number of GPUs per instance.
	GPUCount int

	// TDPPerGPU is the Thermal Design Power per GPU in watts.
	TDPPerGPU float64
}

var (
	gpuSpecs     map[string]GPUSpec
	gpuSpecsOnce sync.Once
)

// parseGPUSpecs initializes the package-level gpuSpecs map by parsing
// the embedded CSV of GPU specifications.
func parseGPUSpecs() {
	gpuSpecs = make(map[string]GPUSpec)

	reader := csv.NewReader(strings.NewReader(gpuSpecsCSV))

	// Skip header row
	_, err := reader.Read()
	if err != nil {
		logger.Error().Err(err).Msg("failed to read GPU specs CSV header")
		return
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn().Err(err).Msg("skipping malformed GPU specs CSV row")
			continue
		}

		// Ensure we have enough columns
		if len(record) <= colTDPPerGPU {
			continue
		}

		instanceType := strings.TrimSpace(record[colGPUInstanceType])
		if instanceType == "" {
			continue
		}

		gpuModel := strings.TrimSpace(record[colGPUModel])

		// Parse GPU count
		gpuCount, err := strconv.Atoi(strings.TrimSpace(record[colGPUCount]))
		if err != nil || gpuCount < 0 {
			continue
		}

		// Parse TDP per GPU
		tdpPerGPU, err := strconv.ParseFloat(strings.TrimSpace(record[colTDPPerGPU]), 64)
		if err != nil || tdpPerGPU < 0 {
			continue
		}

		gpuSpecs[instanceType] = GPUSpec{
			InstanceType: instanceType,
			GPUModel:     gpuModel,
			GPUCount:     gpuCount,
			TDPPerGPU:    tdpPerGPU,
		}
	}
}

// GetGPUSpec retrieves the GPUSpec for the given EC2 instance type.
// Returns the GPUSpec and true if found, or an empty GPUSpec and false otherwise.
// Non-GPU instances will return (empty, false).
func GetGPUSpec(instanceType string) (GPUSpec, bool) {
	gpuSpecsOnce.Do(parseGPUSpecs)
	spec, ok := gpuSpecs[instanceType]
	return spec, ok
}

// HasGPU returns true if the instance type has GPU accelerators.
func HasGPU(instanceType string) bool {
	_, ok := GetGPUSpec(instanceType)
	return ok
}

// GPUSpecCount reports the number of loaded GPU specifications.
func GPUSpecCount() int {
	gpuSpecsOnce.Do(parseGPUSpecs)
	return len(gpuSpecs)
}

// CalculateGPUPowerWatts calculates the total GPU power consumption for an instance.
// Returns the power in watts based on GPU count, TDP, and utilization.
func CalculateGPUPowerWatts(instanceType string, utilization float64) float64 {
	spec, ok := GetGPUSpec(instanceType)
	if !ok {
		return 0
	}

	// GPU power = TDP per GPU × GPU count × utilization
	// Note: This is a simplified model; actual GPU power varies with workload
	return spec.TDPPerGPU * float64(spec.GPUCount) * utilization
}
