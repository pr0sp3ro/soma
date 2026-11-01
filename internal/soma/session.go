package soma

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type Session struct {
	StartedAt       time.Time        `json:"started_at"`
	Kind            string           `json:"kind"`
	Status          string           `json:"status"`
	PlannedSeconds  int64            `json:"planned_seconds"`
	ElapsedSeconds  int64            `json:"elapsed_seconds"`
	Track           string           `json:"track,omitempty"`
	TrackURL        string           `json:"track_url,omitempty"`
	StartSeconds    int              `json:"start_seconds,omitempty"`
	Breathing       *BreathingConfig `json:"breathing,omitempty"`
	CompletedCycles int              `json:"completed_cycles,omitempty"`
}

func withSession(ctx context.Context, paths Paths, session Session, action func(*Session) error) error {
	if err := os.MkdirAll(filepath.Dir(paths.Lock), 0700); err != nil {
		return err
	}

	lock, err := os.OpenFile(paths.Lock, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()

	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return errors.New("another Soma session is running")
		}

		return fmt.Errorf("lock session: %w", err)
	}

	history, err := os.OpenFile(paths.History, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer history.Close()

	if err = ctx.Err(); err != nil {
		return err
	}

	// Wall time includes computer suspend, unlike the monotonic clock on Linux.
	session.StartedAt = time.Now().Round(0)
	err = action(&session)

	if err == nil {
		err = ctx.Err()
	}

	session.ElapsedSeconds = int64(time.Since(session.StartedAt) / time.Second)
	if session.ElapsedSeconds < 0 {
		session.ElapsedSeconds = 0
	}

	session.Status = "finished"
	if err != nil {
		session.Status = "failed"
	}

	if errors.Is(err, context.Canceled) {
		session.Status = "cancelled"
		err = context.Canceled
	}

	saveErr := appendHistory(history, session)
	printSessionSummary(session)

	if saveErr != nil {
		return errors.Join(err, fmt.Errorf("save history: %w", saveErr))
	}

	return err
}

func printSessionSummary(session Session) {
	fmt.Printf(
		"\n%s %s. Elapsed: %s",
		session.Kind,
		session.Status,
		formatTime(time.Duration(session.ElapsedSeconds)*time.Second),
	)

	if session.Breathing != nil {
		fmt.Printf(". Completed cycles: %d", session.CompletedCycles)
	}

	fmt.Println()
}
