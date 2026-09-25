package pricing

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/rs/zerolog"
)

// parseRepetitions is how many times each parser runs over the same input.
// Go randomizes map iteration order, so a parser that resolves colliding SKUs
// by iteration order (last-write-wins or first-write-wins) returns varying
// prices across runs; 64 runs makes a lucky pass vanishingly unlikely.
const parseRepetitions = 64

type fixtureDim struct {
	begin string
	end   string
	unit  string
	usd   string
}

type fixtureSKU struct {
	sku    string
	family string
	attrs  map[string]string
	dims   []fixtureDim
}

// buildPriceList renders fixture SKUs as AWS Price List API JSON.
func buildPriceList(t *testing.T, offerCode string, skus []fixtureSKU) []byte {
	t.Helper()

	data := awsPricing{
		OfferCode: offerCode,
		Products:  make(map[string]product, len(skus)),
		Terms:     map[string]map[string]map[string]term{"OnDemand": {}},
	}
	for _, s := range skus {
		data.Products[s.sku] = product{Sku: s.sku, ProductFamily: s.family, Attributes: s.attrs}
		dims := make(map[string]priceDimension, len(s.dims))
		for i, d := range s.dims {
			code := s.sku + ".JRTCKXETXF.6YS6EN2CT" + string(rune('A'+i))
			dims[code] = priceDimension{
				RateCode:     code,
				BeginRange:   d.begin,
				EndRange:     d.end,
				Unit:         d.unit,
				PricePerUnit: map[string]string{currencyUSD: d.usd},
			}
		}
		data.Terms["OnDemand"][s.sku] = map[string]term{
			s.sku + ".JRTCKXETXF": {OfferTermCode: "JRTCKXETXF", Sku: s.sku, PriceDimensions: dims},
		}
	}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

func newParserTestClient() *Client {
	return &Client{
		logger:           zerolog.Nop(),
		ec2Index:         make(map[string]ec2Price),
		ebsIndex:         make(map[string]ebsPrice),
		rdsInstanceIndex: make(map[string]rdsInstancePrice),
		rdsStorageIndex:  make(map[string]rdsStoragePrice),
		elasticacheIndex: make(map[string]elasticacheInstancePrice),
	}
}

// assertParsedRegion runs a parser and checks it reports the fixture region.
func assertParsedRegion(t *testing.T, parse func([]byte) (string, error), raw []byte) {
	t.Helper()
	region, err := parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if region != "us-east-1" {
		t.Fatalf("parsed region = %q, want us-east-1", region)
	}
}

func hourly(usd string) []fixtureDim {
	return []fixtureDim{{begin: "0", end: "Inf", unit: "Hrs", usd: usd}}
}

func gbMonth(usd string) []fixtureDim {
	return []fixtureDim{{begin: "0", end: "Inf", unit: unitGBMonth, usd: usd}}
}

func cacheNode(sku, locationType, usageType, usd string) fixtureSKU {
	return fixtureSKU{
		sku:    sku,
		family: "Cache Instance",
		attrs: map[string]string{
			"instanceType":  "cache.m6g.large",
			"cacheEngine":   "Redis",
			"locationType":  locationType,
			"usagetype":     usageType,
			"regionCode":    "us-east-1",
			"servicecode":   "AmazonElastiCache",
			"operation":     "CreateCacheCluster:0002",
			"normalization": "4",
		},
		dims: hourly(usd),
	}
}

// TestParseElastiCachePricing_IgnoresNodeVariants verifies that only the
// standard in-region node SKU is indexed for an instanceType:engine key.
//
// AWS publishes several Cache Instance SKUs per node type and engine: the
// standard node, extended-support tiers, sync-durability nodes, and Outposts
// nodes. They share the index key, so without filtering the stored price
// depended on map iteration order (cache.m6g.large Redis varied between the
// standard rate and extended-support rates in the real us-east-1 data).
//
// The us-east-1 standard usage type carries no region prefix
// ("NodeUsage:...") while other regions do ("USW2-NodeUsage:..."); both must
// be accepted.
func TestParseElastiCachePricing_IgnoresNodeVariants(t *testing.T) {
	skus := []fixtureSKU{
		cacheNode("STD", "AWS Region", "NodeUsage:cache.m6g.large", "0.149"),
		cacheNode("EXT12", "AWS Region", "USE1-ExtendedSupportYr1_Yr2-NodeUsage:cache.m6g.large", "0.119"),
		cacheNode("EXT3", "AWS Region", "USE1-ExtendedSupportYr3-NodeUsage:cache.m6g.large", "0.238"),
		cacheNode("SYNC", "AWS Region", "USE1-SyncDurability-NodeUsage:cache.m6g.large", "0.201"),
		cacheNode("OUTPOST", "AWS Outposts", "USE1-Outpost-NodeUsage:cache.m6g.large", "0.310"),
	}
	prefixed := cacheNode("STDUSW2", "AWS Region", "USW2-NodeUsage:cache.t3.micro", "0.017")
	prefixed.attrs["instanceType"] = "cache.t3.micro"
	skus = append(skus, prefixed)
	raw := buildPriceList(t, "AmazonElastiCache", skus)

	for i := range parseRepetitions {
		c := newParserTestClient()
		assertParsedRegion(t, c.parseElastiCachePricing, raw)
		if got := c.elasticacheIndex["cache.m6g.large:Redis"].HourlyRate; got != 0.149 {
			t.Fatalf("run %d: cache.m6g.large:Redis = %v, want standard node rate 0.149", i, got)
		}
		if got := c.elasticacheIndex["cache.t3.micro:Redis"].HourlyRate; got != 0.017 {
			t.Fatalf("run %d: region-prefixed standard node cache.t3.micro:Redis = %v, want 0.017", i, got)
		}
	}
}

// TestParseLambdaPricing_UsesFirstPaidTierAndSkipsFreeTier verifies Lambda
// duration and request prices are deterministic list prices.
//
// Two defects are covered:
//   - Duration SKUs carry three volume tiers in one priceDimensions map, so
//     the returned tier depended on map order (Tier-3 is 20% below list).
//   - The free-tier SKU "Global-Lambda-GB-Second" ($0) shares the
//     AWS-Lambda-Duration group and unit with the paid SKU and could
//     overwrite the x86 rate with zero.
func TestParseLambdaPricing_UsesFirstPaidTierAndSkipsFreeTier(t *testing.T) {
	serverless := func(sku, group, usageType string, dims []fixtureDim) fixtureSKU {
		return fixtureSKU{
			sku:    sku,
			family: "Serverless",
			attrs: map[string]string{
				"group": group, "usagetype": usageType, "regionCode": "us-east-1",
				"locationType": "AWS Region", "servicecode": "AWSLambda",
			},
			dims: dims,
		}
	}
	tiers := func(unit, t1, t2, t3 string) []fixtureDim {
		return []fixtureDim{
			{begin: "0", end: "6000000000", unit: unit, usd: t1},
			{begin: "6000000000", end: "15000000000", unit: unit, usd: t2},
			{begin: "15000000000", end: "Inf", unit: unit, usd: t3},
		}
	}
	raw := buildPriceList(t, "AWSLambda", []fixtureSKU{
		serverless("X86", "AWS-Lambda-Duration", "Lambda-GB-Second",
			tiers("Lambda-GB-Second", "0.0000166667", "0.0000150000", "0.0000133334")),
		serverless("ARM", "AWS-Lambda-Duration-ARM", "Lambda-GB-Second-ARM",
			tiers("Lambda-GB-Second", "0.0000133334", "0.0000120001", "0.0000106667")),
		serverless("FREEGB", "AWS-Lambda-Duration", "Global-Lambda-GB-Second",
			[]fixtureDim{{begin: "0", end: "400000", unit: "Lambda-GB-Second", usd: "0.0000000000"}}),
		serverless("REQ", "AWS-Lambda-Requests", "Request",
			[]fixtureDim{{begin: "0", end: "Inf", unit: "Requests", usd: "0.0000002000"}}),
		serverless("FREEREQ", "AWS-Lambda-Requests", "Global-Request",
			[]fixtureDim{{begin: "0", end: "1000000", unit: "Requests", usd: "0.0000000000"}}),
	})

	for i := range parseRepetitions {
		c := newParserTestClient()
		assertParsedRegion(t, c.parseLambdaPricing, raw)
		if got := c.lambdaPricing.X86GBSecondPrice; got != 0.0000166667 {
			t.Fatalf("run %d: x86 GB-second = %v, want Tier-1 list price 0.0000166667", i, got)
		}
		if got := c.lambdaPricing.ARMGBSecondPrice; got != 0.0000133334 {
			t.Fatalf("run %d: arm64 GB-second = %v, want Tier-1 list price 0.0000133334", i, got)
		}
		if got := c.lambdaPricing.RequestPrice; got != 0.0000002 {
			t.Fatalf("run %d: request price = %v, want 0.0000002", i, got)
		}
	}
}

func dbInstance(sku, engine string, extra map[string]string, usd string) fixtureSKU {
	attrs := map[string]string{
		"instanceType":     "db.m5.large",
		"databaseEngine":   engine,
		"deploymentOption": "Single-AZ",
		"locationType":     "AWS Region",
		"usagetype":        "InstanceUsage:db.m5.large",
		"regionCode":       "us-east-1",
	}
	maps.Copy(attrs, extra)
	return fixtureSKU{sku: sku, family: "Database Instance", attrs: attrs, dims: hourly(usd)}
}

// TestParseRDSPricing_InstanceVariantsResolveDeterministically verifies the
// RDS instance index picks one documented variant per instanceType/engine key.
//
// The key carries no edition, license model, or deployment model, so AWS
// publishes many SKUs per key for Aurora (standard vs I/O-Optimized), Oracle
// and SQL Server (editions x license models, RDS Custom, customer-provided
// media). Without a preference the stored price depended on map order: in the
// real us-east-1 data db.m5.large SQL Server ranged from $0.30 to $1.23/hr.
//
// Policy: exclude RDS Custom, Outposts and Aurora I/O-Optimized; then prefer
// "License included" (the full AWS bill), then "Marketplace", then
// bring-your-own; then Standard editions; then the lowest rate.
func TestParseRDSPricing_InstanceVariantsResolveDeterministically(t *testing.T) {
	li := "License included"
	byol := "Bring your own license"
	raw := buildPriceList(t, "AmazonRDS", []fixtureSKU{
		dbInstance(
			"SQL-WEB-LI",
			"SQL Server",
			map[string]string{"databaseEdition": "Web", "licenseModel": li},
			"0.311",
		),
		dbInstance(
			"SQL-STD-LI",
			"SQL Server",
			map[string]string{"databaseEdition": "Standard", "licenseModel": li},
			"0.977",
		),
		dbInstance(
			"SQL-ENT-LI",
			"SQL Server",
			map[string]string{"databaseEdition": "Enterprise", "licenseModel": li},
			"1.740",
		),
		dbInstance("SQL-STD-BYOM", "SQL Server", map[string]string{
			"databaseEdition": "Standard",
			"licenseModel":    "Bring your own media",
			"engineMediaType": "Customer-provided",
		}, "0.171"),
		dbInstance("SQL-STD-CUSTOM", "SQL Server", map[string]string{
			"databaseEdition": "Standard", "licenseModel": "NA", "deploymentModel": "Custom",
		}, "1.229"),
		dbInstance(
			"ORA-EE-BYOL",
			"Oracle",
			map[string]string{"databaseEdition": "Enterprise", "licenseModel": byol},
			"0.171",
		),
		dbInstance(
			"ORA-SE2-BYOL",
			"Oracle",
			map[string]string{"databaseEdition": "Standard Two", "licenseModel": byol},
			"0.171",
		),
		dbInstance(
			"ORA-SE2-LI",
			"Oracle",
			map[string]string{"databaseEdition": "Standard Two", "licenseModel": li},
			"0.316",
		),
		dbInstance("MYSQL", "MySQL", map[string]string{"licenseModel": "No license required"}, "0.171"),
		dbInstance("AUR-STD", "Aurora PostgreSQL", map[string]string{"licenseModel": "No license required"}, "0.260"),
		dbInstance("AUR-IOOPT", "Aurora PostgreSQL", map[string]string{
			"licenseModel": "No license required", "usagetype": "InstanceUsageIOOptimized:db.m5.large",
		}, "0.338"),
		dbInstance("MYSQL-OUTPOST", "MySQL", map[string]string{
			"licenseModel": "No license required", "locationType": "AWS Outposts",
		}, "0.990"),
	})

	want := map[string]float64{
		"db.m5.large/SQL Server":        0.977,
		"db.m5.large/Oracle":            0.316,
		"db.m5.large/MySQL":             0.171,
		"db.m5.large/Aurora PostgreSQL": 0.260,
	}
	for i := range parseRepetitions {
		c := newParserTestClient()
		assertParsedRegion(t, c.parseRDSPricing, raw)
		for key, w := range want {
			if got := c.rdsInstanceIndex[key].HourlyRate; got != w {
				t.Fatalf("run %d: %s = %v, want %v", i, key, got, w)
			}
		}
	}
}

// TestRDSInstanceVariantRank_BringYourOwnWhenNoLicenseIncluded verifies that
// a key with only bring-your-own-license SKUs (e.g. large Oracle instances,
// which AWS does not sell license-included) still resolves, preferring the
// Standard edition.
func TestRDSInstanceVariantRank_BringYourOwnWhenNoLicenseIncluded(t *testing.T) {
	byol := "Bring your own license"
	raw := buildPriceList(t, "AmazonRDS", []fixtureSKU{
		dbInstance(
			"ORA-EE-BYOL",
			"Oracle",
			map[string]string{"databaseEdition": "Enterprise", "licenseModel": byol},
			"0.180",
		),
		dbInstance(
			"ORA-SE2-BYOL",
			"Oracle",
			map[string]string{"databaseEdition": "Standard Two", "licenseModel": byol},
			"0.171",
		),
	})
	for i := range parseRepetitions {
		c := newParserTestClient()
		assertParsedRegion(t, c.parseRDSPricing, raw)
		if got := c.rdsInstanceIndex["db.m5.large/Oracle"].HourlyRate; got != 0.171 {
			t.Fatalf("run %d: Oracle BYOL-only = %v, want Standard Two rate 0.171", i, got)
		}
	}
}

func dbStorage(sku, volumeType, deployment, usageType, usd string) fixtureSKU {
	return fixtureSKU{
		sku:    sku,
		family: "Database Storage",
		attrs: map[string]string{
			"volumeType":       volumeType,
			"deploymentOption": deployment,
			"usagetype":        usageType,
			"databaseEngine":   "Any",
			"locationType":     "AWS Region",
			"regionCode":       "us-east-1",
		},
		dims: gbMonth(usd),
	}
}

// TestParseRDSPricing_StorageUsesSingleAZAndIndexesGP3IO2 verifies RDS
// storage rates are Single-AZ list prices and that gp3 and io2 are indexed.
//
// Two defects are covered:
//   - Single-AZ, Multi-AZ, readable-standby cluster and SQL Server Mirror
//     storage share a volume-type key; first-write-wins over random map order
//     priced storage at 2-3x the Single-AZ rate on some runs.
//   - The Price List names gp3 and io2 "General Purpose-GP3" and
//     "Provisioned IOPS-IO2" with upper-case usage types, which the parser
//     never matched, so gp3/io2 storage was silently priced at $0.
func TestParseRDSPricing_StorageUsesSingleAZAndIndexesGP3IO2(t *testing.T) {
	raw := buildPriceList(t, "AmazonRDS", []fixtureSKU{
		dbStorage("GP2-SAZ", "General Purpose", "Single-AZ", "RDS:GP2-Storage", "0.115"),
		dbStorage("GP2-MAZ", "General Purpose", "Multi-AZ", "RDS:Multi-AZ-GP2-Storage", "0.230"),
		dbStorage("GP2-MIRROR", "General Purpose", "Multi-AZ (SQL Server Mirror)", "RDS:Mirror-GP2-Storage", "0.230"),
		dbStorage("GP3-SAZ", "General Purpose-GP3", "Single-AZ", "RDS:GP3-Storage", "0.116"),
		dbStorage("GP3-CLUSTER", "General Purpose-GP3", "Multi-AZ (readable standbys)",
			"RDS:Multi-AZCluster-GP3-Storage", "0.345"),
		dbStorage("IO1-SAZ", "Provisioned IOPS", "Single-AZ", "RDS:PIOPS-Storage", "0.125"),
		dbStorage("IO1-MAZ", "Provisioned IOPS", "Multi-AZ", "RDS:Multi-AZ-PIOPS-Storage", "0.250"),
		dbStorage("IO2-SAZ", "Provisioned IOPS-IO2", "Single-AZ", "RDS:PIOPS-Storage-IO2", "0.126"),
		dbStorage("IO2-MAZ", "Provisioned IOPS-IO2", "Multi-AZ", "RDS:Multi-AZ-PIOPS-Storage-IO2", "0.250"),
		dbStorage("MAG-SAZ", "Magnetic", "Single-AZ", "RDS:StorageUsage", "0.100"),
		dbStorage("MAG-MAZ", "Magnetic", "Multi-AZ", "RDS:Multi-AZ-StorageUsage", "0.200"),
	})

	want := map[string]float64{"gp2": 0.115, "gp3": 0.116, "io1": 0.125, "io2": 0.126, "standard": 0.100}
	for i := range parseRepetitions {
		c := newParserTestClient()
		assertParsedRegion(t, c.parseRDSPricing, raw)
		for volType, w := range want {
			got, ok := c.rdsStorageIndex[volType]
			if !ok {
				t.Fatalf("run %d: storage type %q not indexed", i, volType)
			}
			if got.RatePerGBMonth != w {
				t.Fatalf("run %d: %s storage = %v, want Single-AZ rate %v", i, volType, got.RatePerGBMonth, w)
			}
		}
	}
}

// TestGetOnDemandPrice_TierSelection verifies getOnDemandPrice returns the
// lowest-beginning paid tier, skipping a $0 free tier, and still returns $0
// for products whose only price is zero.
func TestGetOnDemandPrice_TierSelection(t *testing.T) {
	tests := []struct {
		name string
		dims []fixtureDim
		want float64
	}{
		{
			name: "volume tiers pick first tier",
			dims: []fixtureDim{
				{begin: "51200", end: "512000", unit: unitGBMonth, usd: "0.022"},
				{begin: "0", end: "51200", unit: unitGBMonth, usd: "0.023"},
				{begin: "512000", end: "Inf", unit: unitGBMonth, usd: "0.021"},
			},
			want: 0.023,
		},
		{
			name: "free tier skipped for first paid tier",
			dims: []fixtureDim{
				{begin: "0", end: "25", unit: unitGBMonth, usd: "0.0000000000"},
				{begin: "25", end: "Inf", unit: unitGBMonth, usd: "0.25"},
			},
			want: 0.25,
		},
		{
			name: "all-zero product stays zero",
			dims: []fixtureDim{{begin: "0", end: "Inf", unit: "Hrs", usd: "0.0000000000"}},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data awsPricing
			raw := buildPriceList(t, "AmazonS3", []fixtureSKU{{sku: "SKU", family: "Storage", dims: tt.dims}})
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			for i := range parseRepetitions {
				got, _, found := getOnDemandPrice(&data, "SKU")
				if !found || got != tt.want {
					t.Fatalf("run %d: getOnDemandPrice = (%v, found=%v), want %v", i, got, found, tt.want)
				}
			}
		})
	}
}

