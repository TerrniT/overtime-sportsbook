package line_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"sportsbook/backend/internal/line"
)

var fixedNow = time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)

func testConfig() line.Config {
	return line.Config{
		Seed:      7,
		MarginBps: 500,
		Tick:      20 * time.Millisecond,
		Now:       func() time.Time { return fixedNow },
	}
}

func TestNewEngineInitialSnapshot(t *testing.T) {
	e := line.NewEngine(testConfig())
	snap := e.Snapshot()

	if snap.Seq != 0 {
		t.Fatalf("initial seq = %d, want 0", snap.Seq)
	}
	if !snap.UpdatedAt.Equal(fixedNow) {
		t.Fatalf("initial updatedAt = %s, want fixed clock %s", snap.UpdatedAt, fixedNow)
	}
	if snap.MarginBps < 500 {
		t.Fatalf("marginBps = %d, want at least the configured target 500", snap.MarginBps)
	}
	if n := len(snap.Matches); n < 6 || n > 8 {
		t.Fatalf("initial line has %d matches, want 6-8 (README describes 7)", n)
	}

	seen := map[string]bool{}
	for _, match := range snap.Matches {
		if match.ID == "" {
			t.Fatal("match id is empty")
		}
		if seen[match.ID] {
			t.Fatalf("duplicate match id %q", match.ID)
		}
		seen[match.ID] = true
		if match.Sport == "" || match.League == "" || match.HomeTeam == "" || match.AwayTeam == "" {
			t.Fatalf("match %q has an empty descriptive field: %+v", match.ID, match)
		}
		if match.StartsAt.IsZero() {
			t.Fatalf("match %q has a zero startsAt", match.ID)
		}

		if len(match.Outcomes) != 3 {
			t.Fatalf("match %q has %d outcomes, want 3", match.ID, len(match.Outcomes))
		}
		wantKeys := []string{"home", "draw", "away"}
		wantLabels := []string{"П1", "Х", "П2"}
		for i, outcome := range match.Outcomes {
			if outcome.Key != wantKeys[i] {
				t.Fatalf("match %q outcome %d key = %q, want %q (order home/draw/away)", match.ID, i, outcome.Key, wantKeys[i])
			}
			if outcome.Label != wantLabels[i] {
				t.Fatalf("match %q outcome %d label = %q, want %q", match.ID, i, outcome.Label, wantLabels[i])
			}
			if outcome.Trend != "flat" {
				t.Fatalf("initial trend for %s/%s = %q, want flat", match.ID, outcome.Key, outcome.Trend)
			}
			assertValidPrice(t, outcome.PriceMilli, "initial outcome price")
			if len(outcome.Sources) != 2 {
				t.Fatalf("match %q outcome %q has %d sources, want 2", match.ID, outcome.Key, len(outcome.Sources))
			}
			sourceIDs := map[string]bool{}
			for _, source := range outcome.Sources {
				if source.ID != "pulse" && source.ID != "sharp" {
					t.Fatalf("unexpected source id %q", source.ID)
				}
				if source.Label == "" {
					t.Fatalf("source %q has an empty label", source.ID)
				}
				if sourceIDs[source.ID] {
					t.Fatalf("duplicate source %q for %s/%s", source.ID, match.ID, outcome.Key)
				}
				sourceIDs[source.ID] = true
				assertValidPrice(t, source.PriceMilli, "initial source price")
			}
			if !sourceIDs["pulse"] || !sourceIDs["sharp"] {
				t.Fatalf("sources for %s/%s = %+v, want pulse and sharp", match.ID, outcome.Key, outcome.Sources)
			}
		}
	}
}

func TestEngineIsDeterministicForSameSeed(t *testing.T) {
	a := line.NewEngine(testConfig())
	b := line.NewEngine(testConfig())

	if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) {
		t.Fatal("two engines with the same seed and clock produced different initial snapshots")
	}

	for i := 0; i < 5; i++ {
		sa := a.Update()
		sb := b.Update()
		if !reflect.DeepEqual(sa, sb) {
			t.Fatalf("snapshots diverged at update %d:\nA: %+v\nB: %+v", i+1, sa, sb)
		}
	}
}

func TestDifferentSeedsProduceDifferentPrices(t *testing.T) {
	cfgA := testConfig()
	cfgA.Seed = 7
	cfgB := testConfig()
	cfgB.Seed = 8

	a := line.NewEngine(cfgA)
	b := line.NewEngine(cfgB)

	for i := 0; i < 5; i++ {
		a.Update()
		b.Update()
	}

	if pricesEqual(a.Snapshot(), b.Snapshot()) {
		t.Fatal("seeds 7 and 8 produced identical outcome prices after 5 updates")
	}
}

