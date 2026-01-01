// Package main provides a tool to update regional grid emission factors from the
// Cloud Carbon Footprint (CCF) cloud-carbon-coefficients repository.
//
// The tool fetches the latest grid emission factors and updates the
// internal/carbon/grid_factors.go file with the new values.
//
// Usage:
//
//	go run ./tools/update-grid-factors [--dry-run] [--validate]
//
// Flags:
//
//	--dry-run   Print changes without writing to file
//	--validate  Validate the fetched values are within expected range
//	--output    Path to grid_factors.go (default: ./internal/carbon/grid_factors.go)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	// CCF grid factors URL - this points to the emissions factors JSON
	// Note: CCF uses a structured data format in their repository
	ccfGridFactorsURL = "https://raw.githubusercontent.com/cloud-carbon-footprint/cloud-carbon-coefficients/main/data/grid-emissions-factors-aws.json"

	// Fallback: EPA eGRID data for US regions
	// https://www.epa.gov/egrid/power-profiler

	// Valid range for grid factors (metric tons CO2e per kWh)
	minValidFactor = 0.0     // Some regions like Iceland have near-zero carbon grids
	maxValidFactor = 2.0     // No grid should exceed 2 metric tons CO2e per kWh
	defaultFactor  = 0.00039 // Global average

	// Template for generating grid_factors.go
	fileTemplate = `package carbon

// GridEmissionFactors maps AWS region codes to grid carbon intensity.
// Values are in metric tons CO2eq per kWh.
//
// Source: Cloud Carbon Footprint methodology
// Data vintage: %s (update annually from CCF repository)
// Reference: https://www.cloudcarbonfootprint.org/docs/methodology
//
// To update these values, run:
//   go run ./tools/update-grid-factors
var GridEmissionFactors = map[string]float64{
%s}

// DefaultGridFactor is used when a region doesn't have a specific factor.
// This is the global average from CCF.
const DefaultGridFactor = %.8f

// GetGridFactor returns the grid carbon emission factor for the given AWS region
// in metric tons CO2e per kWh. If the region is not listed in GridEmissionFactors,
// DefaultGridFactor (global average) is returned.
func GetGridFactor(region string) float64 {
	if factor, ok := GridEmissionFactors[region]; ok {
		return factor
	}
	return DefaultGridFactor
}
`
)

// AWSRegionMapping maps AWS region codes to their location descriptions
// and EPA eGRID subregions (for US regions).
var AWSRegionMapping = map[string]struct {
	location string
	egrid    string // EPA eGRID subregion (for US only)
}{
	"us-east-1":      {"Virginia", "SERC"},
	"us-east-2":      {"Ohio", "RFC"},
	"us-west-1":      {"N. California", "WECC"},
	"us-west-2":      {"Oregon", "WECC"},
	"ca-central-1":   {"Canada", ""},
	"eu-west-1":      {"Ireland", ""},
	"eu-west-2":      {"London", ""},
	"eu-west-3":      {"Paris", ""},
	"eu-central-1":   {"Frankfurt", ""},
	"eu-north-1":     {"Sweden", ""},
	"eu-south-1":     {"Milan", ""},
	"ap-southeast-1": {"Singapore", ""},
	"ap-southeast-2": {"Sydney", ""},
	"ap-northeast-1": {"Tokyo", ""},
	"ap-northeast-2": {"Seoul", ""},
	"ap-northeast-3": {"Osaka", ""},
	"ap-south-1":     {"Mumbai", ""},
	"ap-east-1":      {"Hong Kong", ""},
	"me-south-1":     {"Bahrain", ""},
	"sa-east-1":      {"São Paulo", ""},
	"af-south-1":     {"Cape Town", ""},
}

// GridFactor represents a grid emission factor for a region.
type GridFactor struct {
	Region   string
	Factor   float64
	Location string
	Note     string
}