// TestParseDynamoDBPricing_ProvisionedCapacity verifies provisioned RCU/WCU
// prices are indexed from the current Price List shape.
//
// Three defects are covered:
//   - The unit is "ReadCapacityUnit-Hrs"/"WriteCapacityUnit-Hrs", which the
//     generic hourly-unit check rejects, so provisioned prices were never found.
//   - Each SKU starts with a $0 free tier (first 18,600 RCU-hours), which the
//     tier selection must skip to reach the list price.
//   - The Standard-IA table class ("IA-ReadCapacityUnit-Hrs") shares the
//     substring match and would overwrite the standard rate depending on map
//     order.
func TestParseDynamoDBPricing_ProvisionedCapacity(t *testing.T) {
	capacity := func(sku, usageType, unit string, dims []fixtureDim) fixtureSKU {
		for i := range dims {
			dims[i].unit = unit
		}
		return fixtureSKU{
			sku:    sku,
			family: "Provisioned IOPS",
			attrs: map[string]string{
				"usagetype": usageType, "servicecode": "AmazonDynamoDB",
				"regionCode": "us-east-1", "locationType": "AWS Region",
			},
			dims: dims,
		}
	}
	freeThenPaid := func(usd string) []fixtureDim {
		return []fixtureDim{{begin: "0", end: "18600", usd: "0.0000000000"}, {begin: "18600", end: "Inf", usd: usd}}
	}
	raw := buildPriceList(t, "AmazonDynamoDB", []fixtureSKU{
		capacity("RCU", "ReadCapacityUnit-Hrs", "ReadCapacityUnit-Hrs", freeThenPaid("0.00013")),
		capacity("WCU", "WriteCapacityUnit-Hrs", "WriteCapacityUnit-Hrs", freeThenPaid("0.00065")),
		capacity("RCU-IA", "IA-ReadCapacityUnit-Hrs", "ReadCapacityUnit-Hrs",
			[]fixtureDim{{begin: "0", end: "Inf", usd: "0.00016"}}),
		capacity("WCU-IA", "IA-WriteCapacityUnit-Hrs", "WriteCapacityUnit-Hrs",
			[]fixtureDim{{begin: "0", end: "Inf", usd: "0.00081"}}),
	})

	for i := range parseRepetitions {
		c := newParserTestClient()
		assertParsedRegion(t, c.parseDynamoDBPricing, raw)
		if got := c.dynamoDBPricing.ProvisionedRCUPrice; got != 0.00013 {
			t.Fatalf("run %d: provisioned RCU = %v, want standard list price 0.00013", i, got)
		}
		if got := c.dynamoDBPricing.ProvisionedWCUPrice; got != 0.00065 {
			t.Fatalf("run %d: provisioned WCU = %v, want standard list price 0.00065", i, got)
		}
	}
}
