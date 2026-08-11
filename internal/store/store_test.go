package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jdebrux/nudge/internal/nudge"
)

func TestLoadMissingFileReadsAsIdle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if got.Phase != nudge.Idle {
		t.Fatalf("phase = %v, want Idle", got.Phase)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	since := time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)
	until := since.Add(25 * time.Minute)
	want := nudge.State{Phase: nudge.Focus, Since: since, Until: &until}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Phase != want.Phase {
		t.Fatalf("phase = %v, want %v", got.Phase, want.Phase)
	}
	if !got.Since.Equal(want.Since) {
		t.Fatalf("since = %v, want %v", got.Since, want.Since)
	}
	if got.Until == nil || !got.Until.Equal(*want.Until) {
		t.Fatalf("until = %v, want %v", got.Until, want.Until)
	}
}

func TestSaveOpenEndedStateOmitsUntil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	want := nudge.State{Phase: nudge.Rest, Since: time.Now()}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Until != nil {
		t.Fatalf("until = %v, want nil", *got.Until)
	}
}

func TestSaveLeavesNoTempFilesBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := Save(path, nudge.State{Phase: nudge.Idle}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Fatalf("dir contents = %v, want only state.json", entries)
	}
}
