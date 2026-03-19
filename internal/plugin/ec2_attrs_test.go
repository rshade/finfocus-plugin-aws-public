package plugin

import (
	"testing"

	"github.com/rs/zerolog"
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
				t.Errorf(
					"ExtractEC2AttributesFromTags(%v).Tenancy = %q, want %q",
					tt.tags,
					attrs.Tenancy,
					tt.wantTenancy,
				)
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
				t.Errorf(
					"ExtractEC2AttributesFromTags(%v).Tenancy = %q, want %q",
					tt.tags,
					attrs.Tenancy,
					tt.wantTenancy,
				)
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
				t.Errorf(
					"ExtractEC2AttributesFromTags(%v).Tenancy = %q, want %q",
					tt.tags,
					attrs.Tenancy,
					tt.wantTenancy,
				)
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

// TestParsePositiveInt verifies that parsePositiveInt correctly rejects
// zero, negative, and non-numeric values while accepting valid positive integers.
// This directly tests the boundary behavior documented in the function's docstring:
// values ≤ 0 (including zero) return (0, false).
func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantVal int
		wantOK  bool
	}{
		{name: "positive integer", input: "10", wantVal: 10, wantOK: true},
		{name: "one", input: "1", wantVal: 1, wantOK: true},
		{name: "zero is rejected", input: "0", wantVal: 0, wantOK: false},
		{name: "negative is rejected", input: "-5", wantVal: 0, wantOK: false},
		{name: "non-numeric is rejected", input: "abc", wantVal: 0, wantOK: false},
		{name: "empty string is rejected", input: "", wantVal: 0, wantOK: false},
		{name: "float is rejected", input: "10.5", wantVal: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := parsePositiveInt(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parsePositiveInt(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if val != tt.wantVal {
				t.Errorf("parsePositiveInt(%q) val = %d, want %d", tt.input, val, tt.wantVal)
			}
		})
	}
}

// TestParseGoMapString verifies parsing of Go's fmt.Sprint map format strings.
func TestParseGoMapString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "normal map",
			input: "map[volumeSize:8 volumeType:gp2]",
			want:  map[string]string{"volumeSize": "8", "volumeType": "gp2"},
		},
		{
			name:  "single entry",
			input: "map[volumeType:gp3]",
			want:  map[string]string{"volumeType": "gp3"},
		},
		{
			name:  "empty map",
			input: "map[]",
			want:  map[string]string{},
		},
		{
			name:  "empty string",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "no map prefix",
			input: "volumeSize:8 volumeType:gp2",
			want:  map[string]string{},
		},
		{
			name:  "value with colons",
			input: "map[arn:aws:ec2:us-east-1 volumeType:gp2]",
			want:  map[string]string{"arn": "aws:ec2:us-east-1", "volumeType": "gp2"},
		},
		{
			name:  "multiple fields",
			input: "map[deleteOnTermination:true encrypted:false volumeSize:20 volumeType:gp3]",
			want: map[string]string{
				"deleteOnTermination": "true",
				"encrypted":           "false",
				"volumeSize":          "20",
				"volumeType":          "gp3",
			},
		},
		{
			name:  "whitespace around",
			input: "  map[volumeSize:8 volumeType:gp2]  ",
			want:  map[string]string{"volumeSize": "8", "volumeType": "gp2"},
		},
		{
			// Documents known limitation: values with spaces are silently truncated
			// because strings.Fields splits on whitespace. "My Volume" becomes two
			// tokens: "name:My" (parsed as name→My) and "Volume" (no colon, skipped).
			// This is acceptable because Pulumi's rootBlockDevice only uses scalar values.
			name:  "space in value truncates silently",
			input: "map[name:My Volume volumeType:gp2]",
			want:  map[string]string{"name": "My", "volumeType": "gp2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGoMapString(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseGoMapString(%q) returned %d entries, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseGoMapString(%q)[%q] = %q, want %q", tt.input, k, got[k], v)
				}
			}
		})
	}
}

