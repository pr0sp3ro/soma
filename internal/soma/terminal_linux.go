package soma

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type terminalSize struct {
	Rows    uint16
	Columns uint16
	XPixel  uint16
	YPixel  uint16
}

type breathingView struct {
	original    syscall.Termios
	interactive bool
	lastFrame   string
}

func terminalIOCTL(fd uintptr, request uintptr, value unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, uintptr(value))
	if errno != 0 {
		return errno
	}

	return nil
}

func openBreathingView() (*breathingView, error) {
	var view breathingView
	var size terminalSize

	err := terminalIOCTL(os.Stdin.Fd(), syscall.TCGETS, unsafe.Pointer(&view.original))
	if errors.Is(err, syscall.ENOTTY) {
		return &view, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read terminal settings: %w", err)
	}

	err = terminalIOCTL(os.Stdout.Fd(), syscall.TIOCGWINSZ, unsafe.Pointer(&size))
	if errors.Is(err, syscall.ENOTTY) {
		return &view, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read terminal size: %w", err)
	}

	var settings syscall.Termios = view.original
	// Keep ISIG so Ctrl+C still reaches the application's signal handler.
	settings.Lflag &^= syscall.ICANON | syscall.ECHO
	// Poll keys from the session loop without a blocked input goroutine.
	settings.Cc[syscall.VMIN] = 0
	settings.Cc[syscall.VTIME] = 0

	if err := terminalIOCTL(os.Stdin.Fd(), syscall.TCSETS, unsafe.Pointer(&settings)); err != nil {
		return nil, fmt.Errorf("configure terminal: %w", err)
	}

	view.interactive = true
	fmt.Fprint(os.Stdout, "\x1b[?1049h\x1b[?25l\x1b[2J")
	return &view, nil
}

func (view *breathingView) close() {
	if !view.interactive {
		return
	}

	err := terminalIOCTL(os.Stdin.Fd(), syscall.TCSETS, unsafe.Pointer(&view.original))
	fmt.Fprint(os.Stdout, "\x1b[?25h\x1b[?1049l")

	if err != nil {
		fmt.Fprintln(os.Stderr, "restore terminal:", err)
	}
}

func (view *breathingView) readPause() (bool, error) {
	if !view.interactive {
		return false, nil
	}

	var keys [64]byte
	count, err := syscall.Read(int(os.Stdin.Fd()), keys[:])
	if err != nil {
		if errors.Is(err, syscall.EINTR) || errors.Is(err, syscall.EAGAIN) {
			return false, nil
		}

		return false, fmt.Errorf("read keyboard: %w", err)
	}

	var spaces int
	for _, key := range keys[:count] {
		if key == ' ' {
			spaces++
		}
	}

	return spaces%2 == 1, nil
}

func (view *breathingView) render(phase Phase, cycle int, cycles int, remaining time.Duration, elapsed time.Duration, paused bool) {
	var seconds int = int((remaining + time.Second - 1) / time.Second)
	if seconds < 0 {
		seconds = 0
	}

	if !view.interactive {
		fmt.Printf("\rCycle %d | %-18s | %2ds remaining   ", cycle, phase.Name, seconds)
		return
	}

	var cycleLabel string = fmt.Sprintf("Cycle %d", cycle)
	if cycles > 0 {
		cycleLabel = fmt.Sprintf("Cycle %d / %d", cycle, cycles)
	}

	var state string = "Space: pause | Ctrl+C: stop"
	if paused {
		state = "PAUSED | Space: resume | Ctrl+C: stop"
	}

	var lines = []string{
		"SOMA - BREATHING",
		"",
		cycleLabel,
		"",
		strings.ToUpper(phase.Name),
		fmt.Sprintf("%d seconds", seconds),
		"",
		"Elapsed: " + formatTime(elapsed),
		"",
		state,
	}

	var size = terminalSize{Rows: 24, Columns: 80}
	_ = terminalIOCTL(os.Stdout.Fd(), syscall.TIOCGWINSZ, unsafe.Pointer(&size))
	var rows int = max(1, int(size.Rows))
	var columns int = max(1, int(size.Columns))
	var top int = max(1, (rows-len(lines))/2+1)
	var frame strings.Builder

	fmt.Fprint(&frame, "\x1b[H")
	for index, line := range lines {
		if top+index > rows {
			break
		}

		if len(line) > columns {
			line = line[:columns]
		}

		var left int = max(1, (columns-len(line))/2+1)
		fmt.Fprintf(&frame, "\x1b[%d;%dH%s", top+index, left, line)
	}

	var output string = frame.String()
	if output != view.lastFrame {
		fmt.Fprint(os.Stdout, "\x1b[2J", output)
		view.lastFrame = output
	}
}
