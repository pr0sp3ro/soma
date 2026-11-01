package soma

import "time"

type breathingClock struct {
	Started        time.Time
	PausedAt       time.Time
	PausedDuration time.Duration
}

func (clock *breathingClock) paused() bool {
	return !clock.PausedAt.IsZero()
}

func (clock *breathingClock) elapsed(now time.Time) time.Duration {
	if clock.paused() {
		now = clock.PausedAt
	}

	return now.Sub(clock.Started) - clock.PausedDuration
}

func (clock *breathingClock) togglePause(now time.Time) {
	if clock.paused() {
		clock.PausedDuration += now.Sub(clock.PausedAt)
		clock.PausedAt = time.Time{}
		return
	}

	clock.PausedAt = now
}
