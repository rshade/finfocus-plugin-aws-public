package router

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// maxRegionLength is the maximum allowed length for an AWS region identifier.
const maxRegionLength = 32

var regionPattern = regexp.MustCompile(`^[a-z]{2}(?:-gov)?-[a-z]+-\d+$`)

// validateRegion enforces AWS region-like input (e.g. us-east-1, us-gov-west-1)
// and rejects path separators / traversal sequences before using region in file paths.
func validateRegion(region string) error {
	if region == "" {
		return errors.New("region is empty")
	}
	if len(region) > maxRegionLength {
		return fmt.Errorf("invalid region %q", region)
	}
	if strings.Contains(region, "/") || strings.Contains(region, "\\") || strings.Contains(region, "..") {
		return fmt.Errorf("invalid region %q", region)
	}
	if !regionPattern.MatchString(region) {
		return fmt.Errorf("invalid region %q", region)
	}
	return nil
}
