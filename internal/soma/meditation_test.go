package soma

import (
	"testing"
	"time"
)

func TestMeditationTrackDurationPrecedence(t *testing.T) {
	var cases = []struct {
		name          string
		trackDuration string
		args          []string
		wantDuration  time.Duration
	}{
		{
			name:         "meditation default",
			args:         []string{"--track", "evening"},
			wantDuration: 30 * time.Minute,
		},
		{
			name:          "track default",
			trackDuration: "20m",
			args:          []string{"--track", "evening"},
			wantDuration:  20 * time.Minute,
		},
		{
			name:          "explicit duration",
			trackDuration: "20m",
			args:          []string{"--track", "evening", "--duration", "15m"},
			wantDuration:  15 * time.Minute,
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			config := defaultConfig()
			config.Tracks = []Track{
				{
					Name:     "evening",
					URL:      "https://youtu.be/abcdefghijk?t=1",
					Start:    "2m35s",
					Duration: item.trackDuration,
				},
			}

			options, err := parseMeditationOptions(config, item.args)
			if err != nil {
				t.Fatal(err)
			}

			if options.Duration != item.wantDuration {
				t.Fatalf("duration: got %v, want %v", options.Duration, item.wantDuration)
			}

			if options.StartSeconds != 155 || options.Interval != 0 {
				t.Fatalf("unexpected track timing: %+v", options)
			}

			if options.Track.URL != "https://www.youtube.com/watch?v=abcdefghijk" {
				t.Fatalf("unexpected track URL: %q", options.Track.URL)
			}

			if config.Tracks[0].URL != "https://youtu.be/abcdefghijk?t=1" {
				t.Fatal("parsing must not modify configured tracks")
			}
		})
	}
}
