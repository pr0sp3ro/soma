package soma

import (
	"testing"
	"time"
)

func TestBreathingClockExcludesPauses(t *testing.T) {
	var started time.Time = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	var clock = breathingClock{Started: started}

	clock.togglePause(started.Add(2 * time.Second))

	if elapsed := clock.elapsed(started.Add(12 * time.Second)); elapsed != 2*time.Second {
		t.Fatalf("time advanced during pause: %v", elapsed)
	}

	clock.togglePause(started.Add(12 * time.Second))

	if clock.paused() {
		t.Fatal("clock should resume after the second toggle")
	}

	if elapsed := clock.elapsed(started.Add(15 * time.Second)); elapsed != 5*time.Second {
		t.Fatalf("expected 5 seconds of active time, got %v", elapsed)
	}

	clock.togglePause(started.Add(15 * time.Second))
	clock.togglePause(started.Add(25 * time.Second))

	if elapsed := clock.elapsed(started.Add(28 * time.Second)); elapsed != 8*time.Second {
		t.Fatalf("expected both pauses to be excluded, got %v", elapsed)
	}
}
