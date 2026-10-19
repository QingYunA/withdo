package theme

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The pictures of the Summary scene are files too (see sprites/README.md): actors, goods, props and
// backdrops. Unlike an avatar they are not drawn rectangle by rectangle but as bitmaps, one pixel of
// the file one cell of the scene, so a frame of the scene is a few dozen images and no more.

// Sizes of the scene's pictures, in cells. A good or a prop is its grid and the one-cell drop shadow
// the baker adds down and to the right; an actor is exactly 24 x 24, its shadow cut at the edge.
const (
	ActorGrid  = 24
	GoodGrid   = 16
	goodFile   = GoodGrid + 1 // with its shadow cell
	starGrid   = 7
	binW, binH = 41, 17
	SceneW     = 264 // the scene is laid out for this many cells,
	SceneH     = 80  // this many tall
)

// goodKinds are the goods, by the file under goods/ that draws them. The assistant picks one for
// every finished task (see data.GoodKinds, which a test keeps equal to this list).
var goodKinds = []string{"gem", "jar", "gear", "book", "flask", "mail", "coin", "scroll", "crate"}

// GoodKinds lists the kinds of goods.
func GoodKinds() []string { return append([]string(nil), goodKinds...) }

// actorPoses are the poses each actor folder has a file for. A pose a folder does not have is its
// idle one; idle2 is the pose it alternates with while it sways.
var actorPoses = map[string][]string{
	"you":      {"idle", "lift", "throw", "hop"},
	"claude":   {"idle", "idle2", "lift", "throw", "hop"},
	"codex":    {"idle", "idle2", "lift", "throw", "hop"},
	"gemini":   {"idle", "idle2", "lift", "throw", "hop"},
	"cursor":   {"idle", "idle2", "lift", "throw", "hop"},
	"copilot":  {"idle", "idle2", "lift", "throw", "hop"},
	"opencode": {"idle", "idle2", "lift", "throw", "hop"},
	"bot":      {"idle", "lift", "throw", "hop"},
}

// ActorBot is the folder of the robot every agent without a folder of its own is drawn with. It is
// a template in the gray of avatars/bot.png and is tinted like its avatar (see botIn).
const ActorBot = "bot"

// sceneFiles are the props, with the size of each file.
var sceneFiles = map[string][2]int{
	"props/star":     {starGrid, starGrid},
	"props/bin":      {binW, binH},
	"props/lid-open": {binW, 7},
	"props/lid-half": {binW, 4},
	"props/lid-shut": {binW + 2, 4},
	"scenes/dusk":    {SceneW, SceneH},
	"scenes/night":   {SceneW, SceneH},
	"scenes/day":     {SceneW, SceneH},
	"goods/gem":      {goodFile, goodFile},
	"goods/jar":      {goodFile, goodFile},
	"goods/gear":     {goodFile, goodFile},
	"goods/book":     {goodFile, goodFile},
	"goods/flask":    {goodFile, goodFile},
	"goods/mail":     {goodFile, goodFile},
	"goods/coin":     {goodFile, goodFile},
	"goods/scroll":   {goodFile, goodFile},
	"goods/crate":    {goodFile, goodFile},
}

// sceneFileSizes is every file of the scene with its size, for the test and for spriteFileNames.
func sceneFileSizes() map[string][2]int {
	out := make(map[string][2]int, len(sceneFiles)+16)
	for name, size := range sceneFiles {
		out[name+".png"] = size
	}
	for folder, poses := range actorPoses {
		for _, pose := range poses {
			out["actors/"+folder+"/"+pose+".png"] = [2]int{ActorGrid, ActorGrid}
		}
	}
	return out
}

// bitmaps keeps what was decoded from the files, once: the scene asks for the same pictures every
// frame. A bitmap is pixelated, so it is drawn as squares at any multiple of its size.
var bitmaps = struct {
	sync.Mutex
	by map[bitmapKey]*ui.Bitmap
}{by: map[bitmapKey]*ui.Bitmap{}}

