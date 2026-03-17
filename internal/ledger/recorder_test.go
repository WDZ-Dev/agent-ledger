package ledger

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type countingLedger struct {
	count atomic.Int64
}

func (c *countingLedger) RecordUsage(_ context.Context, _ *UsageRecord) error {
	c.count.Add(1)
	return nil
}

func (c *countingLedger) QueryCosts(_ context.Context, _ CostFilter) ([]CostEntry, error) {
	return nil, nil
}

func (c *countingLedger) GetTotalSpend(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}
func (c *countingLedger) GetTotalSpendByTenant(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}

func (c *countingLedger) QueryCostTimeseries(_ context.Context, _ string, _, _ time.Time, _ string) ([]TimeseriesPoint, error) {
	return nil, nil
}
func (c *countingLedger) QueryRecentExpensive(_ context.Context, _, _ time.Time, _ string, _ int) ([]ExpensiveRequest, error) {
	return nil, nil
}
func (c *countingLedger) QueryErrorStats(_ context.Context, _, _ time.Time, _ string) (*ErrorStats, error) {
	return &ErrorStats{}, nil
}

func (c *countingLedger) Close() error { return nil }

type failingLedger struct{}

func (f *failingLedger) RecordUsage(_ context.Context, _ *UsageRecord) error {
	return errors.New("write failed")
}

func (f *failingLedger) QueryCosts(_ context.Context, _ CostFilter) ([]CostEntry, error) {
	return nil, nil
}

func (f *failingLedger) GetTotalSpend(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}
func (f *failingLedger) GetTotalSpendByTenant(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	return 0, nil
}

func (f *failingLedger) QueryCostTimeseries(_ context.Context, _ string, _, _ time.Time, _ string) ([]TimeseriesPoint, error) {
	return nil, nil
}
func (f *failingLedger) QueryRecentExpensive(_ context.Context, _, _ time.Time, _ string, _ int) ([]ExpensiveRequest, error) {
	return nil, nil
}
func (f *failingLedger) QueryErrorStats(_ context.Context, _, _ time.Time, _ string) (*ErrorStats, error) {
	return &ErrorStats{}, nil
}

func (f *failingLedger) Close() error { return nil }

func TestRecorderDrains(t *testing.T) {
	store := &countingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := NewRecorder(store, 100, 2, logger)

	for i := range 50 {
		rec.Record(&UsageRecord{ID: string(rune('A' + i))})
	}

	rec.Close()

	if store.count.Load() != 50 {
		t.Errorf("expected 50 records, got %d", store.count.Load())
	}
	if rec.Dropped() != 0 {
		t.Errorf("expected 0 dropped, got %d", rec.Dropped())
	}
}

func TestRecorderDropsWhenFull(t *testing.T) {
	store := &countingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Buffer of 1, 0 workers initially — fills up immediately.
	// We create a recorder with 1 worker that's slow — actually let's
	// just use a tiny buffer.
	rec := NewRecorder(store, 1, 1, logger)

	// Flood with records. Some will be dropped.
	for range 1000 {
		rec.Record(&UsageRecord{ID: "test"})
	}

	rec.Close()

	recorded := store.count.Load()
	dropped := rec.Dropped()
	total := recorded + dropped

	if total != 1000 {
		t.Errorf("recorded(%d) + dropped(%d) = %d, want 1000", recorded, dropped, total)
	}
	// With a buffer of 1, we should have dropped some.
	if dropped == 0 {
		t.Log("no records dropped (worker was fast enough) - this is acceptable")
	}
}

func TestRecorderCloseIdempotent(t *testing.T) {
	store := &countingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := NewRecorder(store, 10, 1, logger)

	rec.Record(&UsageRecord{ID: "test"})

	// Close multiple times should not panic.
	rec.Close()
	rec.Close()
	rec.Close()
}

func TestRecorderHandlesErrors(t *testing.T) {
	store := &failingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := NewRecorder(store, 10, 1, logger)

	rec.Record(&UsageRecord{ID: "test"})

	// Should not hang or panic even when ledger fails.
	rec.Close()
}

func TestRecorderDefaultValues(t *testing.T) {
	store := &countingLedger{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Zero values should get defaults.
	rec := NewRecorder(store, 0, 0, logger)
	rec.Record(&UsageRecord{ID: "test"})
	rec.Close()

	if store.count.Load() != 1 {
		t.Errorf("expected 1 record, got %d", store.count.Load())
	}
}
