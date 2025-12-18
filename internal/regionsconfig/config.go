// Package regionsconfig provides types and functions for loading and validating
// AWS region configurations from regions.yaml. This package centralizes region
// configuration handling previously duplicated across generate-embeds and
// generate-goreleaser tools.
package regionsconfig

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// safePattern validates that region fields contain only safe characters
// (alphanumeric, hyphens, and underscores) to prevent YAML injection.
var safePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// RegionConfig represents a single AWS region configuration entry.
// Each region maps to a GoReleaser build with a specific build tag.
type RegionConfig struct {
	// ID is the AWS region identifier (e.g., "us-east-1", "eu-west-1").
	ID string `yaml:"id" json:"id"`

	// Name is a human-readable name for the region (e.g., "US_East_N_Virginia").
	// Used in GoReleaser build naming.
	Name string `yaml:"name" json:"name"`

	// Tag is the Go build tag for this region (e.g., "region_us-east-1").
	// Must be "region_" + ID.
	Tag string `yaml:"tag" json:"tag"`
}

// Config represents the full regions.yaml structure containing all configured
// AWS regions.
type Config struct {
	Regions []RegionConfig `yaml:"regions" json:"regions"`
}

// Load reads and parses a regions.yaml file from the given filename.
// It returns the parsed Config or an error if the file cannot be read or parsed.
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read regions config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse regions config: %w", err)
	}

	return &config, nil
}

// Validate checks that all regions in the slice have valid, non-empty fields
// and conform to expected patterns:
//   - All fields (ID, Name, Tag) must be non-empty
//   - All fields must contain only safe characters (alphanumeric, hyphens, underscores)
//   - Tag must equal "region_" + ID
//   - No duplicate region IDs
//
// Returns nil if all regions are valid, or an error describing the first
// validation failure encountered.
func Validate(regions []RegionConfig) error {
	seen := make(map[string]struct{})

	for _, r := range regions {
		// Check required fields
		if r.ID == "" {
			return fmt.Errorf("region missing id")
		}
		if r.Name == "" {
			return fmt.Errorf("region %s missing name", r.ID)
		}
		if r.Tag == "" {
			return fmt.Errorf("region %s missing tag", r.ID)
		}

		// Validate characters to prevent YAML injection
		if !safePattern.MatchString(r.ID) {
			return fmt.Errorf("region id %q contains invalid characters (only alphanumeric, hyphens, underscores allowed)", r.ID)
		}
		if !safePattern.MatchString(r.Name) {
			return fmt.Errorf("region name %q contains invalid characters (only alphanumeric, hyphens, underscores allowed)", r.Name)
		}
		if !safePattern.MatchString(r.Tag) {
			return fmt.Errorf("region tag %q contains invalid characters (only alphanumeric, hyphens, underscores allowed)", r.Tag)
		}

		// Validate tag format matches expected pattern
		expectedTag := "region_" + r.ID
		if r.Tag != expectedTag {
			return fmt.Errorf("region %s tag mismatch: expected %s, got %s", r.ID, expectedTag, r.Tag)
		}

		// Check for duplicates
		if _, exists := seen[r.ID]; exists {
			return fmt.Errorf("duplicate region id: %s", r.ID)
		}
		seen[r.ID] = struct{}{}
	}

	return nil
}

// LoadAndValidate is a convenience function that loads a regions.yaml file
// and validates its contents in a single call.
// Returns the validated Config or an error if loading or validation fails.
func LoadAndValidate(filename string) (*Config, error) {
	config, err := Load(filename)
	if err != nil {
		return nil, err
	}

	if err := Validate(config.Regions); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return config, nil
}