// CCFGridData represents the structure of CCF's grid emission factors JSON
type CCFGridData struct {
	Region      string  `json:"region"`
	MtCO2ePerKwh float64 `json:"mtCO2ePerKwh"`
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Print changes without writing to file")
	validate := flag.Bool("validate", true, "Validate fetched values are within expected range")
	output := flag.String("output", "./internal/carbon/grid_factors.go", "Path to grid_factors.go")
	flag.Parse()

	fmt.Println("Fetching Cloud Carbon Footprint grid emission factors...")
	fmt.Printf("Source: %s\n", ccfGridFactorsURL)

	factors, err := fetchGridFactors()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching grid factors: %v\n", err)
		fmt.Println("Using default/existing values instead...")
		factors = getDefaultFactors()
	}

	if *validate {
		if err := validateFactors(factors); err != nil {
			fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Validation passed")
	}

	// Generate the file content
	content := generateGridFactorsFile(factors)

	if *dryRun {
		fmt.Println("\n--- Dry run output ---")
		fmt.Println(content)
		return
	}

	// Write the file
	if err := os.WriteFile(*output, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated %s with %d regions\n", *output, len(factors))
	fmt.Println("Run 'go test ./internal/carbon/...' to verify the changes")
}

// fetchGridFactors fetches grid emission factors from CCF repository.
func fetchGridFactors() ([]GridFactor, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(ccfGridFactorsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grid factors: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// If the CCF endpoint doesn't exist, use defaults
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var ccfData []CCFGridData
	if err := json.NewDecoder(resp.Body).Decode(&ccfData); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	var factors []GridFactor
	for _, d := range ccfData {
		// Only include regions we support
		if info, ok := AWSRegionMapping[d.Region]; ok {
			factors = append(factors, GridFactor{
				Region:   d.Region,
				Factor:   d.MtCO2ePerKwh,
				Location: info.location,
			})
		}
	}

	return factors, nil
}

// getDefaultFactors returns the default/current grid factors.
// This is used as a fallback if the CCF endpoint is unavailable.
func getDefaultFactors() []GridFactor {
	return []GridFactor{
		{Region: "us-east-1", Factor: 0.000379, Location: "Virginia", Note: "SERC"},
		{Region: "us-east-2", Factor: 0.000411, Location: "Ohio", Note: "RFC"},
		{Region: "us-west-1", Factor: 0.000322, Location: "N. California", Note: "WECC"},
		{Region: "us-west-2", Factor: 0.000322, Location: "Oregon", Note: "WECC"},
		{Region: "ca-central-1", Factor: 0.00012, Location: "Canada", Note: ""},
		{Region: "eu-west-1", Factor: 0.0002786, Location: "Ireland", Note: ""},
		{Region: "eu-north-1", Factor: 0.0000088, Location: "Sweden", Note: "very low carbon"},
		{Region: "ap-southeast-1", Factor: 0.000408, Location: "Singapore", Note: ""},
		{Region: "ap-southeast-2", Factor: 0.00079, Location: "Sydney", Note: ""},
		{Region: "ap-northeast-1", Factor: 0.000506, Location: "Tokyo", Note: ""},
		{Region: "ap-south-1", Factor: 0.000708, Location: "Mumbai", Note: ""},
		{Region: "sa-east-1", Factor: 0.0000617, Location: "São Paulo", Note: "very low carbon"},
	}
}

// validateFactors validates that all factors are within expected range.
func validateFactors(factors []GridFactor) error {
	var errors []string

	for _, f := range factors {
		if f.Factor < minValidFactor || f.Factor > maxValidFactor {
			errors = append(errors, fmt.Sprintf(
				"%s: factor %.8f is outside valid range [%.4f, %.4f]",
				f.Region, f.Factor, minValidFactor, maxValidFactor,
			))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// generateGridFactorsFile generates the grid_factors.go file content.
func generateGridFactorsFile(factors []GridFactor) string {
	// Sort by region name for consistent output
	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Region < factors[j].Region
	})

	// Generate the map entries
	var entries strings.Builder
	for _, f := range factors {
		comment := f.Location
		if f.Note != "" {
			comment += " (" + f.Note + ")"
		}
		entries.WriteString(fmt.Sprintf("\t%-18q: %.8f, // %s\n",
			f.Region, f.Factor, comment))
	}

	// Get current date for vintage
	vintage := time.Now().Format("2006")

	return fmt.Sprintf(fileTemplate, vintage, entries.String(), defaultFactor)
}
