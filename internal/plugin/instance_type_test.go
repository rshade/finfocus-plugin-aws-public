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
			name:       "multiple dots - invalid",
			input:      "a.b.c",
			wantFamily: "",
			wantSize:   "",
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
		"t2": "t3",
		"t3": "t3a",
		"m4": "m5",
		"m5": "m6i",
		"c4": "c5",
		"c5": "c6i",
		"r4": "r5",
		"r5": "r6i",
		"i3": "i3en",
		"d2": "d3",
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

	for old, new := range rdsGenerationUpgradeMap {
		if old == new {
			t.Errorf("rdsGenerationUpgradeMap has self-reference: %q -> %q", old, new)
		}
	}

	for old, new := range rdsGravitonMap {
		if old == new {
			t.Errorf("rdsGravitonMap has self-reference: %q -> %q", old, new)
		}
	}
}

// TestParseRDSInstanceType validates the parseRDSInstanceType function handles
// various RDS instance type formats correctly.
//
// RDS instance types follow the pattern: db.<family>.<size>
// Examples: db.t3.medium, db.r6g.xlarge, db.m5.large.
func TestParseRDSInstanceType(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFamily string
		wantSize   string
	}{
		{
			name:       "db.t3.medium",
			input:      "db.t3.medium",
			wantFamily: "db.t3",
			wantSize:   "medium",
		},
		{
			name:       "db.m5.xlarge",
			input:      "db.m5.xlarge",
			wantFamily: "db.m5",
			wantSize:   "xlarge",
		},
		{
			name:       "db.r6g.2xlarge",
			input:      "db.r6g.2xlarge",
			wantFamily: "db.r6g",
			wantSize:   "2xlarge",
		},
		{
			name:       "db.t4g.small - Graviton",
			input:      "db.t4g.small",
			wantFamily: "db.t4g",
			wantSize:   "small",
		},
		{
			name:       "invalid - no db prefix",
			input:      "t3.medium",
			wantFamily: "",
			wantSize:   "",
		},
		{
			name:       "invalid - EC2 format",
			input:      "m5.xlarge",
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
			name:       "just db.",
			input:      "db.",
			wantFamily: "",
			wantSize:   "",
		},
		{
			name:       "db.t3 - missing size",
			input:      "db.t3",
			wantFamily: "",
			wantSize:   "",
		},
		{
			name:       "db.t3.medium.extra - invalid",
			input:      "db.t3.medium.extra",
			wantFamily: "",
			wantSize:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			family, size := parseRDSInstanceType(tt.input)
			if family != tt.wantFamily {
				t.Errorf("parseRDSInstanceType(%q) family = %q, want %q", tt.input, family, tt.wantFamily)
			}
			if size != tt.wantSize {
				t.Errorf("parseRDSInstanceType(%q) size = %q, want %q", tt.input, size, tt.wantSize)
			}
		})
	}
}

// TestRDSGenerationUpgradeMapEntries verifies the RDS generation upgrade map
// contains expected mappings for common RDS instance families.
func TestRDSGenerationUpgradeMapEntries(t *testing.T) {
	expected := map[string]string{
		"db.t2": "db.t3",
		"db.t3": "db.t4g",
		"db.m4": "db.m5",
		"db.m5": "db.m6i",
		"db.r4": "db.r5",
		"db.r5": "db.r6i",
	}

	for old, expectedNew := range expected {
		actual, exists := rdsGenerationUpgradeMap[old]
		if !exists {
			t.Errorf("rdsGenerationUpgradeMap missing key %q", old)
			continue
		}
		if actual != expectedNew {
			t.Errorf("rdsGenerationUpgradeMap[%q] = %q, want %q", old, actual, expectedNew)
		}
	}
}

// TestRDSGravitonMapEntries verifies the RDS Graviton map contains expected
// mappings for x86 to ARM instance family migrations.
func TestRDSGravitonMapEntries(t *testing.T) {
	expected := map[string]string{
		"db.m5": "db.m6g",
		"db.r5": "db.r6g",
		"db.t3": "db.t4g",
	}

	for x86, expectedGraviton := range expected {
		actual, exists := rdsGravitonMap[x86]
		if !exists {
			t.Errorf("rdsGravitonMap missing key %q", x86)
			continue
		}
		if actual != expectedGraviton {
			t.Errorf("rdsGravitonMap[%q] = %q, want %q", x86, actual, expectedGraviton)
		}
	}
}

// TestRDSGravitonSupportedEngines verifies the engine support map includes
// common engines and correctly excludes Oracle and SQL Server.
func TestRDSGravitonSupportedEngines(t *testing.T) {
	// Should be supported
	supported := []string{"mysql", "postgres", "postgresql", "mariadb", "aurora"}
	for _, engine := range supported {
		if !rdsGravitonSupportedEngines[engine] {
			t.Errorf("rdsGravitonSupportedEngines should include %q", engine)
		}
	}

	// Should NOT be supported (Oracle and SQL Server don't support Graviton)
	unsupported := []string{"oracle", "sqlserver", "oracle-ee", "sqlserver-se"}
	for _, engine := range unsupported {
		if rdsGravitonSupportedEngines[engine] {
			t.Errorf("rdsGravitonSupportedEngines should NOT include %q", engine)
		}
	}
}
