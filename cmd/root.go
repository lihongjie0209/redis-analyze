package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gituser/redis-analyze/internal/analyzer"
	"github.com/gituser/redis-analyze/internal/connector"
	"github.com/gituser/redis-analyze/internal/models"
	"github.com/gituser/redis-analyze/internal/reporter"
	"github.com/gituser/redis-analyze/internal/scanner"
)

var opts models.ScanOptions

var rootCmd = &cobra.Command{
	Use:   "redis-analyze",
	Short: "Redis memory analysis CLI tool",
	Long: `A CLI tool that connects to Redis, scans keys matching specified prefixes,
and reports memory usage statistics grouped by key type and namespace.

Supports standalone, cluster, and sentinel Redis deployment modes.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(opts.Timeout)*time.Second*60)
		defer cancel()

		if opts.Mode != "standalone" && len(opts.Addrs) == 0 {
			opts.Addrs = []string{fmt.Sprintf("%s:%d", opts.Host, opts.Port)}
		}
		if len(opts.Prefixes) == 0 {
			opts.Prefixes = []string{"*"}
		}

		startTime := time.Now()
		fmt.Fprintf(os.Stderr, "\nConnecting to Redis (%s mode)...\n", opts.Mode)

		client, cleanup, err := connector.Connect(ctx, opts)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer cleanup()

		serverInfo := connector.FetchServerInfo(ctx, client, opts)

		// Periodic ticker for intermediate reports
		var ticker *time.Ticker
		var tickerCh <-chan time.Time
		if opts.ReportInterval > 0 {
			ticker = time.NewTicker(time.Duration(opts.ReportInterval) * time.Second)
			tickerCh = ticker.C
			if !opts.NoProgress {
				fmt.Fprintf(os.Stderr, "Intermediate reports every %ds\n\n", opts.ReportInterval)
			}
		}

		results := make(chan models.KeyInfo, 1000)
		errCh := make(chan error, 1)

		go func() {
			defer close(results)
			errCh <- scanner.ScanStream(ctx, client, opts, results)
		}()

		var allKeys []models.KeyInfo
		var reportID int

	loop:
		for {
			select {
			case info, ok := <-results:
				if !ok {
					break loop
				}
				allKeys = append(allKeys, info)
			case <-tickerCh:
				reportID++
				printIntermediateReport(reportID, allKeys, serverInfo, opts, startTime)
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if ticker != nil {
			ticker.Stop()
		}

		scanErr := <-errCh
		duration := time.Since(startTime)

		if opts.ReportInterval > 0 {
			fmt.Fprintln(os.Stderr, "")
		}

		report := analyzer.Analyze(allKeys, serverInfo, opts, duration)
		outputReport(report)

		if scanErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: scan completed with errors: %v\n", scanErr)
		}
		return nil
	},
}

// printIntermediateReport outputs a partial report during scanning.
func printIntermediateReport(id int, keys []models.KeyInfo, serverInfo models.ServerInfo, opts models.ScanOptions, startTime time.Time) {
	duration := time.Since(startTime)
	report := analyzer.Analyze(keys, serverInfo, opts, duration)

	if opts.Format == "json" {
		fmt.Fprintf(os.Stderr, "\n--- Intermediate Report #%d (scanned: %d, elapsed: %v) ---\n",
			id, len(keys), duration.Round(time.Millisecond))
		return // JSON final report only
	}
	if opts.Format == "csv" {
		fmt.Fprintf(os.Stderr, "\n--- Intermediate Report #%d (scanned: %d, elapsed: %v) ---\n",
			id, len(keys), duration.Round(time.Millisecond))
		return
	}

	// Table: print compact summary
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  ═══════ Intermediate Report #%d ═══════\n", id)
	fmt.Fprintf(os.Stderr, "  Scanned: %d keys  |  Elapsed: %v\n", len(keys), duration.Round(time.Millisecond))
	fmt.Fprintf(os.Stderr, "  Total Memory: %s  |  Avg: %s  |  Max: %s\n",
		reporter.FormatSize(report.Summary.TotalMemory),
		reporter.FormatSize(report.Summary.AvgKeySize),
		reporter.FormatSize(report.Summary.MaxKeySize))
	fmt.Fprintf(os.Stderr, "  By Type: ")
	for i, s := range report.ByType {
		if i > 0 {
			fmt.Fprintf(os.Stderr, ", ")
		}
		fmt.Fprintf(os.Stderr, "%s: %s/%s", s.Type, reporter.FormatInt(s.Count), reporter.FormatSize(s.Total))
	}
	fmt.Fprintf(os.Stderr, "\n")
	if len(report.TopKeys) > 0 {
		fmt.Fprintf(os.Stderr, "  Top Key: %s (%s, %s)\n",
			report.TopKeys[0].Key, report.TopKeys[0].Type, reporter.FormatSize(report.TopKeys[0].Size))
	}
	fmt.Fprintf(os.Stderr, "\n")
}

// outputReport prints the final report to stdout.
func outputReport(report models.Report) {
	switch opts.Format {
	case "json":
		fmt.Println(reporter.RenderJSON(report))
	case "csv":
		fmt.Println(reporter.RenderCSV(report))
	default:
		fmt.Print(reporter.RenderTable(report))
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&opts.Host, "host", "H", "127.0.0.1", "Redis server host")
	rootCmd.Flags().IntVarP(&opts.Port, "port", "p", 6379, "Redis server port")
	rootCmd.Flags().StringVarP(&opts.Password, "password", "a", "", "Redis password")
	rootCmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Redis ACL username (Redis 6.0+)")
	rootCmd.Flags().IntVarP(&opts.DB, "db", "n", 0, "Redis database number (standalone mode)")

	rootCmd.Flags().StringVarP(&opts.Mode, "mode", "m", "standalone", "Redis mode: standalone, cluster, sentinel")
	rootCmd.Flags().StringSliceVar(&opts.Addrs, "addrs", []string{}, "Cluster/sentinel node addresses (comma-separated)")
	rootCmd.Flags().StringVar(&opts.MasterName, "master-name", "mymaster", "Sentinel master name")
	rootCmd.Flags().StringVar(&opts.SentinelPass, "sentinel-password", "", "Sentinel connection password")

	rootCmd.Flags().StringSliceVarP(&opts.Prefixes, "prefix", "P", []string{}, "Key prefix patterns (repeatable, default: *)")
	rootCmd.Flags().BoolVar(&opts.TLS, "tls", false, "Enable TLS connection")
	rootCmd.Flags().IntVarP(&opts.Samples, "samples", "s", 5, "MEMORY USAGE sample count")
	rootCmd.Flags().IntVarP(&opts.TopN, "top", "t", 20, "Number of top largest keys to show")
	rootCmd.Flags().IntVarP(&opts.Concurrency, "concurrency", "c", 10, "Scan concurrency level")
	rootCmd.Flags().IntVar(&opts.Timeout, "timeout", 30, "Connection / scan timeout (seconds)")
	rootCmd.Flags().StringVarP(&opts.ScanMode, "scan-mode", "", "auto", "Scanning strategy: auto (pipeline→sequential), pipeline, sequential")

	rootCmd.Flags().StringVarP(&opts.Format, "format", "f", "table", "Output format: table, json, csv")
	rootCmd.Flags().StringVar(&opts.Separator, "separator", ":", "Key prefix separator for grouping")
	rootCmd.Flags().IntVarP(&opts.PrefixDepth, "depth", "d", 1, "Prefix grouping depth (e.g. depth=2 for a:b:c -> a:b)")
	rootCmd.Flags().BoolVar(&opts.NoProgress, "no-progress", false, "Disable progress bar")
	rootCmd.Flags().IntVar(&opts.ReportInterval, "report-interval", 0, "Seconds between intermediate reports (0 = disabled)")
	rootCmd.Flags().IntVar(&opts.BatchSize, "batch-size", 50, "Keys per pipeline batch (0 = default 50, 1 = sequential mode)")
}

func ValidateFlags() error {
	if opts.Mode != "standalone" && opts.Mode != "cluster" && opts.Mode != "sentinel" {
		return fmt.Errorf("invalid mode: %q (must be standalone, cluster, or sentinel)", opts.Mode)
	}
	if opts.Format != "table" && opts.Format != "json" && opts.Format != "csv" {
		return fmt.Errorf("invalid format: %q (must be table, json, or csv)", opts.Format)
	}
	if opts.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1")
	}
	if opts.Samples < 0 {
		return fmt.Errorf("samples must be >= 0")
	}
	if opts.PrefixDepth < 0 {
		return fmt.Errorf("depth must be >= 0")
	}
	if opts.Timeout < 1 {
		return fmt.Errorf("timeout must be >= 1")
	}
	if opts.ReportInterval < 0 {
		return fmt.Errorf("report-interval must be >= 0")
	}
	for _, p := range opts.Prefixes {
		if strings.Contains(p, "\x00") {
			return fmt.Errorf("prefix contains null byte: %q", p)
		}
	}
	return nil
}
