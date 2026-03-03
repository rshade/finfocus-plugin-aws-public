package plugin

import (
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	defaultRootVolumeType   = "gp2"
	defaultRootVolumeSizeGB = 8
)

// EC2Attributes contains extracted EC2 configuration for pricing lookups.
// OS is normalized to "Linux", "Windows", "RHEL", or "SUSE".
// Tenancy is normalized to "Shared", "Dedicated", or "Host".
type EC2Attributes struct {
	OS      string // "Linux", "Windows", "RHEL", or "SUSE"
	Tenancy string // "Shared", "Dedicated", or "Host"
}

// DefaultEC2Attributes returns EC2 attributes with default values.
// Default OS is "Linux" and default Tenancy is "Shared".
func DefaultEC2Attributes() EC2Attributes {
	return EC2Attributes{
		OS:      defaultOS,
		Tenancy: defaultTenancy,
	}
}

// ExtractEC2AttributesFromTags extracts and normalizes EC2 attributes from a
// ResourceDescriptor.Tags map. Returns default values for missing or invalid fields.
// This function serves as the definitive internal source of truth for mapping
// user-provided tags to AWS pricing identifiers (FR-009).
//
// Platform normalization:
//   - "windows" (case-insensitive) → "Windows"
//   - "rhel", "redhat", "red hat" (case-insensitive) → "RHEL"
//   - "suse" (case-insensitive) → "SUSE"
//   - Any other value or missing → "Linux"
//
// Tenancy normalization:
//   - "dedicated" (case-insensitive) → "Dedicated"
//   - "host" (case-insensitive) → "Host"
//   - Any other value or missing → "Shared"
func ExtractEC2AttributesFromTags(tags map[string]string) EC2Attributes {
	attrs := DefaultEC2Attributes()

	if tags == nil {
		return attrs
	}

	// Extract OS from platform tag
	if platform, ok := tags["platform"]; ok && platform != "" {
		attrs.OS = normalizePlatform(platform)
	}

	// Extract tenancy from tenancy tag
	if tenancy, ok := tags["tenancy"]; ok && tenancy != "" {
		attrs.Tenancy = normalizeTenancy(tenancy)
	}

	return attrs
}

// ExtractEC2AttributesFromStruct extracts and normalizes EC2 attributes from a
// protobuf Struct (used in EstimateCost path). Returns default values for missing
// or invalid fields.
//
// Platform normalization:
//   - "windows" (case-insensitive) → "Windows"
//   - "rhel", "redhat", "red hat" (case-insensitive) → "RHEL"
//   - "suse" (case-insensitive) → "SUSE"
//   - Any other value or missing → "Linux"
//
// Tenancy normalization:
//   - "dedicated" (case-insensitive) → "Dedicated"
//   - "host" (case-insensitive) → "Host"
//   - Any other value or missing → "Shared"
func ExtractEC2AttributesFromStruct(attrs *structpb.Struct) EC2Attributes {
	result := DefaultEC2Attributes()

	if attrs == nil || attrs.GetFields() == nil {
		return result
	}

	// Extract OS from platform attribute
	if val, ok := attrs.GetFields()["platform"]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			result.OS = normalizePlatform(strVal)
		}
	}

	// Extract tenancy from tenancy attribute
	if val, ok := attrs.GetFields()["tenancy"]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			result.Tenancy = normalizeTenancy(strVal)
		}
	}

	return result
}

// normalizePlatform normalizes a platform string to canonical AWS pricing identifiers.
// - "windows" -> "Windows"
// - "rhel" -> "RHEL"
// - "suse" -> "SUSE"
// - All others -> "Linux".
func normalizePlatform(platform string) string {
	p := strings.ToLower(platform)
	switch {
	case strings.Contains(p, "windows"):
		return "Windows"
	case strings.Contains(p, "rhel") || strings.Contains(p, "redhat") || strings.Contains(p, "red hat"):
		return "RHEL"
	case strings.Contains(p, "suse"):
		return "SUSE"
	default:
		return defaultOS
	}
}

