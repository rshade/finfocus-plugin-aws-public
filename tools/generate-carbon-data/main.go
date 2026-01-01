// Package main provides a tool to fetch Cloud Carbon Footprint (CCF) instance
// specifications for carbon emission estimation.
//
// The tool downloads the AWS instances CSV from the cloud-carbon-coefficients
// repository and saves it to internal/carbon/data/ccf_instance_specs.csv for
// embedding at build time.
//
// Usage:
//
//	go run ./tools/generate-carbon-data [--out-dir DIR] [--validate]
//
// Flags:
//
//	--out-dir   Output directory (default: ./internal/carbon/data)
//	--validate  Validate the downloaded CSV has expected columns and row count
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// ccfDataURL is the raw GitHub URL for the CCF AWS instances CSV.
	// Source: https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients
	// License: Apache 2.0
	ccfDataURL = "https://raw.githubusercontent.com/cloud-carbon-footprint/cloud-carbon-coefficients/main/data/aws-instances.csv"

	// outputFileName is the name of the generated CSV file.
	outputFileName = "ccf_instance_specs.csv"

	// expectedMinRows is the minimum number of instance types expected.
	// CCF data contains 500+ instance types; fewer indicates a problem.
	expectedMinRows = 400

	// expectedColumns are the column indices we use from the CSV.
	// See internal/carbon/instance_specs.go for column mapping.
	colInstanceType = 0  // Instance type (e.g., "t3.micro")
	colVCPUCount    = 2  // Instance vCPU
	colMinWatts     = 14 // PkgWatt @ Idle
	colMaxWatts     = 17 // PkgWatt @ 100%
)

// main fetches the CCF AWS instances CSV and saves it to the output directory.
// It validates the data if the --validate flag is set.
func main() {
	outDir := flag.String("out-dir", "./internal/carbon/data", "Output directory for the CSV file")
	validate := flag.Bool("validate", true, "Validate the downloaded CSV has expected structure")
	flag.Parse()

	fmt.Println("Fetching Cloud Carbon Footprint AWS instance specs...")
	fmt.Printf("Source: %s\n", ccfDataURL)

	// Fetch the CSV data
	data, err := fetchCCFData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching CCF data: %v\n", err)
		os.Exit(1)
	}

	// Validate if requested
	if *validate {
		if err := validateCSV(data); err != nil {
			fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Validation passed")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Write the CSV file
	outPath := filepath.Join(*outDir, outputFileName)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote %s (%d bytes)\n", outPath, len(data))

	// Write static GPU and storage specs
	if err := writeStaticSpecs(*outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing static specs: %v\n", err)
		os.Exit(1)
	}
}

// fetchCCFData downloads the CCF AWS instances CSV from GitHub.
func fetchCCFData() ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(ccfDataURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// validateCSV checks that the CSV has the expected structure and content.
func validateCSV(data []byte) error {
	reader := csv.NewReader(strings.NewReader(string(data)))

	// Read and validate header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Check we have enough columns
	if len(header) <= colMaxWatts {
		return fmt.Errorf("CSV has %d columns, expected at least %d", len(header), colMaxWatts+1)
	}

	// Count valid rows and track parsing issues
	validRows := 0
	parseErrors := 0
	totalRows := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			parseErrors++
			continue
		}
		totalRows++

		// Check this row has enough columns
		if len(record) <= colMaxWatts {
			continue
		}

		// Check instance type is non-empty
		instanceType := strings.TrimSpace(record[colInstanceType])
		if instanceType == "" {
			continue
		}

		// Check vCPU is a valid integer
		vcpuStr := strings.TrimSpace(record[colVCPUCount])
		vcpu, err := strconv.Atoi(vcpuStr)
		if err != nil || vcpu < 1 {
			continue
		}

		// Check min/max watts are parseable (European format with commas)
		minWatts := parseEuropeanFloat(record[colMinWatts])
		maxWatts := parseEuropeanFloat(record[colMaxWatts])
		if minWatts < 0 || maxWatts < minWatts {
			continue
		}

		validRows++
	}

	fmt.Printf("CSV stats: %d total rows, %d valid instance specs, %d parse errors\n",
		totalRows, validRows, parseErrors)

	if validRows < expectedMinRows {
		return fmt.Errorf("only %d valid instance specs found, expected at least %d", validRows, expectedMinRows)
	}

	return nil
}

