package plugin

import "testing"

// TestParseInstanceType validates the parseInstanceType function handles various
// EC2 instance type formats correctly.
//
// This test covers:
//   - Standard instance types (t2.medium, m5.xlarge)
//   - Multi-digit sizes (c6i.2xlarge)
//   - Metal instances (i3.metal)
//   - Invalid formats (no dot, empty string)
func TestParseInstanceType(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFamily string
		wantSize   string
	}{
		{
			name:       "t2.medium",
			input:      "t2.medium",
			wantFamily: "t2",
			wantSize:   "medium",
		},
		{
			name:       "m5.xlarge",
			input:      "m5.xlarge",
			wantFamily: "m5",
			wantSize:   "xlarge",
		},
		{
			name:       "c6i.2xlarge",
			input:      "c6i.2xlarge",
			wantFamily: "c6i",
			wantSize:   "2xlarge",
		},
		{
			name:       "r6g.metal",
			input:      "r6g.metal",
			wantFamily: "r6g",
			wantSize:   "metal",
		},
		{
			name:       "invalid - no dot",
			input:      "t2medium",
			wantFamily: "",
			wantSize:   "",
		},
		{
			name:       "empty string",
			input:      "",
			wantFamily: "",
			wantSize:   "",
		},
		{
			name:       "just a dot",
			input:      ".",
			wantFamily: "",
			wantSize:   "",
		},
		{
			name:       "multiple dots - only first split",
			input:      "a.b.c",
			wantFamily: "a",
			wantSize:   "b.c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			family, size := parseInstanceType(tt.input)
			if family != tt.wantFamily {
				t.Errorf("parseInstanceType(%q) family = %q, want %q", tt.input, family, tt.wantFamily)
			}
			if size != tt.wantSize {
				t.Errorf("parseInstanceType(%q) size = %q, want %q", tt.input, size, tt.wantSize)
			}
		})
	}
}

// TestGenerationUpgradeMapEntries verifies the generation upgrade map contains
// expected mappings for common instance families.
func TestGenerationUpgradeMapEntries(t *testing.T) {
	expected := map[string]string{
		"t2":  "t3",
		"t3":  "t3a",
		"m4":  "m5",
		"m5":  "m6i",
		"c4":  "c5",
		"c5":  "c6i",
		"r4":  "r5",
		"r5":  "r6i",
		"i3":  "i3en",
		"d2":  "d3",
	}

	for old, expectedNew := range expected {
		actual, exists := generationUpgradeMap[old]
		if !exists {
			t.Errorf("generationUpgradeMap missing key %q", old)
			continue
		}
		if actual != expectedNew {
			t.Errorf("generationUpgradeMap[%q] = %q, want %q", old, actual, expectedNew)
		}
	}
}

// TestGravitonMapEntries verifies the Graviton map contains expected mappings
// for x86 to ARM instance family migrations.
func TestGravitonMapEntries(t *testing.T) {
	expected := map[string]string{
		"m5":  "m6g",
		"m6i": "m6g",
		"c5":  "c6g",
		"c6i": "c6g",
		"r5":  "r6g",
		"r6i": "r6g",
		"t3":  "t4g",
		"t3a": "t4g",
	}

	for x86, expectedGraviton := range expected {
		actual, exists := gravitonMap[x86]
		if !exists {
			t.Errorf("gravitonMap missing key %q", x86)
			continue
		}
		if actual != expectedGraviton {
			t.Errorf("gravitonMap[%q] = %q, want %q", x86, actual, expectedGraviton)
		}
	}
}

// TestNoSelfReferencesInMaps verifies that maps don't map a family to itself
// which would result in useless recommendations.
func TestNoSelfReferencesInMaps(t *testing.T) {
	for old, new := range generationUpgradeMap {
		if old == new {
			t.Errorf("generationUpgradeMap has self-reference: %q -> %q", old, new)
		}
	}

	for old, new := range gravitonMap {
		if old == new {
			t.Errorf("gravitonMap has self-reference: %q -> %q", old, new)
		}
	}
}
