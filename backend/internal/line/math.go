package line

import "math"

const (
	minPriceMilli = int64(1050)
	maxPriceMilli = int64(20000)
)

// Devig converts decimal prices in milli-odds to implied probabilities
// normalized so they sum to 1.
//
// A non-positive price has no meaningful implied probability; it is treated
// as zero. If nothing usable is left, the result is an all-zero vector of the
// same length, which callers can safely degrade on instead of panicking.
func Devig(pricesMilli []int64) []float64 {
	probs := make([]float64, len(pricesMilli))

	var total float64
	for i, price := range pricesMilli {
		if price <= 0 {
			continue
		}
		implied := 1000 / float64(price)
		probs[i] = implied
		total += implied
	}
	if total <= 0 {
		return probs
	}
	for i := range probs {
		probs[i] /= total
	}
	return probs
}

// Blend combines per-source fair probability vectors with weights.
//
// The vectors are averaged with the given weights and the result is
// normalized, so it sums to 1 even if the weights do not. Mismatched or empty
// input returns an empty slice rather than a partially blended vector.
func Blend(sources [][]float64, weights []float64) []float64 {
	if len(sources) == 0 || len(sources) != len(weights) {
		return []float64{}
	}
	size := len(sources[0])
	out := make([]float64, size)

	for i, source := range sources {
		if len(source) != size {
			return []float64{}
		}
		for j, prob := range source {
			out[j] += prob * weights[i]
		}
	}

	var total float64
	for _, prob := range out {
		total += prob
	}
	if total <= 0 {
		return out
	}
	for i := range out {
		out[i] /= total
	}
	return out
}

// WithMargin applies marginBps to fair probabilities and returns decimal
// prices in milli-odds floored to two decimals and clamped to the valid range.
func WithMargin(fair []float64, marginBps int) []int64 {
	if len(fair) == 0 {
		return []int64{}
	}
	prices := make([]int64, len(fair))
	for i, prob := range fair {
		adjusted := prob * (1 + float64(marginBps)/10_000)
		prices[i] = priceFromAdjustedProbability(adjusted)
	}
	return prices
}

// priceFromAdjustedProbability turns an adjusted (post-margin) probability
// into a displayable price: 1000/adj, floored to a multiple of 10, clamped to
// [1050, 20000]. Invalid probabilities degrade to the longest valid price.
func priceFromAdjustedProbability(adjusted float64) int64 {
	if adjusted <= 0 || math.IsNaN(adjusted) {
		return maxPriceMilli
	}
	raw := 1000 / adjusted
	if raw <= float64(minPriceMilli) {
		return minPriceMilli
	}
	if raw >= float64(maxPriceMilli) {
		return maxPriceMilli
	}
	price := int64(raw) / 10 * 10
	if price < minPriceMilli {
		return minPriceMilli
	}
	if price > maxPriceMilli {
		return maxPriceMilli
	}
	return price
}
