package router

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/rs/zerolog"
)

// binaryPattern matches region binary names like finfocus-plugin-aws-public-us-east-1.
var binaryPattern = regexp.MustCompile(`^finfocus-plugin-aws-public-([a-z]{2}(?:-gov)?-[a-z]+-\d+)(?:\.exe)?$`)

// Discover scans the given directory for region-specific plugin binaries.
// It returns a map of AWS region name to absolute binary path.
// Files must match the naming convention: finfocus-plugin-aws-public-{region}.
func Discover(dir string, logger zerolog.Logger) map[string]string {
	result := make(map[string]string)

	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Warn().Err(err).Str("dir", dir).Msg("failed to read binary directory")
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		matches := binaryPattern.FindStringSubmatch(name)
		if len(matches) != regexSubmatchWithOneGroup {
			continue
		}

		region := matches[1]
		absPath, absErr := filepath.Abs(filepath.Join(dir, name))
		if absErr != nil {
			logger.Warn().Err(absErr).Str("file", name).Msg("failed to resolve absolute binary path")
			continue
		}

		// Verify the file is readable (skip files we cannot stat)
		if _, infoErr := entry.Info(); infoErr != nil {
			continue
		}

		result[region] = absPath
		logger.Debug().
			Str("region", region).
			Str("path", absPath).
			Msg("discovered region binary")
	}

	logger.Info().
		Int("count", len(result)).
		Msg("binary discovery complete")

	return result
}
