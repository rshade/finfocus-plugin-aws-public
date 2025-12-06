package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadRegionsConfig tests the loadRegionsConfig function with various inputs.
func TestLoadRegionsConfig(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		wantCount   int
	}{
		{
			name: "valid config with multiple regions",
			content: `regions:
  - id: use1
    name: us-east-1
    tag: region_use1
  - id: usw2
    name: us-west-2
    tag: region_usw2
`,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "valid config with single region",
			content: `regions:
  - id: use1
    name: us-east-1
    tag: region_use1
`,
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:        "empty regions list",
			content:     "regions: []\n",
			wantErr:     true,
			errContains: "no regions defined",
		},
		{
			name:        "missing regions key",
			content:     "other_key: value\n",
			wantErr:     true,
			errContains: "no regions defined",
		},
		{
			name:        "invalid YAML syntax",
			content:     "regions:\n  - id: use1\n  name: invalid indent\n",
			wantErr:     true,
			errContains: "parsing YAML",
		},
		{
			name:        "empty file",
			content:     "",
			wantErr:     true,
			errContains: "no regions defined",
		},
		{
			name: "region with empty id",
			content: `regions:
  - id: ""
    name: us-east-1
    tag: region_use1
`,
			wantErr:     true,
			errContains: "empty id/name/tag",
		},
		{
			name: "region with empty name",
			content: `regions:
  - id: use1
    name: ""
    tag: region_use1
`,
			wantErr:     true,
			errContains: "empty id/name/tag",
		},
		{
			name: "region with empty tag",
			content: `regions:
  - id: use1
    name: us-east-1
    tag: ""
`,
			wantErr:     true,
			errContains: "empty id/name/tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with test content
			tmpFile, err := os.CreateTemp("", "regions-*.yaml")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			tmpFile.Close()

			// Test loadRegionsConfig
			regions, err := loadRegionsConfig(tmpFile.Name())

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(regions) != tt.wantCount {
				t.Errorf("expected %d regions, got %d", tt.wantCount, len(regions))
			}
		})
	}
}

// TestLoadRegionsConfigNonExistent tests loading a non-existent file.
func TestLoadRegionsConfigNonExistent(t *testing.T) {
	_, err := loadRegionsConfig("/nonexistent/path/regions.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "reading file") {
		t.Errorf("expected error about reading file, got: %v", err)
	}
}

// TestRegionConfigFields verifies that all fields are parsed correctly.
func TestRegionConfigFields(t *testing.T) {
	content := `regions:
  - id: test_id
    name: test-region-1
    tag: region_test
`
	tmpFile, err := os.CreateTemp("", "regions-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	regions, err := loadRegionsConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}

	r := regions[0]
	if r.ID != "test_id" {
		t.Errorf("expected ID 'test_id', got %q", r.ID)
	}
	if r.Name != "test-region-1" {
		t.Errorf("expected Name 'test-region-1', got %q", r.Name)
	}
	if r.Tag != "region_test" {
		t.Errorf("expected Tag 'region_test', got %q", r.Tag)
	}
}

// TestOutputLines tests the outputLines function with different field options.
//
// This test validates that outputLines correctly writes region data in various
// field modes (id, name, tag, all) to the provided io.Writer. The function
// accepts an io.Writer parameter to enable safe parallel test execution without
// race conditions from reassigning os.Stdout.
func TestOutputLines(t *testing.T) {
	t.Parallel()

	regions := []RegionConfig{
		{ID: "use1", Name: "us-east-1", Tag: "region_use1"},
		{ID: "usw2", Name: "us-west-2", Tag: "region_usw2"},
	}

	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{
			name:     "field=id",
			field:    "id",
			expected: "use1\nusw2\n",
		},
		{
			name:     "field=name",
			field:    "name",
			expected: "us-east-1\nus-west-2\n",
		},
		{
			name:     "field=tag",
			field:    "tag",
			expected: "region_use1\nregion_usw2\n",
		},
		{
			name:     "field=all",
			field:    "all",
			expected: "use1\nusw2\n---\nus-east-1\nus-west-2\n---\nregion_use1\nregion_usw2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			outputLines(&buf, regions, tt.field)
			got := buf.String()

			if got != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, got)
			}
		})
	}
}

