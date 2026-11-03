package plugin

import (
	"fmt"
	"math"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/grpc/codes"

	"github.com/rshade/finfocus-plugin-aws-public/internal/carbon"
	"github.com/rshade/finfocus-plugin-aws-public/internal/pricing"
)

// EFS throughput modes.
const (
	efsModeBursting    = "bursting"
	efsModeProvisioned = "provisioned"
	efsModeElastic     = "elastic"
)

// EFS storage class identifiers after normalization.
const (
	efsClassStandard = "std"
	efsClassIA       = "ia"
	efsClassArchive  = "archive"
)

const (
	efsServiceName    = "EFS"
	efsAliasStandard  = "standard"
	efsAliasGeneral   = "generalpurpose"
	efsAliasOneZone   = "onezone"
	efsAliasOneZoneIA = "onezoneia"
)

// efsBaselineMiBpsPerGB is the Bursting baseline AWS includes with Standard
// storage (50 KiB/s per GiB). Provisioned throughput is billed only above it.
const efsBaselineMiBpsPerGB = 50.0 / 1024.0

const bytesPerGB = 1024.0 * 1024.0 * 1024.0

// EFS size_in_bytes keys. Pulumi sends a struct, Terraform state sends a
// one-element list, and a flattening host sends dotted tag keys.
var (
	efsBytesStandardKeys = []string{
		"sizeInBytes.valueInStandard", "sizeInBytes.0.valueInStandard",
		"size_in_bytes.value_in_standard", "size_in_bytes.0.value_in_standard",
	}
	efsBytesIAKeys = []string{
		"sizeInBytes.valueInIa", "sizeInBytes.0.valueInIa",
		"size_in_bytes.value_in_ia", "size_in_bytes.0.value_in_ia",
	}
	efsBytesTotalKeys = []string{
		"sizeInBytes.value", "sizeInBytes.0.value",
		"size_in_bytes.value", "size_in_bytes.0.value",
		"sizeInBytes", "size_in_bytes",
	}
)

// efsInputs holds the resolved usage and configuration of one file system.
type efsInputs struct {
	oneZone          bool
	standardGB       float64
	iaGB             float64
	archiveGB        float64
	archiveIgnoredGB float64
	mode             string
	provisionedMiBps float64
	elasticReadGB    float64
	elasticWriteGB   float64
	iaReadGB         float64
	iaWriteGB        float64
	archiveReadGB    float64
	archiveWriteGB   float64
}

// efsStorageClass normalizes a SKU or storage_class value. It returns the
// class (standard, ia, archive) and whether it names a One Zone class.
func efsStorageClass(raw string) (string, bool, bool) {
	key := strings.NewReplacer(" ", "", "_", "", "-", "").Replace(strings.ToLower(raw))
	switch key {
	case "", efsAliasStandard, efsAliasGeneral:
		return efsClassStandard, false, true
	case "ia", "infrequentaccess":
		return efsClassIA, false, true
	case efsClassArchive:
		return efsClassArchive, false, true
	case efsAliasOneZone, "onezonestandard", "onezonegeneralpurpose":
		return efsClassStandard, true, true
	case efsAliasOneZoneIA, "onezoneinfrequentaccess":
		return efsClassIA, true, true
	}
	return "", false, false
}

// efsTagNumber reads a non-negative numeric input. An empty value is absent.
func (p *AWSPublicPlugin) efsTagNumber(traceID string, tags map[string]string, name string) (float64, bool) {
	raw, ok := tags[name]
	if !ok || raw == "" {
		return 0, false
	}
	return p.validateNonNegativeFloat64(traceID, name, raw), true
}

// efsBytesGB reads the first present size_in_bytes key and converts bytes to GB.
func efsBytesGB(in resourceInputs, keys []string) (float64, bool) {
	v, ok := in.Float(keys...)
	if !ok || v < 0 {
		return 0, false
	}
	return v / bytesPerGB, true
}

