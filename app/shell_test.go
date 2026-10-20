package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/views"
)

// fakeOpen makes OpenIn act as on a Mac and replaces openRunner with one that records the open
// commands and fails those for which fail says so; it restores the real ones when the test ends.
func fakeOpen(t *testing.T, fail func(args []string) bool) *[]string {
	t.Helper()
	var calls []string
	real, realGOOS := openRunner, openGOOS
	openGOOS = "darwin"
	openRunner = func(args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		if fail != nil && fail(args) {
			return errors.New("open failed")
		}
		return nil
	}
	t.Cleanup(func() { openRunner, openGOOS = real, realGOOS })
	return &calls
}

// fakeOpenPath replaces openPath with one that records the paths it is given.
func fakeOpenPath(t *testing.T) *[]string {
	t.Helper()
	var paths []string
	real := openPath
	openPath = func(p string) error {
		paths = append(paths, p)
		return nil
	}
	t.Cleanup(func() { openPath = real })
	return &paths
}

func TestOpenInTheAgentsAppBringsItForward(t *testing.T) {
	calls := fakeOpen(t, nil)
	if err := (&shell{}).OpenIn(views.OpenAgentApp, "Claude", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-a Claude"}; !slices.Equal(*calls, want) {
		t.Errorf("ran %q, want %q: Claude has no folder to open", *calls, want)
	}
}

func TestOpenInAnEditorAgentOpensTheFolder(t *testing.T) {
	calls := fakeOpen(t, nil)
	if err := (&shell{}).OpenIn(views.OpenAgentApp, "Cursor", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-a Cursor /work/tree"}; !slices.Equal(*calls, want) {
		t.Errorf("ran %q, want %q", *calls, want)
	}
}

func TestOpenInAnAgentWithoutAnAppFallsBackToTheTerminal(t *testing.T) {
	calls := fakeOpen(t, nil)
	if err := (&shell{}).OpenIn(views.OpenAgentApp, "", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-a Terminal /work/tree"}; !slices.Equal(*calls, want) {
		t.Errorf("ran %q, want %q", *calls, want)
	}
}

func TestOpenInAnAppThatIsNotInstalledFallsBackToTheTerminal(t *testing.T) {
	calls := fakeOpen(t, func(args []string) bool { return args[1] == "Codex" })
	if err := (&shell{}).OpenIn(views.OpenAgentApp, "Codex", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-a Codex", "-a Terminal /work/tree"}; !slices.Equal(*calls, want) {
		t.Errorf("ran %q, want %q", *calls, want)
	}
}

func TestOpenInWithNoAppAndNoFolderSaysThereIsNothingToOpen(t *testing.T) {
	calls := fakeOpen(t, nil)
	if err := (&shell{}).OpenIn(views.OpenAgentApp, "", ""); !errors.Is(err, views.ErrNoDir) {
		t.Errorf("err = %v, want ErrNoDir", err)
	}
	if len(*calls) != 0 {
		t.Errorf("ran %q", *calls)
	}
}

