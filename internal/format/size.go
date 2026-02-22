package format

import (
	"fmt"
	"math"
)

// FormatBytes converts bytes to human-readable format (KB, MB, GB, TB).
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	exp := int(math.Log(float64(bytes)) / math.Log(unit))
	if exp > 4 {
		exp = 4
	}

	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes) / math.Pow(unit, float64(exp))

	return fmt.Sprintf("%.1f %s", value, units[exp])
}
