package analyzer

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gituser/redis-analyze/internal/models"
)

// Analyze processes a slice of KeyInfo and produces a full Report.
func Analyze(keys []models.KeyInfo, serverInfo models.ServerInfo, opts models.ScanOptions, duration time.Duration) models.Report {
	report := models.Report{
		ServerInfo:   serverInfo,
		Prefixes:     opts.Prefixes,
		ScannedCount: int64(len(keys)),
		ScanDuration: duration.Milliseconds(),
	}

	if len(keys) == 0 {
		return report
	}

	// Sort keys by size descending for Top N
	sorted := make([]models.KeyInfo, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})

	// Top N keys
	n := opts.TopN
	if n > len(sorted) {
		n = len(sorted)
	}
	report.TopKeys = sorted[:n]

	// Type statistics
	typeMap := make(map[string][]models.KeyInfo)
	for _, k := range keys {
		kType := k.Type
		if kType == "" {
			kType = "unknown"
		}
		typeMap[kType] = append(typeMap[kType], k)
	}

	for kType, kList := range typeMap {
		stats := calculateStats(kList)
		stats.Type = kType
		report.ByType = append(report.ByType, stats)
	}
	sort.Slice(report.ByType, func(i, j int) bool {
		return report.ByType[i].Total > report.ByType[j].Total
	})

	// Prefix / namespace statistics
	report.ByPrefix = buildPrefixStats(keys, opts.Separator, opts.PrefixDepth)
	sort.Slice(report.ByPrefix, func(i, j int) bool {
		return report.ByPrefix[i].Total > report.ByPrefix[j].Total
	})

	// Summary
	report.Summary = calculateSummary(keys)

	return report
}

// calculateStats computes TypeStats for a list of keys of the same type.
func calculateStats(keys []models.KeyInfo) models.TypeStats {
	var total, min, max int64
	min = math.MaxInt64
	count := int64(len(keys))

	for _, k := range keys {
		size := k.Size
		if size < 0 {
			continue // skip keys with unknown size
		}
		total += size
		if size < min {
			min = size
		}
		if size > max {
			max = size
		}
	}

	if min == math.MaxInt64 {
		min = 0
	}

	return models.TypeStats{
		Count: count,
		Total: total,
		Avg:   safeDivide(total, count),
		Min:   min,
		Max:   max,
	}
}

// buildPrefixStats groups keys by their prefix namespace.
func buildPrefixStats(keys []models.KeyInfo, separator string, depth int) []models.PrefixStats {
	type prefixAccum struct {
		count int64
		total int64
		types map[string]int64
	}

	prefixMap := make(map[string]*prefixAccum)

	for _, k := range keys {
		prefix := extractPrefix(k.Key, separator, depth)
		if prefix == "" {
			prefix = "(root)"
		}

		pa, ok := prefixMap[prefix]
		if !ok {
			pa = &prefixAccum{
				types: make(map[string]int64),
			}
			prefixMap[prefix] = pa
		}

		pa.count++
		if k.Size >= 0 {
			pa.total += k.Size
		}
		pa.types[k.Type]++
	}

	stats := make([]models.PrefixStats, 0, len(prefixMap))
	for prefix, pa := range prefixMap {
		stats = append(stats, models.PrefixStats{
			Prefix: prefix,
			Count:  pa.count,
			Total:  pa.total,
			Types:  pa.types,
		})
	}

	return stats
}

// extractPrefix derives the namespace prefix from a key.
// e.g., "user:profile:123" with sep=":" depth=2 => "user:profile"
func extractPrefix(key, separator string, depth int) string {
	if separator == "" {
		separator = ":"
	}
	if depth <= 0 {
		return key
	}

	parts := strings.Split(key, separator)
	if len(parts) <= depth {
		return strings.Join(parts, separator)
	}
	return strings.Join(parts[:depth], separator)
}

// calculateSummary computes overall summary statistics.
func calculateSummary(keys []models.KeyInfo) models.SummaryStats {
	var total, min, max int64
	min = math.MaxInt64
	count := int64(len(keys))

	for _, k := range keys {
		if k.Size < 0 {
			continue
		}
		total += k.Size
		if k.Size < min {
			min = k.Size
		}
		if k.Size > max {
			max = k.Size
		}
	}

	if min == math.MaxInt64 {
		min = 0
	}

	return models.SummaryStats{
		TotalKeys:   count,
		TotalMemory: total,
		AvgKeySize:  safeDivide(total, count),
		MinKeySize:  min,
		MaxKeySize:  max,
	}
}

func safeDivide(a, b int64) int64 {
	if b == 0 {
		return 0
	}
	return a / b
}
