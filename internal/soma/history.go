package soma

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"syscall"
	"time"
)

func showHistory(path string, args []string, now time.Time) error {
	var days int
	var kind string

	flags := flag.NewFlagSet("history", flag.ContinueOnError)
	flags.IntVar(&days, "days", 7, "Calendar days including today (1-9999)")
	flags.StringVar(&kind, "kind", "", "Filter: meditation or breathing")

	if err := parseFlags(flags, args); err != nil {
		return err
	}

	if days < 1 || days > 9999 {
		return errors.New("days must be between 1 and 9999")
	}

	if kind != "" && kind != "meditation" && kind != "breathing" {
		return errors.New("kind must be meditation or breathing")
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("No sessions yet.")
		return nil
	}

	if err != nil {
		return err
	}
	defer file.Close()

	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_SH); err != nil {
		return fmt.Errorf("lock history: %w", err)
	}

	var first string = now.AddDate(0, 0, 1-days).Format("2006-01-02")
	var today string = now.Format("2006-01-02")
	var totals map[string]int64 = make(map[string]int64)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	var line int
	fmt.Printf("Sessions: %s to %s (local time)\n", first, today)

	for scanner.Scan() {
		line++

		var session Session
		if err = json.Unmarshal(scanner.Bytes(), &session); err != nil {
			return fmt.Errorf("history line %d: %w", line, err)
		}

		var validElapsed bool = session.ElapsedSeconds >= 0 && session.ElapsedSeconds <= int64((1<<63-1)/time.Second)
		if session.StartedAt.IsZero() || !validElapsed {
			return fmt.Errorf("invalid history entry at line %d", line)
		}

		var started time.Time = session.StartedAt.In(now.Location())
		var day string = started.Format("2006-01-02")
		if day < first || day > today || (kind != "" && session.Kind != kind) {
			continue
		}

		printHistorySession(session, started)
		totals[day] += session.ElapsedSeconds
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	if len(totals) == 0 {
		fmt.Println("No sessions in this period.")
		return nil
	}

	var dates []string
	for day := range totals {
		dates = append(dates, day)
	}
	sort.Strings(dates)

	fmt.Println("\nDaily totals (including cancelled and failed sessions):")

	for _, day := range dates {
		fmt.Printf("%s  %s\n", day, formatTime(time.Duration(totals[day])*time.Second))
	}

	return nil
}

func appendHistory(file *os.File, session Session) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}

	if err := json.NewEncoder(file).Encode(session); err != nil {
		return err
	}

	return file.Sync()
}

func printHistorySession(session Session, started time.Time) {
	fmt.Printf(
		"%s | %-10s | %-9s | %s",
		started.Format("2006-01-02 15:04:05"),
		session.Kind,
		session.Status,
		formatTime(time.Duration(session.ElapsedSeconds)*time.Second),
	)

	if session.Track != "" {
		fmt.Printf(" | %s", session.Track)
	}

	if session.Breathing != nil {
		fmt.Printf(" | %d cycles", session.CompletedCycles)
	}

	fmt.Println()
}
