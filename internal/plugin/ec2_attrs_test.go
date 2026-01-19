package plugin

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// TestDefaultEC2Attributes verifies the default values returned by DefaultEC2Attributes.
func TestDefaultEC2Attributes(t *testing.T) {
	attrs := DefaultEC2Attributes()

	if attrs.OS != "Linux" {
		t.Errorf("DefaultEC2Attributes().OS = %q, want %q", attrs.OS, "Linux")
	}
	if attrs.Tenancy != "Shared" {
		t.Errorf("DefaultEC2Attributes().Tenancy = %q, want %q", attrs.Tenancy, "Shared")
	}
}

// TestExtractEC2AttributesFromTags_PlatformNormalization tests platform/OS normalization
// from tags with various case combinations.
func TestExtractEC2AttributesFromTags_PlatformNormalization(t *testing.T) {
	tests := []struct {
		name   string
		tags   map[string]string
		wantOS string
	}{
		{
			name:   "windows lowercase",
			tags:   map[string]string{"platform": "windows"},
			wantOS: "Windows",
		},
		{
			name:   "windows uppercase",
			tags:   map[string]string{"platform": "WINDOWS"},
			wantOS: "Windows",
		},
		{
			name:   "windows mixed case",
			tags:   map[string]string{"platform": "Windows"},
			wantOS: "Windows",
		},
		{
			name:   "linux lowercase",
			tags:   map[string]string{"platform": "linux"},
			wantOS: "Linux",
		},
		{
			name:   "linux uppercase",
			tags:   map[string]string{"platform": "LINUX"},
			wantOS: "Linux",
		},
		{
			name:   "amazon linux",
			tags:   map[string]string{"platform": "amazon-linux"},
			wantOS: "Linux",
		},
		{
			name:   "rhel",
			tags:   map[string]string{"platform": "rhel"},
			wantOS: "RHEL",
		},
		{
			name:   "suse",
			tags:   map[string]string{"platform": "suse"},
			wantOS: "SUSE",
		},
		{
			name:   "windows server 2019",
			tags:   map[string]string{"platform": "Windows Server 2019"},
			wantOS: "Windows",
		},
		{
			name:   "rhel-8",
			tags:   map[string]string{"platform": "RHEL-8"},
			wantOS: "RHEL",
		},
		{
			name:   "red hat enterprise linux",
			tags:   map[string]string{"platform": "Red Hat Enterprise Linux"},
			wantOS: "RHEL",
		},
		{
			name:   "suse linux enterprise server",
			tags:   map[string]string{"platform": "SUSE Linux Enterprise Server"},
			wantOS: "SUSE",
		},
		{
			name:   "empty platform",
			tags:   map[string]string{"platform": ""},
			wantOS: "Linux",
		},
		{
			name:   "missing platform",
			tags:   map[string]string{},
			wantOS: "Linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ExtractEC2AttributesFromTags(tt.tags)
			if attrs.OS != tt.wantOS {
				t.Errorf("ExtractEC2AttributesFromTags(%v).OS = %q, want %q", tt.tags, attrs.OS, tt.wantOS)
			}
		})
	}
}

// TestExtractEC2AttributesFromTags_TenancyNormalization tests tenancy normalization
// from tags with various case combinations and values.
func TestExtractEC2AttributesFromTags_TenancyNormalization(t *testing.T) {
	tests := []struct {
		name        string
		tags        map[string]string
		wantTenancy string
	}{
		{
			name:        "dedicated lowercase",
			tags:        map[string]string{"tenancy": "dedicated"},
			wantTenancy: "Dedicated",
		},
		{
			name:        "dedicated uppercase",
			tags:        map[string]string{"tenancy": "DEDICATED"},
			wantTenancy: "Dedicated",
		},
		{
			name:        "dedicated mixed case",
			tags:        map[string]string{"tenancy": "Dedicated"},
			wantTenancy: "Dedicated",
		},
		{
			name:        "host lowercase",
			tags:        map[string]string{"tenancy": "host"},
			wantTenancy: "Host",
		},
		{
			name:        "host uppercase",
			tags:        map[string]string{"tenancy": "HOST"},
			wantTenancy: "Host",
		},
		{
			name:        "host mixed case",
			tags:        map[string]string{"tenancy": "Host"},
			wantTenancy: "Host",
		},
		{
			name:        "shared lowercase",
			tags:        map[string]string{"tenancy": "shared"},
			wantTenancy: "Shared",
		},
		{
			name:        "default tenancy",
			tags:        map[string]string{"tenancy": "default"},
			wantTenancy: "Shared",
		},
		{
			name:        "empty tenancy",
			tags:        map[string]string{"tenancy": ""},
			wantTenancy: "Shared",
		},
		{
			name:        "missing tenancy",
			tags:        map[string]string{},
			wantTenancy: "Shared",
		},
		{
			name:        "unknown tenancy value",
			tags:        map[string]string{"tenancy": "unknown"},
			wantTenancy: "Shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ExtractEC2AttributesFromTags(tt.tags)
			if attrs.Tenancy != tt.wantTenancy {
				t.Errorf("ExtractEC2AttributesFromTags(%v).Tenancy = %q, want %q", tt.tags, attrs.Tenancy, tt.wantTenancy)
			}
		})
	}
}

