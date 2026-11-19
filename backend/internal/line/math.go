package line

// Devig converts decimal prices in milli-odds to implied probabilities
// normalized so they sum to 1.
func Devig(pricesMilli []int64) []float64 {
	// TODO: implement (line agent).
	return nil
}

// Blend combines per-source fair probability vectors with weights.
func Blend(sources [][]float64, weights []float64) []float64 {
	// TODO: implement (line agent).
	return nil
}

// WithMargin applies marginBps to fair probabilities and returns decimal
// prices in milli-odds floored to two decimals and clamped to the valid range.
func WithMargin(fair []float64, marginBps int) []int64 {
	// TODO: implement (line agent).
	return nil
}
