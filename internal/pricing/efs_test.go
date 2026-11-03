package pricing

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// efsTestProduct describes one synthetic AmazonEFS product for parser tests.
type efsTestProduct struct {
	family       string
	storageClass string
	throughput   string
	accessType   string
	usagetype    string
	locationType string
	unit         string
	price        string
}

// efsTestJSON builds an AmazonEFS Price List document from the given products.
func efsTestJSON(offerCode string, products []efsTestProduct) []byte {
	var prods, terms strings.Builder
	for i, p := range products {
		sku := fmt.Sprintf("SKU%d", i)
		sep := ""
		if i > 0 {
			sep = ","
		}
		fmt.Fprintf(&prods,
			`%s"%s":{"sku":"%s","productFamily":%q,"attributes":{"regionCode":"us-east-1","locationType":%q,`+
				`"usagetype":%q,"storageClass":%q,"throughputClass":%q,"accessType":%q}}`,
			sep, sku, sku, p.family, p.locationType, p.usagetype, p.storageClass, p.throughput, p.accessType,
		)
		fmt.Fprintf(&terms,
			`%s"%s":{"%s.T":{"offerTermCode":"T","sku":"%s","priceDimensions":{"%s.T.D":`+
				`{"rateCode":"%s.T.D","unit":%q,"pricePerUnit":{"USD":%q}}}}}`,
			sep, sku, sku, sku, sku, sku, p.unit, p.price,
		)
	}
	return fmt.Appendf(
		nil, `{"offerCode":%q,"products":{%s},"terms":{"OnDemand":{%s}}}`,
		offerCode, prods.String(), terms.String(),
	)
}

// TestParseEFSPricing verifies the EFS parser maps usage types to rates, tolerates a missing region
// prefix, and ignores overhead SKUs, Outposts locations, and unknown access types.
func TestParseEFSPricing(t *testing.T) {
	products := []efsTestProduct{
		{"Storage", "General Purpose", "", "", "USE1-TimedStorage-ByteHrs", "AWS Region", "GB-Mo", "0.30"},
		{"Storage", "Infrequent Access", "", "", "USE1-IATimedStorage-ByteHrs", "AWS Region", "GB-Mo", "0.025"},
		{"Storage", "Infrequent Access-ET", "", "", "USE1-IATimedStorage-ET-ByteHrs", "AWS Region", "GB-Mo", "0.016"},
		{"Storage", "Archive", "", "", "USE1-ArchiveTimedStorage-ByteHrs", "AWS Region", "GB-Mo", "0.008"},
		{"Storage", "One Zone-General Purpose", "", "", "USE1-TimedStorage-Z-ByteHrs", "AWS Region", "GB-Mo", "0.16"},
		{"Storage", "One Zone-Infrequent Access", "", "", "IATimedStorage-Z-ByteHrs", "AWS Region", "GB-Mo", "0.0133"},
		{"Storage", "EFS Storage", "", "Read", "USE1-ETDataAccess-Bytes", "AWS Region", "GB", "0.03"},
		{"Storage", "EFS Storage", "", "Write", "USE1-ETDataAccess-Bytes", "AWS Region", "GB", "0.06"},
		{"Storage", "Infrequent Access", "", "Read", "USE1-IADataAccess-Bytes", "AWS Region", "GB", "0.011"},
		{"Storage", "Infrequent Access", "", "Write", "USE1-IADataAccess-Bytes", "AWS Region", "GB", "0.012"},
		{"Storage", "Archive", "", "Read", "USE1-ArchiveDataAccess-Bytes", "AWS Region", "GB", "0.031"},
		{"Storage", "Archive", "", "Write", "USE1-ArchiveDataAccess-Bytes", "AWS Region", "GB", "0.032"},
		{
			"Provisioned Throughput",
			"",
			"Provisioned",
			"",
			"USE1-ProvisionedTP-MiBpsHrs",
			"AWS Region",
			"MiBps-Mo",
			"6.00",
		},
		{"Provisioned Throughput", "", "Included", "", "USE1-IncludedTP-MiBpsHrs", "AWS Region", "MiBps-Mo", "0.00"},
		{
			"Storage",
			"Infrequent Access-SmallFiles",
			"",
			"",
			"USE1-IATimedStorage-SmallFiles",
			"AWS Region",
			"GB-Mo",
			"9.99",
		},
		{"Storage", "General Purpose", "", "", "OUT-TimedStorage-ByteHrs", "AWS Outposts", "GB-Mo", "9.99"},
		{"Storage", "EFS Storage", "", "Other", "USE1-ETDataAccess-Bytes", "AWS Region", "GB", "9.99"},
	}

	c := &Client{logger: zerolog.Nop()}
	region, err := c.parseEFSPricing(efsTestJSON("AmazonEFS", products))
	if err != nil {
		t.Fatalf("parseEFSPricing() error = %v", err)
	}
	if region != "us-east-1" {
		t.Errorf("region = %q, want us-east-1", region)
	}

	want := EFSPrice{
		StandardGBMonth:       0.30,
		IAGBMonth:             0.025,
		IAElasticGBMonth:      0.016,
		ArchiveGBMonth:        0.008,
		OneZoneGBMonth:        0.16,
		OneZoneIAGBMonth:      0.0133,
		ProvisionedMiBpsMonth: 6.0,
		ElasticReadGB:         0.03,
		ElasticWriteGB:        0.06,
		IAReadGB:              0.011,
		IAWriteGB:             0.012,
		ArchiveReadGB:         0.031,
		ArchiveWriteGB:        0.032,
		Currency:              "USD",
	}
	if *c.efsPricing != want {
		t.Errorf("efsPricing = %+v, want %+v", *c.efsPricing, want)
	}
}

// TestParseEFSPricing_InvalidJSON verifies malformed input returns an error.
func TestParseEFSPricing_InvalidJSON(t *testing.T) {
	c := &Client{logger: zerolog.Nop()}
	if _, err := c.parseEFSPricing([]byte("{")); err == nil {
		t.Fatal("parseEFSPricing() error = nil, want error")
	}
}

// TestClient_EFSPrice verifies the embedded data yields Standard, IA, and Provisioned rates.
func TestClient_EFSPrice(t *testing.T) {
	client, err := NewClient(zerolog.Nop())
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	price, found := client.EFSPrice()
	if !found {
		t.Fatal("EFSPrice() not found")
	}
	for name, rate := range map[string]float64{
		"StandardGBMonth":       price.StandardGBMonth,
		"IAGBMonth":             price.IAGBMonth,
		"IAElasticGBMonth":      price.IAElasticGBMonth,
		"ArchiveGBMonth":        price.ArchiveGBMonth,
		"OneZoneGBMonth":        price.OneZoneGBMonth,
		"OneZoneIAGBMonth":      price.OneZoneIAGBMonth,
		"ProvisionedMiBpsMonth": price.ProvisionedMiBpsMonth,
		"ElasticReadGB":         price.ElasticReadGB,
		"ElasticWriteGB":        price.ElasticWriteGB,
	} {
		if rate <= 0 {
			t.Errorf("%s = %v, want > 0", name, rate)
		}
	}
	if price.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", price.Currency)
	}
}
