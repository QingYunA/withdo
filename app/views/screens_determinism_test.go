package views

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"testing"
)

// renderInFreshProcess runs one screen test in a new process of this test binary, writing its
// PNGs into a new directory, and returns the directory.
func renderInFreshProcess(t *testing.T, screen string) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run", "^"+screen+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), "SCREENS_OUT="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("%s: %v\n%s", screen, err, out)
	}
	return dir
}

// differingScreens lists the PNG names that are in one directory and not the other, or differ.
func differingScreens(t *testing.T, a, b string) []string {
	t.Helper()
	names := map[string]bool{}
	for _, dir := range []string{a, b} {
		files, err := filepath.Glob(filepath.Join(dir, "*.png"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			names[filepath.Base(f)] = true
		}
	}
	var differ []string
	for name := range names {
		x, errX := os.ReadFile(filepath.Join(a, name))
		y, errY := os.ReadFile(filepath.Join(b, name))
		if errX != nil || errY != nil || !bytes.Equal(x, y) {
			differ = append(differ, name)
		}
	}
	slices.Sort(differ)
	return differ
}

// Rendering a screen twice, in two fresh processes, must give the same PNGs: a wall clock (the
// calendar's today, a spinner's phase) or an order of goroutines or maps in a picture shows up as
// a diff in unrelated files each time scripts/screens.sh runs. The screens here are the ones that
// flaked (the hand-off drag, the select stack, the reminder calendar) and a menu; rendering every
// screen twice would take several minutes. The two runs of each go side by side.
func TestDesignScreensAreTheSameInTwoFreshProcesses(t *testing.T) {
	screensTest(t)
	if os.Getenv("SCREENS_OUT") != "" {
		t.Skip("rendering into SCREENS_OUT; the check is for plain test runs")
	}
	screens := []string{"TestScreensHandOff", "TestScreensSelect", "TestScreensCalendar", "TestScreensMenu"}
	first, second := make([]string, len(screens)), make([]string, len(screens))
	var wg sync.WaitGroup
	for i, screen := range screens {
		for _, into := range [][]string{first, second} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				into[i] = renderInFreshProcess(t, screen)
			}()
		}
	}
	wg.Wait()
	var differ []string
	for i := range screens {
		differ = append(differ, differingScreens(t, first[i], second[i])...)
	}
	if len(differ) > 0 {
		t.Fatalf("%d screens differ between two fresh renders: %v", len(differ), differ)
	}
}
