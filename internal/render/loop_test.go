package render

import "testing"

func TestLoopMessages(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"started", LoopStarted(), "loop started"},
		{"already active", LoopAlreadyActive(), "loop already running"},
		{"stopped", LoopStopped(), "loop stopped"},
		{"already inactive", LoopAlreadyInactive(), "loop wasn't running"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.got != c.want {
				t.Fatalf("got %q, want %q", c.got, c.want)
			}
		})
	}
}