// parseEuropeanFloat parses a float that may use comma as decimal separator.
func parseEuropeanFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", ".")
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1 // Return negative to indicate parse error
	}
	return val
}

// gpuSpecsCSV is the static GPU specifications data.
// Source: AWS documentation for GPU instance types.
// TDP values from NVIDIA/AWS specifications.
const gpuSpecsCSV = `instance_type,gpu_model,gpu_count,tdp_per_gpu_watts
p4d.24xlarge,A100,8,400
p4de.24xlarge,A100,8,400
p5.48xlarge,H100,8,700
g4dn.xlarge,T4,1,70
g4dn.2xlarge,T4,1,70
g4dn.4xlarge,T4,1,70
g4dn.8xlarge,T4,1,70
g4dn.12xlarge,T4,4,70
g4dn.16xlarge,T4,1,70
g4dn.metal,T4,8,70
g5.xlarge,A10G,1,150
g5.2xlarge,A10G,1,150
g5.4xlarge,A10G,1,150
g5.8xlarge,A10G,1,150
g5.12xlarge,A10G,4,150
g5.16xlarge,A10G,1,150
g5.24xlarge,A10G,4,150
g5.48xlarge,A10G,8,150
inf1.xlarge,Inferentia,1,75
inf1.2xlarge,Inferentia,1,75
inf1.6xlarge,Inferentia,4,75
inf1.24xlarge,Inferentia,16,75
inf2.xlarge,Inferentia2,1,175
inf2.8xlarge,Inferentia2,2,175
inf2.24xlarge,Inferentia2,6,175
inf2.48xlarge,Inferentia2,12,175
trn1.2xlarge,Trainium,1,175
trn1.32xlarge,Trainium,16,175
trn1n.32xlarge,Trainium,16,175
`

// storageSpecsCSV is the static storage specifications data.
// Source: Cloud Carbon Footprint methodology.
// Power coefficients in Wh/TB-hour, replication factors for durability.
// Format: service_type,storage_class,technology,replication_factor,power_coefficient
const storageSpecsCSV = `service_type,storage_class,technology,replication_factor,power_coefficient
ebs,gp2,SSD,2,1.2
ebs,gp3,SSD,2,1.2
ebs,io1,SSD,2,1.2
ebs,io2,SSD,2,1.2
ebs,st1,HDD,2,0.65
ebs,sc1,HDD,2,0.65
s3,STANDARD,SSD,3,1.2
s3,ONEZONE_IA,SSD,1,1.2
s3,GLACIER,HDD,3,0.65
dynamodb,DYNAMODB,SSD,3,1.2
`

// writeStaticSpecs writes the GPU and storage specification CSV files.
func writeStaticSpecs(outDir string) error {
	// Write GPU specs
	gpuPath := filepath.Join(outDir, "gpu_specs.csv")
	if err := os.WriteFile(gpuPath, []byte(gpuSpecsCSV), 0644); err != nil {
		return fmt.Errorf("failed to write GPU specs: %w", err)
	}
	fmt.Printf("Successfully wrote %s (%d bytes)\n", gpuPath, len(gpuSpecsCSV))

	// Write storage specs
	storagePath := filepath.Join(outDir, "storage_specs.csv")
	if err := os.WriteFile(storagePath, []byte(storageSpecsCSV), 0644); err != nil {
		return fmt.Errorf("failed to write storage specs: %w", err)
	}
	fmt.Printf("Successfully wrote %s (%d bytes)\n", storagePath, len(storageSpecsCSV))

	return nil
}
