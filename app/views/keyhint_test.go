package views

import "testing"

func TestKeyHintIsTheMacWritingOnMacAndPlainWordsElsewhere(t *testing.T) {
	cases := []struct{ goos, mac, want string }{
		{"darwin", "⌘⇧A", "⌘⇧A"},
		{"darwin", "⌘↵", "⌘↵"},
		{"windows", "⌘⇧A", "Ctrl+Shift+A"},
		{"windows", "⌘↵", "Ctrl+Enter"},
		{"windows", "⌘⌫", "Ctrl+Backspace"},
		{"windows", "⌘,", "Ctrl+,"},
		{"windows", "⌘F", "Ctrl+F"},
		{"windows", "⌥⌘K", "Alt+Ctrl+K"},
		{"windows", "S", "S"},
		{"linux", "⌘K", "Ctrl+K"},
	}
	for _, tc := range cases {
		t.Run(tc.goos+" "+tc.mac, func(t *testing.T) {
			old := systemGOOS
			systemGOOS = tc.goos
			t.Cleanup(func() { systemGOOS = old })
			if got := keyHint(tc.mac); got != tc.want {
				t.Errorf("keyHint(%q) on %s = %q, want %q", tc.mac, tc.goos, got, tc.want)
			}
		})
	}
}

// The task actions' hints are written when they are shown, not when the package starts, so the
// system they are written for is the one in effect then.
func TestTaskActionHintsFollowTheSystemWhenShown(t *testing.T) {
	old := systemGOOS
	t.Cleanup(func() { systemGOOS = old })
	systemGOOS = "windows"
	if got := keyDone.hint(); got != "Ctrl+Enter" {
		t.Errorf("keyDone on windows hints %q, want Ctrl+Enter", got)
	}
	systemGOOS = "darwin"
	if got := keyDone.hint(); got != "⌘↵" {
		t.Errorf("keyDone on darwin hints %q, want ⌘↵", got)
	}
}
