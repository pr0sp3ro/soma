package soma

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

type meditationOptions struct {
	Duration     time.Duration
	Interval     time.Duration
	Volume       int
	Silent       bool
	Track        Track
	StartSeconds int
}

func meditate(ctx context.Context, paths Paths, config Config, args []string) error {
	options, err := parseMeditationOptions(config, args)
	if err != nil {
		return err
	}

	var bell string
	var youtube bool = options.Track.Name != ""
	var audibleTimer bool = !options.Silent && options.Volume > 0

	if youtube || audibleTimer {
		if err := requirePlayer(youtube); err != nil {
			return err
		}
	}

	if !youtube && audibleTimer {
		bell, err = createTone(528, 1.8)
		if err != nil {
			return err
		}
		defer os.Remove(bell)
	}

	var session = Session{
		Kind:           "meditation",
		PlannedSeconds: int64(options.Duration / time.Second),
		Track:          options.Track.Name,
		TrackURL:       options.Track.URL,
		StartSeconds:   options.StartSeconds,
	}

	return withSession(ctx, paths, session, func(record *Session) error {
		fmt.Printf("Meditation: %s. Ctrl+C to stop.\n", options.Duration)

		if youtube {
			return runYouTubeMeditation(ctx, options, record.StartedAt)
		}

		return runMeditationTimer(ctx, options, record.StartedAt, bell)
	})
}

func parseMeditationOptions(config Config, args []string) (meditationOptions, error) {
	var options meditationOptions
	var durationValue string
	var intervalValue string
	var trackName string

	flags := flag.NewFlagSet("meditate", flag.ContinueOnError)
	flags.StringVar(&durationValue, "duration", config.Meditation.Duration, "Session duration")
	flags.StringVar(&intervalValue, "interval", config.Meditation.Interval, "Reminder interval, 0s disables reminders")
	flags.IntVar(&options.Volume, "volume", config.Meditation.Volume, "Volume (0-100)")
	flags.BoolVar(&options.Silent, "silent", false, "Disable timer bells")
	flags.StringVar(&trackName, "track", "", "Configured YouTube track name")

	if err := parseFlags(flags, args); err != nil {
		return options, err
	}

	if err := validateVolume(options.Volume); err != nil {
		return options, err
	}

	if trackName != "" {
		if options.Silent || flagWasSet(flags, "interval") {
			return options, errors.New("--silent and --interval apply only to timer meditation")
		}

		track, err := findTrack(config.Tracks, trackName)
		if err != nil {
			return options, err
		}

		track.URL, err = youtubeURL(track.URL)
		if err != nil {
			return options, err
		}

		start, err := parseDuration(track.Start, true)
		if err != nil {
			return options, fmt.Errorf("track start: %w", err)
		}

		options.Track = track
		options.StartSeconds = int(start / time.Second)

		if !flagWasSet(flags, "duration") && track.Duration != "" {
			durationValue = track.Duration
		}
	}

	duration, err := parseDuration(durationValue, false)
	if err != nil {
		return options, err
	}
	options.Duration = duration

	if trackName == "" {
		interval, err := parseDuration(intervalValue, true)
		if err != nil {
			return options, err
		}
		options.Interval = interval
	}

	return options, nil
}

func findTrack(tracks []Track, name string) (Track, error) {
	var selected Track
	var matches int

	for _, track := range tracks {
		if track.Name == name {
			selected = track
			matches++
		}
	}

	if matches != 1 {
		return Track{}, fmt.Errorf("track %q must exist exactly once in config, use soma tracks", name)
	}

	return selected, nil
}

func runMeditationTimer(ctx context.Context, options meditationOptions, started time.Time, bell string) error {
	if err := playBell(ctx, bell, options.Volume); err != nil {
		return err
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var lastReminder int64

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		elapsed := time.Since(started)
		if elapsed >= options.Duration {
			return playBell(ctx, bell, options.Volume)
		}

		displayMeditationTime(elapsed, options.Duration)

		if options.Interval > 0 {
			var reminder int64 = int64(elapsed / options.Interval)
			var lateness time.Duration = elapsed - time.Duration(reminder)*options.Interval
			var shouldPlay bool = reminder > lastReminder && lateness <= 5*time.Second

			if reminder > lastReminder {
				lastReminder = reminder
			}

			if shouldPlay {
				if err := playBell(ctx, bell, options.Volume); err != nil {
					return err
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func runYouTubeMeditation(ctx context.Context, options meditationOptions, started time.Time) error {
	player, err := startPlayer(ctx, options.Track.URL, options.Volume, options.StartSeconds)
	if err != nil {
		return err
	}
	defer player.stop()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		elapsed := time.Since(started)
		if elapsed >= options.Duration {
			return nil
		}

		displayMeditationTime(elapsed, options.Duration)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-player.done:
			if err := ctx.Err(); err != nil {
				return err
			}

			if player.err != nil {
				return fmt.Errorf("YouTube playback failed: %w", player.err)
			}

			fmt.Print("\nRecording ended before the timer.\n")
			return nil
		case <-ticker.C:
		}
	}
}

func displayMeditationTime(elapsed time.Duration, duration time.Duration) {
	var remaining time.Duration = duration - elapsed
	var roundedRemaining time.Duration = remaining + time.Second - 1

	fmt.Printf("\rElapsed: %s | Remaining: %s   ", formatTime(elapsed), formatTime(roundedRemaining))
}
