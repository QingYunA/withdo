package agent

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func twoPictures() []Image {
	return []Image{{MIME: "image/png", Data: []byte("bill")}, {MIME: "image/png", Data: []byte("list")}}
}

func (r *rig) askWith(scope Scope, text string, images ...Image) Result {
	r.t.Helper()
	res, err := r.agent.Ask(context.Background(), text, scope, images...)
	if err != nil {
		r.t.Fatalf("Ask(%q): %v", text, err)
	}
	return res
}

func wantKeeps(t *testing.T, got []ImageKeep, want ...ImageKeep) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("keeps = %+v, want %+v", got, want)
	}
}

func TestEachTaskKeepsTheImagesTheModelNamedForIt(t *testing.T) {
	r := newRig(t, "UTC", oct14,
		call("create_task", m{"title": "Pay rent", "images": []int{0}}),
		call("create_task", m{"title": "Buy milk", "images": []int{1}}), say("ok"))
	res := r.askWith(Scope{}, "do these", twoPictures()...)
	wantKeeps(t, res.Images,
		ImageKeep{TaskID: res.Receipts[0].TaskID, Indexes: []int{0}},
		ImageKeep{TaskID: res.Receipts[1].TaskID, Indexes: []int{1}})
}

func TestATaskTheModelNamedNoImagesForKeepsAllOfThem(t *testing.T) {
	r := newRig(t, "UTC", oct14, call("create_task", m{"title": "Pay rent"}), say("ok"))
	res := r.askWith(Scope{}, "do these", twoPictures()...)
	wantKeeps(t, res.Images, ImageKeep{TaskID: res.Receipts[0].TaskID, Indexes: []int{0, 1}})
}

func TestAnEmptyListMeansNoImages(t *testing.T) {
	r := newRig(t, "UTC", oct14, call("create_task", m{"title": "Pay rent", "images": []int{}}), say("ok"))
	res := r.askWith(Scope{}, "do these", twoPictures()...)
	if len(res.Images) != 0 {
		t.Errorf("keeps = %+v, want none", res.Images)
	}
}

func TestAnUpdateKeepsImagesToo(t *testing.T) {
	r := newRig(t, "UTC", oct14)
	tk := r.mustTask(domainTask("old"))
	r.model.steps = []step{call("update_task", m{"task_id": tk.ID, "title": "new", "images": []int{1, 0, 1}}), say("ok")}
	res := r.askWith(Scope{}, "rename it", twoPictures()...)
	wantKeeps(t, res.Images, ImageKeep{TaskID: tk.ID, Indexes: []int{0, 1}})
}

func TestAPictureThatDoesNotExistIsRefusedAndNothingIsCreated(t *testing.T) {
	r := newRig(t, "UTC", oct14, call("create_task", m{"title": "Pay rent", "images": []int{2}}), say("ok"))
	res := r.askWith(Scope{}, "do these", twoPictures()...)
	if len(res.Receipts) != 0 || len(res.Images) != 0 {
		t.Fatalf("a refused call left %+v", res)
	}
	contains(t, "the model was told", lastToolText(r.model.sent(1)), "picture 2 does not exist")
}

func TestWithoutPicturesTheTasksKeepNone(t *testing.T) {
	r := newRig(t, "UTC", oct14, call("create_task", m{"title": "x"}), say("ok"))
	if res := r.ask("x"); len(res.Images) != 0 {
		t.Errorf("keeps = %+v", res.Images)
	}
	r = newRig(t, "UTC", oct14, call("create_task", m{"title": "x", "images": []int{0}}), say("ok"))
	r.ask("x")
	contains(t, "told", lastToolText(r.model.sent(1)), "no pictures")
}

func TestARequestFromTheInspectorAttachesItsImagesToThatTask(t *testing.T) {
	r := newRig(t, "UTC", oct14, say("Done."))
	tk := r.mustTask(domainTask("Pay rent"))
	res := r.askWith(Scope{Kind: ScopeTask, TaskID: tk.ID}, "add these", twoPictures()...)
	wantKeeps(t, res.Images, ImageKeep{TaskID: tk.ID, Indexes: []int{0, 1}})
}

func TestAFailedInspectorRequestAttachesNothing(t *testing.T) {
	r := newRig(t, "UTC", oct14, fail401)
	tk := r.mustTask(domainTask("Pay rent"))
	res, err := r.agent.Ask(context.Background(), "add these", Scope{Kind: ScopeTask, TaskID: tk.ID}, twoPictures()...)
	if err == nil || len(res.Images) != 0 {
		t.Fatalf("err = %v, keeps = %+v", err, res.Images)
	}
}

func TestThePromptTellsAboutThePicturesOnlyWhenThereAreSome(t *testing.T) {
	r := newRig(t, "UTC", oct14, say("a"), say("b"))
	r.askWith(Scope{}, "look", twoPictures()...)
	contains(t, "system prompt", r.model.sent(0)[0].Content, "numbered 0 to 1")
	r.askWith(Scope{}, "again")
	if got := r.model.sent(1)[0].Content; strings.Contains(got, "numbered 0 to") {
		t.Errorf("a request without pictures mentions them")
	}
}

func domainTask(title string) domain.CreateTaskInput { return domain.CreateTaskInput{Title: title} }