// TestOutputCSV tests the outputCSV function.
//
// This test validates that outputCSV correctly writes region data in CSV format
// (id,name,tag per line) to the provided io.Writer. Each line contains exactly
// 3 comma-separated fields with no header row.
func TestOutputCSV(t *testing.T) {
	t.Parallel()

	regions := []RegionConfig{
		{ID: "use1", Name: "us-east-1", Tag: "region_use1"},
		{ID: "usw2", Name: "us-west-2", Tag: "region_usw2"},
	}

	var buf bytes.Buffer
	outputCSV(&buf, regions)
	got := buf.String()

	expected := "use1,us-east-1,region_use1\nusw2,us-west-2,region_usw2\n"
	if got != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, got)
	}
}

// TestOutputJSON tests the outputJSON function for valid JSON output.
//
// This test validates that outputJSON correctly writes region data as a JSON
// array to the provided io.Writer and that the output is properly formatted
// with 2-space indentation.
func TestOutputJSON(t *testing.T) {
	t.Parallel()

	regions := []RegionConfig{
		{ID: "use1", Name: "us-east-1", Tag: "region_use1"},
	}

	var buf bytes.Buffer
	if err := outputJSON(&buf, regions); err != nil {
		t.Fatalf("outputJSON returned error: %v", err)
	}
	got := buf.String()

	// Parse the JSON output to verify structure
	var parsed []RegionConfig
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 region in JSON, got %d", len(parsed))
	}

	if parsed[0].ID != "use1" || parsed[0].Name != "us-east-1" || parsed[0].Tag != "region_use1" {
		t.Errorf("unexpected JSON content: %+v", parsed[0])
	}
}

// TestOutputJSONFormatting tests that outputJSON produces properly indented JSON.
//
// This test validates the formatting of JSON output, specifically verifying
// that the JSON uses 2-space indentation as configured in outputJSON.
func TestOutputJSONFormatting(t *testing.T) {
	t.Parallel()

	regions := []RegionConfig{
		{ID: "use1", Name: "us-east-1", Tag: "region_use1"},
	}

	var buf bytes.Buffer
	if err := outputJSON(&buf, regions); err != nil {
		t.Fatalf("outputJSON returned error: %v", err)
	}
	got := buf.String()

	// Verify indentation - should have 2-space indentation
	expectedJSON := `[
  {
    "id": "use1",
    "name": "us-east-1",
    "tag": "region_use1"
  }
]
`
	if got != expectedJSON {
		t.Errorf("JSON formatting mismatch\nexpected:\n%s\ngot:\n%s", expectedJSON, got)
	}
}

// TestIntegrationWithRealConfig verifies that the real regions.yaml parses into valid RegionConfig entries.
//
// This test exercises the integration between the parse-regions tool and the checked-in
// internal/pricing/regions.yaml file. It ensures that all configured regions have non-empty
// id, name, and tag fields and that at least one region is present.
//
// Test workflow:
//  1. Locate regions.yaml relative to the repository root.
//  2. Skip the test if the file is not present (e.g., in stripped environments).
//  3. Call loadRegionsConfig on the real config file.
//  4. Assert that at least one region is loaded and each has non-empty id/name/tag.
//
// Prerequisites:
//   - Repository checked out with internal/pricing/regions.yaml present.
//
// Run with:
//
//	go test ./tools/parse-regions -run TestIntegrationWithRealConfig
func TestIntegrationWithRealConfig(t *testing.T) {
	// Try to find the real config file relative to test location
	configPaths := []string{
		"../../internal/pricing/regions.yaml",
		"internal/pricing/regions.yaml",
	}

	var configPath string
	for _, p := range configPaths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			configPath = absPath
			break
		}
	}

	if configPath == "" {
		t.Skip("regions.yaml not found, skipping integration test")
	}

	regions, err := loadRegionsConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load real config: %v", err)
	}

	// Verify we have at least some regions
	if len(regions) < 1 {
		t.Error("expected at least 1 region in real config")
	}

	// Verify each region has required fields
	for i, r := range regions {
		if r.ID == "" {
			t.Errorf("region %d: ID is empty", i)
		}
		if r.Name == "" {
			t.Errorf("region %d: Name is empty", i)
		}
		if r.Tag == "" {
			t.Errorf("region %d: Tag is empty", i)
		}
	}
}
