//go:build promo

package views

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// The window frames of the promo video (docs/design/promo.md): the app's own views, driven by a
// clock the test advances by 1/60 s, with the pointer drawn on top. Render with
//
//	PROMO_LANG=zh PROMO_OUT=$PWD/out/promo/work-zh go test -tags promo ./app/views -run TestPromoWindow -count=1
//
// docs/design/tools/promo/render.sh does that and assembles the video. The test writes
// w00000.png... (2200 x 1440, rounded corners in the alpha) and track.json: when the scenes began
// and which sound cues play, all in video seconds.

const (
	promoFPS = 60
	// The window segment runs from promoWinStart to promoWinEnd, in video seconds; a cut into and out
	// of it is a 200 ms crossfade, centred on 2.5 s and 23 s.
	promoWinStart = 2.4
	promoWinEnd   = 23.1
	// The two clicks: on Summary in the sidebar, on This week in the header.
	promoClickSummary = 3.55
	promoClickWeek    = 16.4
	promoClickDip     = 0.14 // seconds the pointer stays dipped
	promoCorner       = 36   // px, the window's corner radius
	promoScale        = 2
)

// promoTrack is what the frames tell the rest of the render.
type promoTrack struct {
	Frames    int         `json:"frames"`
	DayStart  float64     `json:"dayStart"`  // video second the day scene began
	DayEnd    float64     `json:"dayEnd"`    // and the second it had played
	WeekStart float64     `json:"weekStart"` // the same for the week
	WeekEnd   float64     `json:"weekEnd"`
	Cues      []promoCue  `json:"cues"`
	Clicks    [2]float64  `json:"clicks"`
	Stills    []promoStil `json:"stills"`
}

type promoCue struct {
	At    float64 `json:"at"` // video second
	Sound string  `json:"sound"`
	Speed float64 `json:"speed"`
}

type promoStil struct {
	Name string  `json:"name"`
	At   float64 `json:"at"`
}

// promoLang is the language and the folder of the render, from PROMO_LANG and PROMO_OUT.
func promoLang(t *testing.T) (zh bool, out string) {
	t.Helper()
	out = os.Getenv("PROMO_OUT")
	if out == "" {
		t.Skip("PROMO_OUT is not set; docs/design/tools/promo/render.sh runs this")
	}
	switch os.Getenv("PROMO_LANG") {
	case "zh":
		zh = true
	case "en":
	default:
		t.Fatal("PROMO_LANG must be zh or en")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	return zh, out
}

// promoRegion is the region of the window: Monday first, so the week has Saturday and Sunday empty.
func promoRegion(zh bool) i18n.Region {
	if zh {
		return i18n.Region{Locale: "zh-CN", FirstWeekday: time.Monday, HourCycle: 12}
	}
	return i18n.Region{Locale: "en-US", FirstWeekday: time.Monday, HourCycle: 12}
}

// setEngineClock sets the clock mygo's frames read (its animations and transitions), which the
// Tester keeps to itself: the video's time is the test's, not the machine's. A test-only reach into
// ui.Tester; there is no public way to do it in mygo v0.5.1.
func setEngineClock(tt *ui.Tester, clock func() time.Time) {
	rt := reflect.ValueOf(tt).Elem().FieldByName("rt").Elem()
	f := rt.FieldByName("clock")
	reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().Set(reflect.ValueOf(clock))
}

// --- the pointer ---------------------------------------------------------------------------------

// pointerSprite is the mouse pointer, 12 x 12 cells: X the dark outline, W the white body. Its tip
// is the top left cell.
var pointerSprite = [12]string{
	"X...........",
	"XX..........",
	"XWX.........",
	"XWWX........",
	"XWWWX.......",
	"XWWWWX......",
	"XWWWWWX.....",
	"XWWWWWWX....",
	"XWWWWXXXX...",
	"XWWXWWX.....",
	"XWX.XWWX....",
	"XX...XXX....",
}

// pointerCell is a cell of the pointer in pixels: 3 pt at the window's 2x.
const pointerCell = 3 * promoScale

// drawPointer paints the pointer with its tip at (x, y) pixels.
func drawPointer(img *image.NRGBA, x, y int) {
	outline, white := color.NRGBA{22, 22, 28, 255}, color.NRGBA{255, 255, 255, 255}
	for cy, row := range pointerSprite {
		for cx, ch := range row {
			var c color.NRGBA
			switch ch {
			case 'X':
				c = outline
			case 'W':
				c = white
			default:
				continue
			}
			for py := 0; py < pointerCell; py++ {
				for px := 0; px < pointerCell; px++ {
					img.SetNRGBA(x+cx*pointerCell+px, y+cy*pointerCell+py, c)
				}
			}
		}
	}
}

// waypoint is where the pointer is at a video second.
type waypoint struct {
	T    float64
	X, Y float32 // points
}

// pointerPath is the pointer's way: it rests at each waypoint and moves to the next on an eased,
// slightly bowed path that ends at the next waypoint's second.
type pointerPath struct {
	stops []waypoint // the moves: from stops[i] (left at .T) to stops[i+1] (reached at .T)
	rest  []float64  // rest[i] is the second the pointer leaves stops[i]
}

func (p pointerPath) at(t float64) (x, y float32) {
	s := p.stops
	if t <= s[0].T {
		return s[0].X, s[0].Y
	}
	for i := 0; i+1 < len(s); i++ {
		leave, arrive := p.rest[i], s[i+1].T
		if t < leave {
			return s[i].X, s[i].Y
		}
		if t < arrive {
			u := float64(t-leave) / float64(arrive-leave)
			k := u * u * u * (u*(u*6-15) + 10) // smootherstep
			dx, dy := float64(s[i+1].X-s[i].X), float64(s[i+1].Y-s[i].Y)
			bow := math.Sin(math.Pi*k) * 0.08 // a slight arc, as a hand moves
			return s[i].X + float32(dx*k-dy*bow), s[i].Y + float32(dy*k+dx*bow)
		}
	}
	last := s[len(s)-1]
	return last.X, last.Y
}

// --- the window -----------------------------------------------------------------------------------

// framePNG turns the window's pixels into the frame to write: opaque, with the corners rounded in
// the alpha and the three buttons of a macOS window drawn in the title bar.
func framePNG(src *image.RGBA) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			i := y*src.Stride + x*4
			out.Pix[y*out.Stride+x*4+0] = src.Pix[i+0]
			out.Pix[y*out.Stride+x*4+1] = src.Pix[i+1]
			out.Pix[y*out.Stride+x*4+2] = src.Pix[i+2]
			out.Pix[y*out.Stride+x*4+3] = 255
		}
	}
	for i, c := range []color.NRGBA{{255, 95, 87, 255}, {254, 188, 46, 255}, {40, 200, 64, 255}} {
		disc(out, float64(promoScale*(19+20*i)), float64(promoScale*26), float64(promoScale*6), c)
	}
	roundCorners(out, promoCorner)
	return out
}

