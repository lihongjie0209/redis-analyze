package scanner

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/lihongjie0209/redis-analyze/internal/models"
)

// batchProcessor processes a batch of keys and returns their info.
type batchProcessor func(ctx context.Context, client redis.Cmdable, keys []string, opts models.ScanOptions) ([]models.KeyInfo, error)

// buildProcessors returns the processor chain based on scan mode.
func buildProcessors(scanMode string) []batchProcessor {
	switch scanMode {
	case "sequential":
		return []batchProcessor{processSequential}
	default: // "auto" or "pipeline"
		return []batchProcessor{processPipeline, processSequential}
	}
}

// ─── Pipeline processor ────────────────────────────────────────────────────

func processPipeline(ctx context.Context, client redis.Cmdable, keys []string, opts models.ScanOptions) ([]models.KeyInfo, error) {
	pipe := client.Pipeline()
	typeCmds := make([]*redis.StatusCmd, len(keys))
	memCmds := make([]*redis.IntCmd, len(keys))
	idleCmds := make([]*redis.DurationCmd, len(keys))

	for i, key := range keys {
		typeCmds[i] = pipe.Type(ctx, key)
		memCmds[i] = pipe.MemoryUsage(ctx, key, opts.Samples)
		idleCmds[i] = pipe.ObjectIdleTime(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	infos := make([]models.KeyInfo, len(keys))
	for i := range keys {
		info := models.KeyInfo{
			Key:      keys[i],
			Size:     -1,
			IdleTime: -1,
		}
		if kind, err := typeCmds[i].Result(); err == nil {
			info.Type = kind
		}
		if mem, err := memCmds[i].Result(); err == nil {
			info.Size = mem
		}
		if idle, err := idleCmds[i].Result(); err == nil {
			info.IdleTime = int64(idle / 1e9)
		}
		infos[i] = info
	}
	return infos, nil
}

// ─── Sequential processor (last resort) ────────────────────────────────────

func processSequential(ctx context.Context, client redis.Cmdable, keys []string, opts models.ScanOptions) ([]models.KeyInfo, error) {
	infos := make([]models.KeyInfo, len(keys))
	for i, key := range keys {
		info, err := processPipeline(ctx, client, []string{key}, opts)
		if err != nil {
			infos[i] = models.KeyInfo{Key: key, Size: -1, IdleTime: -1}
			continue
		}
		infos[i] = info[0]
	}
	return infos, nil
}

// ─── Processor chain ───────────────────────────────────────────────────────

// processBatch tries each processor in order until one succeeds.
func processBatch(ctx context.Context, client redis.Cmdable, keys []string, opts models.ScanOptions) ([]models.KeyInfo, error) {
	processors := buildProcessors(opts.ScanMode)

	var firstErr error
	for _, proc := range processors {
		infos, err := proc(ctx, client, keys, opts)
		if err == nil {
			return infos, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return nil, fmt.Errorf("all processors failed; first error: %w", firstErr)
}
