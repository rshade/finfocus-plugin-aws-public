package router

import (
	"fmt"
	"regexp"
	"strings"
)

var regionPattern = regexp.MustCompile(`^[a-z]{2}(?:-gov)?-[a-z]+-\d+$`)

func validateRegion(region string) error {
	if region == "" {
		return fmt.Errorf("region is empty")
	}
	if len(region) > 32 {
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
