package views

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The pictures of tasks: the inspector with tiles, with none and with a hovered tile; Mine, a project
// list and a board with their thumbnails and bands; the viewer. Light and dark, English and
// Chinese (-zh). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensTaskImages

// sheetFile writes a picture of a page: a card on a tinted ground, grey lines and a block of an
// accent color, 1200 x 900, so a thumbnail looks like a screenshot of something.
func sheetFile(t *testing.T, name string, ground, accent color.NRGBA, lines []int) string {
	t.Helper()
	const w, h = 1200, 900
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fill := func(r image.Rectangle, c color.NRGBA) {
		draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	fill(img.Bounds(), ground)
	fill(image.Rect(80, 70, w-80, h-70), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	fill(image.Rect(130, 120, 520, 170), color.NRGBA{R: 60, G: 60, B: 66, A: 255})
	for i, width := range lines {
		fill(image.Rect(130, 240+i*90, 130+width, 270+i*90), color.NRGBA{R: 229, G: 229, B: 234, A: 255})
	}
	fill(image.Rect(130, 640, 430, 720), accent)
	path := filepath.Join(t.TempDir(), name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

// imagesApp is the seeded fake with pictures on three tasks: the release notes (three), the design
// feedback (one), the dentist (two, in the Inbox).
func imagesApp(t *testing.T, zh bool) (*App, *recorder) {
	t.Helper()
	app, store, rec := newTestApp(t)
	store.SetImageDir(t.TempDir())
	names := map[bool][]string{
		false: {"rent-notice.png", "weekend-list.png", "whiteboard.png", "design-feedback.png", "dentist.png", "invoice.png"},
		true:  {"房租通知.png", "周末清单.png", "白板.png", "设计反馈.png", "牙医.png", "发票.png"},
	}[zh]
	pink, mint, sand, sky := color.NRGBA{R: 255, G: 107, B: 94, A: 255}, color.NRGBA{R: 167, G: 227, B: 196, A: 255},
		color.NRGBA{R: 255, G: 224, B: 138, A: 255}, color.NRGBA{R: 185, G: 211, B: 255, A: 255}
	give := func(task int, name string, ground, accent color.NRGBA, lines []int) {
		att, err := data.ReadImageFile(sheetFile(t, name, ground, accent, lines))
		if err != nil {
			t.Fatal(err)
		}
		att.Name = name
		if _, err := store.AttachImages(task, []data.Attachment{att}); err != nil {
			t.Fatal(err)
		}
	}
	give(5, names[0], color.NRGBA{R: 251, G: 250, B: 246, A: 255}, pink, []int{700, 520, 640, 380})
	give(5, names[1], color.NRGBA{R: 242, G: 242, B: 247, A: 255}, mint, []int{200, 150, 240, 120})
	give(5, names[2], color.NRGBA{R: 253, G: 252, B: 247, A: 255}, sand, []int{300, 300, 300, 300})
	give(7, names[3], color.NRGBA{R: 238, G: 240, B: 255, A: 255}, sky, []int{500, 420, 360, 240})
	give(8, names[4], color.NRGBA{R: 240, G: 250, B: 244, A: 255}, mint, []int{420, 300, 380, 200})
	give(8, names[5], color.NRGBA{R: 255, G: 246, B: 238, A: 255}, pink, []int{640, 480, 560, 300})
	app.Refresh()
	return app, rec
}

// loadPictures draws until every thumbnail and picture being read has arrived.
func loadPictures(t *testing.T, rec *recorder, tt *ui.Tester, m *Main) {
	t.Helper()
	tt.Frame()
	for len(m.thumbs.loading) > 0 || len(m.viewer.loading) > 0 {
		rec.drain(t)
		tt.Frame()
	}
}

func TestScreensTaskImages(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix + ".png"
			if zh {
				sfx = "-zh" + sfx
			}
			shotOf := func(name string, setup func(tt *ui.Tester, m *Main, rec *recorder)) {
				app, rec := imagesApp(t, zh)
				run := func(tt *ui.Tester, m *Main) {
					setup(tt, m, rec)
					loadPictures(t, rec, tt, m)
				}
				if zh {
					zhShot(t, name+sfx, app, a.dark, run)
					return
				}
				mainShot(t, name+sfx, app, a.dark, run)
			}
			second := "Open weekend-list.png"
			if zh {
				second = "打开 周末清单.png"
			}
			shotOf("inspector-images", func(tt *ui.Tester, m *Main, rec *recorder) { m.OpenTask(5) })
			shotOf("inspector-images-none", func(tt *ui.Tester, m *Main, rec *recorder) { m.OpenTask(6) })
			shotOf("inspector-images-hover", func(tt *ui.Tester, m *Main, rec *recorder) {
				m.OpenTask(5)
				loadPictures(t, rec, tt, m)
				tile, ok := tt.Find(second) // the Mine row behind shows only the first
				if !ok {
					t.Fatal("no second tile")
				}
				tt.Move(tile.X+20, tile.Y+20)
			})
			shotOf("mine-pictures", func(*ui.Tester, *Main, *recorder) {})
			shotOf("project-list-pictures", func(tt *ui.Tester, m *Main, _ *recorder) { toProject(1)(tt, m) })
			shotOf("project-board-pictures", func(tt *ui.Tester, m *Main, _ *recorder) { toProject(1)(tt, m); m.board[1] = true })
			shotOf("viewer", func(tt *ui.Tester, m *Main, _ *recorder) { m.OpenTask(5); m.openViewer(5, 1) })
		}
	}
}
