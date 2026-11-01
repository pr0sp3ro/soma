package soma

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxDuration time.Duration = 7 * 24 * time.Hour

type Config struct {
	Meditation MeditationConfig `json:"meditation"`
	Breathing  BreathingConfig  `json:"breathing"`
	Tracks     []Track          `json:"tracks"`
}

type MeditationConfig struct {
	Duration string `json:"duration"`
	Interval string `json:"interval"`
	Volume   int    `json:"volume"`
}

type BreathingConfig struct {
	Cycles  int `json:"cycles"`
	Inhale  int `json:"inhale_seconds"`
	HoldIn  int `json:"hold_in_seconds"`
	Exhale  int `json:"exhale_seconds"`
	HoldOut int `json:"hold_out_seconds"`
	Volume  int `json:"volume"`
}

type Track struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Start    string `json:"start"`
	Duration string `json:"duration"`
}

type Paths struct {
	Config  string
	History string
	Lock    string
}

func appPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	var configRoot string = os.Getenv("XDG_CONFIG_HOME")
	if configRoot == "" {
		configRoot = filepath.Join(home, ".config")
	}

	var dataRoot string = os.Getenv("XDG_DATA_HOME")
	if dataRoot == "" {
		dataRoot = filepath.Join(home, ".local", "share")
	}

	if !filepath.IsAbs(configRoot) || !filepath.IsAbs(dataRoot) {
		return Paths{}, errors.New("XDG_CONFIG_HOME and XDG_DATA_HOME must be absolute paths")
	}

	return Paths{
		Config:  filepath.Join(configRoot, "soma", "config.json"),
		History: filepath.Join(dataRoot, "soma", "history.jsonl"),
		Lock:    filepath.Join(dataRoot, "soma", "session.lock"),
	}, nil
}

func defaultConfig() Config {
	return Config{
		Meditation: MeditationConfig{
			Duration: "30m",
			Interval: "10m",
			Volume:   50,
		},
		Breathing: BreathingConfig{
			Cycles:  30,
			Inhale:  4,
			HoldIn:  0,
			Exhale:  6,
			HoldOut: 0,
			Volume:  35,
		},
		Tracks: []Track{},
	}
}

func loadConfig(path string) (Config, error) {
	var config Config = defaultConfig()

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return config, nil
	}

	if err != nil {
		return config, err
	}
	defer file.Close()

	var decoder *json.Decoder = json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var document *Config = &config
	if err = decoder.Decode(&document); err != nil {
		return config, fmt.Errorf("read config %s: %w", path, err)
	}

	if document == nil {
		return config, errors.New("config must be a JSON object, not null")
	}

	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return config, errors.New("config must contain exactly one JSON object")
	}

	return config, nil
}

func writeConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(defaultConfig(), "", "  ")
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	err = errors.Join(err, file.Close())
	if err != nil {
		_ = os.Remove(path)
	}

	return err
}

func parseDuration(value string, allowZero bool) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	var invalidZero bool = duration == 0 && !allowZero

	if err != nil || duration < 0 || duration > maxDuration || invalidZero {
		return 0, fmt.Errorf("invalid duration %q: use a duration such as 20m or 2m35s, maximum 168h", value)
	}

	if duration%time.Second != 0 {
		return 0, errors.New("durations must use whole seconds")
	}

	return duration, nil
}

var videoIDPattern *regexp.Regexp = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// Canonicalize the URL so playlist and timestamp parameters cannot change playback.
func youtubeURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" {
		return "", errors.New("use an HTTPS YouTube video URL")
	}

	var id string
	switch parsed.Host {
	case "www.youtube.com", "youtube.com", "music.youtube.com":
		if parsed.Path == "/watch" {
			id = parsed.Query().Get("v")
		}
	case "youtu.be":
		id = strings.TrimPrefix(parsed.Path, "/")
	}

	if !videoIDPattern.MatchString(id) {
		return "", errors.New("use https://www.youtube.com/watch?v=VIDEO_ID or https://youtu.be/VIDEO_ID")
	}

	return "https://www.youtube.com/watch?v=" + id, nil
}