// TestExtractRootVolumeFromTags_GoMapFormat tests root volume extraction from
// the rootBlockDevice tag in Go map string format.
func TestExtractRootVolumeFromTags_GoMapFormat(t *testing.T) {
	tags := map[string]string{
		"rootBlockDevice": "map[volumeSize:20 volumeType:gp3]",
	}
	rv := ExtractRootVolumeFromTags(tags, zerolog.Nop())

	if !rv.Present {
		t.Fatal("Expected Present=true for rootBlockDevice tag")
	}
	if rv.VolumeType != "gp3" {
		t.Errorf("VolumeType = %q, want %q", rv.VolumeType, "gp3")
	}
	if rv.SizeGB != 20 {
		t.Errorf("SizeGB = %d, want %d", rv.SizeGB, 20)
	}
}

// TestExtractRootVolumeFromTags_IndividualTags tests root volume extraction
// from individual root_volume_type and root_volume_size tags.
func TestExtractRootVolumeFromTags_IndividualTags(t *testing.T) {
	tags := map[string]string{
		"root_volume_type": "io1",
		"root_volume_size": "100",
	}
	rv := ExtractRootVolumeFromTags(tags, zerolog.Nop())

	if !rv.Present {
		t.Fatal("Expected Present=true for individual tags")
	}
	if rv.VolumeType != "io1" {
		t.Errorf("VolumeType = %q, want %q", rv.VolumeType, "io1")
	}
	if rv.SizeGB != 100 {
		t.Errorf("SizeGB = %d, want %d", rv.SizeGB, 100)
	}
}

// TestExtractRootVolumeFromTags_IndividualOverridesMap tests that individual
// tags take priority over rootBlockDevice map values.
func TestExtractRootVolumeFromTags_IndividualOverridesMap(t *testing.T) {
	tags := map[string]string{
		"rootBlockDevice":  "map[volumeSize:8 volumeType:gp2]",
		"root_volume_type": "gp3",
		"root_volume_size": "50",
	}
	rv := ExtractRootVolumeFromTags(tags, zerolog.Nop())

	if !rv.Present {
		t.Fatal("Expected Present=true")
	}
	if rv.VolumeType != "gp3" {
		t.Errorf("VolumeType = %q, want %q (individual should override map)", rv.VolumeType, "gp3")
	}
	if rv.SizeGB != 50 {
		t.Errorf("SizeGB = %d, want %d (individual should override map)", rv.SizeGB, 50)
	}
}

// TestExtractRootVolumeFromTags_NoInfo tests that no root volume info is
// returned when no relevant tags are present.
func TestExtractRootVolumeFromTags_NoInfo(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
	}{
		{name: "nil tags", tags: nil},
		{name: "empty tags", tags: map[string]string{}},
		{name: "unrelated tags", tags: map[string]string{"Name": "test"}},
		{name: "empty rootBlockDevice", tags: map[string]string{"rootBlockDevice": ""}},
		{name: "malformed rootBlockDevice", tags: map[string]string{"rootBlockDevice": "not a map"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := ExtractRootVolumeFromTags(tt.tags, zerolog.Nop())
			if rv.Present {
				t.Errorf("Expected Present=false for tags %v", tt.tags)
			}
		})
	}
}

