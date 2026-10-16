package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
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
