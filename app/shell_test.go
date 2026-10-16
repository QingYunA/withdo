package main

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/views"
)

// fakeOpen replaces openRunner with one that records the open commands and fails those for which
// fail says so; it restores the real one when the test ends.
func fakeOpen(t *testing.T, fail func(args []string) bool) *[]string {
	t.Helper()
	var calls []string
	real := openRunner
	openRunner = func(args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		if fail != nil && fail(args) {
			return errors.New("open failed")
		}
		return nil
	}
	t.Cleanup(func() { openRunner = real })
	return &calls
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

func TestOpenInTerminalAndFinderNeedAFolder(t *testing.T) {
	calls := fakeOpen(t, nil)
	s := &shell{}
	if err := s.OpenIn(views.OpenTerminal, "Claude", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if err := s.OpenIn(views.OpenFinder, "Claude", "/work/tree"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"-a Terminal /work/tree", "/work/tree"}; !slices.Equal(*calls, want) {
		t.Errorf("ran %q, want %q", *calls, want)
	}
	for _, how := range []views.OpenWith{views.OpenTerminal, views.OpenEditor, views.OpenFinder} {
		if err := s.OpenIn(how, "Claude", ""); !errors.Is(err, views.ErrNoDir) {
			t.Errorf("%s without a folder: err = %v, want ErrNoDir", how, err)
		}
	}
	if len(*calls) != 2 {
		t.Errorf("ran %q for a missing folder", *calls)
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
