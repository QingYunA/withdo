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

// The window segment of the promo video (docs/design/promo.md): the app's own views, driven by a
// clock the test advances by 1/60 s, with the pointer and a camera. Render with
//
//	PROMO_LANG=zh PROMO_OUT=$PWD/out/promo/work-zh go test -tags promo ./app/views -run TestPromoWindow -count=1
//
// docs/design/tools/promo/render.sh does that and assembles the video. The test writes finished
// 3840 x 2160 frames w00000.png... (the window on its dimmed backdrop, the camera applied), audio.wav,
// track.json (when things happen, the camera's path, the sound cues), timing.env and stills.txt (for
// the script).
//
// The window is rendered by the real renderer at the scale the camera asks for, never resampled: at
// rest 8/3 (a 3 pt cell of the scene is 8 px), zoomed in 4 (12 px), so text and pixels stay sharp.

const (
	promoFPS = 60
	// The window segment starts at promoWinStart (video seconds); a cut into and out of it is a 200 ms
	// crossfade centred 100 ms after its start and 100 ms before its end.
	promoWinStart = 2.4
	// The two clicks: on Summary in the sidebar, on This week in the header.
	promoClickSummary = 3.2
	promoClickWeek    = 15.5
	promoClickDip     = 0.14 // seconds the pointer stays dipped
	promoHold         = 0.5  // seconds the finished week ledger stays before the cut starts

	promoBaseScale = 8.0 / 3.0 // pixels per point at rest: a 3 pt cell is 8 px
	promoWinDIP    = 1176      // points wide: 3136 px at rest, 82 % of the frame's width
	promoWinHDIP   = 720       // points tall: 1920 px, 89 % of its height
	promoWinPxW    = 3136
	promoWinPxH    = 1920
	promoCornerPt  = 18.0 // points, the window's corner radius
	promoZoom      = 1.5  // a 3 pt cell is then 12 px, and the scene band fills the frame
	promoCamEase   = 0.4  // seconds a camera move takes
	promoCamLead   = 0.25 // seconds after a scene begins that the camera starts moving in
	promoCamAbove  = 145  // px of the page kept above the scene band when zoomed in
)

