package risk

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestMaxOrders(t *testing.T) {
	b := NewBreaker(2, 3)
	for i := 0; i < 2; i++ {
		if !b.Allow() {
			t.Fatalf("order %d should pass", i+1)
		}
	}
	if b.Allow() || !b.Tripped() {
		t.Fatal("third should trip")
	}
}

func TestConsecutiveFailures(t *testing.T) {
	b := NewBreaker(100, 2)
	b.Record(false)
	b.Record(true) // resets
	b.Record(false)
	if b.Tripped() {
		t.Fatal("not consecutive")
	}
	b.Record(false)
	if !b.Tripped() || b.Allow() {
		t.Fatal("should be tripped and stay tripped")
	}
}

func TestConcurrentAllowNeverExceedsLimit(t *testing.T) {
	b := NewBreaker(10, 3)
	var ok atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b.Allow() {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 10 {
		t.Fatalf("allowed %d, want 10", ok.Load())
	}
}
