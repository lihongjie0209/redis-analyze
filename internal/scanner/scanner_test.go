package scanner

import (
	"testing"

	"github.com/gituser/redis-analyze/internal/models"
)

// ─── defaultBatchSize ──────────────────────────────────────────────────────

func TestDefaultBatchSize(t *testing.T) {
	t.Run("zero returns default", func(t *testing.T) {
		opts := models.ScanOptions{BatchSize: 0}
		if got := defaultBatchSize(opts); got != 50 {
			t.Errorf("expected 50, got %d", got)
		}
	})

	t.Run("configured value returned", func(t *testing.T) {
		opts := models.ScanOptions{BatchSize: 100}
		if got := defaultBatchSize(opts); got != 100 {
			t.Errorf("expected 100, got %d", got)
		}
	})

	t.Run("negative treated as zero", func(t *testing.T) {
		opts := models.ScanOptions{BatchSize: -5}
		if got := defaultBatchSize(opts); got != 50 {
			t.Errorf("expected 50, got %d", got)
		}
	})
}
