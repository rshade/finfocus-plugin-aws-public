package plugin

import (
	"maps"
	"strconv"
	"strings"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/protobuf/proto"
)

const (
	aliasArch        = "arch"
	aliasSupportType = "support_type"
	skuOnDemand      = "on-demand"
	skuProvisioned   = "provisioned"
	skuGWLB          = "gwlb"
)

// aliasPair is one received key and the canonical key estimators read.
type aliasPair struct {
	from string
	to   string
}

// serviceAliases maps a service to received keys and the canonical tag the
// estimators already read. Pairs are ordered. When two aliases share a
// canonical key, the earlier pair wins if that canonical key is still empty.
// A value is copied only when the canonical tag is absent or empty, so an
// explicit canonical tag wins over an alias tag. Attributes are not copied
// onto the descriptor. pricingTags resolves them for estimators.
//
// Read-only after initialization.
var serviceAliases = map[string][]aliasPair{
	serviceRDS: {
		{from: "storageType", to: "storage_type"},
		{from: "allocatedStorage", to: "storage_size"},
		{from: "multiAz", to: "multi_az"},
	},
	serviceDynamoDB: {
		{from: "readCapacity", to: "read_capacity_units"},
		{from: "readCapacityUnits", to: "read_capacity_units"},
		{from: "writeCapacity", to: "write_capacity_units"},
		{from: "writeCapacityUnits", to: "write_capacity_units"},
		{from: "billingMode", to: "billing_mode"},
		{from: "storageGb", to: "storage_gb"},
		{from: "readRequestsPerMonth", to: "read_requests_per_month"},
		{from: "writeRequestsPerMonth", to: "write_requests_per_month"},
	},
	serviceElastiCache: {
		{from: "numCacheNodes", to: "num_nodes"},
		{from: "num_cache_nodes", to: "num_nodes"},
		{from: "nodeType", to: "node_type"},
	},
	serviceLambda: {
		{from: "architectures", to: aliasArch},
		{from: "architecture", to: aliasArch},
		{from: "memorySize", to: "memory_mb"},
		{from: "requestsPerMonth", to: "requests_per_month"},
		{from: "avgDurationMs", to: "avg_duration_ms"},
	},
	serviceEKS: {
		{from: "upgradePolicy", to: aliasSupportType},
		{from: "supportType", to: aliasSupportType},
	},
	serviceASG: {
		{from: "mixedInstancesPolicy", to: "mixed_instances_policy"},
		{from: "desiredCapacity", to: "desired_capacity"},
		{from: "minSize", to: "min_size"},
	},
	serviceELB: {
		{from: "loadBalancerType", to: "load_balancer_type"},
		{from: "lcuPerHour", to: "lcu_per_hour"},
		{from: "nlcuPerHour", to: "nlcu_per_hour"},
		{from: "capacityUnits", to: "capacity_units"},
	},
	serviceNATGW: {
		{from: "dataProcessedGb", to: "data_processed_gb"},
	},
	serviceCloudWatch: {
		{from: "logIngestionGb", to: "log_ingestion_gb"},
		{from: "logStorageGb", to: "log_storage_gb"},
		{from: "customMetrics", to: "custom_metrics"},
	},
	serviceS3: {
		{from: "storageClass", to: "storage_class"},
	},
	serviceEBS: {
		{from: "volumeSize", to: "size"},
		{from: "volume_size", to: "size"},
	},
}

// s3StorageClass returns the storage class estimators price.
// A non-empty SKU wins. Otherwise the resolved storage_class input wins.
// STANDARD is the recorded default only when both are absent.
func s3StorageClass(resource *pbc.ResourceDescriptor) (string, bool) {
	if resource != nil && resource.GetSku() != "" {
		return resource.GetSku(), false
	}
	if v := pricingTags(resource)["storage_class"]; v != "" {
		return v, false
	}
	return "STANDARD", true
}

// canonicalizeResource returns a cloned descriptor with alias tags copied
// onto canonical keys and, when Sku is empty, a SKU derived from inputs.
// The caller's descriptor is not modified. A non-empty Sku is kept.
// An explicit canonical tag is kept. Values derived here are real inputs,
// not defaults, so they are not recorded in defaults_applied.
func canonicalizeResource(service string, resource *pbc.ResourceDescriptor) *pbc.ResourceDescriptor {
	if resource == nil {
		return nil
	}
	cloned, ok := proto.Clone(resource).(*pbc.ResourceDescriptor)
	if !ok || cloned == nil {
		return nil
	}

	if pairs := serviceAliases[service]; len(pairs) > 0 {
		tags := descriptorTags(cloned)
		for _, pair := range pairs {
			if tags[pair.to] != "" {
				continue
			}
			raw := tags[pair.from]
			if raw == "" {
				continue
			}
			if tags == nil {
				tags = ensureDescriptorTags(cloned)
			}
			tags[pair.to] = normalizeAliasValue(pair.to, raw)
		}
	}

	if cloned.GetSku() == "" {
		if sku := deriveSKU(service, newResourceInputs(resource)); sku != "" {
			cloned.Sku = sku
		}
	}
	return cloned
}

