package views

import "testing"

func TestZZ(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	t.Logf("%q", tt.Texts())
}
