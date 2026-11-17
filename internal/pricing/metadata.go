package pricing

import "time"

// Pricing file keys match the embedded JSON prefix ({service}_{region}.json).
const (
	// ServiceEC2 is the AmazonEC2 offer, which also prices EBS and ASG compute.
	ServiceEC2 = "ec2"
	// ServiceS3 is the AmazonS3 offer.
	ServiceS3 = "s3"
	// ServiceRDS is the AmazonRDS offer.
	ServiceRDS = "rds"
	// ServiceEKS is the AmazonEKS offer.
	ServiceEKS = "eks"
	// ServiceLambda is the AWSLambda offer.
	ServiceLambda = "lambda"
	// ServiceDynamoDB is the AmazonDynamoDB offer.
	ServiceDynamoDB = "dynamodb"
	// ServiceELB is the AWSELB offer.
	ServiceELB = "elb"
	// ServiceVPC is the AmazonVPC offer, which prices NAT Gateway and public IPv4.
	ServiceVPC = "vpc"
	// ServiceCloudWatch is the AmazonCloudWatch offer.
	ServiceCloudWatch = "cloudwatch"
	// ServiceElastiCache is the AmazonElastiCache offer.
	ServiceElastiCache = "elasticache"
	// ServiceECS is the AmazonECS offer, which prices Fargate.
	ServiceECS = "ecs"
	// ServiceEFS is the AmazonEFS offer.
	ServiceEFS = "efs"

	// PricingVersionFallback is the version string embedded in dev builds.
	PricingVersionFallback = "fallback"
)

// Files is the stable order of embedded service files.
func Files() []string {
	return []string{
		ServiceEC2,
		ServiceS3,
		ServiceRDS,
		ServiceEKS,
		ServiceLambda,
		ServiceDynamoDB,
		ServiceELB,
		ServiceVPC,
		ServiceCloudWatch,
		ServiceElastiCache,
		ServiceECS,
		ServiceEFS,
	}
}

// pricingRecord is one service file's header, written by that file's parser.
type pricingRecord struct {
	meta PricingMetadata
	set  bool
}

// rememberPricing stores the header of an already-unmarshaled offer.
// Each service is parsed by one goroutine, and each goroutine writes a
// different field, so this needs no lock.
func (c *Client) rememberPricing(service string, src awsPricing) PricingMetadata {
	meta := PricingMetadata{
		FormatVersion:   src.FormatVersion,
		Version:         src.Version,
		PublicationDate: src.PublicationDate,
		OfferCode:       src.OfferCode,
	}
	rec := pricingRecord{meta: meta, set: true}
	switch service {
	case ServiceEC2:
		c.metaEC2 = rec
	case ServiceS3:
		c.metaS3 = rec
	case ServiceRDS:
		c.metaRDS = rec
	case ServiceEKS:
		c.metaEKS = rec
	case ServiceLambda:
		c.metaLambda = rec
	case ServiceDynamoDB:
		c.metaDynamoDB = rec
	case ServiceELB:
		c.metaELB = rec
	case ServiceVPC:
		c.metaVPC = rec
	case ServiceCloudWatch:
		c.metaCloudWatch = rec
	case ServiceElastiCache:
		c.metaElastiCache = rec
	case ServiceECS:
		c.metaECS = rec
	case ServiceEFS:
		c.metaEFS = rec
	default:
		c.logger.Warn().Str("service", service).Msg("pricing metadata ignored for unknown service file")
	}
	return meta
}

func (c *Client) record(service string) pricingRecord {
	switch service {
	case ServiceEC2:
		return c.metaEC2
	case ServiceS3:
		return c.metaS3
	case ServiceRDS:
		return c.metaRDS
	case ServiceEKS:
		return c.metaEKS
	case ServiceLambda:
		return c.metaLambda
	case ServiceDynamoDB:
		return c.metaDynamoDB
	case ServiceELB:
		return c.metaELB
	case ServiceVPC:
		return c.metaVPC
	case ServiceCloudWatch:
		return c.metaCloudWatch
	case ServiceElastiCache:
		return c.metaElastiCache
	case ServiceECS:
		return c.metaECS
	case ServiceEFS:
		return c.metaEFS
	default:
		return pricingRecord{}
	}
}

// PricingMetadata returns the embedded Price List header for one service file.
// The bool is false when that file was not loaded.
func (c *Client) PricingMetadata(service string) (PricingMetadata, bool) {
	rec := c.record(service)
	if !rec.set {
		return PricingMetadata{}, false
	}
	return rec.meta, true
}

// PricingAsOf returns the oldest publicationDate across loaded service files.
// The oldest date is the conservative freshness of the binary. An unparseable
// date is skipped. The returned string is the value AWS published, unchanged.
func (c *Client) PricingAsOf() string {
	var oldest time.Time
	var raw string
	for _, file := range Files() {
		meta, ok := c.PricingMetadata(file)
		if !ok || meta.PublicationDate == "" {
			continue
		}
		published, err := time.Parse(time.RFC3339, meta.PublicationDate)
		if err != nil {
			continue
		}
		if raw == "" || published.Before(oldest) {
			oldest = published
			raw = meta.PublicationDate
		}
	}
	return raw
}

// NATGatewayOffer returns the file that supplied the selected NAT Gateway price.
func (c *Client) NATGatewayOffer() string {
	if c.natGatewayPricing == nil {
		return ""
	}
	return c.natGatewayPricing.Offer
}