// normalizeAliasValue shapes an alias for the estimator that reads the
// canonical key. EKS support types are lowercased. Lambda architectures
// keep the first list element when the host joined them with commas.
func normalizeAliasValue(canonical, raw string) string {
	value := strings.TrimSpace(raw)
	switch canonical {
	case aliasSupportType:
		return strings.ToLower(value)
	case aliasArch:
		if i := strings.IndexByte(value, ','); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		return value
	default:
		return value
	}
}

// deriveSKU fills an empty SKU from inputs the host already sent.
// S3, NAT Gateway, and CloudWatch are omitted. Their estimators apply and
// record their own defaults. A gateway load balancer becomes the sentinel
// "gwlb" so it is not priced as an ALB.
func deriveSKU(service string, in resourceInputs) string {
	switch service {
	case serviceElastiCache:
		if v, ok := in.String("nodeType", "node_type"); ok && v != "" {
			return v
		}
	case serviceLambda:
		if v, ok := in.String("memorySize", "memory_mb"); ok && v != "" {
			n, err := strconv.Atoi(strings.TrimSpace(strings.Split(v, ".")[0]))
			if err == nil && n > 0 {
				return strconv.Itoa(n)
			}
		}
	case serviceDynamoDB:
		if v, ok := in.String("billingMode", "billing_mode"); ok && v != "" {
			return normalizeBillingMode(v)
		}
	case serviceELB:
		if v, ok := in.String("loadBalancerType", "load_balancer_type"); ok && v != "" {
			return elbSKU(v)
		}
	}
	return ""
}

// normalizeBillingMode maps Terraform billing_mode values onto the SKUs
// estimateDynamoDB already understands.
func normalizeBillingMode(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "PAY_PER_REQUEST", "ON-DEMAND", "ONDEMAND", "ON_DEMAND":
		return skuOnDemand
	case "PROVISIONED":
		return skuProvisioned
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

// elbSKU maps a load balancer type onto alb, nlb, or gwlb.
func elbSKU(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "application", "alb":
		return serviceALB
	case "network", "nlb":
		return serviceNLB
	case "gateway", skuGWLB, "gateway_load_balancer":
		return skuGWLB
	default:
		lower := strings.ToLower(strings.TrimSpace(raw))
		if strings.Contains(lower, "gateway") {
			return skuGWLB
		}
		if strings.Contains(lower, "network") {
			return serviceNLB
		}
		if strings.Contains(lower, "application") {
			return serviceALB
		}
		return lower
	}
}

// pricingTags returns the tag map estimators read.
//
// Tag values are copied first. Attribute scalars replace the same key, so
// attributes win. Each alias is then resolved with the input lookup and
// written to the canonical key when the resolved value is non-empty. The
// descriptor is not modified.
func pricingTags(resource *pbc.ResourceDescriptor) map[string]string {
	if resource == nil {
		return map[string]string{}
	}
	in := newResourceInputs(resource)
	out := make(map[string]string, len(in.tags))
	maps.Copy(out, in.tags)
	if in.attrs != nil {
		for key, value := range in.attrs.GetFields() {
			if s, ok := scalarString(value); ok {
				out[key] = s
			}
		}
	}
	service := detectService(normalizeResourceType(resource.GetResourceType()))
	for canonical, keys := range aliasLookupKeys(service) {
		v, ok := in.String(keys...)
		if !ok || v == "" {
			continue
		}
		normalized := normalizeAliasValue(canonical, v)
		for _, key := range keys {
			out[key] = normalized
		}
	}
	return out
}

// aliasLookupKeys groups alias pairs by canonical key.
//
// RDS checks the Pulumi name before the legacy name, matching the estimator.
// Every other service checks the canonical name before aliases, so an
// explicit canonical tag wins over an alias tag. resourceInputs.String still
// searches every attribute key before any tag, so an attribute wins either way.
func aliasLookupKeys(service string) map[string][]string {
	pairs := serviceAliases[service]
	if len(pairs) == 0 {
		return nil
	}
	var order []string
	grouped := make(map[string][]string)
	for _, pair := range pairs {
		if _, seen := grouped[pair.to]; !seen {
			order = append(order, pair.to)
			grouped[pair.to] = nil
		}
		grouped[pair.to] = append(grouped[pair.to], pair.from)
	}
	result := make(map[string][]string, len(order))
	for _, canonical := range order {
		aliases := grouped[canonical]
		if service == serviceRDS {
			keys := make([]string, 0, len(aliases)+1)
			keys = append(keys, aliases...)
			keys = append(keys, canonical)
			result[canonical] = keys
			continue
		}
		keys := make([]string, 0, len(aliases)+1)
		keys = append(keys, canonical)
		keys = append(keys, aliases...)
		result[canonical] = keys
	}
	return result
}

// skuOptional reports services that can be priced without a SKU.
// The estimator either derives one, applies a recorded default, or returns
// a service-specific error.
func skuOptional(service string) bool {
	switch service {
	case serviceASG, serviceS3, serviceLambda, serviceDynamoDB,
		serviceNATGW, serviceCloudWatch, serviceElastiCache, serviceELB:
		return true
	default:
		return false
	}
}

// emptyRegionUsesPlugin reports services whose empty region means the
// plugin region. Every other service must name a region.
func emptyRegionUsesPlugin(service string) bool {
	return service == serviceS3 || service == serviceIAM || IsZeroCostService(service)
}
