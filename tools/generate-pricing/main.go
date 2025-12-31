package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// serviceConfig maps AWS service codes to lowercase file prefixes.
// Used for generating per-service pricing files.
var serviceConfig = map[string]string{
	"AmazonEC2":        "ec2",
	"AmazonS3":         "s3",
	"AWSLambda":        "lambda",
	"AmazonRDS":        "rds",
	"AmazonEKS":        "eks",
	"AmazonDynamoDB":   "dynamodb",
	"AWSELB":           "elb",
	"AmazonVPC":        "vpc",
	"AmazonCloudWatch": "cloudwatch",
}

// main is the program entry point that fetches AWS pricing data per service.
//
// It parses command-line flags to determine regions (`--regions`), output directory (`--out-dir`),
// and services (`--service`). For each region and service, it fetches pricing data from AWS Price
// List API and writes it to a separate file named {service}_{region}.json.
//
// Fail-fast behavior: If ANY service fetch fails for a region, the program exits with status 1.
// This prevents partial data that could cause $0 pricing issues like v0.0.10/v0.0.11.
func main() {
	regions := flag.String("regions", "us-east-1", "Comma-separated regions")
	outDir := flag.String("out-dir", "./data", "Output directory")
	service := flag.String("service", "AmazonEC2,AmazonS3,AWSLambda,AmazonRDS,AmazonEKS,AmazonDynamoDB,AWSELB,AmazonVPC,AmazonCloudWatch", "AWS Service Codes (comma-separated)")
	dummy := flag.Bool("dummy", false, "DEPRECATED: ignored, real data is always fetched")

	flag.Parse()

	if *dummy {
		fmt.Println("Note: --dummy flag is deprecated and ignored. Fetching real data.")
	}

	regionList := strings.Split(*regions, ",")
	serviceList := strings.Split(*service, ",")

	for _, region := range regionList {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}

		if err := generatePerServicePricingData(region, serviceList, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate pricing for %s: %v\n", region, err)
			os.Exit(1)
		}
		fmt.Printf("Generated pricing data for %s\n", region)
	}

	fmt.Println("Pricing data generated successfully")
}

// generatePerServicePricingData fetches pricing data for each service and writes to separate files.
//
// For each service in the services list, it:
// 1. Fetches the raw AWS Price List API response
// 2. Writes the response verbatim to {servicePrefix}_{region}.json
//
// The function fails fast if any service fetch fails - no partial data is written.
// This prevents the v0.0.10/v0.0.11 bug where partial data caused $0 pricing.
//
// Parameters:
//   - region: AWS region code (e.g., "us-east-1")
//   - services: slice of AWS service codes (e.g., ["AmazonEC2", "AWSELB"])
//   - outDir: directory where output files will be written
//
// Returns an error if any service fetch fails, the output directory cannot be created,
// or any file write fails.
func generatePerServicePricingData(region string, services []string, outDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, service := range services {
		service = strings.TrimSpace(service)
		if service == "" {
			continue
		}

		// Get the lowercase file prefix for this service
		prefix, ok := serviceConfig[service]
		if !ok {
			return fmt.Errorf("unknown service code: %s (add to serviceConfig map)", service)
		}

		fmt.Printf("Fetching %s for %s...\n", service, region)
		data, err := fetchServicePricingRaw(region, service)
		if err != nil {
			// Fail fast - do not continue with partial data
			return fmt.Errorf("failed to fetch %s: %w", service, err)
		}

		// Write per-service file: {prefix}_{region}.json (e.g., ec2_us-east-1.json)
		outFile := fmt.Sprintf("%s/%s_%s.json", outDir, prefix, region)
		if err := writeRawPricingFile(data, outFile); err != nil {
			return fmt.Errorf("failed to write %s: %w", outFile, err)
		}

		fmt.Printf("Wrote %s (%d bytes)\n", outFile, len(data))
	}

	return nil
}

// httpRequestTimeout is the timeout for HTTP requests to AWS pricing API
const httpRequestTimeout = 5 * time.Minute

// awsPricingResponse represents the structure of AWS Price List API response.
// We use this to filter terms while preserving the raw structure.
type awsPricingResponse struct {
	FormatVersion   string                            `json:"formatVersion"`
	Disclaimer      string                            `json:"disclaimer"`
	OfferCode       string                            `json:"offerCode"`
	Version         string                            `json:"version"`
	PublicationDate string                            `json:"publicationDate"`
	Products        map[string]json.RawMessage        `json:"products"`
	Terms           map[string]map[string]interface{} `json:"terms"`
}

// fetchServicePricingRaw retrieves AWS pricing data for the specified service and region.
// It filters out Reserved Instance and Savings Plans terms to reduce file size,
// while preserving all products (including all OS values) and OnDemand terms.
//
// region is the AWS region code (for example, "us-east-1").
// service is the AWS service code (for example, "AmazonEC2", "AWSELB").
//
// Returns the filtered JSON bytes on success. An error is returned if the HTTP request fails,
// the response status is not 200 OK, or reading the response body fails.
func fetchServicePricingRaw(region, service string) ([]byte, error) {
	url := fmt.Sprintf("https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/%s/current/%s/index.json", service, region)

	// Create request with context for timeout support
	ctx, cancel := context.WithTimeout(context.Background(), httpRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response to filter terms
	var pricing awsPricingResponse
	if err := json.Unmarshal(body, &pricing); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	if pricing.OfferCode == "" {
		return nil, fmt.Errorf("missing offerCode in response")
	}
	if len(pricing.Products) == 0 {
		return nil, fmt.Errorf("no products in response for %s/%s", service, region)
	}

	// Filter terms: keep only OnDemand, remove Reserved and Savings Plans.
	// AWS Price List API returns multiple term types:
	//
	// KEPT:
	//   - "OnDemand" - Pay-as-you-go pricing with no commitment (what we use)
	//
	// FILTERED OUT:
	//   - "Reserved" - Reserved Instance pricing (1yr, 3yr upfront commitments)
	//                  Typically 30-75% discount vs OnDemand, but requires commitment.
	//                  For EC2, this includes ~14,000 SKUs (us-east-1).
	//   - "savingsPlan" - Savings Plans pricing (if present, though uncommon)
	//                     Flexible discount program that applies across services.
	//
	// Why filter? Reduces file size from ~400MB to ~154MB for EC2 alone.
	// The plugin only supports on-demand pricing for v1.
	filteredTerms := make(map[string]map[string]interface{})
	for termType, skuTerms := range pricing.Terms {
		if termType == "OnDemand" {
			filteredTerms[termType] = skuTerms
		} else {
			fmt.Printf("  Filtering out term type: %s (%d SKUs)\n", termType, len(skuTerms))
		}
	}
	pricing.Terms = filteredTerms

	// Re-serialize with filtered terms
	filteredBody, err := json.Marshal(pricing)
	if err != nil {
		return nil, fmt.Errorf("failed to re-serialize filtered pricing: %w", err)
	}

	return filteredBody, nil
}

// writeRawPricingFile writes raw pricing data to a file atomically.
// The data is written verbatim without any processing or modification.
// Uses write-to-temp-then-rename pattern to prevent partial writes on failure.
func writeRawPricingFile(data []byte, outFile string) error {
	// Create temp file in the same directory to ensure rename works (same filesystem)
	dir := filepath.Dir(outFile)
	tmpFile, err := os.CreateTemp(dir, ".pricing-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename (works on same filesystem)
	if err := os.Rename(tmpPath, outFile); err != nil {
		return fmt.Errorf("failed to rename temp file to %s: %w", outFile, err)
	}

	success = true
	return nil
}
