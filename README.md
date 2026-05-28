# Morris Counter

A Go implementation of **Morris' probabilistic approximate counting algorithm**, invented by Robert Morris at Bell Labs in 1977.

## The Problem

How do you count a very large number of events when you only have a tiny amount of memory?

A normal 8-bit counter maxes out at 255. If you need to count millions of events, you need more bits — and in constrained environments (network hardware, embedded systems, databases tracking cardinalities), those bits are expensive.

## The Idea

Instead of storing the count `n`, store the exponent `v` where:

```
estimate = 2^v - 1
```

At v=0 the estimate is 0. At v=10 it is 1023. At v=20 it is over a million. A single `uint8` can now represent counts up to `2^255` — astronomically more than a normal 8-bit counter's 255.

The catch: you can't just increment `v` on every event, or the estimate would grow way too fast. So instead, you increment `v` **probabilistically** — with probability `1/2^v`. This keeps the expected value of the estimate tracking the true count:

```
E[estimate after increment] = previous estimate + 1
```

The proof is in Morris' original paper: [Counting large numbers of events in small registers](http://www.inf.ed.ac.uk/teaching/courses/exc/reading/morris.pdf).

## Trade-off

Morris counter is not exact. The relative error per individual estimate is ~50-70%. But the key guarantee is:

> **Relative error stays constant regardless of scale.**

The error at n=100 is roughly the same as at n=100,000,000. It does not blow up as the count grows. This bounded relative error is the core promise Morris provides.

| True Count | Mean Estimate | Mean Relative Error |
|------------|---------------|---------------------|
| 100        | ~100          | ~0.51               |
| 1,000      | ~993          | ~0.45               |
| 10,000     | ~10,094       | ~0.51               |
| 100,000    | ~99,718       | ~0.51               |

Space complexity: **O(log log n)** — the most aggressive compression possible for a counter with bounded relative error.

## Installation

```bash
go get github.com/MonarchRyuzaki/Morris-Counter-Go
```

## Usage

```go
mc := morriscounter.NewMorrisCounter()

// Increment on each event
for i := 0; i < 10000; i++ {
    mc.Incr()
}

// Get approximate count
fmt.Println(mc.Get()) // approximately 10000
```

## Implementation Details

### Storing the exponent as an integer

`v` is always a whole number — there is no such thing as "exponent 3.7" in this algorithm. Floats only appear transiently during the increment decision, never in stored state.

### The bit trick

Instead of computing `rand < 1.0/2^v` using floating point, we use a bitmask:

```go
mask := (uint64(1) << m.v) - 1
if rand.Uint64()&mask == 0 {
    m.v++
}
```

A random `uint64` has each bit independently set with probability 1/2. The probability that all `v` lowest bits are zero is exactly `1/2^v`. This avoids floating point precision loss (float32 loses precision around v=24) and is faster — just a bitwise AND and a comparison.

### Overflow guard

`v` is capped at 63. Beyond that, `uint64(1) << v` overflows, wrapping the mask to `0xFFFFFFFFFFFFFFFF` which would freeze the counter. In practice, `2^63 - 1 ≈ 9.2 × 10^18` — far beyond any realistic counting use case.

## When to Use This

Morris counter is a good fit when:
- You need to count very large numbers of events
- Memory is severely constrained
- An approximate answer within ~50-70% is acceptable
- You need a cardinality estimate, not an exact tally

It is **not** a good fit when:
- You need exact counts
- You need to decrement (the compression is one-way — information is lost on each probabilistic increment)
- Your count fits comfortably in a regular integer

## References

- [Counting large numbers of events in small registers — Robert Morris, 1977](http://www.inf.ed.ac.uk/teaching/courses/exc/reading/morris.pdf)
- [Space-Efficient Counting — Arpit Bhayani](https://arpitbhayani.me/blogs/morris-counter)
- [Approximate Counting Algorithm — Wikipedia](https://en.wikipedia.org/wiki/Approximate_counting_algorithm)