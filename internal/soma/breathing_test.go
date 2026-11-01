package soma

import (
	"context"
	"testing"
	"time"
)

func TestBreathingRejectsInvalidSettingsBeforeSession(t *testing.T) {
	var cases = [][]string{
		{"--cycles", "1", "--duration", "5m"},
		{"--cycles", "0"},
		{"--inhale", "0"},
		{"--hold-out", "-1"},
		{"--duration", "0s"},
		{"--volume", "101"},
	}
	for _, args := range cases {
		if err := breathe(context.Background(), Paths{}, defaultConfig().Breathing, args); err == nil {
			t.Fatalf("expected invalid settings to fail: %v", args)
		}
	}
}

func TestBreathingDurationDoesNotChangeConfiguration(t *testing.T) {
	config := defaultConfig().Breathing

	options, err := parseBreathingOptions(config, []string{"--duration", "5m"})
	if err != nil {
		t.Fatal(err)
	}

	if options.Config.Cycles != 0 || options.Duration != 5*time.Minute {
		t.Fatalf("unexpected duration options: %+v", options)
	}

	if config.Cycles != 30 {
		t.Fatal("parsing must not change the configured cycle count")
	}
}
