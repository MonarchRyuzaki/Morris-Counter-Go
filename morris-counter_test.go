package morriscounter

import (
	"fmt"
	"math"
	"testing"
)

// Runs `trials` counters up to `n` increments each, returns mean estimate and relative error
func runTrials(n int, trials int) (meanEstimate float64, meanRelErr float64) {
	totalEstimate := 0.0
	totalRelErr := 0.0

	for i := 0; i < trials; i++ {
		mc := NewMorrisCounter()
		for j := 0; j < n; j++ {
			mc.Incr()
		}
		est := float64(mc.Get())
		totalEstimate += est
		totalRelErr += math.Abs(est-float64(n)) / float64(n)
	}

	return totalEstimate / float64(trials), totalRelErr / float64(trials)
}

// Core property: relative error should stay roughly constant across scales.
// Arpit's blog explicitly states this is what Morris guarantees.
func TestRelativeErrorIsRoughlyConstant(t *testing.T) {
	trials := 5000
	counts := []int{100, 1000, 10_000, 100_000}
	maxRelErr := 0.6 // 50% relative error tolerance — probabilistic counter, not exact

	fmt.Printf("%-12s %-16s %-16s\n", "True Count", "Mean Estimate", "Mean Rel Error")
	fmt.Printf("%-12s %-16s %-16s\n", "----------", "-------------", "--------------")

	prevRelErr := -1.0
	for _, n := range counts {
		meanEst, meanRelErr := runTrials(n, trials)
		fmt.Printf("%-12d %-16.2f %-16.4f\n", n, meanEst, meanRelErr)

		if meanRelErr > maxRelErr {
			t.Errorf("n=%d: relative error %.4f exceeds threshold %.2f", n, meanRelErr, maxRelErr)
		}

		// Relative error should not blow up as n grows (the whole point of Morris)
		if prevRelErr > 0 && meanRelErr > prevRelErr*2.5 {
			t.Errorf("n=%d: relative error %.4f blew up vs previous %.4f", n, meanRelErr, prevRelErr)
		}
		prevRelErr = meanRelErr
	}
}

// Mean estimate over many trials should converge close to the true count.
// Law of large numbers — averaged across trials, bias should be low.
func TestMeanEstimateConverges(t *testing.T) {
	trials := 10_000
	counts := []int{50, 500, 5000}
	tolerancePct := 0.15 // mean should be within 15% of true count

	for _, n := range counts {
		meanEst, _ := runTrials(n, trials)
		relDiff := math.Abs(meanEst-float64(n)) / float64(n)
		if relDiff > tolerancePct {
			t.Errorf("n=%d: mean estimate %.2f is %.2f%% off (threshold %.0f%%)",
				n, meanEst, relDiff*100, tolerancePct*100)
		}
	}
}

// Zero increments → estimate should be 0 (2^0 - 1 = 0)
func TestZeroIncrements(t *testing.T) {
	mc := NewMorrisCounter()
	if mc.Get() != 0 {
		t.Errorf("expected 0 on fresh counter, got %d", mc.Get())
	}
}

// Single increment → estimate should be 1 (2^1 - 1 = 1), always deterministic
// because at v=0, d = 1/2^0 = 1.0, so rand < 1.0 always
func TestSingleIncrementAlwaysBumps(t *testing.T) {
	for i := 0; i < 1000; i++ {
		mc := NewMorrisCounter()
		mc.Incr()
		if mc.Get() != 1 {
			t.Errorf("first increment should always produce estimate=1, got %d", mc.Get())
		}
	}
}

// Counter should not overflow or panic at high increment counts
func TestHighVolumeStability(t *testing.T) {
	mc := NewMorrisCounter()
	for i := 0; i < 1_000_000; i++ {
		mc.Incr()
	}
	est := mc.Get()
	// Just check it doesn't explode; rough sanity: within 10x of true
	if est == 0 || est > 10_000_000 {
		t.Errorf("estimate %d looks bogus for 1M increments", est)
	}
}