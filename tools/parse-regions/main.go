// Package main provides a CLI tool for parsing regions.yaml and outputting
// region data in shell-consumable formats.
//
// This tool replaces fragile sed-based YAML parsing in shell scripts with
// proper YAML parsing using github.com/goccy/go-yaml.
//
// Usage:
//
//	go run ./tools/parse-regions [OPTIONS]
//
// Options:
//
//	-config string   Path to regions.yaml config file (default "internal/pricing/regions.yaml")
//	-format string   Output format: lines, json, or csv (default "lines")
//	-field string    Field to output when format=lines: id, name, tag, or all (default "all")
//
// Output Formats:
//
//	lines: One value per line, suitable for readarray in bash
//	json:  JSON array of region objects
//	csv:   Comma-separated id,name,tag per line
//
// Examples:
//
//	# Get all region names (one per line)
//	go run ./tools/parse-regions -field name
//
//	# Get all data as JSON
//	go run ./tools/parse-regions -format json
//
//	# Get comma-separated values
//	go run ./tools/parse-regions -format csv
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// RegionConfig represents a single region configuration from regions.yaml.
type RegionConfig struct {
	ID   string `yaml:"id" json:"id"`
	Name string `yaml:"name" json:"name"`
	Tag  string `yaml:"tag" json:"tag"`
}

// RegionsConfig represents the full regions.yaml structure.
type RegionsConfig struct {
	Regions []RegionConfig `yaml:"regions" json:"regions"`
}

// main parses command-line flags, reads the regions YAML configuration,
// and outputs the region data in the specified format.
//
// Exit codes:
//   - 0: Success
//   - 1: Error reading or parsing config file
//   - 2: Invalid command-line arguments
func main() {
	configPath := flag.String("config", "internal/pricing/regions.yaml", "Path to regions config")
	format := flag.String("format", "lines", "Output format: lines, json, or csv")
	field := flag.String("field", "all", "Field to output when format=lines: id, name, tag, or all")
	flag.Parse()

	// Validate format
	validFormats := map[string]bool{"lines": true, "json": true, "csv": true}
	if !validFormats[*format] {
		fmt.Fprintf(os.Stderr, "Error: invalid format %q (must be lines, json, or csv)\n", *format)
		os.Exit(2)
	}

	// Validate field
	validFields := map[string]bool{"id": true, "name": true, "tag": true, "all": true}
	if !validFields[*field] {
		fmt.Fprintf(os.Stderr, "Error: invalid field %q (must be id, name, tag, or all)\n", *field)
		os.Exit(2)
	}

	// Load config
	regions, err := loadRegionsConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Output in requested format
	switch *format {
	case "lines":
		outputLines(regions, *field)
	case "json":
		outputJSON(regions)
	case "csv":
		outputCSV(regions)
	}
}

// loadRegionsConfig reads a YAML file at filename and returns the parsed
// slice of RegionConfig entries.
//
// It returns an error if the file cannot be read, the YAML cannot be
// unmarshaled, or if no regions are defined.
func loadRegionsConfig(filename string) ([]RegionConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var config RegionsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if len(config.Regions) == 0 {
		return nil, fmt.Errorf("no regions defined in config file")
	}

	return config.Regions, nil
}

// outputLines prints region data as lines to stdout.
//
// When field is "all", it prints three groups of lines separated by "---":
// first all IDs, then all names, then all tags. This allows shell scripts
// to split the output into separate arrays.
//
// When field is "id", "name", or "tag", it prints only that field's values,
// one per line.
func outputLines(regions []RegionConfig, field string) {
	switch field {
	case "id":
		for _, r := range regions {
			fmt.Println(r.ID)
		}
	case "name":
		for _, r := range regions {
			fmt.Println(r.Name)
		}
	case "tag":
		for _, r := range regions {
			fmt.Println(r.Tag)
		}
	case "all":
		// Output all fields grouped, separated by ---
		for _, r := range regions {
			fmt.Println(r.ID)
		}
		fmt.Println("---")
		for _, r := range regions {
			fmt.Println(r.Name)
		}
		fmt.Println("---")
		for _, r := range regions {
			fmt.Println(r.Tag)
		}
	}
}

// outputJSON prints all regions as a JSON array to stdout.
func outputJSON(regions []RegionConfig) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(regions); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

// outputCSV prints regions as CSV (id,name,tag) lines to stdout.
// No header is included; each line contains one region's data.
func outputCSV(regions []RegionConfig) {
	for _, r := range regions {
		fmt.Printf("%s,%s,%s\n", r.ID, r.Name, r.Tag)
	}
}
