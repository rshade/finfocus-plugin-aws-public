package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// main is the program entry point that fetches and writes combined AWS pricing data for one or more regions.
//
// It parses command-line flags to determine regions (`--regions`), output directory (`--out-dir`), and
// services (`--service`). The deprecated `--dummy` flag is accepted but ignored. For each region, it calls
// generateCombinedPricingData to fetch pricing for the requested services and write a combined JSON file;
// on any per-region error the program prints the error to stderr and exits with status 1. On success it prints
// per-region and final completion messages.
func main() {
	regions := flag.String("regions", "us-east-1", "Comma-separated regions")
	outDir := flag.String("out-dir", "./data", "Output directory")
	service := flag.String("service", "AmazonEC2,AmazonS3,AWSLambda,AmazonRDS,AmazonEKS,AmazonDynamoDB,AWSELB", "AWS Service Codes (comma-separated, e.g. AmazonEC2,AmazonRDS,AmazonS3,AWSLambda,AmazonEKS,AmazonDynamoDB,AWSELB)")
	dummy := flag.Bool("dummy", false, "DEPRECATED: ignored, real data is always fetched")

	flag.Parse()

	if *dummy {
		fmt.Println("Note: --dummy flag is deprecated and ignored. Fetching real data.")
	}

	regionList := strings.Split(*regions, ",")
	serviceList := strings.Split(*service, ",")

	for _, region := range regionList {
		if err := generateCombinedPricingData(region, serviceList, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate pricing for %s: %v\n", region, err)
			os.Exit(1)
		}
		fmt.Printf("Generated pricing data for %s\n", region)
	}

	fmt.Println("Pricing data generated successfully")
}

// awsPricing represents the structure of AWS Price List API JSON
type awsPricing struct {
	FormatVersion   string                                `json:"formatVersion"`
	Disclaimer      string                                `json:"disclaimer"`
	OfferCode       string                                `json:"offerCode"`
	Version         string                                `json:"version"`
	PublicationDate string                                `json:"publicationDate"`
	Products        map[string]json.RawMessage            `json:"products"`
	Terms           map[string]map[string]json.RawMessage `json:"terms"`
}

// generateCombinedPricingData fetches pricing data for each service in services,
// combines their Products and OnDemand Terms into a single awsPricing value, and
// writes the combined pricing JSON to a file named aws_pricing_<region>.json in outDir.
//
// The function skips empty service entries. The combined data will use "Combined"
// as the OfferCode and inherits Version and PublicationDate from the first
// successfully fetched service.
//
// Parameters:
//   - region: AWS region used to fetch service pricing.
//   - services: slice of AWS service codes to fetch and combine.
//   - outDir: directory where the resulting JSON file will be written.
//
// Returns an error if any service fetch fails, if the output directory or file
// cannot be created, or if encoding the combined pricing to JSON fails.
func generateCombinedPricingData(region string, services []string, outDir string) error {
	// Combined pricing structure
	combined := awsPricing{
		FormatVersion: "v1.0",
		Products:      make(map[string]json.RawMessage),
		Terms:         make(map[string]map[string]json.RawMessage),
	}
	combined.Terms["OnDemand"] = make(map[string]json.RawMessage)

	for _, service := range services {
		service = strings.TrimSpace(service)
		if service == "" {
			continue
		}

		fmt.Printf("Fetching %s for %s...\n", service, region)
		data, err := fetchServicePricing(region, service)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", service, err)
		}

		// Merge all products - no filtering, keep full data for accurate cost estimation
		for sku, product := range data.Products {
			combined.Products[sku] = product
		}

		// Merge OnDemand terms
		if onDemand, ok := data.Terms["OnDemand"]; ok {
			for sku, term := range onDemand {
				combined.Terms["OnDemand"][sku] = term
			}
		}
		fmt.Printf("Merged %d products for %s\n", len(data.Products), service)

		// Keep metadata from first service
		if combined.OfferCode == "" {
			combined.OfferCode = "Combined"
			combined.Version = data.Version
			combined.PublicationDate = data.PublicationDate
		}
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

	// Write combined JSON
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(combined); err != nil {
		return fmt.Errorf("failed to encode combined pricing: %w", err)
	}

	return nil
}

// httpRequestTimeout is the timeout for HTTP requests to AWS pricing API
const httpRequestTimeout = 5 * time.Minute

// fetchServicePricing retrieves AWS pricing data for the specified service and region.
// It requests the Pricing API index JSON for the given service and region and parses it into an awsPricing value.
// region is the AWS region code (for example, "us-east-1").
// service is the AWS service code (for example, "AmazonEC2").
// It returns the parsed awsPricing on success. An error is returned if the HTTP request fails, the response status is not 200 OK, reading the response body fails, or JSON unmarshaling fails.
func fetchServicePricing(region, service string) (*awsPricing, error) {
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

	var data awsPricing
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &data, nil
}
