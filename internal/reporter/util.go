package reporter

import (
	"fmt"
	"time"
)

// timeMs is a shorthand for time.Millisecond used in formatting.
var timeMs = time.Millisecond

// FormatSize converts bytes to a human-readable string (KB, MB, GB, etc.).
func FormatSize(bytes int64) string {
	if bytes < 0 {
		return "N/A"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatInt formats an integer with thousand separators.
func FormatInt(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	in := fmt.Sprintf("%d", n)
	out := make([]byte, len(in)+((len(in)-1)/3))
	for i, j, c := len(in)-1, len(out)-1, 0; i >= 0; {
		out[j] = in[i]
		i--
		j--
		c++
		if c == 3 && i >= 0 {
			out[j] = ','
			j--
			c = 0
		}
	}
	return string(out)
}
