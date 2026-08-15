package watch

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jdebrux/nudge/internal/config"
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

func TestWaitFiresOnceAndStopsWhenRepeatLimitIsOne(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	cfgPath := filepath.Join(dir, "config.json")

	target := time.Now().Add(15 * time.Millisecond)
	s := nudge.State{Phase: nudge.Focus, Since: time.Now(), Until: &target, NextCue: &target}
	mustSave(t, statePath, s)
	mustSaveConfig(t, cfgPath, config.Config{RepeatInterval: 15 * time.Millisecond, MaxRepeats: 1})

	notifications := stubNotify(t)

	out := captureStdout(t, func() {
		if err := Wait(statePath, cfgPath, "focus", s.Since, target); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})

	if out != "\a" {
		t.Fatalf("stdout = %q, want exactly one bell", out)
	}
	if len(*notifications) != 1 {
		t.Fatalf("notifications = %d, want exactly 1", len(*notifications))
	}
}

func TestWaitRepeatsUpToTheConfiguredLimit(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	cfgPath := filepath.Join(dir, "config.json")

	target := time.Now().Add(15 * time.Millisecond)
	s := nudge.State{Phase: nudge.Rest, Since: time.Now(), Until: &target, NextCue: &target}
	mustSave(t, statePath, s)
	mustSaveConfig(t, cfgPath, config.Config{RepeatInterval: 15 * time.Millisecond, MaxRepeats: 3})

	notifications := stubNotify(t)

	out := captureStdout(t, func() {
		if err := Wait(statePath, cfgPath, "rest", s.Since, target); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})

	if got := strings.Count(out, "\a"); got != 3 {
		t.Fatalf("bell count = %d, want 3 (content: %q)", got, out)
	}
	if len(*notifications) != 3 {
		t.Fatalf("notifications = %d, want 3", len(*notifications))
	}
	for _, n := range *notifications {
		if n.title != "nudge" || !strings.Contains(n.message, "back to focus") {
			t.Fatalf("notification = %+v, want a rest-phase message", n)
		}
	}
}

func TestWaitStopsRepeatingWhenStateMovesOnMidway(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	cfgPath := filepath.Join(dir, "config.json")

	sessionSince := time.Now()
	target := sessionSince.Add(15 * time.Millisecond)
	mustSave(t, statePath, nudge.State{Phase: nudge.Focus, Since: sessionSince, Until: &target, NextCue: &target})
	mustSaveConfig(t, cfgPath, config.Config{RepeatInterval: 30 * time.Millisecond, MaxRepeats: 5})

	stubNotify(t)

	// Simulate a `done` landing shortly after the first firing — well
	// before the 5-repeat budget would otherwise be exhausted.
	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = store.Save(statePath, nudge.State{Phase: nudge.Idle, Since: time.Now()})
	}()

	out := captureStdout(t, func() {
		if err := Wait(statePath, cfgPath, "focus", sessionSince, target); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})

	bells := strings.Count(out, "\a")
	if bells < 1 {
		t.Fatalf("expected at least one bell before the state changed, got %d", bells)
	}
	if bells >= 5 {
		t.Fatalf("expected fewer than the 5-repeat limit since state moved on, got %d", bells)
	}
}

func TestWaitStaysQuietWhenStateMovedOn(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	cfgPath := filepath.Join(dir, "config.json")

	sessionSince := time.Now()
	target := sessionSince.Add(15 * time.Millisecond)
	mustSaveConfig(t, cfgPath, config.Config{RepeatInterval: time.Minute, MaxRepeats: 3})

	// Simulate a `done` that landed before the watcher wakes up.
	mustSave(t, statePath, nudge.State{Phase: nudge.Idle, Since: sessionSince})

	stubNotify(t)

	out := captureStdout(t, func() {
		if err := Wait(statePath, cfgPath, "focus", sessionSince, target); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	})

	if out != "" {
		t.Fatalf("stdout = %q, want no output for a stale cue", out)
	}
}

type notification struct{ title, message string }

// stubNotify replaces the package's real osascript call with a
// recorder for the duration of the test — without this, running these
// tests on a Mac would pop a real notification for every firing.
func stubNotify(t *testing.T) *[]notification {
	t.Helper()
	var got []notification
	original := sendNotification
	sendNotification = func(title, message string) {
		got = append(got, notification{title, message})
	}
	t.Cleanup(func() { sendNotification = original })
	return &got
}

func mustSave(t *testing.T, path string, s nudge.State) {
	t.Helper()
	if err := store.Save(path, s); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
}

func mustSaveConfig(t *testing.T, path string, c config.Config) {
	t.Helper()
	if err := config.Save(path, c); err != nil {
		t.Fatalf("config.Save: %v", err)
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
