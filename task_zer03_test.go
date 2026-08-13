package zerolog

import (
	"sync"
	"sync/atomic"
	"testing"
)

// TestTaskZER03NopUpdateContextIsNoop verifies that UpdateContext on a shared
// Nop() logger is a no-op: the update function must not be invoked and no
// state may be written, so concurrent calls from multiple goroutines cannot
// race. Nop() returns a disabled logger that is a different value than the
// package-level disabledLogger singleton, so pointer-equality based detection
// (if any) must not be relied upon.
func TestTaskZER03NopUpdateContextIsNoop(t *testing.T) {
	l := Nop()
	testTaskZER03ConcurrentUpdateContextIsNoop(t, &l)
}

// TestTaskZER03ZeroValueUpdateContextIsNoop verifies that UpdateContext on a
// shared zero-value Logger is a no-op for the same reasons.
func TestTaskZER03ZeroValueUpdateContextIsNoop(t *testing.T) {
	var l Logger
	testTaskZER03ConcurrentUpdateContextIsNoop(t, &l)
}

// testTaskZER03ConcurrentUpdateContextIsNoop runs many goroutines that share a
// single logger and concurrently call UpdateContext, asserting that the update
// function is never invoked on a disabled logger. Together with -race this
// proves that no logger state is written.
func testTaskZER03ConcurrentUpdateContextIsNoop(t *testing.T, l *Logger) {
	const goroutines = 8
	const iterations = 50

	var called int32
	update := func(c Context) Context {
		atomic.AddInt32(&called, 1)
		return c.Str("task", "zer03")
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				l.UpdateContext(update)
			}
		}()
	}
	close(start)
	wg.Wait()

	if n := atomic.LoadInt32(&called); n != 0 {
		t.Fatalf("UpdateContext invoked update %d times on a disabled logger, want 0", n)
	}
}
