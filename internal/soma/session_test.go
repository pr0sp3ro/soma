package soma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStatusAndExclusiveLock(t *testing.T) {
	var failure error = errors.New("playback failure")
	var cases = []struct {
		status string
		result error
	}{
		{status: "finished"},
		{status: "cancelled", result: context.Canceled},
		{status: "failed", result: failure},
	}
	for _, item := range cases {
		t.Run(item.status, func(t *testing.T) {
			var directory string = t.TempDir()
			var paths = Paths{
				History: filepath.Join(directory, "history.jsonl"),
				Lock:    filepath.Join(directory, "session.lock"),
			}
			var session Session = Session{Kind: "meditation", PlannedSeconds: 60}
			var err error = withSession(context.Background(), paths, session, func(record *Session) error {
				var nestedErr error = withSession(context.Background(), paths, session, func(_ *Session) error {
					t.Fatal("second session must not start")
					return nil
				})
				if nestedErr == nil {
					t.Fatal("expected session lock error")
				}
				return item.result
			})
			if !errors.Is(err, item.result) {
				t.Fatalf("unexpected result: %v", err)
			}
			var data []byte
			data, err = os.ReadFile(paths.History)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Count(data, []byte{'\n'}) != 1 {
				t.Fatalf("expected exactly one session: %s", data)
			}
			var saved Session
			if err = json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.Status != item.status || saved.StartedAt.IsZero() || saved.PlannedSeconds != 60 {
				t.Fatalf("unexpected session: %+v", saved)
			}
		})
	}
}
