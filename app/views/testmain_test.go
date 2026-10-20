package views

import (
	"os"
	"testing"
)

// TestMain writes the key hints as on a Mac on every system, so that the view tests and the
// screens read the same on macOS, Linux CI and Windows; keyhint_test.go switches it per case.
func TestMain(m *testing.M) {
	hintGOOS = "darwin"
	os.Exit(m.Run())
}
