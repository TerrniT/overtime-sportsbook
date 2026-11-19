package money

import "errors"

var (
	// ErrOverflow means the payout does not fit into int64.
	ErrOverflow = errors.New("money: amount overflow")
	// ErrStakeOutOfRange means the stake is outside the configured limits.
	ErrStakeOutOfRange = errors.New("money: stake out of range")
)

// Payout returns the potential payout for a stake at a decimal price.
// It is stakeMinor * priceMilli / 1000, floored to whole minor units.
func Payout(stakeMinor, priceMilli int64) (int64, error) {
	// TODO: implement (money agent).
	return 0, errors.New("not implemented")
}

// ValidateStake checks that stake is within [min, max] inclusive.
func ValidateStake(stake, min, max int64) error {
	// TODO: implement (money agent).
	return errors.New("not implemented")
}
