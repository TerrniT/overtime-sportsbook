package line_test

import (
	"math"
	"reflect"
	"testing"

	"sportsbook/backend/internal/line"
)

const epsilon = 1e-9

func approx(t *testing.T, got, want float64, what string) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("%s = %.15f, want %.15f", what, got, want)
	}
}

func sum(v []float64) float64 {
	var total float64
	for _, x := range v {
		total += x
	}
	return total
}

func TestDevigNormalizesAndSumsToOne(t *testing.T) {
	prices := []int64{1950, 3550, 4050}
	got := line.Devig(prices)

	if len(got) != len(prices) {
		t.Fatalf("Devig returned %d probabilities for %d prices", len(got), len(prices))
	}
	if math.Abs(sum(got)-1) > epsilon {
		t.Fatalf("Devig probabilities sum to %.15f, want 1", sum(got))
	}

	totalImplied := 1000/float64(prices[0]) + 1000/float64(prices[1]) + 1000/float64(prices[2])
	for i, price := range prices {
		want := (1000 / float64(price)) / totalImplied
		approx(t, got[i], want, "Devig probability")
	}
}

func TestDevigEqualPricesGiveEqualProbabilities(t *testing.T) {
	got := line.Devig([]int64{2000, 2000, 2000})
	if len(got) != 3 {
		t.Fatalf("Devig returned %d probabilities, want 3", len(got))
	}
	for i := range got {
		approx(t, got[i], 1.0/3.0, "equal-price probability")
	}
}

func TestDevigTwoOutcomes(t *testing.T) {
	got := line.Devig([]int64{1900, 2000})
	if len(got) != 2 {
		t.Fatalf("Devig returned %d probabilities, want 2", len(got))
	}
	if math.Abs(sum(got)-1) > epsilon {
		t.Fatalf("Devig probabilities sum to %.15f, want 1", sum(got))
	}
	assertRatios(t, got, []float64{1000.0 / 1900, 1000.0 / 2000})
}

func TestDevigDoesNotMutateInput(t *testing.T) {
	prices := []int64{1950, 3550, 4050}
	before := append([]int64(nil), prices...)

	line.Devig(prices)

	if !reflect.DeepEqual(prices, before) {
		t.Fatalf("Devig mutated its input: before %v, after %v", before, prices)
	}
}

func TestDevigEmpty(t *testing.T) {
	if got := line.Devig(nil); len(got) != 0 {
		t.Fatalf("Devig(nil) = %v, want empty", got)
	}
	if got := line.Devig([]int64{}); len(got) != 0 {
		t.Fatalf("Devig(empty) = %v, want empty", got)
	}
}

func TestBlendWeightedMean(t *testing.T) {
	sources := [][]float64{
		{0.50, 0.30, 0.20},
		{0.40, 0.35, 0.25},
	}
	weights := []float64{0.6, 0.4}

	got := line.Blend(sources, weights)
	want := []float64{0.46, 0.32, 0.22}
	if len(got) != len(want) {
		t.Fatalf("Blend returned %d probabilities, want %d", len(got), len(want))
	}
	for i := range want {
		approx(t, got[i], want[i], "blended probability")
	}
	if math.Abs(sum(got)-1) > epsilon {
		t.Fatalf("Blend result sums to %.15f, want 1", sum(got))
	}
}

func TestBlendEqualVectorsPassThrough(t *testing.T) {
	v := []float64{0.20, 0.30, 0.50}
	got := line.Blend([][]float64{v, append([]float64(nil), v...)}, []float64{0.6, 0.4})
	if len(got) != len(v) {
		t.Fatalf("Blend returned %d probabilities, want %d", len(got), len(v))
	}
	for i := range v {
		approx(t, got[i], v[i], "pass-through probability")
	}
}

func TestWithMarginKeepsMarginAndValidPrices(t *testing.T) {
	fair := line.Devig([]int64{1950, 3550, 4050})

	for _, marginBps := range []int{500, 300} {
		t.Run(marginName(marginBps), func(t *testing.T) {
			got := line.WithMargin(fair, marginBps)
			if len(got) != len(fair) {
				t.Fatalf("WithMargin returned %d prices, want %d", len(got), len(fair))
			}

			var implied float64
			for _, price := range got {
				assertValidPrice(t, price, "WithMargin price")
				implied += 1000 / float64(price)
			}

			// Flooring prices down (and clamping a favourite up) can only
			// raise implied probability, so the achieved margin is never
			// below the target.
			target := 1 + float64(marginBps)/10_000
			if implied < target-epsilon {
				t.Fatalf("implied sum = %.12f, want >= %.12f for marginBps=%d", implied, target, marginBps)
			}
		})
	}
}

func TestWithMarginPreservesOrdering(t *testing.T) {
	fair := []float64{0.50, 0.30, 0.20}
	got := line.WithMargin(fair, 500)
	if len(got) != 3 {
		t.Fatalf("WithMargin returned %d prices, want 3", len(got))
	}
	if !(got[0] < got[1] && got[1] < got[2]) {
		t.Fatalf("WithMargin reordered outcomes: %v (larger probability must give a smaller price)", got)
	}
}

func TestWithMarginEmpty(t *testing.T) {
	if got := line.WithMargin(nil, 500); len(got) != 0 {
		t.Fatalf("WithMargin(nil) = %v, want empty", got)
	}
	if got := line.WithMargin([]float64{}, 500); len(got) != 0 {
		t.Fatalf("WithMargin(empty) = %v, want empty", got)
	}
}

func TestWithMarginClampsHeavyFavourite(t *testing.T) {
	// 0.999 * 1.05 -> price ~0.953, which would floor to 950. The valid
	// minimum is 1050 and it must not become 0 or a negative value.
	got := line.WithMargin([]float64{0.999, 0.0005, 0.0005}, 500)
	if len(got) != 3 {
		t.Fatalf("WithMargin returned %d prices, want 3", len(got))
	}
	if got[0] != 1050 {
		t.Fatalf("heavy favourite price = %d, want clamped 1050", got[0])
	}
	for _, price := range got {
		assertValidPrice(t, price, "clamped price")
	}
}

func TestWithMarginClampsLongshot(t *testing.T) {
	got := line.WithMargin([]float64{0.000001, 0.4999995, 0.4999995}, 500)
	if len(got) != 3 {
		t.Fatalf("WithMargin returned %d prices, want 3", len(got))
	}
	if got[0] != 20_000 {
		t.Fatalf("longshot price = %d, want clamped 20000", got[0])
	}
	for _, price := range got {
		assertValidPrice(t, price, "clamped price")
	}
}

func assertRatios(t *testing.T, got, implied []float64) {
	t.Helper()
	var total float64
	for _, x := range implied {
		total += x
	}
	for i := range got {
		approx(t, got[i], implied[i]/total, "normalized probability")
	}
}

func assertValidPrice(t *testing.T, price int64, what string) {
	t.Helper()
	if price < 1050 || price > 20_000 {
		t.Fatalf("%s = %d, want within [1050, 20000]", what, price)
	}
	if price%10 != 0 {
		t.Fatalf("%s = %d, want a multiple of 10", what, price)
	}
}

func marginName(bps int) string {
	if bps == 500 {
		return "margin_500"
	}
	return "margin_300"
}