// camKey is the camera at a video second: the zoom, and the point of the window, in its pixels at
// rest, that is in the middle of the frame. The camera eases (smoothstep) from one key to the next.
type camKey struct {
	T    float64 `json:"t"`
	Zoom float64 `json:"zoom"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// promoTrack is what the frames tell the rest of the render.
type promoTrack struct {
	Frames    int        `json:"frames"`
	WinEnd    float64    `json:"winEnd"` // video second the window segment ends
	Total     float64    `json:"total"`  // video second the video ends
	DayStart  float64    `json:"dayStart"`
	DayEnd    float64    `json:"dayEnd"`
	DayDone   float64    `json:"dayDone"` // the ledger has lit and counted
	WeekStart float64    `json:"weekStart"`
	WeekEnd   float64    `json:"weekEnd"`
	WeekDone  float64    `json:"weekDone"`
	Clicks    [2]float64 `json:"clicks"`
	Camera    []camKey   `json:"camera"`
	Cues      []promoCue `json:"cues"`
}

type promoCue struct {
	At    float64 `json:"at"` // video second
	Sound string  `json:"sound"`
	Speed float64 `json:"speed"`
}

func (tr *promoTrack) addCues(cues []theme.Cue, start float64) {
	for _, c := range cues {
		tr.Cues = append(tr.Cues, promoCue{At: start + c.At.Seconds(), Sound: string(c.Sound), Speed: c.Speed()})
	}
}

func ease(u float64) float64 { u = math.Max(0, math.Min(1, u)); return u * u * (3 - 2*u) }

// camAt is the camera at video second t. The keys are in order of time; the camera holds the last.
func camAt(keys []camKey, t float64) camKey {
	cur := keys[0]
	for i := 0; i+1 < len(keys); i++ {
		a, b := keys[i], keys[i+1]
		if t < a.T {
			break
		}
		cur = b
		if t < b.T {
			k := ease((t - a.T) / (b.T - a.T))
			return camKey{T: t, Zoom: math.Exp(math.Log(a.Zoom) + (math.Log(b.Zoom)-math.Log(a.Zoom))*k),
				X: a.X + (b.X-a.X)*k, Y: a.Y + (b.Y-a.Y)*k}
		}
	}
	return cur
}

// sceneView is where the camera looks to have the scene band fill the frame (the window's pixels at
// rest): the band's left edge at the frame's left, and a little of the page above it.
func sceneView(scene ui.Rect) (x, y float64) {
	w, h := float64(promoW)/promoZoom, float64(promoH)/promoZoom
	x = math.Min(float64(scene.X)*promoBaseScale+w/2, promoWinPxW-w/2)
	y = float64(scene.Y)*promoBaseScale - promoCamAbove + h/2
	return x, math.Max(h/2, math.Min(promoWinPxH-h/2, y))
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
// is the top left cell. A cell is 3 pt, as everywhere in the pixel language.
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

// drawPointer paints the pointer with its tip at (x, y) pixels, a cell being cell pixels.
func drawPointer(img *image.RGBA, x, y, cell float64) {
	outline, white := color.RGBA{22, 22, 28, 255}, color.RGBA{255, 255, 255, 255}
	for cy, row := range pointerSprite {
		for cx, ch := range row {
			var c color.RGBA
			switch ch {
			case 'X':
				c = outline
			case 'W':
				c = white
			default:
				continue
			}
			x0, x1 := int(math.Round(x+float64(cx)*cell)), int(math.Round(x+float64(cx+1)*cell))
			y0, y1 := int(math.Round(y+float64(cy)*cell)), int(math.Round(y+float64(cy+1)*cell))
			for py := y0; py < y1; py++ {
				for px := x0; px < x1; px++ {
					img.SetRGBA(px, py, c)
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
	stops []waypoint // the moves: from stops[i] (left at rest[i]) to stops[i+1] (reached at its .T)
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

// --- the frame ------------------------------------------------------------------------------------

const promoWinX, promoWinY = (promoW - promoWinPxW) / 2, (promoH - promoWinPxH) / 2

// disc paints an anti-aliased disc.
func disc(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
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
			if cover == 0 || !(image.Point{x, y}).In(img.Rect) {
				continue
			}
			o := img.RGBAAt(x, y)
			mix := func(a, b uint8) uint8 { return uint8(float64(a)*(1-cover) + float64(b)*cover + 0.5) }
			img.SetRGBA(x, y, color.RGBA{mix(o.R, c.R), mix(o.G, c.G), mix(o.B, c.B), 255})
		}
	}
}

// cornerCover is how much of the pixel (x, y), counted from the corner of a window, is inside a
// corner of radius r.
func cornerCover(x, y int, r float64) float64 {
	d := math.Hypot(float64(x)+0.5-r, float64(y)+0.5-r)
	if d < r-1.5 {
		return 1
	}
	if d > r+1.5 {
		return 0
	}
	cover := 0.0
	for sy := 0; sy < 4; sy++ {
		for sx := 0; sx < 4; sx++ {
			if math.Hypot(float64(x)+(float64(sx)+0.5)/4-r, float64(y)+(float64(sy)+0.5)/4-r) <= r {
				cover += 1.0 / 16
			}
		}
	}
	return cover
}

// frameMaker puts the window on the backdrop as the camera sees them.
type frameMaker struct {
	back *image.RGBA // the dimmed backdrop with the window's shadow, at rest
}

// backAt is the backdrop pixel at output pixel (X, Y) when the camera is at c.
func (f frameMaker) backAt(c camKey, X, Y int) color.RGBA {
	bx := int(float64(promoWinX) + c.X + (float64(X)-promoW/2)/c.Zoom)
	by := int(float64(promoWinY) + c.Y + (float64(Y)-promoH/2)/c.Zoom)
	bx, by = max(0, min(promoW-1, bx)), max(0, min(promoH-1, by))
	return f.back.RGBAAt(bx, by)
}

// make is the frame: win is the window as the renderer drew it at the scale the camera asked for.
// It returns the frame and where the window's top left corner is in it.
func (f frameMaker) make(win *image.RGBA, c camKey, scale float64) (*image.RGBA, int, int) {
	ox := int(math.Round(promoW/2 - c.X*c.Zoom))
	oy := int(math.Round(promoH/2 - c.Y*c.Zoom))
	wb := win.Bounds()
	ww, wh := wb.Dx(), wb.Dy()
	out := image.NewRGBA(image.Rect(0, 0, promoW, promoH))
	covered := ox <= 0 && oy <= 0 && ox+ww >= promoW && oy+wh >= promoH
	switch {
	case covered:
	case math.Abs(c.Zoom-1) < 1e-9:
		copy(out.Pix, f.back.Pix)
	default:
		var wg sync.WaitGroup
		for y0 := 0; y0 < promoH; y0 += 135 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for y := y0; y < y0+135; y++ {
					for x := 0; x < promoW; x++ {
						out.SetRGBA(x, y, f.backAt(c, x, y))
					}
				}
			}()
		}
		wg.Wait()
	}
	// The window's rows, clipped to the frame.
	x0, x1 := max(0, ox), min(promoW, ox+ww)
	for y := max(0, oy); y < min(promoH, oy+wh); y++ {
		if x1 > x0 {
			copy(out.Pix[y*out.Stride+x0*4:y*out.Stride+x1*4], win.Pix[(y-oy)*win.Stride+(x0-ox)*4:(y-oy)*win.Stride+(x1-ox)*4])
		}
	}
	// Its rounded corners: the backdrop shows through where the corner is cut.
	r := promoCornerPt * scale
	n := int(math.Ceil(r))
	for _, corner := range [4][2]int{{0, 0}, {ww - n, 0}, {0, wh - n}, {ww - n, wh - n}} {
		for dy := 0; dy < n; dy++ {
			for dx := 0; dx < n; dx++ {
				wx, wy := corner[0]+dx, corner[1]+dy
				X, Y := ox+wx, oy+wy
				if X < 0 || Y < 0 || X >= promoW || Y >= promoH {
					continue
				}
				// The corner's own coordinates: from the corner of the window, towards its inside.
				cx, cy := dx, dy
				if corner[0] > 0 {
					cx = n - 1 - dx
				}
				if corner[1] > 0 {
					cy = n - 1 - dy
				}
				cover := cornerCover(cx, cy, r)
				if cover >= 1 {
					continue
				}
				b, w := f.backAt(c, X, Y), win.RGBAAt(wx, wy)
				mix := func(a, b uint8) uint8 { return uint8(float64(a)*(1-cover) + float64(b)*cover + 0.5) }
				out.SetRGBA(X, Y, color.RGBA{mix(b.R, w.R), mix(b.G, w.G), mix(b.B, w.B), 255})
			}
		}
	}
	// The renderer's frame may carry some alpha (a translucent sidebar): the frame is opaque, so every
	// PNG has the same pixel format.
	for i := 3; i < len(out.Pix); i += 4 {
		out.Pix[i] = 255
	}
	return out, ox, oy
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
	w := &pngWriter{jobs: make(chan pngJob, workers), enc: png.Encoder{CompressionLevel: png.BestSpeed}}
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
	tt := ui.NewTester(m.View, promoWinDIP, promoWinHDIP)
	tt.SetScale(promoBaseScale)
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

	// The pointer rests on Mine, goes to Summary, rests there through the day, goes to This week.
	// The week's stop is moved to its real place once Summary has been laid out.
	path := pointerPath{
		stops: []waypoint{{promoWinStart, 760, 440}, {promoClickSummary - 0.08, summaryX, summaryY}, {promoClickWeek - 0.08, 0, 0}},
		rest:  []float64{promoWinStart + 0.1, promoClickWeek - 0.9, promoClickWeek},
	}
	centre := camKey{Zoom: 1, X: promoWinPxW / 2, Y: promoWinPxH / 2}
	track := promoTrack{Clicks: [2]float64{promoClickSummary, promoClickWeek}, Camera: []camKey{centre}}
	zoomed := func(x, y float64) camKey { return camKey{Zoom: promoZoom, X: x, Y: y} }
	// moveIn and moveOut add the camera's moves: in from lead after a scene began, out when it is over.
	moveCamera := func(in, out float64, view camKey) {
		for _, k := range []camKey{
			{T: in, Zoom: 1, X: centre.X, Y: centre.Y}, {T: in + promoCamEase, Zoom: view.Zoom, X: view.X, Y: view.Y},
			{T: out, Zoom: view.Zoom, X: view.X, Y: view.Y}, {T: out + promoCamEase, Zoom: 1, X: centre.X, Y: centre.Y},
		} {
			track.Camera = append(track.Camera, k)
		}
	}

	back := promoBackdrop(t, promoDim, true)
	maker := frameMaker{back: back}
	wr := newPNGWriter(6)
	winEnd := math.Inf(1)
	clickedSummary, clickedWeek := false, false
	began := time.Now()
	k := 0
	for ; ; k++ {
		T := promoWinStart + float64(k)/promoFPS
		if T >= winEnd {
			break
		}
		vt = time.Duration(float64(time.Second) * (T - promoWinStart))
		cam := camAt(track.Camera, T)
		scale := promoBaseScale * cam.Zoom
		tt.SetScale(float32(scale))
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
			track.DayDone = track.DayStart + ledgerDone(v.sceneEnd(), len(v.sum.People)+1, v.sum.Total()).Seconds()
			track.addCues(v.day.SoundCues(false), track.DayStart)
			tt.Frame()
			week, ok := tt.Find(m.say.SummarySegmentWeek())
			if !ok {
				t.Fatalf("no %q in %q", m.say.SummarySegmentWeek(), tt.Texts())
			}
			path.stops[2].X, path.stops[2].Y = week.X+week.W*0.5, week.Y+week.H/2
			x, y := sceneView(m.sum.scene)
			moveCamera(track.DayStart+promoCamLead, track.DayEnd, zoomed(x, y))
		case !clickedWeek && T >= promoClickWeek:
			clickedWeek = true
			tt.ClickAt(px, py)
			v := m.summaryFor(app.Now())
			track.WeekStart = promoWinStart + m.sum.start.Sub(promoBase).Seconds()
			track.WeekEnd = track.WeekStart + v.sceneEnd().Seconds()
			track.WeekDone = track.WeekStart + ledgerDone(v.sceneEnd(), len(v.sum.People)+1, v.sum.Total()).Seconds()
			track.addCues(v.week.SoundCues(false), track.WeekStart)
			// The ledger has lit and counted; the video holds a moment, then cuts (200 ms crossfade, the
			// middle of it 100 ms before winEnd).
			winEnd = track.WeekDone + promoHold + 0.2
			tt.Frame()
			x, y := sceneView(m.sum.scene)
			moveCamera(track.WeekStart+promoCamLead, track.WeekEnd, zoomed(x, y))
		default:
			tt.Frame()
		}

		frame, ox, oy := maker.make(tt.Image(), cam, scale)
		for i, c := range []color.RGBA{{255, 95, 87, 255}, {254, 188, 46, 255}, {40, 200, 64, 255}} {
			disc(frame, float64(ox)+float64(19+20*i)*scale, float64(oy)+26*scale, 6*scale, c)
		}
		dip := 0.0
		for _, c := range []float64{promoClickSummary, promoClickWeek} {
			if T >= c && T < c+promoClickDip {
				dip = 3 // the pointer dips a cell as it clicks
			}
		}
		drawPointer(frame, float64(ox)+float64(px)*scale, float64(oy)+(float64(py)+dip)*scale, 3*scale)
		wr.put(filepath.Join(out, fmt.Sprintf("w%05d.png", k)), frame)
		if k%120 == 0 {
			t.Logf("frame %d, T=%.2fs, zoom %.2f, %.0f ms a frame", k, T, cam.Zoom, float64(time.Since(began).Milliseconds())/float64(k+1))
		}
	}
	wr.close(t)
	if !clickedSummary || !clickedWeek {
		t.Fatalf("clicks: summary %v, week %v", clickedSummary, clickedWeek)
	}
	track.Frames, track.WinEnd, track.Total = k, winEnd, winEnd+4.9
	data, err := json.MarshalIndent(track, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"track.json": string(data),
		"timing.env": fmt.Sprintf("WIN_START=%.3f\nWIN_END=%.3f\nTOTAL=%.3f\n", promoWinStart, track.WinEnd, track.Total),
		// name and video second of the stills the script takes from the finished video
		"stills.txt": fmt.Sprintf("01-title-hop 1.2\n02-crossfade 2.5\n03-pointer-at-summary %.3f\n04-camera-in-throws %.3f\n05-camera-hop %.3f\n"+
			"06-paragraph %.3f\n07-day-ledger %.3f\n08-pointer-at-week %.3f\n09-week-falling %.3f\n10-week-camera-out %.3f\n11-week-ledger-lit %.3f\n12-end-card %.3f\n",
			promoClickSummary+0.07, track.DayStart+3.2, track.DayEnd-0.3, track.DayEnd+1.8, track.DayDone-0.2,
			promoClickWeek+0.07, track.WeekStart+3.0, track.WeekEnd+0.7, track.WeekDone+0.3, track.Total-1.5),
		"audio.wav": string(promoMix(t, track)),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(out, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
