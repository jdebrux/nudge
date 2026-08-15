package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	got := Default()
	want := Config{Focus: 25 * time.Minute, Break: 5 * time.Minute, LongBreak: 15 * time.Minute, SessionsPerLongBreak: 4}
	if got != want {
		t.Fatalf("Default() = %+v, want %+v", got, want)
	}
}

func TestMarshalJSONUsesDurationStrings(t *testing.T) {
	c := Config{Focus: 30 * time.Minute, Break: 10 * time.Minute, LongBreak: 20 * time.Minute, SessionsPerLongBreak: 3}

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
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	want := Config{Focus: 45 * time.Minute, Break: 15 * time.Minute, LongBreak: 30 * time.Minute, SessionsPerLongBreak: 5}

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
	want := Config{Focus: 50 * time.Minute, Break: 10 * time.Minute, LongBreak: 25 * time.Minute, SessionsPerLongBreak: 4}

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
