// Package morriscounter implements Morris' probabilistic approximate counting algorithm.
//
// Invented by Robert Morris at Bell Labs in 1977, this algorithm solves a deceptively
// simple problem: how do you count a very large number of events when you only have
// a tiny amount of memory?
//
// The key insight is to store the *exponent* instead of the count itself.
// Instead of storing n, we store v where n ≈ 2^v - 1.
// This means an 8-bit counter (v up to 255) can approximate counts up to 2^255,
// which is astronomically larger than what a normal 8-bit counter (max 255) could do.
//
// The trade-off: estimates are approximate with a relative error of ~50-70%.
// But crucially, this relative error stays *constant* regardless of scale —
// the error at n=100 is roughly the same as at n=100,000,000.
// This bounded relative error is the core guarantee Morris provides.
//
// Space complexity: O(log log n) — the most aggressive compression possible
// for a counter with bounded relative error.
package morriscounter

import "math/rand"

// MorrisCounter holds the internal state of a Morris approximate counter.
//
// v is the exponent — the only thing we store.
// The true count is never stored; it is approximated as 2^v - 1 on demand.
// A uint8 is sufficient since meaningful values of v only go up to 63
// (beyond that, 2^v overflows uint64).
type MorrisCounter struct {
	v uint8
}

// NewMorrisCounter returns a fresh counter with an initial estimate of 0.
// At v=0, Get() returns 2^0 - 1 = 0, which correctly represents "no events seen".
func NewMorrisCounter() MorrisCounter {
	return MorrisCounter{
		v: 0,
	}
}

// Get returns the approximate count of events seen so far.
//
// The estimate is 2^v - 1, derived from the internal exponent v.
// This is not the exact count — it is a probabilistic approximation.
// The expected relative error per individual estimate is ~50-70%, but
// averaged over many trials the mean converges close to the true count
// (the estimator is unbiased).
//
// Example estimates:
//
//	v=0  → 0
//	v=1  → 1
//	v=10 → 1023
//	v=20 → 1048575
func (m MorrisCounter) Get() uint64 {
	return (uint64(1) << m.v) - 1
}

// Incr probabilistically increments the internal exponent v.
//
// The core idea: instead of always incrementing v, we only increment it
// with probability 1/2^v. This keeps the expected value of 2^v - 1
// tracking the true count n, because:
//
//	E[2^(v+1) - 1 | increment happens with prob 1/2^v]
//	= (1/2^v) * 2^(v+1) + (1 - 1/2^v) * 2^v - 1
//	= 2 + 2^v - 1 - 1
//	= 2^v  →  which is (2^v - 1) + 1  →  previous estimate + 1 ✓
//
// So on average, each call to Incr moves the estimate forward by exactly 1,
// even though we are updating a compressed exponent, not a raw count.
//
// Implementation — the bit trick:
// Instead of using floating point (rand < 1.0/2^v), we use a bitmask.
// A random uint64 has each bit independently set with probability 1/2.
// The probability that all v lowest bits are zero is (1/2)^v = 1/2^v exactly.
// This avoids floating point precision loss (float32 loses precision around v=24)
// and is faster — just a bitwise AND and a comparison.
//
// The guard v >= 63 prevents uint64 overflow: 1 << 64 wraps to 0,
// which would make the mask 0xFFFFFFFFFFFFFFFF and effectively freeze the counter.
func (m *MorrisCounter) Incr() {
	if m.v >= 63 {
		return
	}

	// Build a mask of v ones in the lowest v bits.
	// E.g., v=3 → mask = 0b111 (decimal 7)
	// This represents the condition "all v lowest bits must be zero".
	mask := (uint64(1) << m.v) - 1

	// Sample the increment probabilistically.
	// rand.Uint64() produces a uniformly random 64-bit value.
	// ANDing with mask isolates the lowest v bits.
	// All v bits being zero happens with probability exactly 1/2^v.
	if rand.Uint64()&mask == 0 {
		m.v++
	}
}