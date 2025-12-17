package plugin

import "strings"

// parseInstanceType splits an EC2 instance type into family and size.
// Example: "t2.medium" → ("t2", "medium")
// Returns empty strings if the format is invalid.
func parseInstanceType(instanceType string) (family, size string) {
	parts := strings.SplitN(instanceType, ".", 2)
	if len(parts) != 2 {
		return "", ""
	}
	// Validate both parts are non-empty (handles edge case like "." input)
	if parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// generationUpgradeMap maps old instance families to newer generations.
// Only includes mappings where the newer generation is typically the same price
// or cheaper with better performance.
var generationUpgradeMap = map[string]string{
	// T-series (burstable general purpose)
	"t2": "t3",  // 2014 → 2018
	"t3": "t3a", // Intel → AMD (often cheaper)

	// M-series (general purpose)
	"m4":  "m5",  // 2015 → 2017
	"m5":  "m6i", // 2017 → 2021
	"m5a": "m6a", // AMD 2018 → AMD 2022
	"m6i": "m7i", // 2021 → 2023
	"m6a": "m7a", // AMD 2022 → AMD 2023

	// C-series (compute optimized)
	"c4":  "c5",  // 2015 → 2017
	"c5":  "c6i", // 2017 → 2021
	"c5a": "c6a", // AMD 2020 → AMD 2022
	"c6i": "c7i", // 2021 → 2023
	"c6a": "c7a", // AMD 2022 → AMD 2023

	// R-series (memory optimized)
	"r4":  "r5",  // 2016 → 2018
	"r5":  "r6i", // 2018 → 2021
	"r5a": "r6a", // AMD 2019 → AMD 2022
	"r6i": "r7i", // 2021 → 2023
	"r6a": "r7a", // AMD 2022 → AMD 2023

	// I-series (storage optimized)
	"i3": "i3en", // 2017 → 2019

	// D-series (dense storage)
	"d2": "d3", // 2015 → 2020
}

// gravitonMap maps x86 instance families to Graviton (ARM) equivalents.
// Graviton instances typically offer ~20% cost savings with comparable performance.
var gravitonMap = map[string]string{
	// M-series → M6g/M7g
	"m5":  "m6g",
	"m5a": "m6g",
	"m5n": "m6g",
	"m6i": "m6g",
	"m6a": "m6g",
	"m7i": "m7g", // 7th gen Intel → Graviton3
	"m7a": "m7g", // 7th gen AMD → Graviton3

	// C-series → C6g/C6gn/C7g
	"c5":  "c6g",
	"c5a": "c6g",
	"c5n": "c6gn",
	"c6i": "c6g",
	"c6a": "c6g",
	"c7i": "c7g", // 7th gen Intel → Graviton3
	"c7a": "c7g", // 7th gen AMD → Graviton3

	// R-series → R6g/R7g
	"r5":  "r6g",
	"r5a": "r6g",
	"r5n": "r6g",
	"r6i": "r6g",
	"r6a": "r6g",
	"r7i": "r7g", // 7th gen Intel → Graviton3
	"r7a": "r7g", // 7th gen AMD → Graviton3

	// T-series → T4g
	"t3":  "t4g",
	"t3a": "t4g",
}
