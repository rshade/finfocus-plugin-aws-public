package main

import (
	"encoding/json"
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
	service := flag.String("service", "AmazonEC2", "AWS Service Codes (comma-separated, e.g. AmazonEC2,AmazonRDS)")
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

// generateCombinedPricingData fetches and combines pricing data from multiple AWS services
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

		// Merge products
		for sku, product := range data.Products {
			combined.Products[sku] = product
		}

		// Merge OnDemand terms
		if onDemand, ok := data.Terms["OnDemand"]; ok {
			for sku, term := range onDemand {
				combined.Terms["OnDemand"][sku] = term
			}
		}

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

// fetchServicePricing retrieves AWS pricing data for the specified service and region.
// It requests the Pricing API index JSON for the given service and region and parses it into an awsPricing value.
// region is the AWS region code (for example, "us-east-1").
// service is the AWS service code (for example, "AmazonEC2").
// It returns the parsed awsPricing on success. An error is returned if the HTTP request fails, the response status is not 200 OK, reading the response body fails, or JSON unmarshaling fails.
func fetchServicePricing(region, service string) (*awsPricing, error) {
	url := fmt.Sprintf("https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/%s/current/%s/index.json", service, region)

	resp, err := http.Get(url)
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