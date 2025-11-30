package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	regions := flag.String("regions", "us-east-1", "Comma-separated regions")
	outDir := flag.String("out-dir", "./data", "Output directory")
	service := flag.String("service", "AmazonEC2", "AWS Service Code (e.g. AmazonEC2)")
	dummy := flag.Bool("dummy", false, "DEPRECATED: ignored, real data is always fetched")

	flag.Parse()

	if *dummy {
		fmt.Println("Note: --dummy flag is deprecated and ignored. Fetching real data.")
	}

	regionList := strings.Split(*regions, ",")

	for _, region := range regionList {
		if err := generatePricingData(region, *service, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate pricing for %s: %v\n", region, err)
			os.Exit(1)
		}
		fmt.Printf("Generated pricing data for %s\n", region)
	}

	fmt.Println("Pricing data generated successfully")
}

func generatePricingData(region, service, outDir string) error {
	// URL: https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/{Service}/current/{Region}/index.json
	// Note: The base URL is always us-east-1 for the API, but the path changes.
	url := fmt.Sprintf("https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/%s/current/%s/index.json", service, region)

	fmt.Printf("Fetching %s...\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
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

	// Stream directly to file
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}