// normalizeTenancy normalizes a tenancy string to "Shared", "Dedicated", or "Host".
// "dedicated" and "host" (case-insensitive) map to their canonical forms;
// all others map to "Shared".
func normalizeTenancy(tenancy string) string {
	switch strings.ToLower(tenancy) {
	case "dedicated":
		return "Dedicated"
	case "host":
		return "Host"
	default:
		return defaultTenancy
	}
}

// RootVolumeInfo contains extracted root EBS volume configuration for an EC2 instance.
// When Present is true, the root volume cost should be included in EC2 cost estimates.
type RootVolumeInfo struct {
	VolumeType string // EBS volume type (e.g., "gp2", "gp3")
	SizeGB     int    // Volume size in GB
	Present    bool   // Whether root volume info was found in tags
}

// parseGoMapString parses Go's fmt.Sprint map format ("map[key1:val1 key2:val2]")
// into a map[string]string. This is used to parse the rootBlockDevice tag that Pulumi
// serializes as a Go map string representation.
//
// For each space-separated token, the first colon splits key from value.
// Values containing colons are preserved (e.g., "arn:aws:..." keeps the full ARN).
// Returns an empty map for empty or malformed input.
func parseGoMapString(s string) map[string]string {
	result := make(map[string]string)

	// Strip "map[" prefix and "]" suffix
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "map[") || !strings.HasSuffix(s, "]") {
		return result
	}
	s = s[4 : len(s)-1] // Remove "map[" and "]"
	if s == "" {
		return result
	}

	// Split by space, then split each token on first ":"
	tokens := strings.Fields(s)
	for _, token := range tokens {
		idx := strings.Index(token, ":")
		if idx < 0 {
			continue // No colon found, skip malformed token
		}
		key := token[:idx]
		value := token[idx+1:]
		if key != "" {
			result[key] = value
		}
	}

	return result
}

// parsePositiveIntField parses positive integer tag values used for root volume size fields.
// It logs a warning for malformed or non-positive values.
func parsePositiveIntField(fieldName, raw string) (int, bool) {
	size, err := strconv.Atoi(raw)
	if err != nil || size <= 0 {
		log.Warn().
			Str("field", fieldName).
			Str("value", raw).
			Err(err).
			Msg("invalid root volume size value")
		return 0, false
	}
	return size, true
}

// ExtractRootVolumeFromTags extracts root EBS volume configuration from a
// ResourceDescriptor.Tags map. This is used by the GetProjectedCost path.
//
// Tag priority:
//  1. Individual tags "root_volume_type" and "root_volume_size" (explicit overrides)
//  2. Parsed "rootBlockDevice" tag via parseGoMapString (Go map format from Pulumi)
//  3. Individual tags override map-parsed values when both are present
//
// Default behavior:
//   - If no root volume source found → RootVolumeInfo{Present: false} (no cost added)
//   - If source present but missing type → defaults to "gp2"
//   - If source present but missing/invalid size → defaults to 8 GB
func ExtractRootVolumeFromTags(tags map[string]string) RootVolumeInfo {
	if tags == nil {
		return RootVolumeInfo{}
	}

	var volumeType string
	var sizeGB int
	var hasSource bool

	// Check for rootBlockDevice map tag first (lower priority, overridden by individual tags)
	if rbdStr, ok := tags["rootBlockDevice"]; ok && rbdStr != "" {
		rbdMap := parseGoMapString(rbdStr)
		if len(rbdMap) > 0 {
			hasSource = true
			if vt, vtOK := rbdMap["volumeType"]; vtOK && vt != "" {
				volumeType = vt
			}
			if vs, vsOK := rbdMap["volumeSize"]; vsOK && vs != "" {
				if size, sizeOK := parsePositiveIntField("rootBlockDevice.volumeSize", vs); sizeOK {
					sizeGB = size
				}
			}
		}
	}

	// Individual tags override map-parsed values (higher priority)
	if rvt, rvtOK := tags["root_volume_type"]; rvtOK && rvt != "" {
		hasSource = true
		volumeType = rvt
	}
	if rvs, rvsOK := tags["root_volume_size"]; rvsOK && rvs != "" {
		hasSource = true
		if size, sizeOK := parsePositiveIntField("root_volume_size", rvs); sizeOK {
			sizeGB = size
		}
	}

	if !hasSource {
		return RootVolumeInfo{}
	}

	// Apply defaults for present-but-missing fields
	if volumeType == "" {
		volumeType = defaultRootVolumeType
	}
	if sizeGB <= 0 {
		sizeGB = defaultRootVolumeSizeGB
	}

	return RootVolumeInfo{
		VolumeType: volumeType,
		SizeGB:     sizeGB,
		Present:    true,
	}
}