// disc paints an anti-aliased disc.
func disc(img *image.NRGBA, cx, cy, r float64, c color.NRGBA) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			cover := 0.0
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					px, py := float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4
					if math.Hypot(px-cx, py-cy) <= r {
						cover += 1.0 / 16
					}
				}
			}
			if cover == 0 {
				continue
			}
			o := img.NRGBAAt(x, y)
			mix := func(a, b uint8) uint8 { return uint8(float64(a)*(1-cover) + float64(b)*cover + 0.5) }
			img.SetNRGBA(x, y, color.NRGBA{mix(o.R, c.R), mix(o.G, c.G), mix(o.B, c.B), 255})
		}
	}
}

// roundCorners clears the corners of img to a radius r, anti-aliased, in the alpha channel.
func roundCorners(img *image.NRGBA, r int) {
	b := img.Bounds()
	for _, corner := range [4][2]int{{0, 0}, {b.Dx() - r, 0}, {0, b.Dy() - r}, {b.Dx() - r, b.Dy() - r}} {
		cx, cy := float64(corner[0]), float64(corner[1])
		if corner[0] == 0 {
			cx = float64(r)
		}
		if corner[1] == 0 {
			cy = float64(r)
		}
		for y := corner[1]; y < corner[1]+r; y++ {
			for x := corner[0]; x < corner[0]+r; x++ {
				cover := 0.0
				for sy := 0; sy < 4; sy++ {
					for sx := 0; sx < 4; sx++ {
						px, py := float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4
						if math.Hypot(px-cx, py-cy) <= float64(r) {
							cover += 1.0 / 16
						}
					}
				}
				// Only the outer quarter of each r x r square is outside the arc.
				inCornerSquare := (x < r && corner[0] == 0 || x >= b.Dx()-r && corner[0] != 0) &&
					(y < r && corner[1] == 0 || y >= b.Dy()-r && corner[1] != 0)
				if inCornerSquare {
					i := (y-b.Min.Y)*img.Stride + (x-b.Min.X)*4
					img.Pix[i+3] = uint8(float64(img.Pix[i+3])*cover + 0.5)
				}
			}
		}
	}
}

// pngWriter encodes frames on a few goroutines, so the renderer never waits for a file.
type pngWriter struct {
	jobs chan pngJob
	wg   sync.WaitGroup
	enc  png.Encoder
	mu   sync.Mutex
	err  error
}

type pngJob struct {
	path string
	img  image.Image
}

func newPNGWriter(workers int) *pngWriter {
	w := &pngWriter{jobs: make(chan pngJob, workers*2), enc: png.Encoder{CompressionLevel: png.BestSpeed}}
	for i := 0; i < workers; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for j := range w.jobs {
				if err := w.write(j); err != nil {
					w.mu.Lock()
					w.err = err
					w.mu.Unlock()
				}
			}
		}()
	}
	return w
}

