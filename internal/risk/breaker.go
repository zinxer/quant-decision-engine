// Package risk holds the global kill switch.
package risk

import "sync/atomic"

// Breaker halts trading once a limit is hit. Safe for concurrent use.
type Breaker struct {
	maxOrders, maxFailures int64
	orders, failures       atomic.Int64
	tripped                atomic.Bool
}

// NewBreaker trips after maxOrders orders or maxConsecutiveFailures failures in a row.
func NewBreaker(maxOrders, maxConsecutiveFailures int) *Breaker {
	return &Breaker{maxOrders: int64(maxOrders), maxFailures: int64(maxConsecutiveFailures)}
}

// Allow reserves an order slot; false means trading is halted.
func (b *Breaker) Allow() bool {
	if b.tripped.Load() {
		return false
	}
	if b.orders.Add(1) > b.maxOrders {
		b.tripped.Store(true)
		return false
	}
	return true
}

// Record reports an order outcome; consecutive failures trip the breaker.
func (b *Breaker) Record(ok bool) {
	if ok {
		b.failures.Store(0)
		return
	}
	if b.failures.Add(1) >= b.maxFailures {
		b.tripped.Store(true)
	}
}

// Tripped reports whether trading is halted.
func (b *Breaker) Tripped() bool { return b.tripped.Load() }

// Trip is the manual kill switch.
func (b *Breaker) Trip() { b.tripped.Store(true) }