// resolveEFSInputs reads attributes then tags and records every default.
func (p *AWSPublicPlugin) resolveEFSInputs(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (efsInputs, *DefaultsTracker, error) {
	tags := pricingTags(resource)
	var dt DefaultsTracker
	var out efsInputs

	classRaw := resource.GetSku()
	if classRaw == "" {
		classRaw = tags["storage_class"]
	}
	class, classOneZone, ok := efsStorageClass(classRaw)
	if !ok {
		return out, nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			fmt.Sprintf("unsupported EFS storage class %q", classRaw),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}
	out.oneZone = classOneZone || tags["availability_zone_name"] != ""

	p.resolveEFSStorage(traceID, resource, tags, class, &out, &dt)

	mode := strings.ToLower(strings.TrimSpace(tags["throughput_mode"]))
	switch mode {
	case "":
		mode = efsModeBursting
		dt.Add("throughput_mode", efsModeBursting, KindConfig)
	case efsModeBursting, efsModeProvisioned, efsModeElastic:
	default:
		return out, nil, p.newErrorWithID(
			traceID,
			codes.InvalidArgument,
			fmt.Sprintf("unsupported EFS throughput_mode %q: use bursting, provisioned, or elastic", mode),
			pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
		)
	}
	out.mode = mode

	usage := func(name string, dst *float64, tracked bool) {
		if v, found := p.efsTagNumber(traceID, tags, name); found {
			*dst = v
		} else if tracked {
			dt.Add(name, "0", KindUsageZero)
		}
	}
	usage("provisioned_throughput_in_mibps", &out.provisionedMiBps, mode == efsModeProvisioned)
	usage("elastic_read_gb", &out.elasticReadGB, mode == efsModeElastic)
	usage("elastic_write_gb", &out.elasticWriteGB, mode == efsModeElastic)
	usage("ia_read_gb", &out.iaReadGB, false)
	usage("ia_write_gb", &out.iaWriteGB, false)
	usage("archive_read_gb", &out.archiveReadGB, false)
	usage("archive_write_gb", &out.archiveWriteGB, false)

	if mode != efsModeProvisioned {
		out.provisionedMiBps = 0
	}
	if mode != efsModeElastic {
		out.elasticReadGB, out.elasticWriteGB = 0, 0
	}
	return out, &dt, nil
}

// resolveEFSStorage fills the per-class GB. storage_gb belongs to the SKU
// class. When it is absent a Standard file system falls back to size_in_bytes,
// and a total without a split is priced as Standard after removing the IA part.
func (p *AWSPublicPlugin) resolveEFSStorage(
	traceID string,
	resource *pbc.ResourceDescriptor,
	tags map[string]string,
	class string,
	out *efsInputs,
	dt *DefaultsTracker,
) {
	gb, hasGB := p.efsTagNumber(traceID, tags, "storage_gb")
	iaGB, hasIA := p.efsTagNumber(traceID, tags, "ia_storage_gb")
	archiveGB, hasArchive := p.efsTagNumber(traceID, tags, "archive_storage_gb")

	if hasIA {
		out.iaGB = iaGB
	}
	if hasArchive {
		out.archiveGB = archiveGB
	}

	known := hasGB || hasIA || hasArchive
	if hasGB {
		switch class {
		case efsClassIA:
			out.iaGB += gb
		case efsClassArchive:
			out.archiveGB += gb
		default:
			out.standardGB += gb
		}
	} else if class == efsClassStandard {
		in := newResourceInputs(resource)
		stdBytes, hasStd := efsBytesGB(in, efsBytesStandardKeys)
		iaBytes, hasIABytes := efsBytesGB(in, efsBytesIAKeys)
		total, hasTotal := efsBytesGB(in, efsBytesTotalKeys)
		switch {
		case hasStd:
			out.standardGB = stdBytes
		case hasTotal:
			out.standardGB = math.Max(0, total-iaBytes)
		}
		if hasIABytes && !hasIA {
			out.iaGB = iaBytes
		}
		known = known || hasStd || hasTotal || hasIABytes
	}

	if !known {
		switch class {
		case efsClassIA:
			out.iaGB = 1
		case efsClassArchive:
			out.archiveGB = 1
		default:
			out.standardGB = 1
		}
		dt.Add("storage_gb", "1", KindConfig)
	}

	if out.oneZone && out.archiveGB > 0 {
		out.archiveIgnoredGB = out.archiveGB
		out.archiveGB = 0
	}
}

// efsRates picks the storage rates that apply to the file system.
func efsRates(rates *pricing.EFSPrice, in efsInputs) (float64, float64) {
	if in.oneZone {
		return rates.OneZoneGBMonth, rates.OneZoneIAGBMonth
	}
	iaRate := rates.IAGBMonth
	if in.mode == efsModeElastic && rates.IAElasticGBMonth > 0 {
		iaRate = rates.IAElasticGBMonth
	}
	return rates.StandardGBMonth, iaRate
}

// efsMissingRate returns the first storage class that has GB but no rate.
func efsMissingRate(in efsInputs, std, ia, archive float64) string {
	switch {
	case in.standardGB > 0 && std == 0:
		return "Standard"
	case in.iaGB > 0 && ia == 0:
		return "Infrequent Access"
	case in.archiveGB > 0 && archive == 0:
		return "Archive"
	}
	return ""
}

// efsMissingAccessRate returns the first throughput or data access charge that
// has positive usage but no rate.
func efsMissingAccessRate(in efsInputs, rates *pricing.EFSPrice) string {
	checks := []struct {
		name  string
		usage float64
		rate  float64
	}{
		{"Elastic read", in.elasticReadGB, rates.ElasticReadGB},
		{"Elastic write", in.elasticWriteGB, rates.ElasticWriteGB},
		{"IA read", in.iaReadGB, rates.IAReadGB},
		{"IA write", in.iaWriteGB, rates.IAWriteGB},
		{"Archive read", in.archiveReadGB, rates.ArchiveReadGB},
		{"Archive write", in.archiveWriteGB, rates.ArchiveWriteGB},
	}
	for _, c := range checks {
		if c.usage > 0 && c.rate == 0 {
			return c.name
		}
	}
	return ""
}

// efsBillableMiBps returns provisioned throughput above the included baseline.
func efsBillableMiBps(in efsInputs) (float64, float64) {
	baseline := in.standardGB * efsBaselineMiBpsPerGB
	return math.Max(0, in.provisionedMiBps-baseline), baseline
}

// efsBillingDetail describes the storage, throughput, and ignored inputs.
func efsBillingDetail(in efsInputs, billable, baseline float64, dt *DefaultsTracker) string {
	scope := "Regional"
	if in.oneZone {
		scope = "One Zone"
	}
	parts := []string{fmt.Sprintf("%gGB Standard", in.standardGB)}
	if in.iaGB > 0 {
		parts = append(parts, fmt.Sprintf("%gGB IA", in.iaGB))
	}
	if in.archiveGB > 0 {
		parts = append(parts, fmt.Sprintf("%gGB Archive", in.archiveGB))
	}
	switch in.mode {
	case efsModeProvisioned:
		parts = append(parts, fmt.Sprintf(
			"provisioned throughput %g MiB/s (%.2f MiB/s billable above %.2f MiB/s baseline)",
			in.provisionedMiBps, billable, baseline,
		))
	case efsModeElastic:
		parts = append(parts, fmt.Sprintf(
			"elastic throughput (%gGB read, %gGB write)", in.elasticReadGB, in.elasticWriteGB,
		))
	default:
		parts = append(parts, "bursting throughput")
	}
	if in.iaReadGB > 0 || in.iaWriteGB > 0 {
		parts = append(parts, fmt.Sprintf("IA access %gGB read, %gGB write", in.iaReadGB, in.iaWriteGB))
	}
	if in.archiveReadGB > 0 || in.archiveWriteGB > 0 {
		parts = append(parts, fmt.Sprintf("Archive access %gGB read, %gGB write", in.archiveReadGB, in.archiveWriteGB))
	}
	detail := fmt.Sprintf("EFS %s: %s", scope, strings.Join(parts, ", "))
	if in.archiveIgnoredGB > 0 {
		detail += fmt.Sprintf(" (%gGB Archive ignored: One Zone has no Archive class)", in.archiveIgnoredGB)
	}
	if dt != nil && dt.Metadata() != nil {
		detail += fmt.Sprintf(" (defaulted: %s)", dt.Metadata()[metadataKeyDefaultsApplied])
	}
	return detail
}

// estimateEFS calculates projected monthly cost for an EFS file system.
//
// The estimate covers storage per class, provisioned throughput above the
// Bursting baseline, Elastic Throughput read and write GB, and optional IA and
// Archive data access GB. Backups, replication, data transfer, and the
// small-file overhead SKUs are excluded. The IA rate with Elastic Throughput
// comes from the Price List class "Infrequent Access-ET".
func (p *AWSPublicPlugin) estimateEFS(
	traceID string,
	resource *pbc.ResourceDescriptor,
) (*pbc.GetProjectedCostResponse, error) {
	rates, found := p.pricing.EFSPrice()
	if !found {
		return nil, &PricingUnavailableError{
			Service:       efsServiceName,
			SKU:           p.region,
			BillingDetail: fmt.Sprintf(PricingUnavailableTemplate, efsServiceName, p.region),
		}
	}

	in, dt, err := p.resolveEFSInputs(traceID, resource)
	if err != nil {
		return nil, err
	}

	stdRate, iaRate := efsRates(rates, in)
	if missing := efsMissingRate(in, stdRate, iaRate, rates.ArchiveGBMonth); missing != "" {
		return nil, &PricingUnavailableError{
			Service:       efsServiceName,
			SKU:           missing,
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "EFS storage class", missing),
		}
	}

	billable, baseline := efsBillableMiBps(in)
	if billable > 0 && rates.ProvisionedMiBpsMonth == 0 {
		return nil, &PricingUnavailableError{
			Service:       efsServiceName,
			SKU:           efsModeProvisioned,
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "EFS throughput mode", efsModeProvisioned),
		}
	}

	if missing := efsMissingAccessRate(in, rates); missing != "" {
		return nil, &PricingUnavailableError{
			Service:       efsServiceName,
			SKU:           missing,
			BillingDetail: fmt.Sprintf(PricingNotFoundTemplate, "EFS access rate", missing),
		}
	}

	storage := in.standardGB*stdRate + in.iaGB*iaRate + in.archiveGB*rates.ArchiveGBMonth
	throughput := billable*rates.ProvisionedMiBpsMonth +
		in.elasticReadGB*rates.ElasticReadGB + in.elasticWriteGB*rates.ElasticWriteGB
	access := in.iaReadGB*rates.IAReadGB + in.iaWriteGB*rates.IAWriteGB +
		in.archiveReadGB*rates.ArchiveReadGB + in.archiveWriteGB*rates.ArchiveWriteGB
	total := storage + throughput + access

	p.traceLogger(traceID, "GetProjectedCost").Debug().
		Bool("one_zone", in.oneZone).
		Str("throughput_mode", in.mode).
		Float64("storage_cost", storage).
		Float64("throughput_cost", throughput).
		Float64("access_cost", access).
		Msg("EFS cost estimated")

	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth:  total,
		UnitPrice:     stdRate,
		Currency:      currencyUSD,
		BillingDetail: efsBillingDetail(in, billable, baseline, dt),
		Metadata:      dt.Metadata(),
	}

	carbonGrams, carbonOK := carbon.NewEFSEstimator().EstimateCarbonGrams(carbon.EFSStorageConfig{
		StandardGB: in.standardGB,
		IAGB:       in.iaGB,
		ArchiveGB:  in.archiveGB,
		OneZone:    in.oneZone,
		Region:     resource.GetRegion(),
		Hours:      HoursPerMonthProd,
	})
	if carbonOK {
		resp.ImpactMetrics = []*pbc.ImpactMetric{
			{
				Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
				Value: carbonGrams,
				Unit:  carbonUnitGCO2e,
			},
		}
	}

	setGrowthHint(p.logger.With().Str(pluginsdk.FieldTraceID, traceID).Logger(), "aws:efs:filesystem", resp)

	return resp, nil
}

