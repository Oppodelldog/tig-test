package ccount

import (
	"sync/atomic"
	"github.com/rcrowley/go-metrics"
)

// NewConcurrentCounter constructs a new StandardCounter.
func NewConcurrentCounter() metrics.Counter {
	if metrics.UseNilMetrics {
		return NilCounter{}
	}
	return &StandardCounter{}
}

// CounterSnapshot is a read-only copy of another Counter.
type CounterSnapshot int64

// Clear panics.
func (CounterSnapshot) Clear() {
	panic("Clear called on a CounterSnapshot")
}

// Count returns the count at the time the snapshot was taken.
func (c CounterSnapshot) Count() int64 { return int64(c) }

// Dec panics.
func (CounterSnapshot) Dec(int64) {
	panic("Dec called on a CounterSnapshot")
}

// Inc panics.
func (CounterSnapshot) Inc(int64) {
	panic("Inc called on a CounterSnapshot")
}

// Snapshot returns the snapshot.
func (c CounterSnapshot) Snapshot() metrics.Counter { return c }

// NilCounter is a no-op Counter.
type NilCounter struct{}

// Clear is a no-op.
func (NilCounter) Clear() {}

// Count is a no-op.
func (NilCounter) Count() int64 { return 0 }

// Dec is a no-op.
func (NilCounter) Dec(i int64) {}

// Inc is a no-op.
func (NilCounter) Inc(i int64) {}

// Snapshot is a no-op.
func (NilCounter) Snapshot() metrics.Counter { return NilCounter{} }

// StandardCounter is the standard implementation of a Counter and uses the
// sync/atomic package to manage a single int64 value.
type StandardCounter struct {
	count    int64
	maxCount int64
}

// Clear sets the counter to zero.
func (c *StandardCounter) Clear() {
	count := atomic.LoadInt64(&c.count)
	atomic.StoreInt64(&c.maxCount, count)
}

// Count returns the maxCount which represents the maximum number of concurrent counts during since the last snapshot.
func (c *StandardCounter) Count() int64 {
	count := atomic.LoadInt64(&c.maxCount)
	c.Clear()
	return count
}

// Dec decrements the counter by the given amount.
func (c *StandardCounter) Dec(i int64) {
	atomic.AddInt64(&c.count, -i)
}

// Inc increments the counter and updates maxCount
func (c *StandardCounter) Inc(i int64) {
	atomic.AddInt64(&c.count, i)
	count := atomic.LoadInt64(&c.count)
	maxCount := atomic.LoadInt64(&c.maxCount)
	if count > maxCount {
		atomic.StoreInt64(&c.maxCount, count)
	}
}

// Snapshot returns a read-only copy of the counter.
func (c *StandardCounter) Snapshot() metrics.Counter {
	return CounterSnapshot(c.Count())
}
