package plugin

import "testing"

// TestDefaultsTracker_Empty verifies that a tracker with no defaults returns nil
// metadata and qualityHigh quality, ensuring backward compatibility for responses
// where all values are explicit.
func TestDefaultsTracker_Empty(t *testing.T) {
	var dt DefaultsTracker

	if dt.Quality() != qualityHigh {
		t.Errorf("Quality() = %q, want %q", dt.Quality(), qualityHigh)
	}
	if dt.Metadata() != nil {
		t.Errorf("Metadata() = %v, want nil", dt.Metadata())
	}
}

// TestDefaultsTracker_SingleConfig verifies that a single KindConfig default
// produces qualityMedium quality with correct defaults_applied metadata.
func TestDefaultsTracker_SingleConfig(t *testing.T) {
	var dt DefaultsTracker
	dt.Add("size", "8", KindConfig)

	if dt.Quality() != qualityMedium {
		t.Errorf("Quality() = %q, want %q", dt.Quality(), qualityMedium)
	}

	m := dt.Metadata()
	if m == nil {
		t.Fatal("Metadata() returned nil, want non-nil")
	}
	if m[metadataKeyDefaultsApplied] != "size=8" {
		t.Errorf("defaults_applied = %q, want %q", m[metadataKeyDefaultsApplied], "size=8")
	}
	if m[metadataKeyEstimateQuality] != qualityMedium {
		t.Errorf("estimate_quality = %q, want %q", m[metadataKeyEstimateQuality], qualityMedium)
	}
}

// TestDefaultsTracker_SingleUsageZero verifies that a single KindUsageZero
// default produces qualityLow quality.
func TestDefaultsTracker_SingleUsageZero(t *testing.T) {
	var dt DefaultsTracker
	dt.Add("requests_per_month", "0", KindUsageZero)

	if dt.Quality() != qualityLow {
		t.Errorf("Quality() = %q, want %q", dt.Quality(), qualityLow)
	}

	m := dt.Metadata()
	if m == nil {
		t.Fatal("Metadata() returned nil, want non-nil")
	}
	if m[metadataKeyEstimateQuality] != qualityLow {
		t.Errorf("estimate_quality = %q, want %q", m[metadataKeyEstimateQuality], qualityLow)
	}
}

// TestDefaultsTracker_MixedKinds verifies that when both KindConfig and
// KindUsageZero defaults are present, UsageZero dominates → qualityLow quality.
func TestDefaultsTracker_MixedKinds(t *testing.T) {
	var dt DefaultsTracker
	dt.Add("engine", "mysql", KindConfig)
	dt.Add("read_capacity_units", "0", KindUsageZero)

	if dt.Quality() != qualityLow {
		t.Errorf("Quality() = %q, want %q", dt.Quality(), qualityLow)
	}

	m := dt.Metadata()
	if m == nil {
		t.Fatal("Metadata() returned nil, want non-nil")
	}
	if m[metadataKeyDefaultsApplied] != "engine=mysql,read_capacity_units=0" {
		t.Errorf("defaults_applied = %q, want %q",
			m[metadataKeyDefaultsApplied], "engine=mysql,read_capacity_units=0")
	}
}

// TestDefaultsTracker_MultipleConfig verifies that multiple KindConfig defaults
// are all listed and quality remains qualityMedium.
func TestDefaultsTracker_MultipleConfig(t *testing.T) {
	var dt DefaultsTracker
	dt.Add("engine", "mysql", KindConfig)
	dt.Add("storage_type", "gp2", KindConfig)
	dt.Add("storage_size", "20", KindConfig)

	if dt.Quality() != qualityMedium {
		t.Errorf("Quality() = %q, want %q", dt.Quality(), qualityMedium)
	}

	m := dt.Metadata()
	if m == nil {
		t.Fatal("Metadata() returned nil, want non-nil")
	}
	expected := "engine=mysql,storage_type=gp2,storage_size=20"
	if m[metadataKeyDefaultsApplied] != expected {
		t.Errorf("defaults_applied = %q, want %q", m[metadataKeyDefaultsApplied], expected)
	}
}
