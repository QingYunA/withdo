package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newMem(t *testing.T) *Memory {
	t.Helper()
	return NewMemory(filepath.Join(t.TempDir(), "sub", "assistant-memory.md"))
}

func mustAdd(t *testing.T, m *Memory, section, note string) MemoryEdit {
	t.Helper()
	e, err := m.Add(section, note)
	if err != nil {
		t.Fatalf("Add(%q): %v", note, err)
	}
	return e
}

func TestMemoryMissingFileIsNoMemory(t *testing.T) {
	m := newMem(t)
	if got := m.Text(); got != "" {
		t.Errorf("Text = %q", got)
	}
	var off *Memory
	if off.Text() != "" {
		t.Error("nil memory has text")
	}
}

func TestMemoryAddsToSectionsInTheFileFormat(t *testing.T) {
	m := newMem(t)
	mustAdd(t, m, "about", "叫我 Cyan")
	mustAdd(t, m, "prefs", "健身归「生活」类，不是工作")
	mustAdd(t, m, "about", "用中文回复")
	want := "# 助手记忆\n\n## 关于我\n- 叫我 Cyan\n- 用中文回复\n\n## 纠正与偏好\n- 健身归「生活」类，不是工作\n"
	if got := m.Text(); got != want {
		t.Errorf("file:\n%s\nwant:\n%s", got, want)
	}
}

func TestMemorySkipsNormalizedDuplicates(t *testing.T) {
	m := newMem(t)
	mustAdd(t, m, "prefs", "Always reply in Chinese.")
	if e := mustAdd(t, m, "prefs", "  always reply  in chinese "); e.Changed != 0 {
		t.Errorf("duplicate added: %q", e.Text)
	}
	if n := countNotes(splitLines(m.Text())); n != 1 {
		t.Errorf("%d notes", n)
	}
}

func TestMemoryRejectsBadNotes(t *testing.T) {
	m := newMem(t)
	if _, err := m.Add("prefs", " \n "); err == nil {
		t.Error("empty note accepted")
	}
	if _, err := m.Add("prefs", strings.Repeat("长", 201)); err == nil {
		t.Error("201-character note accepted")
	}
	if _, err := m.Add("nope", "x"); err == nil {
		t.Error("unknown section accepted")
	}
	mustAdd(t, m, "prefs", "line one\nline two")
	contains(t, "file", m.Text(), "- line one line two\n")
}

func TestMemoryLimitsAskToMergeFirst(t *testing.T) {
	m := newMem(t)
	for i := 0; i < maxNotes; i++ {
		mustAdd(t, m, "prefs", fmt.Sprintf("note %d", i))
	}
	if _, err := m.Add("prefs", "one more"); err == nil || !strings.Contains(err.Error(), "forget") {
		t.Errorf("41st note: %v", err)
	}
	big := newMem(t)
	for i := 0; i < 30; i++ {
		_, err := big.Add("prefs", fmt.Sprintf("%d", i)+strings.Repeat("x", 190))
		if err != nil {
			contains(t, "error", err.Error(), "characters")
			if len(big.Text()) > maxMemoryText {
				t.Error("file grew past the limit")
			}
			return
		}
	}
	t.Error("no limit on characters")
}

func TestMemoryForget(t *testing.T) {
	m := newMem(t)
	mustAdd(t, m, "prefs", "健身归生活类")
	mustAdd(t, m, "prefs", "跑步归生活类")
	mustAdd(t, m, "about", "叫我 Cyan")
	if e, err := m.Forget("不存在"); err != nil || e.Changed != 0 {
		t.Errorf("no match: %+v %v", e, err)
	}
	e, err := m.Forget("生活类")
	if err != nil || e.Changed != 2 {
		t.Fatalf("forget: %+v %v", e, err)
	}
	if strings.Contains(m.Text(), "生活类") || !strings.Contains(m.Text(), "叫我 Cyan") {
		t.Errorf("file: %s", m.Text())
	}
	if _, err := m.Forget(" "); err == nil {
		t.Error("empty match accepted")
	}
}

