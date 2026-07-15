package scanner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"github.com/schollz/progressbar/v3"

	"github.com/lihongjie0209/redis-analyze/internal/models"
)

const scanCount = 100 // keys per SCAN call

// ScanResult holds the result of a scan operation.
type ScanResult struct {
	Keys  []models.KeyInfo
	Total int64
}

// Scan performs a complete scan and returns all keys at once.
func Scan(ctx context.Context, client redis.UniversalClient, opts models.ScanOptions) (*ScanResult, error) {
	results := make(chan models.KeyInfo, 1000)
	errCh := make(chan error, 1)

	go func() {
		defer close(results)
		errCh <- ScanStream(ctx, client, opts, results)
	}()

	var allKeys []models.KeyInfo
	for info := range results {
		allKeys = append(allKeys, info)
	}

	err := <-errCh
	return &ScanResult{Keys: allKeys, Total: int64(len(allKeys))}, err
}

// ScanStream streams scanned keys into the provided channel.
// results channel is NOT closed by this function; the caller controls it.
func ScanStream(ctx context.Context, client redis.UniversalClient, opts models.ScanOptions, results chan<- models.KeyInfo) error {
	var (
		total    int64
		mu       sync.Mutex
		errorsCh = make(chan error, 100)
	)

	// Progress bar
	var bar *progressbar.ProgressBar
	if !opts.NoProgress {
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("Scanning Redis keys..."),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(100),
			progressbar.OptionShowCount(),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionEnableColorCodes(true),
		)
	}

	switch opts.Mode {
	case "cluster":
		clusterClient, ok := client.(*redis.ClusterClient)
		if !ok {
			return fmt.Errorf("expected ClusterClient but got %T", client)
		}
		err := clusterClient.ForEachMaster(ctx, func(ctx context.Context, node *redis.Client) error {
			for _, prefix := range opts.Prefixes {
				if err := scanNodeStream(ctx, node, prefix, opts, bar, &total, &mu, errorsCh, results); err != nil {
					return fmt.Errorf("node %s prefix %s: %w", node.Options().Addr, prefix, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

	default:
		for _, prefix := range opts.Prefixes {
			if err := scanNodeStream(ctx, client, prefix, opts, bar, &total, &mu, errorsCh, results); err != nil {
				return fmt.Errorf("prefix %s: %w", prefix, err)
			}
		}
	}

	close(errorsCh)
	var errs []string
	for e := range errorsCh {
		errs = append(errs, e.Error())
	}

	if bar != nil {
		_ = bar.Finish()
		fmt.Println()
	}

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors: %s", len(errs), strings.Join(errs, "; "))
	}
	return nil
}

// defaultBatchSize returns the configured batch size, defaulting to 50.
func defaultBatchSize(opts models.ScanOptions) int {
	if opts.BatchSize > 0 {
		return opts.BatchSize
	}
	return 50
}

// scanNodeStream iterates SCAN for a prefix and sends each key info to the channel.
// Key info is fetched via a configurable processor chain (Lua → pipeline → sequential).
func scanNodeStream(
	ctx context.Context,
	client redis.Cmdable,
	prefix string,
	opts models.ScanOptions,
	bar *progressbar.ProgressBar,
	total *int64,
	mu *sync.Mutex,
	errorsCh chan<- error,
	results chan<- models.KeyInfo,
) error {
	iter := client.Scan(ctx, 0, prefix, scanCount).Iterator()
	batchSize := defaultBatchSize(opts)

	var batch []string

	// flushBatch tries the processor chain for the current batch.
	// If a processor fails, the next one in the chain is tried.
	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}

		infos, err := processBatch(ctx, client, batch, opts)
		if err != nil {
			// All processors failed — report each key as an error
			for _, k := range batch {
				errorsCh <- fmt.Errorf("error fetching key %q (all processors): %w", k, err)
			}
			if bar != nil {
				_ = bar.Add(len(batch))
			}
			atomic.AddInt64(total, int64(len(batch)))
			batch = batch[:0]
			return nil
		}

		for _, info := range infos {
			select {
			case results <- info:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		atomic.AddInt64(total, int64(len(infos)))
		if bar != nil {
			_ = bar.Add(len(infos))
		}
		batch = batch[:0]
		return nil
	}

	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= batchSize {
			if err := flushBatch(); err != nil {
				return err
			}
		}
	}

	if err := flushBatch(); err != nil {
		return err
	}
	return iter.Err()
}
