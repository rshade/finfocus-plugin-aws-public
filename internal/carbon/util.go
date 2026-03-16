package carbon

import (
	"fmt"
	"strconv"
)

// formatFloat formats a float for display.
// If the float is an integer, it is formatted as an integer.
// Otherwise, it is formatted with 2 decimal places.
func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return strconv.Itoa(int(f))
	}
	return fmt.Sprintf("%.2f", f)
}

// formatInt formats an integer for display.
func formatInt(i int) string {
	return strconv.Itoa(i)
}