func (w *pngWriter) write(j pngJob) error {
	f, err := os.Create(j.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return w.enc.Encode(f, j.img)
}

func (w *pngWriter) put(path string, img image.Image) { w.jobs <- pngJob{path, img} }

func (w *pngWriter) close(t *testing.T) {
	t.Helper()
	close(w.jobs)
	w.wg.Wait()
	if w.err != nil {
		t.Fatal(w.err)
	}
}

// --- the render -----------------------------------------------------------------------------------

func TestPromoWindow(t *testing.T) {
	zh, out := promoLang(t)
	var vt time.Duration // the app's time since the window segment began
	clock := func() time.Time { return promoBase.Add(vt) }
	app, err := promoApp(t, clock, zh)
	if err != nil {
		t.Fatal(err)
	}
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	tt.SetScale(promoScale)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	tt.SetPreferences(ui.Preferences{})
	if zh {
		tt.SetLocale("zh-Hans")
	} else {
		tt.SetLocale("en")
	}
	tt.SetRegion(promoRegion(zh))
	setEngineClock(tt, clock)
	// Some warm-up frames on Mine, a moment before the first one is taken.
	for i := -30; i < 0; i++ {
		vt = time.Duration(i) * time.Second / promoFPS
		tt.Frame()
	}

	summary, ok := tt.Find(m.say.NavSummary())
	if !ok {
		t.Fatalf("no %q in %q", m.say.NavSummary(), tt.Texts())
	}
	summaryX, summaryY := summary.X+summary.W*0.55, summary.Y+summary.H/2
	var weekX, weekY float32 // known once Summary is open

	// The pointer rests on Mine, goes to Summary, rests there through the day, goes to This week.
	// The week's stop is moved to its real place once Summary has been laid out; until then it is a
	// stand-in (the path is only asked for it after that).
	path := pointerPath{
		stops: []waypoint{{promoWinStart, 760, 440}, {promoClickSummary - 0.08, summaryX, summaryY}, {promoClickWeek - 0.08, 0, 0}},
		rest:  []float64{promoWinStart + 0.1, promoClickWeek - 1.0, promoClickWeek},
	}

	track := promoTrack{Clicks: [2]float64{promoClickSummary, promoClickWeek}}
	frames := int(math.Round((promoWinEnd - promoWinStart) * promoFPS))
	track.Frames = frames
	wr := newPNGWriter(6)
	clickedSummary, clickedWeek := false, false
	began := time.Now()
	for k := 0; k < frames; k++ {
		T := promoWinStart + float64(k)/promoFPS
		vt = time.Duration(float64(time.Second) * (T - promoWinStart))
		px, py := path.at(T)
		tt.Move(px, py)

		switch {
		case !clickedSummary && T >= promoClickSummary:
			clickedSummary = true
			tt.ClickAt(px, py)
			v := m.summaryFor(app.Now())
			// The paragraph types in once the scene has played, as the page does for a new one.
			m.paragraphOf(v.period).typedAt = m.sum.start.Add(v.sceneEnd())
			track.DayStart = promoWinStart + m.sum.start.Sub(promoBase).Seconds()
			track.DayEnd = track.DayStart + v.sceneEnd().Seconds()
			track.addCues(v.day.SoundCues(false), track.DayStart)
			tt.Frame()
			week, ok := tt.Find(m.say.SummarySegmentWeek())
			if !ok {
				t.Fatalf("no %q in %q", m.say.SummarySegmentWeek(), tt.Texts())
			}
			weekX, weekY = week.X+week.W*0.5, week.Y+week.H/2
			path.stops[2].X, path.stops[2].Y = weekX, weekY
		case !clickedWeek && T >= promoClickWeek:
			clickedWeek = true
			tt.ClickAt(px, py)
			v := m.summaryFor(app.Now())
			track.WeekStart = promoWinStart + m.sum.start.Sub(promoBase).Seconds()
			track.WeekEnd = track.WeekStart + v.sceneEnd().Seconds()
			track.addCues(v.week.SoundCues(false), track.WeekStart)
			tt.Frame()
		default:
			tt.Frame()
		}

		img := framePNG(tt.Image())
		dip := float64(0)
		for _, c := range []float64{promoClickSummary, promoClickWeek} {
			if T >= c && T < c+promoClickDip {
				dip = pointerCell // the pointer dips a cell as it clicks
			}
		}
		drawPointer(img, int(math.Round(float64(px*promoScale))), int(math.Round(float64(py*promoScale)+dip)))
		wr.put(filepath.Join(out, fmt.Sprintf("w%05d.png", k)), img)
		if k%120 == 0 {
			t.Logf("frame %d/%d, T=%.2fs, %.0f ms a frame", k, frames, T, float64(time.Since(began).Milliseconds())/float64(k+1))
		}
	}
	wr.close(t)
	if !clickedSummary || !clickedWeek {
		t.Fatalf("clicks: summary %v, week %v", clickedSummary, clickedWeek)
	}
	data, err := json.MarshalIndent(track, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "track.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "audio.wav"), promoMix(t, track), 0o644); err != nil {
		t.Fatal(err)
	}
}

// addCues adds the cues of a scene that began at start (video seconds).
func (tr *promoTrack) addCues(cues []theme.Cue, start float64) {
	for _, c := range cues {
		tr.Cues = append(tr.Cues, promoCue{At: start + c.At.Seconds(), Sound: string(c.Sound), Speed: c.Speed()})
	}
}
