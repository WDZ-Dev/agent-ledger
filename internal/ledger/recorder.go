package ledger

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Recorder provides non-blocking, asynchronous usage recording.
// It buffers records in a channel and writes them via background workers.
type Recorder struct {
	ch      chan *UsageRecord
	ledger  Ledger
	wg      sync.WaitGroup
	logger  *slog.Logger
	dropped atomic.Int64
	closed  sync.Once
}

// NewRecorder starts background workers that drain records into the ledger.
func NewRecorder(ledger Ledger, bufSize, workers int, logger *slog.Logger) *Recorder {
	if bufSize <= 0 {
		bufSize = 10000
	}
	if workers <= 0 {
		workers = 4
	}

	r := &Recorder{
		ch:     make(chan *UsageRecord, bufSize),
		ledger: ledger,
		logger: logger,
	}

	for range workers {
		r.wg.Add(1)
		go r.worker()
	}

	return r
}

// Record enqueues a usage record for async persistence.
// If the buffer is full the record is dropped and counted.
func (r *Recorder) Record(record *UsageRecord) {
	select {
	case r.ch <- record:
	default:
		r.dropped.Add(1)
		r.logger.Warn("usage record dropped, buffer full")
	}
}

// Dropped returns the number of records that were dropped because the
// buffer was full.
func (r *Recorder) Dropped() int64 {
	return r.dropped.Load()
}

// Close signals workers to stop and waits for them to drain.
// Safe to call multiple times.
func (r *Recorder) Close() {
	r.closed.Do(func() {
		close(r.ch)
		r.wg.Wait()
	})
}

func (r *Recorder) worker() {
	defer r.wg.Done()
	for record := range r.ch {
		if err := r.ledger.RecordUsage(context.Background(), record); err != nil {
			r.logger.Error("failed to record usage", "error", err, "id", record.ID)
		}
	}
}