// ExtractRootVolumeFromStruct extracts root EBS volume configuration from a
// protobuf Struct (used in the EstimateCost path). Checks the "rootBlockDevice"
// attribute for volume type and size.
//
// Default behavior matches ExtractRootVolumeFromTags:
//   - If no rootBlockDevice attribute → RootVolumeInfo{Present: false}
//   - If present but missing type → defaults to "gp2"
//   - If present but missing/invalid size → defaults to 8 GB
func ExtractRootVolumeFromStruct(attrs *structpb.Struct) RootVolumeInfo {
	if attrs == nil || attrs.GetFields() == nil {
		return RootVolumeInfo{}
	}

	rbdVal, ok := attrs.GetFields()["rootBlockDevice"]
	if !ok {
		return RootVolumeInfo{}
	}

	// rootBlockDevice can be a struct or a list of structs
	var volumeType string
	var sizeGB int

	switch v := rbdVal.GetKind().(type) {
	case *structpb.Value_StructValue:
		volumeType, sizeGB = extractRootVolumeFromStructValue(v.StructValue)
	case *structpb.Value_ListValue:
		// Take the first element if it's a list
		if v.ListValue != nil && len(v.ListValue.GetValues()) > 0 {
			if sv := v.ListValue.GetValues()[0].GetStructValue(); sv != nil {
				volumeType, sizeGB = extractRootVolumeFromStructValue(sv)
			}
		}
	case *structpb.Value_StringValue:
		// Handle Go map string format (e.g., "map[volumeSize:8 volumeType:gp2]")
		rbdMap := parseGoMapString(v.StringValue)
		if len(rbdMap) == 0 {
			return RootVolumeInfo{}
		}
		if vt, vtOK := rbdMap["volumeType"]; vtOK && vt != "" {
			volumeType = vt
		}
		if vs, vsOK := rbdMap["volumeSize"]; vsOK && vs != "" {
			if size, sizeOK := parsePositiveIntField("rootBlockDevice.volumeSize", vs); sizeOK {
				sizeGB = size
			}
		}
	default:
		return RootVolumeInfo{}
	}

	// Apply defaults
	if volumeType == "" {
		volumeType = defaultRootVolumeType
	}
	if sizeGB <= 0 {
		sizeGB = defaultRootVolumeSizeGB
	}

	return RootVolumeInfo{
		VolumeType: volumeType,
		SizeGB:     sizeGB,
		Present:    true,
	}
}

// extractRootVolumeFromStructValue extracts volume type and size from a structpb.Struct.
func extractRootVolumeFromStructValue(s *structpb.Struct) (string, int) {
	if s == nil {
		return "", 0
	}

	var volumeType string
	var sizeGB int

	if val, vtOK := s.GetFields()["volumeType"]; vtOK {
		volumeType = val.GetStringValue()
	}

	if val, vsOK := s.GetFields()["volumeSize"]; vsOK {
		switch sv := val.GetKind().(type) {
		case *structpb.Value_NumberValue:
			if sv.NumberValue > 0 {
				sizeGB = int(sv.NumberValue)
			}
		case *structpb.Value_StringValue:
			if size, sizeOK := parsePositiveIntField("rootBlockDevice.volumeSize", sv.StringValue); sizeOK {
				sizeGB = size
			}
		}
	}

	return volumeType, sizeGB
}