// efsPricingSpec returns the pricing specification for EFS Standard storage.
func (p *AWSPublicPlugin) efsPricingSpec(resource *pbc.ResourceDescriptor) *pbc.PricingSpec {
	spec := &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingModePerGBMonth,
		Currency:     currencyUSD,
		Unit:         unitGBMonth,
		Source:       sourceAWSPublic,
	}

	rates, found := p.pricing.EFSPrice()
	if !found {
		spec.Description = fmt.Sprintf(PricingUnavailableTemplate, efsServiceName, p.region)
		spec.Assumptions = []string{"EFS pricing data not available"}
		return spec
	}

	tags := pricingTags(resource)
	classRaw := resource.GetSku()
	if classRaw == "" {
		classRaw = tags["storage_class"]
	}
	class, oneZone, ok := efsStorageClass(classRaw)
	if !ok {
		spec.Description = fmt.Sprintf(PricingNotFoundTemplate, "EFS storage class", classRaw)
		spec.Assumptions = []string{"Storage class not recognized"}
		return spec
	}
	oneZone = oneZone || tags["availability_zone_name"] != ""

	rate, label := rates.StandardGBMonth, "Standard"
	switch {
	case class == efsClassIA && oneZone:
		rate, label = rates.OneZoneIAGBMonth, "One Zone Infrequent Access"
	case class == efsClassIA:
		rate, label = rates.IAGBMonth, "Infrequent Access"
	case class == efsClassArchive:
		rate, label = rates.ArchiveGBMonth, "Archive"
	case oneZone:
		rate, label = rates.OneZoneGBMonth, "One Zone Standard"
	}

	spec.RatePerUnit = rate
	spec.Description = fmt.Sprintf("EFS %s storage", label)
	spec.Assumptions = []string{
		"Storage rate per GB-month for the storage class",
		fmt.Sprintf(
			"Provisioned throughput: $%.4f per MiB/s-month above the Standard storage baseline",
			rates.ProvisionedMiBpsMonth,
		),
		fmt.Sprintf(
			"Elastic throughput: $%.4f per GB read, $%.4f per GB written",
			rates.ElasticReadGB,
			rates.ElasticWriteGB,
		),
		"Backups, replication, and data transfer billed separately",
	}
	return spec
}
