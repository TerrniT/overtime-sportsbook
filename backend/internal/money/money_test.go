package money_test

import (
	"errors"
	"math"
	"testing"

	"sportsbook/backend/internal/money"
)

func TestPayoutFloorsAndNeverRoundsUp(t *testing.T) {
	tests := []struct {
		name       string
		stake      int64
		priceMilli int64
		want       int64
	}{
		{name: "contract example", stake: 50_000, priceMilli: 1910, want: 95_500},
		{name: "single kopeck at minimum price", stake: 1, priceMilli: 1050, want: 1},
		{name: "fraction is truncated", stake: 3, priceMilli: 1050, want: 3},
		{name: "999 kopecks at 1.05 is 1048 not 1049", stake: 999, priceMilli: 1050, want: 1048},
		{name: "seven kopecks at 1.91 is 13 not 14", stake: 7, priceMilli: 1910, want: 13},
		{name: "three kopecks at 1.99 is 5 not 6", stake: 3, priceMilli: 1990, want: 5},
		{name: "maximum price", stake: 10_000, priceMilli: 20_000, want: 200_000},
		{name: "maximum allowed stake", stake: 500_000, priceMilli: 1910, want: 955_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := money.Payout(tt.stake, tt.priceMilli)
			if err != nil {
				t.Fatalf("Payout(%d, %d) returned unexpected error: %v", tt.stake, tt.priceMilli, err)
			}
			if got != tt.want {
				t.Fatalf("Payout(%d, %d) = %d, want %d (must floor, never round up)", tt.stake, tt.priceMilli, got, tt.want)
			}
			// The result can never exceed the exact real-valued payout.
			exact := float64(tt.stake) * float64(tt.priceMilli) / 1000.0
			if float64(got) > exact {
				t.Fatalf("Payout(%d, %d) = %d rounds up past %f", tt.stake, tt.priceMilli, got, exact)
			}
		})
	}
}

func TestPayoutOverflow(t *testing.T) {
	// MaxInt64 * 1910 does not fit in int64; the multiplication must be
	// detected and reported instead of silently wrapping.
	if _, err := money.Payout(math.MaxInt64, 1910); !errors.Is(err, money.ErrOverflow) {
		t.Fatalf("Payout(MaxInt64, 1910) error = %v, want ErrOverflow", err)
	}

	// Also overflow at the minimum valid price.
	if _, err := money.Payout(math.MaxInt64, 1050); !errors.Is(err, money.ErrOverflow) {
		t.Fatalf("Payout(MaxInt64, 1050) error = %v, want ErrOverflow", err)
	}
}

func TestPayoutNonPositiveInputs(t *testing.T) {
	// The contract only pins the happy path and overflow. It allows either a
	// rejection or a documented zero/non-positive result, but a non-positive
	// stake or price must never produce a positive payout.
	cases := []struct {
		name       string
		stake      int64
		priceMilli int64
	}{
		{name: "zero stake", stake: 0, priceMilli: 1910},
		{name: "negative stake", stake: -50_000, priceMilli: 1910},
		{name: "zero price", stake: 50_000, priceMilli: 0},
		{name: "negative price", stake: 50_000, priceMilli: -1910},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := money.Payout(tt.stake, tt.priceMilli)
			if err != nil {
				return // explicit rejection is fine
			}
			if got > 0 {
				t.Fatalf("Payout(%d, %d) = %d with nil error; must not pay out on non-positive input", tt.stake, tt.priceMilli, got)
			}
			if tt.stake == 0 || tt.priceMilli == 0 {
				if got != 0 {
					t.Fatalf("Payout(%d, %d) = %d, want 0", tt.stake, tt.priceMilli, got)
				}
			}
		})
	}
}

func TestValidateStakeBoundaries(t *testing.T) {
	const minStake, maxStake int64 = 10_000, 500_000

	for _, stake := range []int64{minStake, maxStake, minStake + 1, maxStake - 1, 50_000} {
		if err := money.ValidateStake(stake, minStake, maxStake); err != nil {
			t.Errorf("ValidateStake(%d, %d, %d) = %v, want nil (bounds are inclusive)", stake, minStake, maxStake, err)
		}
	}

	for _, stake := range []int64{minStake - 1, maxStake + 1, 0, -1, -10_000, 1_000_000} {
		err := money.ValidateStake(stake, minStake, maxStake)
		if !errors.Is(err, money.ErrStakeOutOfRange) {
			t.Errorf("ValidateStake(%d, %d, %d) = %v, want ErrStakeOutOfRange", stake, minStake, maxStake, err)
		}
	}
}

func TestValidateStakeInvertedLimits(t *testing.T) {
	// min > max describes an empty interval: every stake is out of range.
	for _, stake := range []int64{0, 100, 200, 300, 1000} {
		err := money.ValidateStake(stake, 300, 100)
		if !errors.Is(err, money.ErrStakeOutOfRange) {
			t.Errorf("ValidateStake(%d, 300, 100) = %v, want ErrStakeOutOfRange", stake, err)
		}
	}
}
