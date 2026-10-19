//go:build promo

package views

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The backdrop, the title card and the end card of the promo video (docs/design/promo.md). The
// backdrop is the dusk scene scaled up with nearest neighbour to fill 3840 x 2160, one scene cell 27
// px; the cards draw it as it is with native text and the pixel figures of app/theme/sprites on it,
// through the same mygo renderer as the window, at 1920 x 1080 points and 2x.
//
//	PROMO_LANG=zh PROMO_OUT=$PWD/out/promo/work-zh go test -tags promo ./app/views -run TestPromoCards -count=1

const (
	promoW, promoH = 3840, 2160
	promoSceneCell = 27 // px of one cell of the scene
	// The window sits centred on the backdrop, at 1:1 (2200 x 1440).
	promoWinX, promoWinY = (promoW - 2200) / 2, (promoH - 1440) / 2
	promoDim             = 0.42 // what is left of the backdrop's light behind the window

	titleLen = 2.6 // seconds of title frames: the cut into the window crossfades from 2.4 s
)

// promoSprite is a picture of app/theme/sprites, drawn pixelated.
func promoSprite(t *testing.T, name string) (*image.NRGBA, *ui.Bitmap) {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "theme", "sprites", name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, b.Min, draw.Src)
	return nrgba, ui.NewBitmap(nrgba).Pixelated()
}

// promoBackdrop is the dusk scene filling the frame, dimmed by dim (1 leaves it as it is), with a
// soft shadow where the window will be when shadowed.
func promoBackdrop(t *testing.T, dim float64, shadowed bool) *image.RGBA {
	t.Helper()
	src, _ := promoSprite(t, "scenes/dusk")
	out := image.NewRGBA(image.Rect(0, 0, promoW, promoH))
	left := (src.Bounds().Dx()*promoSceneCell - promoW) / 2 // centred crop
	for y := 0; y < promoH; y++ {
		for x := 0; x < promoW; x++ {
			c := src.NRGBAAt((x+left)/promoSceneCell, y/promoSceneCell)
			out.SetRGBA(x, y, color.RGBA{uint8(float64(c.R) * dim), uint8(float64(c.G) * dim), uint8(float64(c.B) * dim), 255})
		}
	}
	if shadowed {
		// The shadow of helpers_test, wider and lower: a window floating well above the desktop.
		shadow(out, image.Rect(promoWinX, promoWinY+50, promoWinX+2200, promoWinY+1440+50), 200, true)
	}
	return out
}

// --- the cards ------------------------------------------------------------------------------------

type promoWords struct{ title, name, line, url string }

func promoText(zh bool) promoWords {
	if zh {
		return promoWords{"你和 Agent，今天做了什么", "adhd-todo", "看见每个 Agent 在做什么", "github.com/QingYunA/adhd-todo"}
	}
	return promoWords{"What you and your agents got done today", "adhd-todo", "See what every agent is doing", "github.com/QingYunA/adhd-todo"}
}

// cardFigures are the six agents with a figure of their own, in the order of the title card's row.
var cardFigures = []string{"claude", "codex", "gemini", "cursor", "copilot", "opencode"}

const (
	figCell    = 9    // points of one cell of a figure on the cards
	figGrid    = 24   // cells a figure is across
	figGap     = 270  // points between two figures' left edges
	figFeet    = 22   // cells from a figure's top to its feet
	pathRow    = 71   // the row of the scene where feet stand
	hopStart   = 0.65 // seconds: the first figure hops
	hopEvery   = 0.16 // and the next one, after
	hopLen     = 0.6  // how long a hop lasts
	hopHeight  = 9    // cells a hop rises
	cardCellPt = float32(promoSceneCell) / promoScale
)

// cardView is a card: the dusk backdrop, then draw (the figures), then the texts.
type cardView struct {
	t       *testing.T
	dusk    *ui.Bitmap
	poses   map[string]*ui.Bitmap
	icon    *ui.Bitmap
	words   promoWords
	title   bool    // the title card; otherwise the end card
	elapsed float64 // seconds into the title card
}