func TestMemoryKeepsHandEditsAndRestores(t *testing.T) {
	m := newMem(t)
	hand := "# 助手记忆\n## 纠正与偏好\n- 手写的\n\n随手写的一段\n"
	if err := os.MkdirAll(filepath.Dir(m.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.Path(), []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}
	e := mustAdd(t, m, "prefs", "新的")
	contains(t, "file", m.Text(), "- 手写的\n- 新的")
	contains(t, "file", m.Text(), "随手写的一段")
	if err := m.Restore(e); err != nil || m.Text() != hand {
		t.Errorf("restore: %v\n%s", err, m.Text())
	}
	fresh := newMem(t)
	e = mustAdd(t, fresh, "about", "x")
	if err := fresh.Restore(e); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fresh.Path()); !os.IsNotExist(err) {
		t.Errorf("file left behind: %v", err)
	}
}

// TestUndoingAForgetKeepsALaterNote: undoing only the forget of "forget, then remember" brings the
// old note back and keeps the new one, instead of rolling the file back past the remember.
func TestUndoingAForgetKeepsALaterNote(t *testing.T) {
	m := newMem(t)
	mustAdd(t, m, "about", "叫我 Cyan")
	mustAdd(t, m, "prefs", "健身归「工作」类")
	forgot, err := m.Forget("健身")
	if err != nil || forgot.Changed != 1 {
		t.Fatalf("forget: %+v %v", forgot, err)
	}
	mustAdd(t, m, "prefs", "健身归「生活」类")
	if err := m.Restore(forgot); err != nil {
		t.Fatal(err)
	}
	contains(t, "file", m.Text(), "## 纠正与偏好\n- 健身归「生活」类\n- 健身归「工作」类")
	contains(t, "file", m.Text(), "## 关于我\n- 叫我 Cyan")
}

// TestUndoingARememberKeepsLaterHandEdits: undoing a remember takes out that note only.
func TestUndoingARememberKeepsLaterHandEdits(t *testing.T) {
	m := newMem(t)
	e := mustAdd(t, m, "prefs", "新的")
	if err := os.WriteFile(m.Path(), []byte(m.Text()+"- 后来手写的\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Restore(e); err != nil {
		t.Fatal(err)
	}
	if text := m.Text(); strings.Contains(text, "新的") || !strings.Contains(text, "- 后来手写的") {
		t.Errorf("file:\n%s", text)
	}
}

// TestUndoingARememberKeepsAFileWithHandWrittenText: the file the remember made stays once the
// user has written in it.
func TestUndoingARememberKeepsAFileWithHandWrittenText(t *testing.T) {
	m := newMem(t)
	e := mustAdd(t, m, "about", "x")
	if err := os.WriteFile(m.Path(), []byte(m.Text()+"\n随手写的一段\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Restore(e); err != nil {
		t.Fatal(err)
	}
	if text := m.Text(); strings.Contains(text, "- x") || !strings.Contains(text, "随手写的一段") {
		t.Errorf("file:\n%s", text)
	}
}

// TestUndoingAForgetOfANoteAboveTheHeadingsFilesItUnderPreferences: a note the user wrote above
// every heading comes back in the preferences section.
func TestUndoingAForgetOfANoteAboveTheHeadingsFilesItUnderPreferences(t *testing.T) {
	m := newMem(t)
	if err := os.MkdirAll(filepath.Dir(m.Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.Path(), []byte("- 顶上的\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := m.Forget("顶上")
	if err != nil || e.Changed != 1 {
		t.Fatalf("forget: %+v %v", e, err)
	}
	if err := m.Restore(e); err != nil {
		t.Fatal(err)
	}
	contains(t, "file", m.Text(), "## 纠正与偏好\n- 顶上的")
}