// The folder opens through the system (Finder, File Explorer), not the macOS open command.
func TestOpenInTerminalAndFinderNeedAFolder(t *testing.T) {
	calls := fakeOpen(t, nil)
	paths := fakeOpenPath(t)
	s := &shell{}
	if err := s.OpenIn(views.OpenTerminal, "Claude", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if err := s.OpenIn(views.OpenFinder, "Claude", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-a Terminal /work/tree"}; !slices.Equal(*calls, want) {
		t.Errorf("ran %q, want %q", *calls, want)
	}
	if want := []string{"/work/tree"}; !slices.Equal(*paths, want) {
		t.Errorf("opened %q, want %q", *paths, want)
	}
	for _, how := range []views.OpenWith{views.OpenTerminal, views.OpenEditor, views.OpenFinder} {
		if err := s.OpenIn(how, "Claude", ""); !errors.Is(err, views.ErrNoDir) {
			t.Errorf("%s without a folder: err = %v, want ErrNoDir", how, err)
		}
	}
	if len(*calls) != 1 || len(*paths) != 1 {
		t.Errorf("ran %q and opened %q for a missing folder", *calls, *paths)
	}
}

func TestOpenInEditorUsesTheFirstInstalledOne(t *testing.T) {
	calls := fakeOpen(t, func(args []string) bool { return args[0] == "-Ra" && args[1] == "Cursor" })
	if err := (&shell{}).OpenIn(views.OpenEditor, "Claude", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-Ra Cursor", "-Ra Visual Studio Code", "-a Visual Studio Code /work/tree"}; !slices.Equal(*calls, want) {
		t.Errorf("ran %q, want %q", *calls, want)
	}
}

func TestOpenInEditorWithNoneInstalledSaysSo(t *testing.T) {
	calls := fakeOpen(t, func(args []string) bool { return args[0] == "-Ra" })
	if err := (&shell{}).OpenIn(views.OpenEditor, "Claude", "/work/tree"); !errors.Is(err, views.ErrNoEditor) {
		t.Errorf("err = %v, want ErrNoEditor", err)
	}
	if len(*calls) != 3 {
		t.Errorf("ran %q, want a look for each of the three editors", *calls)
	}
}

// fakeStart makes OpenIn act as on Windows and replaces startRunner with one that records what it
// starts, as "dir: name args", and fails those for which fail says so. It never starts anything.
func fakeStart(t *testing.T, fail func(name string) bool) *[]string {
	t.Helper()
	var calls []string
	realGOOS, realStart := openGOOS, startRunner
	openGOOS = "windows"
	startRunner = func(dir, name string, args ...string) error {
		calls = append(calls, fmt.Sprintf("%s: %s %q", dir, name, args))
		if fail != nil && fail(name) {
			return errors.New("exec: not found")
		}
		return nil
	}
	t.Cleanup(func() { openGOOS, startRunner = realGOOS, realStart })
	return &calls
}

func TestOnWindowsTheTerminalIsWindowsTerminal(t *testing.T) {
	calls := fakeStart(t, nil)
	if err := (&shell{}).OpenIn(views.OpenTerminal, "Claude", `C:\work\tree`); err != nil {
		t.Fatal(err)
	}
	if want := []string{`C:\work\tree: wt ["-d" "."]`}; !slices.Equal(*calls, want) {
		t.Errorf("started %q, want %q", *calls, want)
	}
}

func TestOnWindowsWithoutWindowsTerminalTheTerminalIsTheCommandPrompt(t *testing.T) {
	calls := fakeStart(t, func(name string) bool { return name == "wt" })
	if err := (&shell{}).OpenIn(views.OpenTerminal, "Claude", `C:\work\tree`); err != nil {
		t.Fatal(err)
	}
	want := []string{`C:\work\tree: wt ["-d" "."]`, `C:\work\tree: cmd ["/c" "start" "" "cmd"]`}
	if !slices.Equal(*calls, want) {
		t.Errorf("started %q, want %q", *calls, want)
	}
}

func TestOnWindowsTheEditorIsTheFirstCommandFound(t *testing.T) {
	calls := fakeStart(t, func(name string) bool { return name == "cursor" })
	if err := (&shell{}).OpenIn(views.OpenEditor, "Claude", `C:\work\tree`); err != nil {
		t.Fatal(err)
	}
	want := []string{`C:\work\tree: cursor ["."]`, `C:\work\tree: code ["."]`}
	if !slices.Equal(*calls, want) {
		t.Errorf("started %q, want %q", *calls, want)
	}
}

func TestOnWindowsWithNoEditorFoundItSaysSo(t *testing.T) {
	calls := fakeStart(t, func(string) bool { return true })
	if err := (&shell{}).OpenIn(views.OpenEditor, "Claude", `C:\work\tree`); !errors.Is(err, views.ErrNoEditor) {
		t.Errorf("err = %v, want ErrNoEditor", err)
	}
	if len(*calls) != 3 {
		t.Errorf("started %q, want a try of each of the three editors", *calls)
	}
}

func TestOnWindowsAnEditorAgentOpensTheFolderWithItsCommand(t *testing.T) {
	calls := fakeStart(t, nil)
	s := &shell{}
	if err := s.OpenIn(views.OpenAgentApp, "Visual Studio Code", `C:\work\tree`); err != nil {
		t.Fatal(err)
	}
	if err := s.OpenIn(views.OpenAgentApp, "Zed", ""); err != nil {
		t.Fatal(err)
	}
	want := []string{`C:\work\tree: code ["."]`, `: zed []`}
	if !slices.Equal(*calls, want) {
		t.Errorf("started %q, want %q", *calls, want)
	}
}

// Claude and Codex have no command that brings their desktop app forward (claude is Claude Code).
func TestOnWindowsAnAgentAppWithoutACommandFallsBackToTheTerminal(t *testing.T) {
	calls := fakeStart(t, nil)
	if err := (&shell{}).OpenIn(views.OpenAgentApp, "Claude", `C:\work\tree`); err != nil {
		t.Fatal(err)
	}
	if want := []string{`C:\work\tree: wt ["-d" "."]`}; !slices.Equal(*calls, want) {
		t.Errorf("started %q, want %q", *calls, want)
	}
}

func TestOnWindowsAnEditorAgentNotFoundFallsBackToTheTerminal(t *testing.T) {
	calls := fakeStart(t, func(name string) bool { return name == "windsurf" })
	if err := (&shell{}).OpenIn(views.OpenAgentApp, "Windsurf", `C:\work\tree`); err != nil {
		t.Fatal(err)
	}
	want := []string{`C:\work\tree: windsurf ["."]`, `C:\work\tree: wt ["-d" "."]`}
	if !slices.Equal(*calls, want) {
		t.Errorf("started %q, want %q", *calls, want)
	}
}

func TestOnWindowsTheMacOpenCommandIsNeverRun(t *testing.T) {
	opened := fakeOpen(t, nil)
	fakeStart(t, nil)
	fakeOpenPath(t)
	s := &shell{}
	for _, how := range []views.OpenWith{views.OpenAgentApp, views.OpenTerminal, views.OpenEditor, views.OpenFinder} {
		if err := s.OpenIn(how, "Cursor", `C:\work\tree`); err != nil {
			t.Errorf("%s: %v", how, err)
		}
		if err := s.OpenIn(how, "", ""); !errors.Is(err, views.ErrNoDir) {
			t.Errorf("%s without a folder: err = %v, want ErrNoDir", how, err)
		}
	}
	if len(*opened) != 0 {
		t.Errorf("ran open %q on Windows", *opened)
	}
}
