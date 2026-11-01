package soma

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

type Phase struct {
	Name    string
	Seconds int
	Tone    string
}

type breathingOptions struct {
	Config   BreathingConfig
	Duration time.Duration
	Silent   bool
}

func breathe(ctx context.Context, paths Paths, config BreathingConfig, args []string) error {
	options, err := parseBreathingOptions(config, args)
	if err != nil {
		return err
	}

	var settings BreathingConfig = options.Config
	var phases = []Phase{
		{Name: "Inhale", Seconds: settings.Inhale},
		{Name: "Hold after inhale", Seconds: settings.HoldIn},
		{Name: "Exhale", Seconds: settings.Exhale},
		{Name: "Hold after exhale", Seconds: settings.HoldOut},
	}

	if !options.Silent && settings.Volume > 0 {
		if err := requirePlayer(false); err != nil {
			return err
		}

		var frequencies = []float64{660, 550, 440, 330}

		for index := range phases {
			if phases[index].Seconds == 0 {
				continue
			}

			tone, err := createTone(frequencies[index], 0.22)
			if err != nil {
				return err
			}
			defer os.Remove(tone)

			phases[index].Tone = tone
		}
	}

	var session = Session{
		Kind:           "breathing",
		PlannedSeconds: int64(options.Duration / time.Second),
		Breathing:      &settings,
	}

	return withSession(ctx, paths, session, func(record *Session) error {
		return runBreathing(ctx, options, phases, record)
	})
}

func parseBreathingOptions(config BreathingConfig, args []string) (breathingOptions, error) {
	var options = breathingOptions{Config: config}
	var durationValue string

	flags := flag.NewFlagSet("breathe", flag.ContinueOnError)
	flags.IntVar(&options.Config.Cycles, "cycles", config.Cycles, "Number of full cycles (1-9999)")
	flags.IntVar(&options.Config.Inhale, "inhale", config.Inhale, "Inhale seconds (1-3600)")
	flags.IntVar(&options.Config.HoldIn, "hold-in", config.HoldIn, "Hold after inhale (0-3600 seconds)")
	flags.IntVar(&options.Config.Exhale, "exhale", config.Exhale, "Exhale seconds (1-3600)")
	flags.IntVar(&options.Config.HoldOut, "hold-out", config.HoldOut, "Hold after exhale (0-3600 seconds)")
	flags.IntVar(&options.Config.Volume, "volume", config.Volume, "Volume (0-100)")
	flags.StringVar(&durationValue, "duration", "", "Minimum duration, completes the current cycle")
	flags.BoolVar(&options.Silent, "silent", false, "Disable phase tones")

	if err := parseFlags(flags, args); err != nil {
		return options, err
	}

	if err := validateVolume(options.Config.Volume); err != nil {
		return options, err
	}

	if flagWasSet(flags, "duration") && flagWasSet(flags, "cycles") {
		return options, errors.New("choose either --duration or --cycles")
	}

	if err := validateBreathingPhases(options.Config); err != nil {
		return options, err
	}

	if flagWasSet(flags, "duration") {
		duration, err := parseDuration(durationValue, false)
		if err != nil {
			return options, err
		}

		options.Duration = duration
		options.Config.Cycles = 0
		return options, nil
	}

	if options.Config.Cycles < 1 || options.Config.Cycles > 9999 {
		return options, errors.New("cycles must be between 1 and 9999")
	}

	var settings BreathingConfig = options.Config
	var cycleSeconds int = settings.Inhale + settings.HoldIn + settings.Exhale + settings.HoldOut
	var cycleDuration time.Duration = time.Duration(cycleSeconds) * time.Second

	options.Duration = time.Duration(settings.Cycles) * cycleDuration
	if options.Duration > maxDuration {
		return options, errors.New("planned breathing session exceeds 168h")
	}

	return options, nil
}

func validateBreathingPhases(config BreathingConfig) error {
	var validInhale bool = config.Inhale >= 1 && config.Inhale <= 3600
	var validExhale bool = config.Exhale >= 1 && config.Exhale <= 3600
	var validHoldIn bool = config.HoldIn >= 0 && config.HoldIn <= 3600
	var validHoldOut bool = config.HoldOut >= 0 && config.HoldOut <= 3600

	if !validInhale || !validExhale || !validHoldIn || !validHoldOut {
		return errors.New("inhale and exhale must be 1-3600 seconds, holds must be 0-3600 seconds")
	}

	return nil
}

func runBreathing(ctx context.Context, options breathingOptions, phases []Phase, record *Session) error {
	view, err := openBreathingView()
	if err != nil {
		return err
	}
	defer view.close()

	var settings BreathingConfig = options.Config
	var clock = breathingClock{Started: time.Now()}

	for cycle := 1; ; cycle++ {
		for _, phase := range phases {
			if phase.Seconds == 0 {
				continue
			}

			if err := runPhase(ctx, phase, cycle, options, &clock, view); err != nil {
				return err
			}
		}

		record.CompletedCycles = cycle

		if settings.Cycles > 0 && cycle >= settings.Cycles {
			return nil
		}

		if settings.Cycles == 0 && clock.elapsed(time.Now()) >= options.Duration {
			return nil
		}
	}
}

func runPhase(ctx context.Context, phase Phase, cycle int, options breathingOptions, clock *breathingClock, view *breathingView) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var phaseDuration time.Duration = time.Duration(phase.Seconds) * time.Second
	var phaseStarted time.Duration = clock.elapsed(time.Now())
	var playbackDone <-chan struct{}
	var player *Player

	if phase.Tone != "" {
		var err error
		player, err = startPlayer(ctx, phase.Tone, options.Config.Volume, 0)
		if err != nil {
			return err
		}
		defer player.stop()

		playbackDone = player.done
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		pausePressed, err := view.readPause()
		if err != nil {
			return err
		}

		if pausePressed {
			clock.togglePause(time.Now())

			if clock.paused() && player != nil {
				player.stop()
				playbackDone = nil
			}
		}

		var elapsed time.Duration = clock.elapsed(time.Now())
		var remaining time.Duration = phaseDuration - (elapsed - phaseStarted)
		if remaining <= 0 && !clock.paused() {
			return nil
		}

		view.render(phase, cycle, options.Config.Cycles, remaining, elapsed, clock.paused())

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-playbackDone:
			if err := ctx.Err(); err != nil {
				return err
			}

			if player.err != nil {
				return fmt.Errorf("play phase tone: %w", player.err)
			}

			playbackDone = nil
		case <-ticker.C:
		}
	}
}