// TestExtractEC2AttributesFromTags_NilAndEmpty tests handling of nil and empty input.
func TestExtractEC2AttributesFromTags_NilAndEmpty(t *testing.T) {
	tests := []struct {
		name        string
		tags        map[string]string
		wantOS      string
		wantTenancy string
	}{
		{
			name:        "nil tags",
			tags:        nil,
			wantOS:      "Linux",
			wantTenancy: "Shared",
		},
		{
			name:        "empty tags",
			tags:        map[string]string{},
			wantOS:      "Linux",
			wantTenancy: "Shared",
		},
		{
			name:        "unrelated tags only",
			tags:        map[string]string{"Name": "test", "Environment": "prod"},
			wantOS:      "Linux",
			wantTenancy: "Shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ExtractEC2AttributesFromTags(tt.tags)
			if attrs.OS != tt.wantOS {
				t.Errorf("ExtractEC2AttributesFromTags(%v).OS = %q, want %q", tt.tags, attrs.OS, tt.wantOS)
			}
			if attrs.Tenancy != tt.wantTenancy {
				t.Errorf("ExtractEC2AttributesFromTags(%v).Tenancy = %q, want %q", tt.tags, attrs.Tenancy, tt.wantTenancy)
			}
		})
	}
}

// TestExtractEC2AttributesFromTags_Combined tests combined platform and tenancy extraction.
func TestExtractEC2AttributesFromTags_Combined(t *testing.T) {
	tests := []struct {
		name        string
		tags        map[string]string
		wantOS      string
		wantTenancy string
	}{
		{
			name:        "windows dedicated",
			tags:        map[string]string{"platform": "windows", "tenancy": "dedicated"},
			wantOS:      "Windows",
			wantTenancy: "Dedicated",
		},
		{
			name:        "linux host",
			tags:        map[string]string{"platform": "linux", "tenancy": "host"},
			wantOS:      "Linux",
			wantTenancy: "Host",
		},
		{
			name:        "windows shared",
			tags:        map[string]string{"platform": "WINDOWS", "tenancy": "shared"},
			wantOS:      "Windows",
			wantTenancy: "Shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ExtractEC2AttributesFromTags(tt.tags)
			if attrs.OS != tt.wantOS {
				t.Errorf("ExtractEC2AttributesFromTags(%v).OS = %q, want %q", tt.tags, attrs.OS, tt.wantOS)
			}
			if attrs.Tenancy != tt.wantTenancy {
				t.Errorf("ExtractEC2AttributesFromTags(%v).Tenancy = %q, want %q", tt.tags, attrs.Tenancy, tt.wantTenancy)
			}
		})
	}
}

