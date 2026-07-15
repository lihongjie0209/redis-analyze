package reporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/lihongjie0209/redis-analyze/internal/models"
)

// RenderCSV renders the report in CSV format with separate sections.
func RenderCSV(report models.Report) string {
	var b strings.Builder
	w := csv.NewWriter(&b)

	// Summary section
	w.Write([]string{"Section", "Key", "Value"})
	w.Write([]string{"summary", "total_keys", fmt.Sprintf("%d", report.Summary.TotalKeys)})
	w.Write([]string{"summary", "total_memory_bytes", fmt.Sprintf("%d", report.Summary.TotalMemory)})
	w.Write([]string{"summary", "avg_key_size_bytes", fmt.Sprintf("%d", report.Summary.AvgKeySize)})
	w.Write([]string{"summary", "min_key_size_bytes", fmt.Sprintf("%d", report.Summary.MinKeySize)})
	w.Write([]string{"summary", "max_key_size_bytes", fmt.Sprintf("%d", report.Summary.MaxKeySize)})
	w.Write([]string{"summary", "scanned_keys", fmt.Sprintf("%d", report.ScannedCount)})
	w.Write([]string{"summary", "scan_duration_ms", fmt.Sprintf("%d", report.ScanDuration)})
	w.Write([]string{"summary", "redis_version", report.ServerInfo.Version})
	w.Write([]string{"summary", "redis_mode", report.ServerInfo.Mode})

	// By Type section
	if len(report.ByType) > 0 {
		w.Write([]string{})
		w.Write([]string{"by_type", "type", "count", "memory_bytes", "avg_bytes", "min_bytes", "max_bytes"})
		for _, s := range report.ByType {
			w.Write([]string{
				"by_type",
				s.Type,
				fmt.Sprintf("%d", s.Count),
				fmt.Sprintf("%d", s.Total),
				fmt.Sprintf("%d", s.Avg),
				fmt.Sprintf("%d", s.Min),
				fmt.Sprintf("%d", s.Max),
			})
		}
	}

	// By Prefix section
	if len(report.ByPrefix) > 0 {
		w.Write([]string{})
		w.Write([]string{"by_prefix", "prefix", "count", "memory_bytes"})
		for _, s := range report.ByPrefix {
			w.Write([]string{
				"by_prefix",
				s.Prefix,
				fmt.Sprintf("%d", s.Count),
				fmt.Sprintf("%d", s.Total),
			})
		}
	}

	// Top Keys section
	if len(report.TopKeys) > 0 {
		w.Write([]string{})
		w.Write([]string{"top_keys", "key", "type", "size_bytes", "idle_time_seconds"})
		for _, k := range report.TopKeys {
			w.Write([]string{
				"top_keys",
				k.Key,
				k.Type,
				fmt.Sprintf("%d", k.Size),
				fmt.Sprintf("%d", k.IdleTime),
			})
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return b.String()
}

// WriteCSV writes CSV output to the given writer (useful for streaming).
func WriteCSV(w io.Writer, report models.Report) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	cw.Write([]string{"section", "key", "type", "count", "memory_bytes", "avg_bytes", "min_bytes", "max_bytes", "size_bytes", "idle_seconds"})

	for _, s := range report.ByType {
		cw.Write([]string{"by_type", s.Type, s.Type, fmt.Sprintf("%d", s.Count), fmt.Sprintf("%d", s.Total), fmt.Sprintf("%d", s.Avg), fmt.Sprintf("%d", s.Min), fmt.Sprintf("%d", s.Max), "", ""})
	}

	for _, k := range report.TopKeys {
		cw.Write([]string{"top_keys", k.Key, k.Type, "", "", "", "", "", fmt.Sprintf("%d", k.Size), fmt.Sprintf("%d", k.IdleTime)})
	}

	return cw.Error()
}
