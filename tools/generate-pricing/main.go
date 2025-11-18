package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	regions := flag.String("regions", "us-east-1", "Comma-separated regions")
	outDir := flag.String("out-dir", "./data", "Output directory")
	dummy := flag.Bool("dummy", false, "Generate dummy data")

	flag.Parse()

	regionList := strings.Split(*regions, ",")

	for _, region := range regionList {
		if err := generatePricingData(region, *outDir, *dummy); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate pricing for %s: %v\n", region, err)
			os.Exit(1)
		}
		fmt.Printf("Generated pricing data for %s\n", region)
	}

	fmt.Println("Pricing data generated successfully")
}

func generatePricingData(region, outDir string, dummy bool) error {
	// For v1, only dummy mode is implemented
	if !dummy {
		return fmt.Errorf("real AWS pricing fetch not implemented yet - use --dummy flag")
	}

	// Create dummy pricing data with a few instance types and volume types
	data := map[string]interface{}{
		"region":   region,
		"currency": "USD",
		"ec2": map[string]interface{}{
			"t3.micro": map[string]interface{}{
				"instance_type":    "t3.micro",
				"operating_system": "Linux",
				"tenancy":          "Shared",
				"hourly_rate":      0.0104,
			},
			"t3.small": map[string]interface{}{
				"instance_type":    "t3.small",
				"operating_system": "Linux",
				"tenancy":          "Shared",
				"hourly_rate":      0.0208,
			},
			"t3.medium": map[string]interface{}{
				"instance_type":    "t3.medium",
				"operating_system": "Linux",
				"tenancy":          "Shared",
				"hourly_rate":      0.0416,
			},
			"m5.large": map[string]interface{}{
				"instance_type":    "m5.large",
				"operating_system": "Linux",
				"tenancy":          "Shared",
				"hourly_rate":      0.096,
			},
		},
		"ebs": map[string]interface{}{
			"gp3": map[string]interface{}{
				"volume_type":       "gp3",
				"rate_per_gb_month": 0.08,
			},
			"gp2": map[string]interface{}{
				"volume_type":       "gp2",
				"rate_per_gb_month": 0.10,
			},
			"io1": map[string]interface{}{
				"volume_type":       "io1",
				"rate_per_gb_month": 0.125,
			},
			"io2": map[string]interface{}{
				"volume_type":       "io2",
				"rate_per_gb_month": 0.125,
			},
		},
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outFile := fmt.Sprintf("%s/aws_pricing_%s.json", outDir, region)
	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", outFile, closeErr)
		}
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("failed to encode pricing data: %w", err)
	}

	return nil
}
