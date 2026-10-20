package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// filedReply is a turn in which the assistant sorted task 5 under Life by itself (auto) or the
// user named the category (not auto). The script files the task the way the real assistant does.
func filedReply(store *data.Real, auto bool) func(string) (data.Reply, error) {
	return func(string) (data.Reply, error) {
		life := 2
		if _, err := store.UpdateTask(5, data.Patch{Category: &life}); err != nil {
			return data.Reply{}, err
		}
		return data.Reply{Text: "ok", Receipts: []data.Receipt{{
			ID: 11, TaskID: 5, Title: "Pay rent", Status: data.Todo,
			Detail: data.Detail{Kind: data.DetailAdded, Category: data.BuiltinLife, AutoFiled: auto},
		}}}, nil
	}
}

func newFiledRig(t *testing.T, auto bool) *assistRig {
	t.Helper()
	app, store, rec := newTestApp(t)
	fa := &data.FakeAssistant{Ready: true, Script: filedReply(store, auto)}
	app.Assistant, app.Models = fa, newFakeModels(t)
	r := &assistRig{app: app, rec: rec, fa: fa, now: fixedNow()}
	app.Now = func() time.Time { return r.now }
	r.tt, r.m, _ = newTesterFor(app, 1100)
	return r
}

func TestAnAutoFiledTaskGetsAQuietToastWithUndoThatUnsortsIt(t *testing.T) {
	r := newFiledRig(t, true)
	r.ask("pay rent")
	r.answer(t)
	r.tt.Frame()
	has(t, r.tt, "Filed under Life")
	if task, _ := r.app.Task(5); task.CategoryID != 2 {
		t.Fatalf("category = %d, want Life (2)", task.CategoryID)
	}
	r.m.closeSheet() // its receipt line has an Undo of its own
	r.tt.Frame()
	r.tt.Frame() // the toast rides on the dock's height in the frame before, which closing shrank
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if task, _ := r.app.Task(5); task.CategoryID != 0 {
		t.Errorf("category = %d after Undo, want 0", task.CategoryID)
	}
	hasNot(t, r.tt, "Filed under Life")
}

func TestATaskTheUserFiledGetsNoFilingToast(t *testing.T) {
	r := newFiledRig(t, false)
	r.ask("pay rent, under life")
	r.answer(t)
	r.tt.Frame()
	hasNot(t, r.tt, "Filed under")
}

func TestTheFilingToastSitsAboveTheComposerDock(t *testing.T) {
	r := newFiledRig(t, true)
	r.ask("pay rent")
	r.answer(t)
	r.tt.Frame()
	toast, ok := r.tt.Find("Filed under Life")
	if !ok {
		t.Fatalf("no toast in %q", r.tt.Texts())
	}
	input, ok := r.tt.Find("Ask the assistant")
	if !ok {
		t.Fatalf("no composer in %q", r.tt.Texts())
	}
	if toast.Y+toast.H > input.Y {
		t.Errorf("toast %+v covers the composer %+v", toast, input)
	}
}