// TestExtractRootVolumeFromTags_Defaults tests that defaults are applied when
// root volume source is present but fields are missing or invalid.
func TestExtractRootVolumeFromTags_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		wantType string
		wantSize int
	}{
		{
			name:     "type only from individual tag",
			tags:     map[string]string{"root_volume_type": "gp3"},
			wantType: "gp3",
			wantSize: 8, // default
		},
		{
			name:     "size only from individual tag",
			tags:     map[string]string{"root_volume_size": "50"},
			wantType: "gp2", // default
			wantSize: 50,
		},
		{
			name:     "invalid size defaults to 8",
			tags:     map[string]string{"root_volume_type": "gp3", "root_volume_size": "invalid"},
			wantType: "gp3",
			wantSize: 8, // default
		},
		{
			name:     "negative size defaults to 8",
			tags:     map[string]string{"root_volume_type": "gp3", "root_volume_size": "-10"},
			wantType: "gp3",
			wantSize: 8, // default
		},
		{
			name:     "zero size defaults to 8",
			tags:     map[string]string{"root_volume_type": "gp3", "root_volume_size": "0"},
			wantType: "gp3",
			wantSize: 8, // default
		},
		{
			name:     "rootBlockDevice with type only",
			tags:     map[string]string{"rootBlockDevice": "map[volumeType:io2]"},
			wantType: "io2",
			wantSize: 8, // default
		},
		{
			name:     "rootBlockDevice with size only",
			tags:     map[string]string{"rootBlockDevice": "map[volumeSize:100]"},
			wantType: "gp2", // default
			wantSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := ExtractRootVolumeFromTags(tt.tags, zerolog.Nop())
			if !rv.Present {
				t.Fatal("Expected Present=true")
			}
			if rv.VolumeType != tt.wantType {
				t.Errorf("VolumeType = %q, want %q", rv.VolumeType, tt.wantType)
			}
			if rv.SizeGB != tt.wantSize {
				t.Errorf("SizeGB = %d, want %d", rv.SizeGB, tt.wantSize)
			}
		})
	}
}

// TestExtractRootVolumeFromStruct tests root volume extraction from protobuf Struct.
func TestExtractRootVolumeFromStruct(t *testing.T) {
	tests := []struct {
		name        string
		attrs       *structpb.Struct
		wantPresent bool
		wantType    string
		wantSize    int
	}{
		{
			name:        "nil struct",
			attrs:       nil,
			wantPresent: false,
		},
		{
			name:        "no rootBlockDevice",
			attrs:       mustStruct(map[string]interface{}{"instanceType": "t3.micro"}),
			wantPresent: false,
		},
		{
			name: "rootBlockDevice as struct",
			attrs: mustStruct(map[string]interface{}{
				"rootBlockDevice": map[string]interface{}{
					"volumeType": "gp3",
					"volumeSize": float64(20),
				},
			}),
			wantPresent: true,
			wantType:    "gp3",
			wantSize:    20,
		},
		{
			name: "rootBlockDevice as list",
			attrs: mustStruct(map[string]interface{}{
				"rootBlockDevice": []interface{}{
					map[string]interface{}{
						"volumeType": "io1",
						"volumeSize": float64(100),
					},
				},
			}),
			wantPresent: true,
			wantType:    "io1",
			wantSize:    100,
		},
		{
			name: "rootBlockDevice as Go map string",
			attrs: mustStruct(map[string]interface{}{
				"rootBlockDevice": "map[volumeSize:30 volumeType:gp2]",
			}),
			wantPresent: true,
			wantType:    "gp2",
			wantSize:    30,
		},
		{
			name: "rootBlockDevice struct with defaults",
			attrs: mustStruct(map[string]interface{}{
				"rootBlockDevice": map[string]interface{}{},
			}),
			wantPresent: true,
			wantType:    "gp2",
			wantSize:    8,
		},
		{
			name: "rootBlockDevice struct with type only",
			attrs: mustStruct(map[string]interface{}{
				"rootBlockDevice": map[string]interface{}{
					"volumeType": "gp3",
				},
			}),
			wantPresent: true,
			wantType:    "gp3",
			wantSize:    8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := ExtractRootVolumeFromStruct(tt.attrs, zerolog.Nop())
			if rv.Present != tt.wantPresent {
				t.Fatalf("Present = %v, want %v", rv.Present, tt.wantPresent)
			}
			if !tt.wantPresent {
				return
			}
			if rv.VolumeType != tt.wantType {
				t.Errorf("VolumeType = %q, want %q", rv.VolumeType, tt.wantType)
			}
			if rv.SizeGB != tt.wantSize {
				t.Errorf("SizeGB = %d, want %d", rv.SizeGB, tt.wantSize)
			}
		})
	}
}
