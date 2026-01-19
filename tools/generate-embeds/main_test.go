package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/rshade/finfocus-plugin-aws-public/internal/regionsconfig"
)

func TestGenerateEmbedFile(t *testing.T) {
	// Create a temporary directory for output
	tmpDir := t.TempDir()

	// Parse the actual template file to test integration
	tmpl, err := template.ParseFiles("embed_template.go.tmpl")
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	tests := []struct {
		name      string
		region    regionsconfig.RegionConfig
		wantFile  string
		wantConts []string
	}{
		{
			name: "us-east-1",
			region: regionsconfig.RegionConfig{
				ID:   "use1",
				Name: "us-east-1",
				Tag:  "region_use1",
			},
			wantFile: "embed_use1.go",
			// All 10 services must be present to catch template/fallback sync issues
			wantConts: []string{
				"//go:build region_use1",
				"package pricing",
				"//go:embed data/ec2_us-east-1.json",
				"var rawEC2JSON []byte",
				"//go:embed data/s3_us-east-1.json",
				"var rawS3JSON []byte",
				"//go:embed data/rds_us-east-1.json",
				"var rawRDSJSON []byte",
				"//go:embed data/eks_us-east-1.json",
				"var rawEKSJSON []byte",
				"//go:embed data/lambda_us-east-1.json",
				"var rawLambdaJSON []byte",
				"//go:embed data/dynamodb_us-east-1.json",
				"var rawDynamoDBJSON []byte",
				"//go:embed data/elb_us-east-1.json",
				"var rawELBJSON []byte",
				"//go:embed data/vpc_us-east-1.json",
				"var rawVPCJSON []byte",
				"//go:embed data/cloudwatch_us-east-1.json",
				"var rawCloudWatchJSON []byte",
				"//go:embed data/elasticache_us-east-1.json",
				"var rawElastiCacheJSON []byte",
			},
		},
		{
			name: "eu-west-1",
			region: regionsconfig.RegionConfig{
				ID:   "euw1",
				Name: "eu-west-1",
				Tag:  "region_euw1",
			},
			wantFile: "embed_euw1.go",
			wantConts: []string{
				"//go:build region_euw1",
				"package pricing",
				"//go:embed data/rds_eu-west-1.json",
				"var rawRDSJSON []byte",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generateEmbedFile(tt.region, tmpl, tmpDir)
			if err != nil {
				t.Fatalf("generateEmbedFile() error = %v", err)
			}

			// Verify file exists
			filePath := filepath.Join(tmpDir, tt.wantFile)
			contentBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			content := string(contentBytes)
			for _, want := range tt.wantConts {
				if !strings.Contains(content, want) {
					t.Errorf("Generated file missing content %q", want)
				}
			}
		})
	}
}

func TestGenerateEmbedFile_Errors(t *testing.T) {
	// Test case: Output directory doesn't exist - os.Create fails when parent dir is missing
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "missing")

	tmpl := template.Must(template.New("test").Parse("package pricing"))
	region := regionsconfig.RegionConfig{ID: "test", Name: "test", Tag: "test"}

	err := generateEmbedFile(region, tmpl, nonExistentDir)
	if err == nil {
		t.Error("generateEmbedFile() expected error when directory is missing, got nil")
	}
}

// TestGenerateEmbedFile_TemplateError verifies that template execution errors are handled.
//
// This catches cases where a template references fields that don't exist in RegionConfig.
// The test validates both that an error is returned AND that the error message contains
// the expected "executing template" text from text/template.
func TestGenerateEmbedFile_TemplateError(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a template with an invalid field to test error handling
	tmpl := template.Must(template.New("test").Parse("{{.InvalidField}}"))
	region := regionsconfig.RegionConfig{ID: "test", Name: "test-region", Tag: "region_test"}

	err := generateEmbedFile(region, tmpl, tmpDir)
	if err == nil {
		t.Error("generateEmbedFile() expected error when template has invalid field, got nil")
	}
	// Verify the error message indicates template execution failure
	if !strings.Contains(err.Error(), "executing template") {
		t.Errorf("expected error to contain 'executing template', got: %v", err)
	}
}
