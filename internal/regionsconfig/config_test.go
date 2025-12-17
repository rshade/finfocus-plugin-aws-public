package regionsconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_ValidYAML tests loading a valid regions.yaml file.
func TestLoad_ValidYAML(t *testing.T) {
	// Create a temporary YAML file
	content := `regions:
  - id: us-east-1
    name: US_East_N_Virginia
    tag: region_us-east-1
  - id: eu-west-1
    name: EU_West_Ireland
    tag: region_eu-west-1
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "regions.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(config.Regions) != 2 {
		t.Errorf("Load() returned %d regions, want 2", len(config.Regions))
	}

	if config.Regions[0].ID != "us-east-1" {
		t.Errorf("config.Regions[0].ID = %q, want %q", config.Regions[0].ID, "us-east-1")
	}
	if config.Regions[0].Name != "US_East_N_Virginia" {
		t.Errorf("config.Regions[0].Name = %q, want %q", config.Regions[0].Name, "US_East_N_Virginia")
	}
	if config.Regions[0].Tag != "region_us-east-1" {
		t.Errorf("config.Regions[0].Tag = %q, want %q", config.Regions[0].Tag, "region_us-east-1")
	}
}

// TestLoad_FileNotFound tests loading a non-existent file.
func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/regions.yaml")
	if err == nil {
		t.Error("Load() expected error for non-existent file, got nil")
	}
}

// TestLoad_InvalidYAML tests loading an invalid YAML file.
func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(tmpFile, []byte("not: valid: yaml: {{"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}

// TestValidate_ValidRegions tests validation of valid regions.
func TestValidate_ValidRegions(t *testing.T) {
	regions := []RegionConfig{
		{ID: "us-east-1", Name: "US_East_N_Virginia", Tag: "region_us-east-1"},
		{ID: "eu-west-1", Name: "EU_West_Ireland", Tag: "region_eu-west-1"},
	}

	err := Validate(regions)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestValidate_EmptyRegions tests validation of an empty region list.
func TestValidate_EmptyRegions(t *testing.T) {
	err := Validate([]RegionConfig{})
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for empty slice", err)
	}
}

// TestValidate_MissingID tests validation with missing ID field.
func TestValidate_MissingID(t *testing.T) {
	regions := []RegionConfig{
		{ID: "", Name: "US_East", Tag: "region_"},
	}

	err := Validate(regions)
	if err == nil {
		t.Error("Validate() expected error for missing ID, got nil")
	}
	if !strings.Contains(err.Error(), "missing id") {
		t.Errorf("Validate() error = %q, want to contain %q", err.Error(), "missing id")
	}
}

// TestValidate_MissingName tests validation with missing Name field.
func TestValidate_MissingName(t *testing.T) {
	regions := []RegionConfig{
		{ID: "us-east-1", Name: "", Tag: "region_us-east-1"},
	}

	err := Validate(regions)
	if err == nil {
		t.Error("Validate() expected error for missing Name, got nil")
	}
	if !strings.Contains(err.Error(), "missing name") {
		t.Errorf("Validate() error = %q, want to contain %q", err.Error(), "missing name")
	}
}

// TestValidate_MissingTag tests validation with missing Tag field.
func TestValidate_MissingTag(t *testing.T) {
	regions := []RegionConfig{
		{ID: "us-east-1", Name: "US_East", Tag: ""},
	}

	err := Validate(regions)
	if err == nil {
		t.Error("Validate() expected error for missing Tag, got nil")
	}
	if !strings.Contains(err.Error(), "missing tag") {
		t.Errorf("Validate() error = %q, want to contain %q", err.Error(), "missing tag")
	}
}

// TestValidate_InvalidCharacters tests validation with invalid characters.
func TestValidate_InvalidCharacters(t *testing.T) {
	tests := []struct {
		name    string
		regions []RegionConfig
		want    string
	}{
		{
			name:    "invalid ID with spaces",
			regions: []RegionConfig{{ID: "us east 1", Name: "US_East", Tag: "region_us east 1"}},
			want:    "id",
		},
		{
			name:    "invalid Name with special chars",
			regions: []RegionConfig{{ID: "us-east-1", Name: "US$East!", Tag: "region_us-east-1"}},
			want:    "name",
		},
		{
			name:    "invalid Tag with special chars",
			regions: []RegionConfig{{ID: "us-east-1", Name: "US_East", Tag: "region@us-east-1"}},
			want:    "tag",
		},
		{
			name:    "ID with newline injection",
			regions: []RegionConfig{{ID: "us-east-1\ninjected", Name: "US_East", Tag: "region_us-east-1"}},
			want:    "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.regions)
			if err == nil {
				t.Error("Validate() expected error for invalid characters, got nil")
			}
			if !strings.Contains(err.Error(), "invalid characters") {
				t.Errorf("Validate() error = %q, want to contain %q", err.Error(), "invalid characters")
			}
		})
	}
}

// TestValidate_TagMismatch tests validation with tag not matching expected pattern.
func TestValidate_TagMismatch(t *testing.T) {
	regions := []RegionConfig{
		{ID: "us-east-1", Name: "US_East", Tag: "region_wrong"},
	}

	err := Validate(regions)
	if err == nil {
		t.Error("Validate() expected error for tag mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "tag mismatch") {
		t.Errorf("Validate() error = %q, want to contain %q", err.Error(), "tag mismatch")
	}
}

// TestValidate_DuplicateIDs tests validation with duplicate region IDs.
func TestValidate_DuplicateIDs(t *testing.T) {
	regions := []RegionConfig{
		{ID: "us-east-1", Name: "US_East_1", Tag: "region_us-east-1"},
		{ID: "eu-west-1", Name: "EU_West", Tag: "region_eu-west-1"},
		{ID: "us-east-1", Name: "US_East_2", Tag: "region_us-east-1"}, // duplicate
	}

	err := Validate(regions)
	if err == nil {
		t.Error("Validate() expected error for duplicate ID, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate region id") {
		t.Errorf("Validate() error = %q, want to contain %q", err.Error(), "duplicate region id")
	}
}

// TestLoadAndValidate_Success tests the combined load and validate function.
func TestLoadAndValidate_Success(t *testing.T) {
	content := `regions:
  - id: us-east-1
    name: US_East_N_Virginia
    tag: region_us-east-1
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "regions.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadAndValidate(tmpFile)
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}

	if len(config.Regions) != 1 {
		t.Errorf("LoadAndValidate() returned %d regions, want 1", len(config.Regions))
	}
}

// TestLoadAndValidate_ValidationFailure tests that validation errors are returned.
func TestLoadAndValidate_ValidationFailure(t *testing.T) {
	content := `regions:
  - id: us-east-1
    name: US_East_N_Virginia
    tag: wrong_tag
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "regions.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadAndValidate(tmpFile)
	if err == nil {
		t.Error("LoadAndValidate() expected error for invalid config, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("LoadAndValidate() error = %q, want to contain %q", err.Error(), "validation failed")
	}
}