func TestUpdateIncrementsSeqAndValidatesTrends(t *testing.T) {
	e := line.NewEngine(testConfig())
	prev := e.Snapshot()
	changed := false

	for tick := 1; tick <= 5; tick++ {
		cur := e.Update()

		if cur.Seq != prev.Seq+1 {
			t.Fatalf("update %d: seq = %d, want %d", tick, cur.Seq, prev.Seq+1)
		}
		if !cur.UpdatedAt.Equal(fixedNow) {
			t.Fatalf("update %d: updatedAt = %s, want fixed clock", tick, cur.UpdatedAt)
		}
		if got := e.Snapshot(); got.Seq != cur.Seq {
			t.Fatalf("Snapshot() seq = %d after Update returned %d", got.Seq, cur.Seq)
		}
		if len(cur.Matches) != len(prev.Matches) {
			t.Fatalf("update %d: match count changed from %d to %d", tick, len(prev.Matches), len(cur.Matches))
		}

		prevPrice := priceMap(prev)
		for _, match := range cur.Matches {
			if len(match.Outcomes) != 3 {
				t.Fatalf("update %d: match %q has %d outcomes", tick, match.ID, len(match.Outcomes))
			}
			for _, outcome := range match.Outcomes {
				assertValidPrice(t, outcome.PriceMilli, "updated outcome price")

				key := match.ID + "|" + outcome.Key
				old, ok := prevPrice[key]
				if !ok {
					t.Fatalf("update %d: missing previous price for %q", tick, key)
				}
				wantTrend := "flat"
				switch {
				case outcome.PriceMilli > old:
					wantTrend = "up"
					changed = true
				case outcome.PriceMilli < old:
					wantTrend = "down"
					changed = true
				}
				if outcome.Trend != wantTrend {
					t.Fatalf("update %d: %s price %d -> %d has trend %q, want %q", tick, key, old, outcome.PriceMilli, outcome.Trend, wantTrend)
				}
				if outcome.Trend != "up" && outcome.Trend != "down" && outcome.Trend != "flat" {
					t.Fatalf("update %d: %s has invalid trend %q", tick, key, outcome.Trend)
				}
			}
		}

		prev = cur
	}

	if !changed {
		t.Fatal("no outcome price changed over 5 updates; the simulated line is frozen")
	}
}

func TestSubscribeReceivesNextUpdate(t *testing.T) {
	e := line.NewEngine(testConfig())
	ch, cancel := e.Subscribe()
	defer cancel()

	want := e.Update()

	select {
	case got := <-ch:
		if got.Seq != want.Seq {
			t.Fatalf("subscriber received seq %d, want %d", got.Seq, want.Seq)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("subscriber snapshot differs from Update result:\n got: %+v\nwant: %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive the snapshot from Update within 1s")
	}
}

func TestCancelStopsDeliveryWithoutPanic(t *testing.T) {
	e := line.NewEngine(testConfig())
	ch, cancel := e.Subscribe()

	// Drain the buffer and make sure the subscription is live.
	e.Update()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("subscription did not deliver before cancel")
	}

	cancel()

	// Must not panic even if the engine keeps ticking after unsubscribe.
	e.Update()

	select {
	case got, ok := <-ch:
		if ok {
			t.Fatalf("received snapshot after cancel: %+v", got)
		}
	case <-time.After(100 * time.Millisecond):
		// No delivery (open or closed channel): that is the contract.
	}
}

func TestSlowSubscriberDoesNotBlockUpdate(t *testing.T) {
	e := line.NewEngine(testConfig())
	_, cancel := e.Subscribe() // never drained
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			e.Update()
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Update blocked for more than 1s with an undrained subscriber")
	}
}

func TestRunBroadcastsAndStopsOnCancel(t *testing.T) {
	cfg := testConfig()
	cfg.Tick = 10 * time.Millisecond
	e := line.NewEngine(cfg)

	ch, unsubscribe := e.Subscribe()
	defer unsubscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Run(ctx)
	}()

	var seqs []int64
	deadline := time.After(500 * time.Millisecond)
	for len(seqs) < 2 {
		select {
		case snap := <-ch:
			if len(snap.Matches) == 0 {
				t.Fatal("Run broadcast an empty snapshot")
			}
			seqs = append(seqs, snap.Seq)
		case <-deadline:
			t.Fatalf("Run delivered %d snapshots in 500ms with a 10ms tick, want at least 2", len(seqs))
		}
	}
	if seqs[1] <= seqs[0] {
		t.Fatalf("Run seq did not increase: %v", seqs)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return within 1s after context cancellation")
	}
}

func pricesEqual(a, b line.Snapshot) bool {
	pa := priceMap(a)
	pb := priceMap(b)
	if len(pa) != len(pb) {
		return false
	}
	for key, price := range pa {
		if pb[key] != price {
			return false
		}
	}
	return true
}

func priceMap(snap line.Snapshot) map[string]int64 {
	out := make(map[string]int64)
	for _, match := range snap.Matches {
		for _, outcome := range match.Outcomes {
			out[match.ID+"|"+outcome.Key] = outcome.PriceMilli
		}
	}
	return out
}
