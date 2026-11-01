package soma

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestYouTubeURL(t *testing.T) {
	var cases = []struct {
		input string
		valid bool
	}{
		{input: "https://www.youtube.com/watch?v=abcdefghijk&t=15&list=ignored", valid: true},
		{input: "https://youtu.be/abcdefghijk?si=ignored", valid: true},
		{input: "https://youtu.be"},
		{input: "https://youtu.be/"},
		{input: "https://www.youtube.com/playlist?list=abcdefghijk"},
		{input: "https://youtube.com.evil.example/watch?v=abcdefghijk"},
		{input: "https://user@youtube.com/watch?v=abcdefghijk"},
		{input: "http://youtube.com/watch?v=abcdefghijk"},
		{input: "--script=anything"},
	}
	for _, item := range cases {
		t.Run(item.input, func(t *testing.T) {
			var actual string
			var err error
			actual, err = youtubeURL(item.input)
			if !item.valid {
				if err == nil {
					t.Fatal("expected invalid URL to be rejected")
				}
				return
			}
			if err != nil || actual != "https://www.youtube.com/watch?v=abcdefghijk" {
				t.Fatalf("unexpected canonical URL %q, error: %v", actual, err)
			}
		})
	}
}

func TestDurationValidation(t *testing.T) {
	var cases = []struct {
		value     string
		allowZero bool
		valid     bool
		seconds   int64
	}{
		{value: "2m35s", valid: true, seconds: 155},
		{value: "0s", allowZero: true, valid: true},
		{value: "168h", valid: true, seconds: 604800},
		{value: "0s"},
		{value: "-1s"},
		{value: "1.5s"},
		{value: "169h"},
		{value: "30"},
	}
	for _, item := range cases {
		t.Run(item.value, func(t *testing.T) {
			var duration time.Duration
			var err error
			duration, err = parseDuration(item.value, item.allowZero)
			if (err == nil) != item.valid {
				t.Fatalf("duration validity: %v, error: %v", item.valid, err)
			}
			if item.valid && int64(duration/time.Second) != item.seconds {
				t.Fatalf("unexpected duration: %v", duration)
			}
		})
	}
}

func TestConfigDoesNotOverwriteAndRejectsInvalidDocuments(t *testing.T) {
	var path string = filepath.Join(t.TempDir(), "config.json")
	if err := writeConfig(path); err != nil {
		t.Fatal(err)
	}
	var config Config
	var err error
	config, err = loadConfig(path)
	if err != nil || config.Breathing.Inhale != 4 || config.Meditation.Duration != "30m" {
		t.Fatalf("invalid defaults: %+v, error: %v", config, err)
	}
	if err = writeConfig(path); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected existing config protection, got %v", err)
	}
	for _, document := range []string{"null", `{"unknown":true}`, `{} {}`, `{"meditation":{"volume":"50"}}`} {
		if err = os.WriteFile(path, []byte(document), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = loadConfig(path); err == nil {
			t.Fatalf("accepted invalid config: %s", document)
		}
	}
}