func newCardView(t *testing.T, zh bool) *cardView {
	_, dusk := promoSprite(t, "scenes/dusk")
	v := &cardView{t: t, dusk: dusk, poses: map[string]*ui.Bitmap{}, words: promoText(zh)}
	for _, f := range cardFigures {
		for _, pose := range []string{"idle", "hop"} {
			_, v.poses[f+"/"+pose] = promoSprite(t, "actors/"+f+"/"+pose)
		}
	}
	iconFile, err := os.Open(filepath.Join("..", "resources", "icon.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer iconFile.Close()
	img, err := png.Decode(iconFile)
	if err != nil {
		t.Fatal(err)
	}
	v.icon = ui.NewBitmap(img)
	return v
}

// panel is a dark scrim with its corners cut a step, behind a text that the sky would drown. Three
// strips that do not overlap, so that the scrim is one even shade.
func panel(p *ui.Painter, r ui.Rect) {
	const step = 12
	dark := ui.RGBA(14, 8, 24, 0.7)
	p.Fill(ui.Rect{X: r.X + step, Y: r.Y, W: r.W - 2*step, H: step}, dark, 0)
	p.Fill(ui.Rect{X: r.X, Y: r.Y + step, W: r.W, H: r.H - 2*step}, dark, 0)
	p.Fill(ui.Rect{X: r.X + step, Y: r.Y + r.H - step, W: r.W - 2*step, H: step}, dark, 0)
}

// hopOf is how many cells figure i is lifted, at the card's time.
func (v *cardView) hopOf(i int) (lift int, hopping bool) {
	u := (v.elapsed - hopStart - float64(i)*hopEvery) / hopLen
	if u <= 0 || u >= 1 {
		return 0, false
	}
	return int(math.Round(hopHeight * 4 * u * (1 - u))), true
}

func (v *cardView) view(c *ui.Context) {
	const sceneCellPt = float32(promoSceneCell) / promoScale
	ui.Box(c.Key("card")).FillWidth().FillHeight().Draw(func(p *ui.Painter, r ui.Rect) {
		w, h := v.dusk.Size()
		sceneW := float32(w) * sceneCellPt
		p.ImageRegion(v.dusk, ui.Rect{}, ui.Rect{X: (r.W - sceneW) / 2, Y: 0, W: sceneW, H: float32(h) * sceneCellPt}, ui.ImageOptions{})
		if v.title {
			v.drawFigures(p, r)
		}
	}).Children(func() {
		if v.title {
			v.titleTexts(c)
		} else {
			v.endTexts(c)
		}
	})
}

func (v *cardView) drawFigures(p *ui.Painter, r ui.Rect) {
	feet := float32(pathRow) * cardCellPt
	left := (r.W - float32((len(cardFigures)-1)*figGap+figGrid*figCell)) / 2
	for i, f := range cardFigures {
		pose := "idle"
		lift, hopping := v.hopOf(i)
		if hopping {
			pose = "hop"
		}
		x := left + float32(i*figGap)
		y := feet - figFeet*figCell - float32(lift*figCell)
		p.ImageRegion(v.poses[f+"/"+pose], ui.Rect{}, ui.Rect{X: x, Y: y, W: figGrid * figCell, H: figGrid * figCell}, ui.ImageOptions{})
	}
}

// line is a centred line of text at y, white, with a darker copy under it for the sky.
func cardLine(c *ui.Context, key string, s theme.Style, y float32, alpha float32, text string) {
	ui.Column(c.Key(key)).Absolute().Left(0).Right(0).Top(y).AlignItems(ui.Center).PassThrough().Children(func() {
		theme.Text(c, s, ui.Hex("#FFFFFF").Alpha(alpha), text)
	})
}

func (v *cardView) titleTexts(c *ui.Context) {
	s := theme.Style{Size: 64, Line: 80, Weight: 700}
	textW, _ := ui.ShapeText(v.words.title, ui.Font{Size: s.Size, Weight: s.Weight}, 0).Size()
	ui.Box(c.Key("title-panel")).Absolute().Left(0).Right(0).Top(325).Height(130).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
		w := textW + 120
		panel(p, ui.Rect{X: r.X + (r.W-w)/2, Y: r.Y, W: w, H: r.H})
	})
	cardLine(c, "title", s, 350, 1, v.words.title)
}

func (v *cardView) endTexts(c *ui.Context) {
	ui.Box(c.Key("end-panel")).Absolute().Left(0).Right(0).Top(120).Height(660).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
		const w = 900
		panel(p, ui.Rect{X: r.X + (r.W-w)/2, Y: r.Y, W: w, H: r.H})
	})
	ui.Box(c.Key("icon")).Absolute().Left(0).Right(0).Top(170).Height(300).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
		const side = 300
		p.ImageRegion(v.icon, ui.Rect{}, ui.Rect{X: r.X + (r.W-side)/2, Y: r.Y, W: side, H: side}, ui.ImageOptions{})
	})
	cardLine(c, "name", theme.Style{Size: 84, Line: 100, Weight: 700}, 490, 1, v.words.name)
	cardLine(c, "line", theme.Style{Size: 36, Line: 48, Weight: 400}, 610, 1, v.words.line)
	cardLine(c, "url", theme.Style{Size: 28, Line: 40, Weight: 400}, 690, 0.8, v.words.url)
}

// shootCard renders a card at a time.
func shootCard(t *testing.T, v *cardView, zh bool) func(elapsed float64) *image.RGBA {
	tt := ui.NewTester(v.view, 1920, 1080)
	tt.SetScale(promoScale)
	tt.SetPreferences(ui.Preferences{})
	if zh {
		tt.SetLocale("zh-Hans")
	} else {
		tt.SetLocale("en")
	}
	tt.SetRegion(promoRegion(zh))
	var vt time.Duration
	setEngineClock(tt, func() time.Time { return promoBase.Add(vt) })
	return func(elapsed float64) *image.RGBA {
		v.elapsed = elapsed
		vt = time.Duration(elapsed * float64(time.Second))
		tt.Frame()
		return tt.Image()
	}
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestPromoCards(t *testing.T) {
	zh, out := promoLang(t)
	writePNG(t, filepath.Join(out, "backdrop-window.png"), promoBackdrop(t, promoDim, true))

	v := newCardView(t, zh)
	v.title = false
	writePNG(t, filepath.Join(out, "end.png"), shootCard(t, v, zh)(0))

	title := newCardView(t, zh)
	title.title = true
	shoot := shootCard(t, title, zh)
	wr := newPNGWriter(4)
	n := int(math.Round(titleLen * promoFPS))
	for k := 0; k < n; k++ {
		src := shoot(float64(k) / promoFPS)
		frame := image.NewRGBA(src.Bounds())
		copy(frame.Pix, src.Pix) // the renderer reuses its buffer
		wr.put(filepath.Join(out, fmt.Sprintf("t%04d.png", k)), frame)
	}
	wr.close(t)
}
