package views

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// fixedNow is the clock of every test and screenshot, so relative times read the same each run.
func fixedNow() time.Time { return time.Date(2026, 10, 14, 14, 30, 0, 0, time.Local) }

// openCall is one OpenIn the views asked for.
type openCall struct {
	how      OpenWith
	app, dir string
}

// savedFile is one SaveText the views asked for.
type savedFile struct{ name, text string }

// recorder is an Actions that remembers what the views asked for.
type recorder struct {
	opened   []int
	urls     []string
	panes    []Pane      // what OpenSettings was asked to show
	folders  []string    // what ShowInFinder was asked to reveal
	files    []string    // what OpenFile was asked to open
	language string      // what SetLanguage last chose
	opens    []openCall  // what OpenIn was asked for
	saved    []savedFile // what SaveText was asked to save
	openErr  error       // what OpenIn answers
	denied   bool        // what NotificationsDenied answers

	mu     sync.Mutex
	posted []func()
	wake   chan struct{}
}

func (r *recorder) OpenTask(id int)          { r.opened = append(r.opened, id) }
func (r *recorder) OpenSettings(p Pane)      { r.panes = append(r.panes, p) }
func (r *recorder) OpenURL(url string)       { r.urls = append(r.urls, url) }
func (r *recorder) ShowInFinder(path string) { r.folders = append(r.folders, path) }
func (r *recorder) OpenFile(path string)     { r.files = append(r.files, path) }

func (r *recorder) SaveText(name, text string) { r.saved = append(r.saved, savedFile{name, text}) }

func (r *recorder) OpenIn(how OpenWith, agentApp, dir string) error {
	r.opens = append(r.opens, openCall{how, agentApp, dir})
	return r.openErr
}

func (r *recorder) Languages() []string    { return []string{"en", "zh-Hans"} }
func (r *recorder) Language() string       { return r.language }
func (r *recorder) SetLanguage(tag string) { r.language = tag }

func (r *recorder) NotificationsDenied() bool { return r.denied }

// Post queues fn, as the real shell hands it to the UI thread; drain runs the queue.
func (r *recorder) Post(fn func()) {
	r.mu.Lock()
	r.posted = append(r.posted, fn)
	if r.wake != nil {
		select {
		case r.wake <- struct{}{}:
		default:
		}
	}
	r.mu.Unlock()
}

// drain waits up to a second for something to be posted, then runs everything posted.
func (r *recorder) drain(t *testing.T) {
	t.Helper()
	r.mu.Lock()
	if r.wake == nil {
		r.wake = make(chan struct{}, 1)
	}
	r.mu.Unlock()
	deadline := time.After(time.Second)
	for {
		r.mu.Lock()
		fns := r.posted
		r.posted = nil
		r.mu.Unlock()
		if len(fns) > 0 {
			for _, fn := range fns {
				fn()
			}
			return
		}
		select {
		case <-r.wake:
		case <-deadline:
			t.Fatal("nothing was posted to the UI thread")
		}
	}
}

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
	cfg := testConfig()
	cfg.AgentSetup = newAgentRig(t).env
	app := NewApp(store, rec, cfg)
	app.Now = fixedNow
	return app, store, rec
}

func newEmptyApp(t *testing.T) (*App, *recorder) {
	t.Helper()
	rec := &recorder{}
	cfg := testConfig()
	cfg.AgentSetup = newAgentRig(t).env
	app := NewApp(data.NewEmptyFake(fixedNow), rec, cfg)
	app.Now = fixedNow
	return app, rec
}

// still turns motion off so frames are deterministic.
func still(tt *ui.Tester) { tt.SetPreferences(ui.Preferences{ReduceMotion: true}) }

// screensTest skips a design-screenshot test under -short: they render every screen at 2x and
// take minutes, whether or not SCREENS_OUT asks for the PNGs.
func screensTest(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("design screens; run without -short")
	}
}

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
	out := img
	if float {
		out = onBackdrop(img, dark)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// 256 colors and the best compression: a third of the bytes of the RGBA default, same look
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(f, quantizeScreen(out)); err != nil {
		t.Fatal(err)
	}
}

// onBackdrop sets a floating surface on a soft wallpaper-like gradient with room around it, the
// way it appears on the desktop.
func onBackdrop(img *image.RGBA, dark bool) *image.RGBA {
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
			d := math.Hypot(float64(dx), float64(dy))
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

// settle waits out the 160 ms color transitions that a change of appearance starts, then draws
// the frame that follows.
func settle(tt *ui.Tester) {
	time.Sleep(300 * time.Millisecond)
	tt.Frame()
}

func newMainTester(t *testing.T, width int) (*ui.Tester, *Main, *App) {
	t.Helper()
	app, _, _ := newTestApp(t)
	return newTesterFor(app, width)
}

func newTesterFor(app *App, width int) (*ui.Tester, *Main, *App) {
	m := NewMain(app)
	tt := ui.NewTester(m.View, width, 720)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	return tt, m, app
}

func center(r ui.Rect) (float32, float32) { return r.X + r.W/2, r.Y + r.H/2 }

// above reports whether the text a is drawn higher on the screen than b.
func above(t *testing.T, tt *ui.Tester, a, b string) bool {
	t.Helper()
	ra, ok := tt.Find(a)
	if !ok {
		t.Fatalf("no %q in %q", a, tt.Texts())
	}
	rb, ok := tt.Find(b)
	if !ok {
		t.Fatalf("no %q in %q", b, tt.Texts())
	}
	return ra.Y < rb.Y
}

// dragTo drags the card showing text onto the column named column.
func dragTo(t *testing.T, tt *ui.Tester, text, column string) {
	t.Helper()
	from, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no card %q in %q", text, tt.Texts())
	}
	to, ok := tt.Find(column)
	if !ok {
		t.Fatalf("no column %q in %q", column, tt.Texts())
	}
	fx, fy := center(from)
	tx, ty := center(to)
	tt.Press(fx, fy)
	tt.Move(fx+8, fy+8)
	tt.Move(tx, ty+60)
	tt.Release(tx, ty+60)
}

// emptyProjectApp is an empty store with one project and nothing in it.
func emptyProjectApp(t *testing.T) (*App, data.Project) {
	t.Helper()
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("scratch", "me/scratch")
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	return app, p
}
