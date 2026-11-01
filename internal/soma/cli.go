package soma

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const helpText string = `Soma - meditation and breathing in the terminal (Linux).

Usage:
  soma meditate [--duration 30m] [--interval 10m] [--volume 50] [--silent]
  soma meditate --track NAME [--duration 20m] [--volume 50]
  soma breathe [--cycles 30 | --duration 5m] [--inhale 4] [--hold-in 0]
               [--exhale 6] [--hold-out 0] [--volume 35] [--silent]
  soma tracks
  soma history [--days 7] [--kind meditation|breathing]
  soma config init
  soma config path
  soma help

Times use h, m and s, e.g. 20m or 2m35s. Breathing phases use seconds.
Duration-limited breathing completes the current cycle.
Breathing uses the full terminal. Space pauses or resumes the exercise.
YouTube plays audio only, without any additional bells.
Ctrl+C stops playback and saves the interrupted session.
`

// Run executes a Soma command.
func Run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(helpText)
		return nil
	}

	paths, err := appPaths()
	if err != nil {
		return err
	}

	if args[0] == "config" {
		return configCommand(paths.Config, args[1:])
	}

	if args[0] == "history" {
		return showHistory(paths.History, args[1:], time.Now())
	}

	config, err := loadConfig(paths.Config)
	if err != nil {
		return err
	}

	if args[0] == "tracks" {
		return listTracks(config.Tracks, paths.Config, args[1:])
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch args[0] {
	case "meditate":
		return meditate(ctx, paths, config, args[1:])
	case "breathe":
		return breathe(ctx, paths, config.Breathing, args[1:])
	default:
		return fmt.Errorf("unknown command %q, use soma help", args[0])
	}
}

func configCommand(path string, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: soma config init|path")
	}

	switch args[0] {
	case "path":
		fmt.Println(path)
		return nil
	case "init":
		if err := writeConfig(path); err != nil {
			return err
		}

		fmt.Println("Created", path)
		return nil
	default:
		return errors.New("usage: soma config init|path")
	}
}

func listTracks(tracks []Track, configPath string, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: soma tracks")
	}

	if len(tracks) == 0 {
		fmt.Println("No tracks. Run soma config init, then edit", configPath)
		return nil
	}

	for _, track := range tracks {
		fmt.Printf("%s | start %s | duration %s\n", track.Name, track.Start, track.Duration)
	}

	return nil
}

func parseFlags(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return err
	}

	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q, options must come before positional arguments", flags.Arg(0))
	}

	return nil
}

func flagWasSet(flags *flag.FlagSet, name string) bool {
	var found bool

	flags.Visit(func(item *flag.Flag) {
		if item.Name == name {
			found = true
		}
	})

	return found
}

func formatTime(duration time.Duration) string {
	var seconds int64 = int64(duration / time.Second)
	if seconds < 0 {
		seconds = 0
	}

	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
}

func validateVolume(volume int) error {
	if volume < 0 || volume > 100 {
		return errors.New("volume must be between 0 and 100")
	}

	return nil
}
