# Soma

A Linux terminal app for meditation timers, breathing exercises and session history.

## Install

Requires Go 1.22+ and `mpv`. YouTube playback also requires `yt-dlp`.
With `make`, install from the repository directory:

```bash
make install
```

Installs `soma` to `~/.local/bin`. Add that directory to your `PATH` if needed.
Use `make install PREFIX=/usr/local` to choose another installation directory.

Alternatively, install directly from GitHub:

```bash
go install github.com/pr0sp3ro/soma/cmd/soma@latest
```

This installs to `GOBIN`, or `~/go/bin` by default. To use `~/.local/bin` instead:

```bash
GOBIN="$HOME/.local/bin" go install github.com/pr0sp3ro/soma/cmd/soma@latest
```

## Usage

```bash
soma meditate
soma meditate --duration 20m --interval 5m --volume 40
soma meditate --duration 10m --silent
soma breathe
soma breathe --cycles 10 --inhale 4 --hold-in 4 --exhale 4 --hold-out 4
soma breathe --duration 5m
soma history --days 30
soma history --kind breathing
soma help
```

Meditation defaults to 30 minutes, with start and end bells and reminders every
10 minutes. Set `--interval 0s` to disable reminders. Volume ranges from 0 to 100.

Breathing defaults to 30 cycles of 4-second inhales and 6-second exhales, without
holds. Phase lengths are in seconds, with `0` skipping a hold. Choose a cycle
count or a duration. Duration-based sessions finish their current cycle.

Press `Ctrl+C` to stop a session. Finished and interrupted sessions are saved
in history, which also shows daily totals. Times accept values such as `20m`
and `2m35s`.

## Configuration and YouTube

```bash
soma config init
soma config path
```

Edit the generated configuration to change defaults or add recordings to `tracks`.
Replace the example URL with your recording's YouTube URL:

```json
"tracks": [
  {
    "name": "evening",
    "url": "https://www.youtube.com/watch?v=VIDEO_ID",
    "start": "2m35s",
    "duration": "20m"
  }
]
```

```bash
soma tracks
soma meditate --track evening
soma meditate --track evening --duration 15m
```

`start` sets the playback offset, use `0s` for the beginning. Track duration is
optional and defaults to the meditation duration. YouTube sessions play audio
without additional bells and stop when the timer or recording ends.

Configuration: `~/.config/soma/config.json`.
History: `~/.local/share/soma/history.jsonl`.
Both locations respect `XDG_CONFIG_HOME` and `XDG_DATA_HOME`, respectively.
