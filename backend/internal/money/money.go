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
//
// Multiplication is guarded before it happens: stake > MaxInt64/price means
// the exact product would overflow int64, and the caller gets ErrOverflow
// instead of a silently wrapped result.
func Payout(stakeMinor, priceMilli int64) (int64, error) {
	if stakeMinor <= 0 || priceMilli <= 0 {
		return 0, errors.New("money: stake and price must be positive")
	}

	// stake*price must not overflow. Both factors are positive here.
	const maxInt64 = int64(^uint64(0) >> 1)
	if stakeMinor > maxInt64/priceMilli {
		return 0, ErrOverflow
	}

	// The product fits, so the division is the only remaining operation.
	// Go integer division truncates toward zero, which for positive numbers
	// is exactly the floor the contract requires.
	return stakeMinor * priceMilli / 1000, nil
}

// ValidateStake checks that stake is within [min, max] inclusive.
func ValidateStake(stake, min, max int64) error {
	if min > max || stake < min || stake > max {
		return ErrStakeOutOfRange
	}
	return nil
}
