package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	got := Default()
	want := Config{
		Focus: 25 * time.Minute, Break: 5 * time.Minute, LongBreak: 15 * time.Minute, SessionsPerLongBreak: 4,
		RepeatInterval: 5 * time.Minute, MaxRepeats: 6,
	}
	if got != want {
		t.Fatalf("Default() = %+v, want %+v", got, want)
	}
}

func TestMarshalJSONUsesDurationStrings(t *testing.T) {
	c := Config{
		Focus: 30 * time.Minute, Break: 10 * time.Minute, LongBreak: 20 * time.Minute, SessionsPerLongBreak: 3,
		RepeatInterval: 90 * time.Second, MaxRepeats: 8,
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal into map: %v", err)
	}
	if raw["focus"] != "30m0s" {
		t.Fatalf("focus = %v, want %q", raw["focus"], "30m0s")
	}
	if raw["long_break"] != "20m0s" {
		t.Fatalf("long_break = %v, want %q", raw["long_break"], "20m0s")
	}
	if raw["repeat_interval"] != "1m30s" {
		t.Fatalf("repeat_interval = %v, want %q", raw["repeat_interval"], "1m30s")
	}
	if raw["max_repeats"] != float64(8) {
		t.Fatalf("max_repeats = %v, want %v", raw["max_repeats"], 8)
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	want := Config{
		Focus: 45 * time.Minute, Break: 15 * time.Minute, LongBreak: 30 * time.Minute, SessionsPerLongBreak: 5,
		RepeatInterval: 3 * time.Minute, MaxRepeats: 10,
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadMissingFileReadsAsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if got != Default() {
		t.Fatalf("got %+v, want Default() %+v", got, Default())
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := Config{
		Focus: 50 * time.Minute, Break: 10 * time.Minute, LongBreak: 25 * time.Minute, SessionsPerLongBreak: 4,
		RepeatInterval: 5 * time.Minute, MaxRepeats: 6,
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