// bitmapKey names a picture in the cache: a file, how many quarter turns it was turned (with the
// shadow cell cut), and the tint of a template robot.
type bitmapKey struct {
	file string
	spin int // 0 is the file as it is, 1 to 4 are quarter turns of its 16 x 16 body
	tint ui.Color
}

// loadNRGBA reads and decodes sprites/<file>.
func loadNRGBA(file string) *image.NRGBA {
	data, err := spriteFS.ReadFile("sprites/" + file)
	if err != nil {
		panic("theme: " + err.Error())
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(fmt.Sprintf("theme: sprites/%s: %v", file, err))
	}
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.SetNRGBA(x, y, color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA))
		}
	}
	return out
}

// bitmap is the picture of a file, made once.
func bitmap(key bitmapKey, build func() *image.NRGBA) *ui.Bitmap {
	bitmaps.Lock()
	defer bitmaps.Unlock()
	if b, ok := bitmaps.by[key]; ok {
		return b
	}
	b := ui.NewBitmap(build()).Pixelated()
	bitmaps.by[key] = b
	return b
}

// fileBitmap is the bitmap of sprites/<file>.png as it is.
func fileBitmap(file string) *ui.Bitmap {
	return bitmap(bitmapKey{file: file}, func() *image.NRGBA { return loadNRGBA(file + ".png") })
}

// spinBitmap is a good turned k quarter turns clockwise, as it flies: its 16 x 16 body, without the
// shadow it casts on the ground.
func spinBitmap(kind string, k int) *ui.Bitmap {
	file := "goods/" + kind
	return bitmap(bitmapKey{file: file, spin: k%4 + 1}, func() *image.NRGBA {
		img := cropNRGBA(loadNRGBA(file+".png"), GoodGrid)
		for i := 0; i < k%4; i++ {
			img = turnNRGBA(img)
		}
		return img
	})
}

// actorBitmap is the picture of an actor in a pose; a pose its folder has no file for is its idle.
// Robots are tinted.
func actorBitmap(folder, pose string, tint ui.Color) *ui.Bitmap {
	if !hasPose(folder, pose) {
		pose = "idle"
	}
	file := "actors/" + folder + "/" + pose
	if folder != ActorBot {
		return fileBitmap(file)
	}
	return bitmap(bitmapKey{file: file, tint: tint}, func() *image.NRGBA { return tintNRGBA(loadNRGBA(file+".png"), tint) })
}

// hasPose reports whether an actor folder has a file for a pose.
func hasPose(folder, pose string) bool {
	for _, p := range actorPoses[folder] {
		if p == pose {
			return true
		}
	}
	return false
}

// cropNRGBA is the top left n x n of img.
func cropNRGBA(img *image.NRGBA, n int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			out.SetNRGBA(x, y, img.NRGBAAt(x, y))
		}
	}
	return out
}

// turnNRGBA turns a square image a quarter turn clockwise: the top left corner goes to the top
// right.
func turnNRGBA(img *image.NRGBA) *image.NRGBA {
	n := img.Bounds().Dx()
	out := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			out.SetNRGBA(x, y, img.NRGBAAt(y, n-1-x))
		}
	}
	return out
}

// tintNRGBA recolors a template robot: the three grays botShades names become the same three
// shades of tint, every other color stays.
func tintNRGBA(img *image.NRGBA, tint ui.Color) *image.NRGBA {
	from, to := botShades(ui.Hex(botTints[0])), botShades(tint)
	out := image.NewNRGBA(img.Bounds())
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c := img.NRGBAAt(x, y)
			for i := range from {
				if c.A == 255 && c.R == from[i].R && c.G == from[i].G && c.B == from[i].B {
					c = color.NRGBA{R: to[i].R, G: to[i].G, B: to[i].B, A: 255}
					break
				}
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out
}
