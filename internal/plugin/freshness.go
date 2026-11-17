package plugin

import (
	"maps"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
)

const (
	metadataKeyPricingAsOf    = "pricing_as_of"
	metadataKeyPricingVersion = "pricing_version"
	metadataKeyPricingSource  = "pricing_source"
	pricingSourceFallback     = "fallback"
)

// pricingFileForService maps a detected service to the embedded offer that
// priced it. EBS and ASG use the EC2 file. NAT Gateway and Elastic IP use
// the VPC file. Fargate uses the ECS file. Zero-cost and unknown types
// return "" and are left unstamped.
func pricingFileForService(serviceType string) string {
	switch serviceType {
	case serviceEC2, serviceEBS, serviceASG:
		return pricing.ServiceEC2
	case serviceS3:
		return pricing.ServiceS3
	case serviceRDS:
		return pricing.ServiceRDS
	case serviceEKS:
		return pricing.ServiceEKS
	case serviceLambda:
		return pricing.ServiceLambda
	case serviceDynamoDB:
		return pricing.ServiceDynamoDB
	case serviceELB, serviceALB, serviceNLB:
		return pricing.ServiceELB
	case serviceNATGW, serviceEIP:
		return pricing.ServiceVPC
	case serviceCloudWatch:
		return pricing.ServiceCloudWatch
	case serviceElastiCache:
		return pricing.ServiceElastiCache
	case serviceEKSFargate, serviceFargate:
		return pricing.ServiceECS
	case serviceEFS:
		return pricing.ServiceEFS
	default:
		return ""
	}
}

// stampProjectedFreshness merges the pricing snapshot into a successful
// cost response. Existing defaults_applied and estimate_quality keys are
// left as they are. A client with no metadata, and a zero-cost or unknown
// service, leave the map untouched.
func (p *AWSPublicPlugin) stampProjectedFreshness(resp *pbc.GetProjectedCostResponse, serviceType string) {
	if resp == nil || p.pricing == nil {
		return
	}
	meta, ok := p.freshnessMeta(serviceType)
	if !ok {
		return
	}
	if resp.GetMetadata() == nil {
		resp.Metadata = map[string]string{}
	}
	applyFreshness(resp.GetMetadata(), meta)
}

// stampSpecFreshness sets PricingSpec.plugin_metadata for the same snapshot.
func (p *AWSPublicPlugin) stampSpecFreshness(spec *pbc.PricingSpec, serviceType string) {
	if spec == nil || p.pricing == nil {
		return
	}
	meta, ok := p.freshnessMeta(serviceType)
	if !ok {
		return
	}
	if spec.GetPluginMetadata() == nil {
		spec.PluginMetadata = map[string]string{}
	}
	applyFreshness(spec.GetPluginMetadata(), meta)
}

func (p *AWSPublicPlugin) freshnessMeta(serviceType string) (pricing.PricingMetadata, bool) {
	file := pricingFileForService(serviceType)
	if file == "" {
		return pricing.PricingMetadata{}, false
	}
	meta, ok := p.pricing.PricingMetadata(file)
	if !ok || (meta.PublicationDate == "" && meta.Version == "") {
		return pricing.PricingMetadata{}, false
	}
	return meta, true
}

func applyFreshness(dst map[string]string, meta pricing.PricingMetadata) {
	if meta.PublicationDate != "" {
		dst[metadataKeyPricingAsOf] = meta.PublicationDate
	}
	if meta.Version != "" {
		dst[metadataKeyPricingVersion] = meta.Version
	}
	if meta.Version == pricing.PricingVersionFallback {
		dst[metadataKeyPricingSource] = pricingSourceFallback
	}
}

// PricingInfoMetadata copies base and adds the binary-wide freshness keys:
// pricing_as_of (oldest publication date) and pricing_version.<service>.
// A fallback embed also sets pricing_source=fallback.
func PricingInfoMetadata(client pricing.PricingClient, base map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(pricing.Files())+2)
	maps.Copy(out, base)
	if client == nil {
		return out
	}
	if asOf := client.PricingAsOf(); asOf != "" {
		out[metadataKeyPricingAsOf] = asOf
	}
	fallback := false
	for _, file := range pricing.Files() {
		meta, ok := client.PricingMetadata(file)
		if !ok || meta.Version == "" {
			continue
		}
		out["pricing_version."+file] = meta.Version
		if meta.Version == pricing.PricingVersionFallback {
			fallback = true
		}
	}
	if fallback {
		out[metadataKeyPricingSource] = pricingSourceFallback
	}
	return out
}
