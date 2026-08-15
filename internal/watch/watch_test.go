package watch

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jdebrux/nudge/internal/nudge"
	"github.com/jdebrux/nudge/internal/store"
)

var since = time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)
var cueAt = since.Add(25 * time.Minute)

func TestStillCurrentMatches(t *testing.T) {
	s := nudge.State{Phase: nudge.Focus, Since: since, Until: &cueAt, NextCue: &cueAt}
	if !stillCurrent(s, "focus", since, cueAt) {
		t.Fatal("expected matching state to be current")
	}
}

func TestStillCurrentDetectsPhaseChange(t *testing.T) {
	s := nudge.State{Phase: nudge.Idle, Since: since}
	if stillCurrent(s, "focus", since, cueAt) {
		t.Fatal("expected phase change to make the cue stale")
	}
}

func TestStillCurrentDetectsNewSession(t *testing.T) {
	newSince := since.Add(30 * time.Minute)
	newCueAt := newSince.Add(10 * time.Minute)
	s := nudge.State{Phase: nudge.Rest, Since: newSince, Until: &newCueAt, NextCue: &newCueAt}
	if stillCurrent(s, "focus", since, cueAt) {
		t.Fatal("expected a superseding session to make the cue stale")
	}
}

func TestStillCurrentDetectsLaterPostponement(t *testing.T) {
	// Same session (Phase/Since unchanged), but `later` moved NextCue
	// forward without touching Until — the original watcher's cue is
	// now stale even though nothing else about the session changed.
	postponed := cueAt.Add(5 * time.Minute)
	s := nudge.State{Phase: nudge.Focus, Since: since, Until: &cueAt, NextCue: &postponed}
	if stillCurrent(s, "focus", since, cueAt) {
		t.Fatal("expected a later-postponed cue to make the original watcher stale")
	}
	if !stillCurrent(s, "focus", since, postponed) {
		t.Fatal("expected the new watcher scheduled for the postponed time to be current")
	}
}

func TestWaitFiresBellWhenStateStillMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	target := time.Now().Add(20 * time.Millisecond)
	s := nudge.State{Phase: nudge.Focus, Since: time.Now(), Until: &target, NextCue: &target}
	if err := store.Save(path, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Wait(path, "focus", s.Since, target); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})

	if out != "\a" {
		t.Fatalf("stdout = %q, want a bell character", out)
	}
}

func TestWaitStaysQuietWhenStateMovedOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	sessionSince := time.Now()
	target := sessionSince.Add(20 * time.Millisecond)

	// Simulate a `done` that landed before the watcher wakes up.
	if err := store.Save(path, nudge.State{Phase: nudge.Idle, Since: sessionSince}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Wait(path, "focus", sessionSince, target); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})

	if out != "" {
		t.Fatalf("stdout = %q, want no output for a stale cue", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(data)
}
