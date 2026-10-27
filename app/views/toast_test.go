package views

import (
	"testing"
	"time"
)

func TestCopyForAgentNamesTheTasksTheWayAPersonWouldSayIt(t *testing.T) {
	for _, c := range []struct {
		ids  []int
		want string
	}{
		{[]int{12}, "Do Withdo task #12."},
		{[]int{31, 32}, "Do Withdo tasks #31 and #32."},
		{[]int{33, 31, 32}, "Do Withdo tasks #31, #32 and #33."},
	} {
		if got := copyForAgent(c.ids); got != c.want {
			t.Errorf("copyForAgent(%v) = %q, want %q", c.ids, got, c.want)
		}
	}
}

func TestTheHandOffToastOfOneTaskCopiesForTheAgentAndKeepsShowing(t *testing.T) {
	r := newHandRig(t, true)
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	has(t, r.tt, "Handed to an agent", "Copy for the agent", "Undo")
	if err := r.tt.Click("Copy for the agent"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if got := r.tt.Clipboard(); got != "Do Withdo task #5." {
		t.Errorf("the clipboard holds %q", got)
	}
	has(t, r.tt, "Copied", "Undo", "Handed to an agent")
	hasNot(t, r.tt, "Copy for the agent")
	r.after(copiedFor + 100*time.Millisecond)
	has(t, r.tt, "Copy for the agent")
}

func TestTheHandOffToastOfAStackCopiesAllTheirNumbers(t *testing.T) {
	r := newHandRig(t, true)
	cmdClick(t, r.tt, "Decide on the activation policy", "Write the release notes")
	r.tt.Frame()
	r.dragOnto(t, "Write the release notes", "Port core to Go", false)
	if err := r.tt.Click("Copy for the agent"); err != nil {
		t.Fatal(err)
	}
	if got := r.tt.Clipboard(); got != "Do Withdo tasks #5 and #6." {
		t.Errorf("the clipboard holds %q", got)
	}
}

func TestOtherToastsHaveNoCopyButton(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	cmdClick(t, tt, "Reply to design feedback", "Book dentist")
	tt.Frame()
	if err := tt.Click("Mark important"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	has(t, tt, "Undo")
	hasNot(t, tt, "Copy for the agent")
}
