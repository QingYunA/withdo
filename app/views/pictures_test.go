package views

import (
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Pictures of tasks: tiles in the inspector, thumbnails in lists, the band on a card, the viewer.

// picRig is the main window over the seeded fake, with a folder for pictures.
type picRig struct {
	tt    *ui.Tester
	m     *Main
	app   *App
	store *data.Fake
	rec   *recorder
}

var (
	red  = color.NRGBA{R: 230, G: 40, B: 40, A: 255}
	blue = color.NRGBA{R: 40, G: 60, B: 230, A: 255}
)

func newPicRig(t *testing.T) *picRig {
	t.Helper()
	app, store, rec := newTestApp(t)
	store.SetImageDir(t.TempDir())
	tt, m, _ := newTesterFor(app, 1100)
	return &picRig{tt: tt, m: m, app: app, store: store, rec: rec}
}

// give attaches a one-color picture to a task, as a turn of the assistant would.
func (r *picRig) give(t *testing.T, task int, name string, c color.NRGBA) data.TaskImage {
	t.Helper()
	att, err := data.PrepareImage(name, pngBytes(t, 120, 90, c))
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.store.AttachImages(task, []data.Attachment{att})
	if err != nil || len(got) != 1 {
		t.Fatalf("AttachImages = %v, %v", got, err)
	}
	r.app.Refresh()
	return got[0]
}

// settle draws the frame and waits for the thumbnails and pictures being read, then draws again.
func (r *picRig) settle(t *testing.T) {
	t.Helper()
	r.tt.Frame()
	for len(r.m.thumbs.loading) > 0 || len(r.m.viewer.loading) > 0 {
		r.rec.drain(t)
		r.tt.Frame()
	}
}

func (r *picRig) open(t *testing.T, id int) {
	t.Helper()
	r.m.OpenTask(id)
	r.settle(t)
}

// openAside opens a task in the inspector over the Completed page, which does not list it: a
// picture's label then names its tile alone, not also its row.
func (r *picRig) openAside(t *testing.T, id int) {
	t.Helper()
	r.m.nav, r.m.openID, r.m.sel = nav{screen: screenCompleted}, id, id
	r.m.inspector.reset()
	r.settle(t)
}

func hasExactly(tt *ui.Tester, text string) bool { return slices.Contains(tt.Texts(), text) }

func TestATaskWithoutImagesHasOnlyAQuietLineToAddThem(t *testing.T) {
	r := newPicRig(t)
	r.openAside(t, 5)
	has(t, r.tt, "Add images…")
	if hasExactly(r.tt, "Add images") {
		t.Error("the + tile shows with no picture")
	}
}

func TestTheInspectorShowsTilesAndADashedPlusAfterThem(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.give(t, 5, "list.png", blue)
	r.openAside(t, 5)
	hasNot(t, r.tt, "Add images…")
	for _, label := range []string{"Open bill.png", "Open list.png", "Add images"} {
		if _, ok := r.tt.Find(label); !ok {
			t.Errorf("no %q in %q", label, r.tt.Texts())
		}
	}
	tile, _ := r.tt.Find("Open bill.png")
	if tile.W != tileW || tile.H != tileH {
		t.Errorf("tile is %v x %v, want 96 x 72", tile.W, tile.H)
	}
	if !paintedIn(r.tt.Image(), 1100, tile, ui.Hex("#E62828"), 12) {
		t.Error("the picture is not drawn in its tile")
	}
	list, _ := r.tt.Find("Open list.png")
	add, _ := r.tt.Find("Add images")
	if list.X-tile.X != tileW+tileGap || add.X-list.X != tileW+tileGap || add.Y != tile.Y {
		t.Errorf("tiles at %v %v %v: want 3 a row, 12 apart", tile, list, add)
	}
}

func TestAFourthPictureStartsTheSecondRowOfTiles(t *testing.T) {
	r := newPicRig(t)
	for i, n := range []string{"a.png", "b.png", "c.png"} {
		r.give(t, 5, n, color.NRGBA{R: uint8(60 * (i + 1)), G: 90, B: 90, A: 255}) // the same picture would be one
	}
	r.give(t, 5, "d.png", blue)
	r.openAside(t, 5)
	first, _ := r.tt.Find("Open a.png")
	fourth, _ := r.tt.Find("Open d.png")
	add, _ := r.tt.Find("Add images")
	if fourth.X != first.X || fourth.Y <= first.Y || add.Y != fourth.Y || add.X-fourth.X != tileW+tileGap {
		t.Errorf("rows: first %v, fourth %v, plus %v", first, fourth, add)
	}
}

func TestAHoveredTileShowsTheXThatDetachesWithUndo(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.give(t, 5, "list.png", blue)
	r.openAside(t, 5)
	if _, ok := r.tt.Find("Remove bill.png from this task"); ok {
		t.Fatal("the x shows before the tile is hovered")
	}
	tile, _ := r.tt.Find("Open bill.png")
	r.tt.Move(tile.X+10, tile.Y+10)
	r.tt.Frame()
	if err := r.tt.Click("Remove bill.png from this task"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if n := len(r.app.Images(5)); n != 1 {
		t.Fatalf("%d pictures after the x, want 1", n)
	}
	has(t, r.tt, "Removed bill.png", "Undo")
	if err := r.tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	r.settle(t)
	got := r.app.Images(5)
	if len(got) != 2 || !slices.ContainsFunc(got, func(i data.TaskImage) bool { return i.Name == "bill.png" }) {
		t.Errorf("after Undo: %+v", got)
	}
}

// waitImages lets the goroutine that reads and keeps dropped files finish, until the task has n pictures.
func (r *picRig) waitImages(t *testing.T, task, n int) {
	t.Helper()
	for i := 0; len(r.app.Images(task)) < n && i < 20; i++ {
		r.rec.drain(t)
	}
	r.settle(t)
	if got := len(r.app.Images(task)); got != n {
		t.Fatalf("task %d has %d pictures, want %d", task, got, n)
	}
}

func TestImageFilesDroppedOnTheInspectorAreKeptWithTheTask(t *testing.T) {
	r := newPicRig(t)
	r.openAside(t, 5)
	r.m.attachToTask(5, picFiles(t, "one.png", "two.png"))
	r.waitImages(t, 5, 2)
	names := []string{r.app.Images(5)[0].Name, r.app.Images(5)[1].Name}
	if !slices.Equal(names, []string{"one.png", "two.png"}) {
		t.Errorf("names %v", names)
	}
	has(t, r.tt, "Open one.png", "Open two.png")
}

func TestAFileThatIsNotAnImageIsSaidInAQuietLineAndNotTaken(t *testing.T) {
	r := newPicRig(t)
	r.openAside(t, 5)
	pdf := filepath.Join(t.TempDir(), "bill.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.m.attachToTask(5, append(picFiles(t, "ok.png"), pdf))
	r.waitImages(t, 5, 1)
	has(t, r.tt, "That file isn't a PNG, JPEG, GIF or WebP image.")
	r.m.attachToTask(5, picFiles(t, "1.png", "2.png", "3.png", "4.png", "5.png"))
	r.waitImages(t, 5, 4) // the fifth is left, and 1.png is the picture ok.png already was
	has(t, r.tt, "At most 4 images at a time.")
}

func TestThePlusTileAndTheQuietLineOpenTheFilePanel(t *testing.T) {
	r := newPicRig(t)
	var asked []string
	defer func(old func(string) []string) { pickFiles = old }(pickFiles)
	files := picFiles(t, "chosen.png")
	pickFiles = func(title string) []string { asked = append(asked, title); return files }
	r.openAside(t, 5)
	if err := r.tt.Click("Add images…"); err != nil {
		t.Fatal(err)
	}
	r.rec.drain(t) // the panel closed
	r.waitImages(t, 5, 1)
	if !slices.Equal(asked, []string{"Choose images"}) {
		t.Errorf("the panel was asked for %v", asked)
	}
	if err := r.tt.Click("Add images"); err != nil {
		t.Fatal(err)
	}
	r.rec.drain(t)
	if len(asked) != 2 {
		t.Errorf("the + tile did not open the panel")
	}
}

func TestClickingATileOpensTheViewerThatStepsAndCloses(t *testing.T) {
	r := newPicRig(t)
	first := r.give(t, 5, "bill.png", red)
	second := r.give(t, 5, "list.png", blue)
	r.openAside(t, 5)
	if err := r.tt.Click("Open list.png"); err != nil {
		t.Fatal(err)
	}
	r.settle(t)
	if !r.m.viewer.open || r.m.viewer.at != 1 {
		t.Fatalf("viewer %+v", r.m.viewer)
	}
	has(t, r.tt, "list.png · 2 / 2 · 120 × 90", "Open in Preview", "Previous image", "Close")
	hasNot(t, r.tt, "Next image") // the last one has none after it
	r.tt.Key(0, ui.KeyLeft)
	r.settle(t)
	has(t, r.tt, "bill.png · 1 / 2 · 120 × 90", "Next image")
	hasNot(t, r.tt, "Previous image")
	r.tt.Key(0, ui.KeyRight)
	r.settle(t)
	has(t, r.tt, "list.png · 2 / 2 · 120 × 90")
	if err := r.tt.Click("Open in Preview"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rec.files, []string{second.File}) {
		t.Errorf("opened %v, want %v", r.rec.files, second.File)
	}
	r.tt.Key(0, ui.KeyEscape)
	r.tt.Frame()
	if r.m.viewer.open || r.tt.HasText("Open in Preview") {
		t.Error("Esc did not close the viewer")
	}
	if r.m.openID != 5 {
		t.Error("Esc closed the inspector with the viewer")
	}
	_ = first
}

func TestTheViewerClosesWithItsXAndWithAClickOutsideThePicture(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.openAside(t, 5)
	r.m.openViewer(5, 0)
	r.settle(t)
	if err := r.tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.m.viewer.open {
		t.Error("the x did not close it")
	}
	r.m.openViewer(5, 0)
	r.settle(t)
	r.tt.ClickAt(8, 360) // the dim at the window's edge
	r.tt.Frame()
	if r.m.viewer.open {
		t.Error("a click outside did not close it")
	}
	r.m.openViewer(5, 0)
	r.settle(t)
	r.tt.ClickAt(550, 360) // on the picture
	r.tt.Frame()
	if !r.m.viewer.open {
		t.Error("a click on the picture closed it")
	}
}

func TestAViewerOfOnePictureHasNoChevrons(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.m.openViewer(5, 0)
	r.settle(t)
	has(t, r.tt, "bill.png · 1 / 1 · 120 × 90")
	hasNot(t, r.tt, "Previous image", "Next image")
}

func TestTheViewerShowsTheFullPictureOnceItIsRead(t *testing.T) {
	r := newPicRig(t)
	im := r.give(t, 5, "bill.png", red)
	r.m.openViewer(5, 0)
	r.tt.Frame()
	if len(r.m.viewer.full) != 0 {
		t.Fatal("the picture was read on the UI thread")
	}
	r.settle(t)
	if r.m.viewer.full[im.File] == nil {
		t.Error("the full picture never arrived")
	}
}

func TestTheViewerClosesWhenItsTaskLosesItsPictures(t *testing.T) {
	r := newPicRig(t)
	im := r.give(t, 5, "bill.png", red)
	r.m.openViewer(5, 0)
	r.settle(t)
	if err := r.store.DetachImage(5, im.Hash); err != nil {
		t.Fatal(err)
	}
	r.app.Refresh()
	r.tt.Frame()
	if r.m.viewer.open {
		t.Error("the viewer stays on a picture that is gone")
	}
}

func TestARowShowsItsFirstPictureBeforeTheDueLabelWithAPlusNForTheRest(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.give(t, 5, "list.png", blue)
	r.give(t, 7, "reply.png", color.NRGBA{R: 40, G: 200, B: 90, A: 255})
	r.m.nav = nav{} // Mine
	r.settle(t)
	thumb, ok := r.tt.Find("Open bill.png")
	if !ok || thumb.W != rowThumbW || thumb.H != rowThumbH {
		t.Fatalf("thumbnail %v, %v", thumb, ok)
	}
	if !paintedIn(r.tt.Image(), 1100, thumb, ui.Hex("#E62828"), 12) {
		t.Error("the first picture is not drawn in the row")
	}
	due, _ := r.tt.Find("Fri")
	if thumb.X+thumb.W > due.X {
		t.Errorf("the thumbnail at %v is not before the due label at %v", thumb, due)
	}
	has(t, r.tt, "+1")
	if r.tt.HasText("+2") {
		t.Error("a task with one picture has no badge")
	}
	if _, ok := r.tt.Find("Open list.png"); ok {
		t.Error("only the first picture shows in a row")
	}
	hasNot(t, r.tt, "Open dentist.png")
	if got := len(r.m.thumbs.items); got != 2 {
		t.Errorf("%d thumbnails kept, want 2 (rows without pictures read none)", got)
	}
}

func TestARowWithoutPicturesIsAsItWas(t *testing.T) {
	r := newPicRig(t)
	r.settle(t)
	before, _ := r.tt.Find("Book dentist")
	dueBefore, _ := r.tt.Find("Tomorrow")
	r.give(t, 5, "bill.png", red) // another task's
	r.settle(t)
	after, _ := r.tt.Find("Book dentist")
	dueAfter, _ := r.tt.Find("Tomorrow")
	if before.X != after.X || dueBefore.X != dueAfter.X {
		t.Errorf("the row of a task without pictures moved: %v %v -> %v %v", before, dueBefore, after, dueAfter)
	}
	if len(r.m.thumbs.items) != 1 {
		t.Errorf("thumbnails kept: %d", len(r.m.thumbs.items))
	}
}

func TestOneThumbnailIsReadOnceWhateverNumberOfRowsShowIt(t *testing.T) {
	r := newPicRig(t)
	im := r.give(t, 5, "bill.png", red)
	r.m.nav = nav{}
	r.settle(t)
	r.m.openRow(5) // the tile too
	r.settle(t)
	if len(r.m.thumbs.items) != 1 || r.m.thumbs.items[im.Thumb] == nil || r.m.thumbs.items[im.Thumb].bmp == nil {
		t.Errorf("thumbnails: %+v", r.m.thumbs.items)
	}
}

func TestTheThumbnailCacheLetsTheLeastRecentlyDrawnGo(t *testing.T) {
	var tc thumbCache
	tc.items = map[string]*thumbEntry{}
	for i := 0; i < thumbCacheMax; i++ {
		tc.put("p"+itoa(i), nil)
	}
	tc.items["p0"].used = tc.tick + 1000 // drawn just now
	tc.put("new", nil)
	if len(tc.items) != thumbCacheMax+1-thumbCacheDrop {
		t.Errorf("%d kept", len(tc.items))
	}
	if tc.items["p0"] == nil || tc.items["new"] == nil || tc.items["p1"] != nil {
		t.Error("it dropped something drawn lately, or kept the oldest")
	}
}

func TestACardShowsItsFirstPictureAsABandWithAPlusN(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.give(t, 5, "list.png", blue)
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.m.board[1] = true
	r.settle(t)
	band, ok := r.tt.Find("Open bill.png")
	if !ok || band.H != cardBandH || band.W < 120 {
		t.Fatalf("band %v, %v", band, ok)
	}
	if !paintedIn(r.tt.Image(), 1100, band, ui.Hex("#E62828"), 12) {
		t.Error("the picture is not drawn in the band")
	}
	title, _ := r.tt.Find("Write the release notes")
	if title.Y < band.Y+band.H {
		t.Errorf("the title at %v is not under the band at %v", title, band)
	}
	has(t, r.tt, "+1")
	if err := r.tt.Click("Open bill.png"); err != nil {
		t.Fatal(err)
	}
	r.settle(t)
	if !r.m.viewer.open || r.m.openID == 5 {
		t.Errorf("a click on the band opens the viewer (open %v, inspector %d)", r.m.viewer.open, r.m.openID)
	}
}

func TestProjectColumnsAndListCarryTheThumbnailToo(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.m.goTo(nav{screen: screenProject, projectID: 1})
	r.settle(t)
	if _, ok := r.tt.Find("Open bill.png"); !ok {
		t.Errorf("no thumbnail on the project page: %q", r.tt.Texts())
	}
}

func TestTheInboxRowsCarryTheThumbnail(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 8, "tooth.png", red)
	r.m.goTo(nav{screen: screenInbox})
	r.settle(t)
	if _, ok := r.tt.Find("Open tooth.png"); !ok {
		t.Errorf("no thumbnail in the Inbox: %q", r.tt.Texts())
	}
}

func TestSearchResultsAndCompletedRowsCarryTheThumbnail(t *testing.T) {
	r := newPicRig(t)
	r.give(t, 5, "bill.png", red)
	r.give(t, 13, "spec.png", blue) // done
	r.settle(t)
	searchFor(t, r.tt, "release")
	r.settle(t)
	if _, ok := r.tt.Find("Open bill.png"); !ok {
		t.Errorf("no thumbnail in the search results: %q", r.tt.Texts())
	}
	r.m.clearSearch()
	r.m.goTo(nav{screen: screenCompleted})
	r.settle(t)
	if _, ok := r.tt.Find("Open spec.png"); !ok {
		t.Errorf("no thumbnail in Completed: %q", r.tt.Texts())
	}
}

func TestAReceiptSaysTheTaskKeptItsImages(t *testing.T) {
	reply := data.Reply{Text: "ok", Receipts: []data.Receipt{
		{ID: 31, TaskID: 5, Title: "Pay rent", Status: data.Todo, Detail: data.Detail{Kind: data.DetailAdded}, Images: 2},
		{ID: 32, TaskID: 7, Title: "Buy milk", Status: data.Todo, Detail: data.Detail{Kind: data.DetailAdded}, Images: 1},
		{ID: 33, TaskID: 8, Title: "Call mum", Status: data.Todo, Detail: data.Detail{Kind: data.DetailAdded}},
	}}
	a := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reply, nil)})
	a.ask("these")
	a.answer(t)
	has(t, a.tt, "· with its images", "· with its image")
	if n := countEnding(a.tt, "with its image"); n != 1 {
		t.Errorf("%d lines say it kept one image", n)
	}
}

func countEnding(tt *ui.Tester, suffix string) int {
	n := 0
	for _, s := range tt.Texts() {
		if strings.HasSuffix(s, suffix) {
			n++
		}
	}
	return n
}

func TestTheReceiptsInChineseSaySoToo(t *testing.T) {
	reply := data.Reply{Receipts: []data.Receipt{
		{ID: 31, TaskID: 5, Title: "交房租", Status: data.Todo, Detail: data.Detail{Kind: data.DetailAdded}, Images: 2},
	}}
	a := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(reply, nil)})
	a.tt.SetLocale("zh-Hans")
	a.ask("这些")
	a.answer(t)
	has(t, a.tt, "图片也挂上了")
}
