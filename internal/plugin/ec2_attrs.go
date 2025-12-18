package plugin

import (
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// EC2Attributes contains extracted EC2 configuration for pricing lookups.
// OS is normalized to "Linux" or "Windows".
// Tenancy is normalized to "Shared", "Dedicated", or "Host".
type EC2Attributes struct {
	OS      string // "Linux" or "Windows"
	Tenancy string // "Shared", "Dedicated", or "Host"
}

// DefaultEC2Attributes returns EC2 attributes with default values.
// Default OS is "Linux" and default Tenancy is "Shared".
func DefaultEC2Attributes() EC2Attributes {
	return EC2Attributes{
		OS:      "Linux",
		Tenancy: "Shared",
	}
}

// ExtractEC2AttributesFromTags extracts and normalizes EC2 attributes from a
// ResourceDescriptor.Tags map. Returns default values for missing or invalid fields.
//
// Platform normalization:
//   - "windows" (case-insensitive) → "Windows"
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
//   - Any other value or missing → "Linux"
//
// Tenancy normalization:
//   - "dedicated" (case-insensitive) → "Dedicated"
//   - "host" (case-insensitive) → "Host"
//   - Any other value or missing → "Shared"
func ExtractEC2AttributesFromStruct(attrs *structpb.Struct) EC2Attributes {
	result := DefaultEC2Attributes()

	if attrs == nil || attrs.Fields == nil {
		return result
	}

	// Extract OS from platform attribute
	if val, ok := attrs.Fields["platform"]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			result.OS = normalizePlatform(strVal)
		}
	}

	// Extract tenancy from tenancy attribute
	if val, ok := attrs.Fields["tenancy"]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			result.Tenancy = normalizeTenancy(strVal)
		}
	}

	return result
}

// normalizePlatform normalizes a platform string to "Linux" or "Windows".
// Only "windows" (case-insensitive) maps to "Windows"; all others map to "Linux".
func normalizePlatform(platform string) string {
	if strings.EqualFold(platform, "windows") {
		return "Windows"
	}
	return "Linux"
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
		return "Shared"
	}
}
