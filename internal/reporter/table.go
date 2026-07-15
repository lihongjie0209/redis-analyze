package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/lihongjie0209/redis-analyze/internal/models"
)

var (
	headerColor = color.New(color.FgCyan, color.Bold)
	labelColor  = color.New(color.FgYellow)
	valueColor  = color.New(color.FgWhite)
	keyColor    = color.New(color.FgGreen)
	accentColor = color.New(color.FgMagenta)
)

// RenderTable renders the report as a nicely formatted terminal table.
func RenderTable(report models.Report) string {
	var b strings.Builder

	// Title
	headerColor.Fprintf(&b, "\n  Redis Memory Analysis Report\n")
	accentColor.Fprintf(&b, "  %s\n\n", strings.Repeat("─", 55))

	// Connection info
	fmt.Fprintf(&b, "  %s %s %s %d  %s %s  %s %s\n",
		labelColor.Sprint("Host:"), valueColor.Sprint(report.ServerInfo.Host+":"+fmt.Sprint(report.ServerInfo.Port)),
		labelColor.Sprint("DB:"), report.ServerInfo.DB,
		labelColor.Sprint("Mode:"), valueColor.Sprint(report.ServerInfo.Mode),
		labelColor.Sprint("Duration:"), valueColor.Sprint(time.Duration(report.ScanDuration)*time.Millisecond),
	)
	if report.ServerInfo.Version != "" {
		fmt.Fprintf(&b, "  %s %s  %s %s  %s %s\n",
			labelColor.Sprint("Redis:"), valueColor.Sprint(report.ServerInfo.Version),
			labelColor.Sprint("OS:"), valueColor.Sprint(report.ServerInfo.Os),
			labelColor.Sprint("Uptime:"), valueColor.Sprint(formatUptime(report.ServerInfo.Uptime)),
		)
	}
	fmt.Fprintf(&b, "  %s %s  %s %s\n",
		labelColor.Sprint("Prefixes:"), valueColor.Sprint(strings.Join(report.Prefixes, ", ")),
		labelColor.Sprint("Scanned:"), valueColor.Sprint(FormatInt(report.ScannedCount)+" keys"),
	)

	// Summary
	accentColor.Fprintf(&b, "\n  ── %s ──\n\n", "Summary")
	t := table.NewWriter()
	t.SetStyle(table.StyleBold)
	t.Style().Box.PaddingLeft = "  "
	t.Style().Box.PaddingRight = " "
	t.Style().Options.SeparateRows = false
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = false
	t.Style().Options.SeparateColumns = false

	t.AppendRow(table.Row{
		labelColor.Sprint("  Total Keys:"),
		valueColor.Sprint(FormatInt(report.Summary.TotalKeys)),
	})
	t.AppendRow(table.Row{
		labelColor.Sprint("  Total Memory:"),
		keyColor.Sprint(FormatSize(report.Summary.TotalMemory)),
	})
	t.AppendRow(table.Row{
		labelColor.Sprint("  Avg Key Size:"),
		valueColor.Sprint(FormatSize(report.Summary.AvgKeySize)),
	})
	t.AppendRow(table.Row{
		labelColor.Sprint("  Min Key Size:"),
		valueColor.Sprint(FormatSize(report.Summary.MinKeySize)),
	})
	t.AppendRow(table.Row{
		labelColor.Sprint("  Max Key Size:"),
		keyColor.Sprint(FormatSize(report.Summary.MaxKeySize)),
	})
	b.WriteString(t.Render())
	b.WriteString("\n")

	// By Type
	b.WriteString(renderTypeTable(report.ByType))

	// By Prefix
	b.WriteString(renderPrefixTable(report.ByPrefix))

	// Top N
	b.WriteString(renderTopKeys(report.TopKeys))

	return b.String()
}

func renderTypeTable(stats []models.TypeStats) string {
	var b strings.Builder
	if len(stats) == 0 {
		return ""
	}

	accentColor.Fprintf(&b, "\n  ── %s ──\n\n", "By Type")
	t := table.NewWriter()
	t.Style().Box.PaddingLeft = "  "
	t.Style().Options.SeparateRows = false
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = true

	t.AppendHeader(table.Row{"Type", "Count", "Memory", "Avg", "Min", "Max"})
	for _, s := range stats {
		t.AppendRow(table.Row{
			s.Type,
			FormatInt(s.Count),
			FormatSize(s.Total),
			FormatSize(s.Avg),
			FormatSize(s.Min),
			FormatSize(s.Max),
		})
	}

	b.WriteString(t.Render())
	b.WriteString("\n")
	return b.String()
}

func renderPrefixTable(stats []models.PrefixStats) string {
	var b strings.Builder
	if len(stats) == 0 {
		return ""
	}

	accentColor.Fprintf(&b, "\n  ── %s ──\n\n", "By Prefix")
	t := table.NewWriter()
	t.Style().Box.PaddingLeft = "  "
	t.Style().Options.SeparateRows = false
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = true

	t.AppendHeader(table.Row{"Prefix", "Count", "Memory", "Types"})
	for _, s := range stats {
		typeStrs := make([]string, 0, len(s.Types))
		for k, v := range s.Types {
			typeStrs = append(typeStrs, fmt.Sprintf("%s:%d", k, v))
		}
		t.AppendRow(table.Row{
			s.Prefix,
			FormatInt(s.Count),
			FormatSize(s.Total),
			strings.Join(typeStrs, ", "),
		})
	}

	b.WriteString(t.Render())
	b.WriteString("\n")
	return b.String()
}

func renderTopKeys(keys []models.KeyInfo) string {
	var b strings.Builder
	if len(keys) == 0 {
		return ""
	}

	accentColor.Fprintf(&b, "\n  ── %s ──\n\n", "Top Largest Keys")
	t := table.NewWriter()
	t.Style().Box.PaddingLeft = "  "
	t.Style().Options.SeparateRows = false
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = true

	t.AppendHeader(table.Row{"#", "Key", "Type", "Size", "Idle"})
	for i, k := range keys {
		idleStr := fmt.Sprintf("%ds", k.IdleTime)
		if k.IdleTime < 0 {
			idleStr = "N/A"
		} else if k.IdleTime > 3600 {
			idleStr = fmt.Sprintf("%dh", k.IdleTime/3600)
		} else if k.IdleTime > 60 {
			idleStr = fmt.Sprintf("%dm", k.IdleTime/60)
		}
		t.AppendRow(table.Row{
			i + 1,
			text.WrapSoft(keyColor.Sprint(k.Key), 60),
			k.Type,
			FormatSize(k.Size),
			idleStr,
		})
	}

	b.WriteString(t.Render())
	b.WriteString("\n")
	return b.String()
}

func formatUptime(seconds int64) string {
	if seconds <= 0 {
		return "N/A"
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh", hours)
}