// TestExtractEC2AttributesFromStruct_PlatformNormalization tests platform normalization
// from protobuf Struct attributes.
func TestExtractEC2AttributesFromStruct_PlatformNormalization(t *testing.T) {
	tests := []struct {
		name   string
		attrs  *structpb.Struct
		wantOS string
	}{
		{
			name:   "windows lowercase",
			attrs:  mustStruct(map[string]interface{}{"platform": "windows"}),
			wantOS: "Windows",
		},
		{
			name:   "windows uppercase",
			attrs:  mustStruct(map[string]interface{}{"platform": "WINDOWS"}),
			wantOS: "Windows",
		},
		{
			name:   "linux lowercase",
			attrs:  mustStruct(map[string]interface{}{"platform": "linux"}),
			wantOS: "Linux",
		},
		{
			name:   "rhel lowercase",
			attrs:  mustStruct(map[string]interface{}{"platform": "rhel"}),
			wantOS: "RHEL",
		},
		{
			name:   "rhel with version",
			attrs:  mustStruct(map[string]interface{}{"platform": "RHEL-8"}),
			wantOS: "RHEL",
		},
		{
			name:   "red hat enterprise linux",
			attrs:  mustStruct(map[string]interface{}{"platform": "Red Hat Enterprise Linux"}),
			wantOS: "RHEL",
		},
		{
			name:   "suse lowercase",
			attrs:  mustStruct(map[string]interface{}{"platform": "suse"}),
			wantOS: "SUSE",
		},
		{
			name:   "suse linux enterprise server",
			attrs:  mustStruct(map[string]interface{}{"platform": "SUSE Linux Enterprise Server"}),
			wantOS: "SUSE",
		},
		{
			name:   "empty platform",
			attrs:  mustStruct(map[string]interface{}{"platform": ""}),
			wantOS: "Linux",
		},
		{
			name:   "missing platform",
			attrs:  mustStruct(map[string]interface{}{}),
			wantOS: "Linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ExtractEC2AttributesFromStruct(tt.attrs)
			if attrs.OS != tt.wantOS {
				t.Errorf("ExtractEC2AttributesFromStruct().OS = %q, want %q", attrs.OS, tt.wantOS)
			}
		})
	}
}

// TestExtractEC2AttributesFromStruct_TenancyNormalization tests tenancy normalization
// from protobuf Struct attributes.
func TestExtractEC2AttributesFromStruct_TenancyNormalization(t *testing.T) {
	tests := []struct {
		name        string
		attrs       *structpb.Struct
		wantTenancy string
	}{
		{
			name:        "dedicated lowercase",
			attrs:       mustStruct(map[string]interface{}{"tenancy": "dedicated"}),
			wantTenancy: "Dedicated",
		},
		{
			name:        "host lowercase",
			attrs:       mustStruct(map[string]interface{}{"tenancy": "host"}),
			wantTenancy: "Host",
		},
		{
			name:        "shared lowercase",
			attrs:       mustStruct(map[string]interface{}{"tenancy": "shared"}),
			wantTenancy: "Shared",
		},
		{
			name:        "unknown value",
			attrs:       mustStruct(map[string]interface{}{"tenancy": "unknown"}),
			wantTenancy: "Shared",
		},
		{
			name:        "missing tenancy",
			attrs:       mustStruct(map[string]interface{}{}),
			wantTenancy: "Shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ExtractEC2AttributesFromStruct(tt.attrs)
			if attrs.Tenancy != tt.wantTenancy {
				t.Errorf("ExtractEC2AttributesFromStruct().Tenancy = %q, want %q", attrs.Tenancy, tt.wantTenancy)
			}
		})
	}
}

// TestExtractEC2AttributesFromStruct_NilAndEmpty tests handling of nil and empty structs.
func TestExtractEC2AttributesFromStruct_NilAndEmpty(t *testing.T) {
	tests := []struct {
		name        string
		attrs       *structpb.Struct
		wantOS      string
		wantTenancy string
	}{
		{
			name:        "nil struct",
			attrs:       nil,
			wantOS:      "Linux",
			wantTenancy: "Shared",
		},
		{
			name:        "empty struct",
			attrs:       &structpb.Struct{Fields: nil},
			wantOS:      "Linux",
			wantTenancy: "Shared",
		},
		{
			name:        "struct with empty fields",
			attrs:       &structpb.Struct{Fields: make(map[string]*structpb.Value)},
			wantOS:      "Linux",
			wantTenancy: "Shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ExtractEC2AttributesFromStruct(tt.attrs)
			if attrs.OS != tt.wantOS {
				t.Errorf("ExtractEC2AttributesFromStruct().OS = %q, want %q", attrs.OS, tt.wantOS)
			}
			if attrs.Tenancy != tt.wantTenancy {
				t.Errorf("ExtractEC2AttributesFromStruct().Tenancy = %q, want %q", attrs.Tenancy, tt.wantTenancy)
			}
		})
	}
}

// mustStruct creates a structpb.Struct from a map, panicking on error.
// This is a test helper for creating test data.
func mustStruct(m map[string]interface{}) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		panic(err)
	}
	return s
}
