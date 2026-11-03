package carbon

// EFSEstimator estimates carbon footprint for Amazon EFS storage.
type EFSEstimator struct{}

// NewEFSEstimator creates a new EFS carbon estimator.
func NewEFSEstimator() *EFSEstimator {
	return &EFSEstimator{}
}

// EstimateCarbonGrams calculates the carbon footprint for EFS storage.
//
// Each storage class is estimated with the shared CCF storage formula. CCF
// classifies every EFS usage type (Standard, IA, Archive, One Zone) as HDD
// storage with a replication factor of 3 for regional file systems and 2 for
// One Zone, so unlike S3 STANDARD, EFS Standard is not treated as SSD.
//
// Returns (0, false) when a size or the hours are negative, or when a storage
// spec is missing.
func (e *EFSEstimator) EstimateCarbonGrams(config EFSStorageConfig) (float64, bool) {
	if config.StandardGB < 0 || config.IAGB < 0 || config.ArchiveGB < 0 || config.Hours < 0 {
		return 0, false
	}

	type classSize struct {
		class string
		gb    float64
	}
	sizes := []classSize{{"STANDARD", config.StandardGB}, {"IA", config.IAGB}, {"ARCHIVE", config.ArchiveGB}}
	if config.OneZone {
		sizes = []classSize{{"ONEZONE", config.StandardGB}, {"ONEZONE_IA", config.IAGB}}
	}

	total := 0.0
	for _, s := range sizes {
		if _, ok := GetEFSStorageSpec(s.class); !ok {
			return 0, false
		}
		total += CalculateStorageCarbonGrams(s.gb, config.Hours, "efs", s.class, config.Region)
	}
	return total, true
}
