package soma

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type Player struct {
	command *exec.Cmd
	done    chan struct{}
	err     error
}

func requirePlayer(youtube bool) error {
	if _, err := exec.LookPath("mpv"); err != nil {
		return errors.New("mpv is required for audio, install it or use --silent for timers and breathing")
	}

	if youtube {
		if _, err := exec.LookPath("yt-dlp"); err != nil {
			return errors.New("yt-dlp is required for YouTube playback")
		}
	}

	return nil
}

func startPlayer(ctx context.Context, source string, volume int, startSeconds int) (*Player, error) {
	var args []string = []string{
		"--no-config",
		"--no-video",
		"--input-terminal=no",
		"--term-osd=no",
		"--msg-level=all=error",
		"--no-input-default-bindings",
		"--no-resume-playback",
		"--ytdl-format=bestaudio/best",
		"--ytdl-raw-options=check-formats=",
		fmt.Sprintf("--volume=%d", volume),
		fmt.Sprintf("--start=%d", startSeconds),
		"--",
		source,
	}

	command := exec.CommandContext(ctx, "mpv", args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Kill the process group so cancellation also stops mpv's extractor helpers.
	command.Cancel = func() error {
		var err error = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}

		return err
	}

	command.Stderr = os.Stderr
	command.Stdout = os.Stderr

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start mpv: %w", err)
	}

	var player = &Player{
		command: command,
		done:    make(chan struct{}),
	}
	go func() {
		player.err = command.Wait()
		close(player.done)
	}()

	return player, nil
}

func (player *Player) stop() {
	select {
	case <-player.done:
		return
	default:
	}

	_ = player.command.Cancel()
	<-player.done
}

func playBell(ctx context.Context, path string, volume int) error {
	if path == "" {
		return ctx.Err()
	}

	player, err := startPlayer(ctx, path, volume, 0)
	if err != nil {
		return err
	}
	defer player.stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-player.done:
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if player.err != nil {
			return fmt.Errorf("play bell: %w", player.err)
		}

		return nil
	}
}
