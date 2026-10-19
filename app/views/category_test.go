package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The category of a task is set by hand in the inspector and in the task menu, like its priority.

func TestInspectorCategoryRowSitsUnderPriorityAndSetsTheCategory(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(18) // Look into Liquid Glass: nobody classified it
	tt.Frame()
	if !above(t, tt, "Priority", "Category") || !above(t, tt, "Category", "Project") {
		t.Errorf("Category goes between Priority and Project: %q", tt.Texts())
	}
	click(t, tt, "Category: None")
	for _, want := range []string{"Set category to Work", "Set category to Life", "Set category to None"} {
		if !tt.HasText(want) {
			t.Errorf("the Category value should show its strip, missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText("Set status to Done") {
		t.Error("the Category value opened more than its own piece")
	}
	click(t, tt, "Set category to Life")
	if got, _ := app.Task(18); got.Category != data.CategoryLife {
		t.Errorf("category %q", got.Category)
	}
	if !tt.HasText("Category: Life") {
		t.Errorf("the row does not show the new value: %q", tt.Texts())
	}
	click(t, tt, "Category: Life")
	click(t, tt, "Set category to None")
	if got, _ := app.Task(18); got.Category != "" {
		t.Errorf("category %q after None", got.Category)
	}
}

func TestTaskMenuMainPageHasACategoryStrip(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	for _, want := range []string{"Category", "Set category to Work", "Set category to Life", "Set category to None"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	if !above(t, tt, "Priority", "Category") || !above(t, tt, "Category", "Move to…") {
		t.Errorf("the strip sits under the priority strip, above Move to: %q", tt.Texts())
	}
	click(t, tt, "Set category to Life")
	if got, _ := app.Task(3); got.Category != data.CategoryLife {
		t.Errorf("category %q", got.Category)
	}
}

func TestCategoryMovesATaskBetweenTheSidebarSlices(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenMine, sub: subUnsorted})
	tt.Frame()
	if !tt.HasText("Look into Liquid Glass for the sidebar") {
		t.Fatalf("Unsorted: %q", tt.Texts())
	}
	rightClick(t, tt, "Look into Liquid Glass for the sidebar")
	click(t, tt, "Set category to Work")
	if tt.HasText("Look into Liquid Glass for the sidebar") {
		t.Errorf("a classified task is still under Unsorted: %q", tt.Texts())
	}
	m.goTo(nav{screen: screenMine, sub: subWork})
	tt.Frame()
	if !tt.HasText("Look into Liquid Glass for the sidebar") {
		t.Errorf("Work does not have it: %q", tt.Texts())
	}
}

func TestInChineseTheCategoryRowAndSlicesSpeakChinese(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	inChinese(tt)
	m.OpenTask(18)
	tt.Frame()
	for _, want := range []string{"工作", "生活", "未分类", "分类"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
}

// Work and Life carry their icons, a briefcase and a house, in the sidebar, the inspector and the
// strip; a task with no category shows none.

func TestWorkAndLifeSubRowsShowTheirIconAndUnsortedNone(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	for _, row := range []struct {
		label string
		icon  bool
	}{{"Work", true}, {"Life", true}, {"Unsorted", false}} {
		r, ok := tt.Find(row.label)
		if !ok {
			t.Fatalf("no %q in %q", row.label, tt.Texts())
		}
		slot := ui.Rect{X: r.X - sidebarRowGap - theme.AvatarSmall, Y: r.Y, W: theme.AvatarSmall, H: r.H}
		if got := paintedAnything(tt.Image(), 1100, slot); got != row.icon {
			t.Errorf("%s: an icon in its slot is %v, want %v", row.label, got, row.icon)
		}
	}
}

func TestTheCategoryStripPutsAnIconBeforeWorkAndLife(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	none := inkWidth(t, tt, "Set category to None")
	for _, label := range []string{"Set category to Work", "Set category to Life"} {
		if w := inkWidth(t, tt, label); w < none+12 {
			t.Errorf("%s draws %v pt against None's %v pt: no icon before its name", label, w, none)
		}
	}
}

func TestTheInspectorCategoryValueHasTheIconOfItsCategory(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(18) // Look into Liquid Glass: nobody classified it
	tt.Frame()
	none := inkWidth(t, tt, "Category: None")
	_ = app.Update(18, data.Patch{Category: data.CategoryPtr(data.CategoryLife)})
	tt.Frame()
	// Life is a shorter word than None; with the house before it, it draws wider.
	if life := inkWidth(t, tt, "Category: Life"); life < none+12 {
		t.Errorf("Life draws %v pt against None's %v pt: no icon before it", life, none)
	}
}

// inkWidth is how wide, in points, the drawing inside an element is: from its leftmost to its
// rightmost pixel that differs from the element's own background, read just under its top edge.
func inkWidth(t *testing.T, tt *ui.Tester, label string) float32 {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("no %q in %q", label, tt.Texts())
	}
	img := tt.Image()
	scale := float32(img.Bounds().Dx()) / 1100
	base := img.RGBAAt(int((r.X+r.W/2)*scale), int((r.Y+1)*scale))
	left, right := -1, -1
	for y := int((r.Y + 1) * scale); y < int((r.Y+r.H-1)*scale); y++ {
		for x := int((r.X + 8) * scale); x < int((r.X+r.W-8)*scale); x++ {
			p := img.RGBAAt(x, y)
			if absInt(int(p.R)-int(base.R))+absInt(int(p.G)-int(base.G))+absInt(int(p.B)-int(base.B)) > 60 {
				if left < 0 || x < left {
					left = x
				}
				right = max(right, x)
			}
		}
	}
	if left < 0 {
		return 0
	}
	return float32(right-left+1) / scale
}
