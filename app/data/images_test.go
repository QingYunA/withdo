package data

import (
	"bytes"
	"context"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func noisePNG(t *testing.T, w, h int) []byte { t.Helper(); return pngOf(t, noise(w, h)) }

// bigPNG is a picture that PrepareImage shrinks: 2400 x 1200, so its prepared copy differs. Only
// the tests of that difference need one; the race detector makes its pixels slow.
func bigPNG(t *testing.T) []byte {
	t.Helper()
	return pngOf(t, image.NewNRGBA(image.Rect(0, 0, 2400, 1200)))
}

// smallPNG is a picture PrepareImage leaves as it came, for the tests of how pictures are kept,
// linked and collected. noisePNG(t, 50, 50) is another one.
func smallPNG(t *testing.T) []byte { t.Helper(); return noisePNG(t, 40, 30) }

// skipUnderRace skips a test that only turns pixels, single-threaded: the race detector has
// nothing to find there and makes it many times slower. The pass without it still runs it.
func skipUnderRace(t *testing.T) {
	t.Helper()
	if raceDetector {
		t.Skip("pixel work with nothing concurrent; it runs without -race")
	}
}

func openImagesReal(t *testing.T, grace time.Duration) *Real {
	t.Helper()
	r, err := OpenReal(bg, filepath.Join(t.TempDir(), "t.db"), RealOptions{DataFolder: t.TempDir(), ImageGrace: grace})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func newTask(t *testing.T, r *Real, title string) Task {
	t.Helper()
	task, err := r.CreateTask(CreateTaskInput{Title: title})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func attach(t *testing.T, r *Real, task Task, a Attachment) TaskImage {
	t.Helper()
	got, err := r.AttachImages(task.ID, []Attachment{a})
	if err != nil || len(got) != 1 {
		t.Fatalf("AttachImages = %v, %v", got, err)
	}
	return got[0]
}

func prepared(t *testing.T, name string, raw []byte) Attachment {
	t.Helper()
	a, err := PrepareImage(name, raw)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestAPreparedPictureCarriesItsOriginalOnlyWhenTheyDiffer(t *testing.T) {
	skipUnderRace(t)
	small := pngOf(t, image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if a := prepared(t, "a.png", small); a.Original != nil {
		t.Errorf("a picture that went out as it came has no separate original")
	}
	raw := bigPNG(t)
	a := prepared(t, "big.png", raw)
	if !bytes.Equal(a.Original, raw) || a.OriginalMIME != "image/png" || bytes.Equal(a.Data, raw) {
		t.Fatalf("original %d bytes (%s), prepared %d bytes", len(a.Original), a.OriginalMIME, len(a.Data))
	}
	if data, mime := a.Stored(); !bytes.Equal(data, raw) || mime != "image/png" {
		t.Errorf("Stored is the original")
	}
}

func TestAnOriginalOverTheLimitIsLeftOutAndThePreparedCopyIsKept(t *testing.T) {
	skipUnderRace(t)
	defer func(v int) { originalLimit = v }(originalLimit)
	originalLimit = 100
	a := prepared(t, "big.png", bigPNG(t))
	if a.Original != nil {
		t.Fatalf("the original is over the limit but was kept: %d bytes", len(a.Original))
	}
	if data, mime := a.Stored(); !bytes.Equal(data, a.Data) || mime != a.MIME {
		t.Errorf("Stored is the prepared copy")
	}
}

func TestAttachKeepsTheOriginalWithAThumbnailBesideIt(t *testing.T) {
	r := openImagesReal(t, 0)
	task := newTask(t, r, "Pay rent")
	raw := bigPNG(t)
	link := attach(t, r, task, prepared(t, "bill.png", raw))

	if link.Name != "bill.png" || link.MIME != "image/png" || link.Width != 2400 || link.Height != 1200 || link.TaskID != task.ID {
		t.Errorf("link = %+v", link)
	}
	if got, err := os.ReadFile(link.File); err != nil || !bytes.Equal(got, raw) {
		t.Errorf("the stored file is not the original: %v", err)
	}
	if filepath.Base(filepath.Dir(link.File)) != "images" || filepath.Dir(link.File) != filepath.Dir(link.Thumb) {
		t.Errorf("files in %s and %s", link.File, link.Thumb)
	}
	thumb, err := LoadThumbnail(link.Thumb)
	if err != nil || thumb.Bounds().Dx() != ThumbEdge || thumb.Bounds().Dy() != ThumbEdge/2 {
		t.Errorf("thumbnail = %v, %v", thumb, err)
	}
}

func TestAttachFallsBackToThePreparedCopyWithoutAnOriginal(t *testing.T) {
	r := openImagesReal(t, 0)
	task := newTask(t, r, "Pay rent")
	a := prepared(t, "bill.png", pngOf(t, image.NewNRGBA(image.Rect(0, 0, 40, 30))))
	link := attach(t, r, task, a)
	if got, _ := os.ReadFile(link.File); !bytes.Equal(got, a.Data) {
		t.Error("the stored file is not the prepared copy")
	}
}

func TestOnePictureOnTwoTasksIsStoredOnce(t *testing.T) {
	r := openImagesReal(t, 0)
	a, b := newTask(t, r, "a"), newTask(t, r, "b")
	pic := prepared(t, "bill.png", smallPNG(t))
	first := attach(t, r, a, pic)
	again := attach(t, r, a, prepared(t, "renamed.png", smallPNG(t)))
	second := attach(t, r, b, pic)
	if again.Hash != first.Hash || again.Name != "bill.png" || second.Hash != first.Hash || second.File != first.File {
		t.Fatalf("links %+v %+v %+v", first, again, second)
	}
	files, _ := os.ReadDir(filepath.Dir(first.File))
	if len(files) != 2 {
		t.Errorf("%d files, want the picture and its thumbnail", len(files))
	}
	if links, _ := r.TaskImages(a.ID); len(links) != 1 {
		t.Errorf("task a has %d pictures", len(links))
	}
}

func TestImagesOfReadsManyTasksAtOnce(t *testing.T) {
	r := openImagesReal(t, 0)
	a, b, none := newTask(t, r, "a"), newTask(t, r, "b"), newTask(t, r, "none")
	attach(t, r, a, prepared(t, "x.png", smallPNG(t)))
	attach(t, r, b, prepared(t, "y.png", noisePNG(t, 50, 50)))
	got, err := r.ImagesOf([]int{a.ID, b.ID, none.ID})
	if err != nil || len(got) != 2 || got[a.ID][0].Name != "x.png" || got[b.ID][0].Name != "y.png" {
		t.Fatalf("ImagesOf = %+v, %v", got, err)
	}
}

func TestCollectDeletesTheFilesNoTaskLinks(t *testing.T) {
	r := openImagesReal(t, 0)
	a, b := newTask(t, r, "a"), newTask(t, r, "b")
	shared := attach(t, r, a, prepared(t, "s.png", smallPNG(t)))
	attach(t, r, b, prepared(t, "s.png", smallPNG(t)))
	own := attach(t, r, a, prepared(t, "o.png", noisePNG(t, 50, 50)))

	if err := r.DetachImage(a.ID, shared.Hash); err != nil {
		t.Fatal(err)
	}
	if err := r.DetachImage(a.ID, own.Hash); err != nil {
		t.Fatal(err)
	}
	if !exists(own.File) {
		t.Fatal("the files stay until the collection, for Undo")
	}
	if n, err := r.CollectImages(); err != nil || n != 1 {
		t.Fatalf("CollectImages = %d, %v", n, err)
	}
	if exists(own.File) || exists(own.Thumb) {
		t.Error("the files of a picture no task links are still there")
	}
	if !exists(shared.File) || !exists(shared.Thumb) {
		t.Error("task b still has the shared picture")
	}
}

func TestDeletingATaskCollectsItsPicturesAfterTheGrace(t *testing.T) {
	r := openImagesReal(t, 20*time.Millisecond)
	task := newTask(t, r, "a")
	link := attach(t, r, task, prepared(t, "s.png", smallPNG(t)))
	if err := r.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the files to go", func() bool { return !exists(link.File) && !exists(link.Thumb) })
}

func TestUndoingTheDeleteOfATaskBringsItsPicturesBack(t *testing.T) {
	r := openImagesReal(t, time.Hour)
	task := newTask(t, r, "a")
	link := attach(t, r, task, prepared(t, "s.png", smallPNG(t)))
	snap, err := r.SnapshotTask(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RestoreTask(snap); err != nil {
		t.Fatal(err)
	}
	got, _ := r.TaskImages(task.ID)
	if len(got) != 1 || got[0].Hash != link.Hash || !exists(got[0].Thumb) {
		t.Errorf("pictures after the Undo = %+v", got)
	}
}

func TestUndoingADetachLinksThePictureAgain(t *testing.T) {
	r := openImagesReal(t, time.Hour)
	task := newTask(t, r, "a")
	link := attach(t, r, task, prepared(t, "s.png", smallPNG(t)))
	if err := r.DetachImage(task.ID, link.Hash); err != nil {
		t.Fatal(err)
	}
	if err := r.RestoreImage(link); err != nil {
		t.Fatal(err)
	}
	got, _ := r.TaskImages(task.ID)
	if len(got) != 1 || got[0] != link {
		t.Errorf("pictures after the Undo = %+v, want %+v", got, link)
	}
}

func TestWithoutADataFolderThereIsNowhereToKeepPictures(t *testing.T) {
	r, err := OpenReal(bg, filepath.Join(t.TempDir(), "t.db"), RealOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	task := newTask(t, r, "a")
	if _, err := r.AttachImages(task.ID, []Attachment{prepared(t, "s.png", smallPNG(t))}); err != ErrNoImageFolder {
		t.Errorf("err = %v", err)
	}
}

func TestPicturesFollowATaskThroughDeleteAndRestore(t *testing.T) {
	f := openImagesReal(t, time.Hour) // only the CollectImages below collects, not a sweep a slow machine lets run first
	task, _ := f.CreateTask(CreateTaskInput{Title: "a"})
	got, err := f.AttachImages(task.ID, []Attachment{prepared(t, "s.png", smallPNG(t)), prepared(t, "s2.png", noisePNG(t, 50, 50))})
	if err != nil || len(got) != 2 {
		t.Fatalf("AttachImages = %v, %v", got, err)
	}
	all, _ := f.ImagesOf([]int{task.ID})
	if len(all[task.ID]) != 2 || !exists(got[0].Thumb) {
		t.Fatalf("ImagesOf = %+v", all)
	}
	snap, _ := f.SnapshotTask(task.ID)
	_ = f.DeleteTask(task.ID)
	if all, _ := f.ImagesOf([]int{task.ID}); len(all) != 0 {
		t.Fatal("a deleted task has no pictures")
	}
	_, _ = f.RestoreTask(snap)
	_ = f.DetachImage(task.ID, got[1].Hash)
	if n, _ := f.CollectImages(); n != 1 || exists(got[1].File) || !exists(got[0].File) {
		t.Errorf("collected %d", n)
	}
}

func imageAssistant(t *testing.T, steps ...func(context.Context) (*schema.Message, error)) (*AgentAssistant, *Real) {
	t.Helper()
	r := openImagesReal(t, 20*time.Millisecond)
	sm := &scriptModel{steps: steps}
	ag := agent.New(agent.Config{
		Core: r.Core(), Model: func(context.Context) (model.ToolCallingChatModel, error) { return sm, nil },
	})
	return &AgentAssistant{Agent: ag, Provider: newManager(t, providers.NewMemoryKeyring(), nil), Images: r}, r
}

func said(text string) func(context.Context) (*schema.Message, error) {
	return func(context.Context) (*schema.Message, error) { return schema.AssistantMessage(text, nil), nil }
}

func TestTheTurnsPicturesAreKeptByTheTasksItCreated(t *testing.T) {
	a, r := imageAssistant(t,
		toolCall("create_task", `{"title":"Pay rent","images":[0]}`),
		toolCall("create_task", `{"title":"Buy milk"}`), said("ok"))
	bill, list := prepared(t, "bill.png", smallPNG(t)), prepared(t, "list.png", noisePNG(t, 50, 50))
	reply, err := a.Ask(bg, "these", Place{}, bill, list)
	if err != nil || len(reply.Receipts) != 2 {
		t.Fatalf("reply %+v, %v", reply, err)
	}
	rent, milk := reply.Receipts[0], reply.Receipts[1]
	if rent.Images != 1 || milk.Images != 2 {
		t.Errorf("receipts say %d and %d pictures", rent.Images, milk.Images)
	}
	got, _ := r.TaskImages(rent.TaskID)
	if len(got) != 1 || got[0].Name != "bill.png" {
		t.Errorf("Pay rent keeps %+v", got)
	}
	got, _ = r.TaskImages(milk.TaskID)
	if len(got) != 2 {
		t.Errorf("Buy milk keeps %+v", got)
	}
}

func TestARequestFromTheInspectorGivesItsPicturesToThatTask(t *testing.T) {
	a, r := imageAssistant(t, said("Added."))
	task := newTask(t, r, "Pay rent")
	if _, err := a.Ask(bg, "add this", Place{Kind: PlaceTask, TaskID: task.ID}, prepared(t, "bill.png", smallPNG(t))); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.TaskImages(task.ID); len(got) != 1 {
		t.Errorf("the task keeps %+v", got)
	}
}

func TestUndoingATaskTheTurnCreatedCollectsItsPictures(t *testing.T) {
	a, r := imageAssistant(t, toolCall("create_task", `{"title":"Pay rent"}`), said("ok"))
	reply, err := a.Ask(bg, "this", Place{}, prepared(t, "bill.png", smallPNG(t)))
	if err != nil {
		t.Fatal(err)
	}
	links, _ := r.TaskImages(reply.Receipts[0].TaskID)
	if err := a.Undo(reply.Receipts[0].ID); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the files to go", func() bool { return !exists(links[0].File) })
}

func TestPicturesThatCannotBeKeptAreSaidButTheTasksStay(t *testing.T) {
	a, r := imageAssistant(t, toolCall("create_task", `{"title":"Pay rent"}`), said("ok"))
	r.images = imageFiles{} // nowhere to write
	reply, err := a.Ask(bg, "this", Place{}, prepared(t, "bill.png", smallPNG(t)))
	if err == nil || len(reply.Receipts) != 1 {
		t.Fatalf("reply %+v, err %v", reply, err)
	}
	if reply.Receipts[0].Images != 0 {
		t.Error("the receipt claims pictures that were not kept")
	}
}

func TestCollectSoonTakesThePicturesAnotherProcessLetGo(t *testing.T) {
	r := openImagesReal(t, 20*time.Millisecond)
	task := newTask(t, r, "a")
	link := attach(t, r, task, prepared(t, "s.png", smallPNG(t)))
	if _, err := r.Core().DetachImage(bg, int64(task.ID), link.Hash); err != nil { // the MCP server's doing: no sweep is scheduled
		t.Fatal(err)
	}
	r.CollectSoon()
	waitFor(t, "the files to go", func() bool { return !exists(link.File) })
}
