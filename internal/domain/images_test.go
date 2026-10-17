package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func pic(hash, name string) domain.AttachImageInput {
	return domain.AttachImageInput{
		Image: domain.Image{Hash: hash, Ext: "png", MIME: "image/png", Bytes: 10},
		Name:  name, Width: 1200, Height: 900,
	}
}

func hashes(imgs []domain.TaskImage) []string {
	out := []string{}
	for _, im := range imgs {
		out = append(out, im.Hash)
	}
	return out
}

func TestAttachImageIsIdempotentPerTask(t *testing.T) {
	c, clk := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "Pay rent"})
	first := must[*domain.TaskImage](t)(c.AttachImage(ctx, task.ID, pic("aa", "bill.png")))
	eq(t, "name", first.Name, "bill.png")
	eq(t, "size", [2]int{first.Width, first.Height}, [2]int{1200, 900})
	clk.advance(time.Hour)
	again := must[*domain.TaskImage](t)(c.AttachImage(ctx, task.ID, pic("aa", "other-name.png")))
	eq(t, "kept link", *again, *first)
	same(t, "images", hashes(must[[]domain.TaskImage](t)(c.ListTaskImages(ctx, task.ID))), []string{"aa"})
}

func TestOnePictureCanBelongToSeveralTasks(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic("aa", "x.png")))
	must[*domain.TaskImage](t)(c.AttachImage(ctx, b.ID, pic("aa", "y.png")))
	eq(t, "a", must[[]domain.TaskImage](t)(c.ListTaskImages(ctx, a.ID))[0].Name, "x.png")
	eq(t, "b", must[[]domain.TaskImage](t)(c.ListTaskImages(ctx, b.ID))[0].Name, "y.png")
	if gone := must[[]domain.Image](t)(c.CollectImages(ctx)); len(gone) != 0 {
		t.Errorf("a picture both tasks use is not collected: %+v", gone)
	}
}

func same(t *testing.T, name string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %+v, want %+v", name, got, want)
	}
}

func TestAttachImageToAMissingTask(t *testing.T) {
	c, _ := newCore(t)
	if _, err := c.AttachImage(ctx, 99, pic("aa", "x.png")); !domain.IsNotFound(err) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestListImagesOfManyTasksInTheOrderAdded(t *testing.T) {
	c, clk := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	none := mustTask(t, c, domain.CreateTaskInput{Title: "none"})
	for _, h := range []string{"c3", "a1", "b2"} {
		must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic(h, h)))
		clk.advance(time.Second)
	}
	must[*domain.TaskImage](t)(c.AttachImage(ctx, b.ID, pic("a1", "a1")))
	got := must[map[int64][]domain.TaskImage](t)(c.ListImagesOf(ctx, []int64{a.ID, b.ID, none.ID}))
	same(t, "a", hashes(got[a.ID]), []string{"c3", "a1", "b2"})
	same(t, "b", hashes(got[b.ID]), []string{"a1"})
	if _, ok := got[none.ID]; ok || len(got) != 2 {
		t.Errorf("a task without pictures is not in the map: %v", got)
	}
	if len(must[map[int64][]domain.TaskImage](t)(c.ListImagesOf(ctx, nil))) != 0 {
		t.Error("no ids, no pictures")
	}
}

func TestDetachAndCollect(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic("aa", "x")))
	must[*domain.TaskImage](t)(c.AttachImage(ctx, b.ID, pic("aa", "x")))
	must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic("bb", "y")))

	if removed := must[bool](t)(c.DetachImage(ctx, a.ID, "bb")); !removed {
		t.Fatal("detach found no link")
	}
	if removed := must[bool](t)(c.DetachImage(ctx, a.ID, "bb")); removed {
		t.Fatal("a second detach finds nothing")
	}
	gone := must[[]domain.Image](t)(c.CollectImages(ctx))
	if len(gone) != 1 || gone[0].Hash != "bb" || gone[0].Ext != "png" {
		t.Fatalf("collected %+v, want only bb", gone)
	}
	must[bool](t)(c.DetachImage(ctx, a.ID, "aa"))
	if gone := must[[]domain.Image](t)(c.CollectImages(ctx)); len(gone) != 0 {
		t.Fatalf("aa is still on task b: %+v", gone)
	}
}

func TestDeletingATaskDropsItsLinksAndTheCollectorTakesTheRest(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic("aa", "x")))
	if err := c.DeleteTask(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if gone := must[[]domain.Image](t)(c.CollectImages(ctx)); len(gone) != 1 || gone[0].Hash != "aa" {
		t.Fatalf("collected %+v", gone)
	}
}

func TestDeletingAProjectDropsTheLinksOfItsTasks(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a", ProjectID: &p.ID})
	must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic("aa", "x")))
	if _, err := c.DeleteProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if gone := must[[]domain.Image](t)(c.CollectImages(ctx)); len(gone) != 1 {
		t.Fatalf("collected %+v", gone)
	}
}

func TestRestoreTaskBringsBackItsPictures(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	want := must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic("aa", "x")))
	snap := must[*domain.TaskSnapshot](t)(c.SnapshotTask(ctx, a.ID))
	if err := c.DeleteTask(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	must[*domain.Task](t)(c.RestoreTask(ctx, *snap))
	same(t, "images", must[[]domain.TaskImage](t)(c.ListTaskImages(ctx, a.ID)), []domain.TaskImage{*want})
}

func TestRestoreImageIsTheUndoOfDetach(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	im := must[*domain.TaskImage](t)(c.AttachImage(ctx, a.ID, pic("aa", "x")))
	must[bool](t)(c.DetachImage(ctx, a.ID, "aa"))
	if err := c.RestoreImage(ctx, *im); err != nil {
		t.Fatal(err)
	}
	if err := c.RestoreImage(ctx, *im); err != nil {
		t.Fatalf("restoring twice: %v", err)
	}
	same(t, "images", must[[]domain.TaskImage](t)(c.ListTaskImages(ctx, a.ID)), []domain.TaskImage{*im})
}
