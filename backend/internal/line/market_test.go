package line

import (
	"testing"
	"time"
)

func TestRenderClosesUnpriceableMarket(t *testing.T) {
	original := fixtures[0].base
	fixtures[0].base = [3]float64{0.97, 0.015, 0.015}
	t.Cleanup(func() { fixtures[0].base = original })

	e := NewEngine(Config{
		Seed:      7,
		MarginBps: 500,
		Tick:      time.Second,
		Now:       func() time.Time { return time.Unix(1_700_000_000, 0) },
	})

	snap := e.Snapshot()
	if len(snap.Matches) != len(fixtures)-1 {
		t.Fatalf("line has %d matches, want %d: the 97%% favourite market must be closed", len(snap.Matches), len(fixtures)-1)
	}
	for _, match := range snap.Matches {
		if match.ID == fixtures[0].id {
			t.Fatalf("market %q is still in the line", fixtures[0].id)
		}
	}
}
