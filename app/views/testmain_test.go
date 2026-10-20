package views

import (
	"os"
	"testing"
)

// TestMain writes the words as on a Mac on every system (key hints, Finder, Preview, brew), so that
// the view tests and the screens read the same on macOS, Linux CI and Windows; keyhint_test.go and
// system_words_test.go switch it per case.
func TestMain(m *testing.M) {
	systemGOOS = "darwin"
	os.Exit(m.Run())
}
