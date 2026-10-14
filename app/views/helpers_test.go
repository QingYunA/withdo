package views

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// fixedNow is the clock of every test and screenshot, so relative times read the same each run.
func fixedNow() time.Time { return time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local) }

// recorder is an Actions that remembers what the views asked for.
type recorder struct {
	opened      []int
	urls        []string
	panelClosed int
	main        int
	settings    int
}

func (r *recorder) OpenTask(id int)    { r.opened = append(r.opened, id) }
func (r *recorder) OpenMain()          { r.main++ }
func (r *recorder) OpenSettings()      { r.settings++ }
func (r *recorder) ClosePanel()        { r.panelClosed++ }
func (r *recorder) OpenURL(url string) { r.urls = append(r.urls, url) }

func testConfig() Config {
	return Config{
		ConnectCommand: "claude mcp add --scope user adhd-todo -- ~/.local/bin/adhd-core mcp",
		DataFolder:     "~/Library/Application Support/adhd-todo",
		Version:        "0.2.0",
	}
}

// newTestApp is an App over the seeded fake with the fixed clock.
func newTestApp(t *testing.T) (*App, *data.Fake, *recorder) {
	t.Helper()
	store := data.NewFake(fixedNow)
	rec := &recorder{}
	app := NewApp(store, rec, testConfig())
	app.Now = fixedNow
	return app, store, rec
}

func newEmptyApp(t *testing.T) (*App, *recorder) {
	t.Helper()
	rec := &recorder{}
	app := NewApp(data.NewEmptyFake(fixedNow), rec, testConfig())
	app.Now = fixedNow
	return app, rec
}

// still turns motion off so frames are deterministic.
func still(tt *ui.Tester) { tt.SetPreferences(ui.Preferences{ReduceMotion: true}) }

// shot renders a surface at 2x into dir, over a backdrop for floating surfaces.
func shot(t *testing.T, name string, img *image.RGBA, float bool, dark bool) {
	t.Helper()
	dir := os.Getenv("SCREENS_OUT")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	out := image.Image(img)
	if float {
		out = onBackdrop(img, dark)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		t.Fatal(err)
	}
}

// onBackdrop sets a floating surface on a soft wallpaper-like gradient with room around it, the
// way it appears on the desktop.
func onBackdrop(img *image.RGBA, dark bool) image.Image {
	const pad = 96
	b := img.Bounds()
	w, h := b.Dx()+2*pad, b.Dy()+2*pad
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	top, bottom := color.RGBA{214, 218, 232, 255}, color.RGBA{232, 220, 226, 255}
	if dark {
		top, bottom = color.RGBA{38, 40, 58, 255}, color.RGBA{58, 44, 52, 255}
	}
	for y := 0; y < h; y++ {
		k := float64(y) / float64(h)
		row := color.RGBA{
			uint8(float64(top.R)*(1-k) + float64(bottom.R)*k),
			uint8(float64(top.G)*(1-k) + float64(bottom.G)*k),
			uint8(float64(top.B)*(1-k) + float64(bottom.B)*k), 255}
		for x := 0; x < w; x++ {
			out.SetRGBA(x, y, row)
		}
	}
	shadow(out, image.Rect(pad, pad+16, pad+b.Dx(), pad+b.Dy()+16), 48, dark)
	draw.Draw(out, image.Rect(pad, pad, pad+b.Dx(), pad+b.Dy()), img, b.Min, draw.Over)
	return out
}

// shadow darkens the area around r, softer with distance.
func shadow(dst *image.RGBA, r image.Rectangle, spread int, dark bool) {
	strength := 0.22
	if dark {
		strength = 0.5
	}
	for y := r.Min.Y - spread; y < r.Max.Y+spread; y++ {
		for x := r.Min.X - spread; x < r.Max.X+spread; x++ {
			dx := max(r.Min.X-x, 0, x-r.Max.X+1)
			dy := max(r.Min.Y-y, 0, y-r.Max.Y+1)
			d := float64(max(dx, dy))
			if d >= float64(spread) || !(image.Point{x, y}).In(dst.Bounds()) {
				continue
			}
			a := strength * (1 - d/float64(spread)) * (1 - d/float64(spread))
			c := dst.RGBAAt(x, y)
			c.R, c.G, c.B = uint8(float64(c.R)*(1-a)), uint8(float64(c.G)*(1-a)), uint8(float64(c.B)*(1-a))
			dst.SetRGBA(x, y, c)
		}
	}
}
