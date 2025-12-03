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
func TestOutputLines(t *testing.T) {
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
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			outputLines(regions, tt.field)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			got := buf.String()

			if got != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, got)
			}
		})
	}
}

// TestOutputCSV tests the outputCSV function.
func TestOutputCSV(t *testing.T) {
	regions := []RegionConfig{
		{ID: "use1", Name: "us-east-1", Tag: "region_use1"},
		{ID: "usw2", Name: "us-west-2", Tag: "region_usw2"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCSV(regions)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	got := buf.String()

	expected := "use1,us-east-1,region_use1\nusw2,us-west-2,region_usw2\n"
	if got != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, got)
	}
}

// TestOutputJSON tests the outputJSON function.
func TestOutputJSON(t *testing.T) {
	regions := []RegionConfig{
		{ID: "use1", Name: "us-east-1", Tag: "region_use1"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputJSON(regions)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
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

// TestIntegrationWithRealConfig tests loading the actual regions.yaml if it exists.
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
