package views

import (
	"sort"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The pictures of tasks (see ui-spec, Task images): a thumbnail in lists and on cards, tiles in the
// inspector, the viewer. Every one is drawn from the 256 px thumbnail file next to the original; the
// original is read only by the viewer, and never on the UI thread.

// Sizes, in points.
const (
	rowThumbW, rowThumbH float32 = 32, 24 // in a list row
	rowThumbRadius       float32 = 4      // of the row's thumbnail
	cardBandH            float32 = 96     // the band across the top of a card
	tileW, tileH         float32 = 96, 72 // in the inspector
	tileGap              float32 = 12     // between tiles
	tileRadius           float32 = 8      // of a tile
	tileX                float32 = 18     // the round x that detaches a picture
	tilesPerRow                  = 3      // a row of tiles in the inspector
	thumbCacheMax                = 300    // bitmaps kept, least recently drawn go first
	thumbCacheDrop               = 40     // how many go at once when it is full
	viewerDim                    = 0.9    // opacity of the near-black behind the viewer
)

// thumbCache keeps the decoded thumbnails, by file, so that drawing never decodes: a picture is
// read once, off the UI thread, and its bitmap reused by every row and card that shows it.
type thumbCache struct {
	items   map[string]*thumbEntry
	loading map[string]bool
	tick    uint64
}

type thumbEntry struct {
	bmp  *ui.Bitmap // nil when the file could not be read
	used uint64
}

// thumbOf is the bitmap of a thumbnail file, or nil while it is being read (or when it cannot be:
// the tile is then just its well).
func (m *Main) thumbOf(path string) *ui.Bitmap {
	tc := &m.thumbs
	tc.tick++
	if e, ok := tc.items[path]; ok {
		e.used = tc.tick
		return e.bmp
	}
	if tc.loading[path] {
		return nil
	}
	if tc.loading == nil {
		tc.loading, tc.items = map[string]bool{}, map[string]*thumbEntry{}
	}
	tc.loading[path] = true
	post := m.app.Act.Post
	go func() {
		img, err := data.LoadThumbnail(path)
		post(func() {
			delete(tc.loading, path)
			var bmp *ui.Bitmap
			if err == nil {
				bmp = ui.NewBitmap(img)
			}
			tc.put(path, bmp)
		})
	}()
	return nil
}

// put keeps a bitmap, and lets the least recently drawn ones go when there are too many.
func (tc *thumbCache) put(path string, bmp *ui.Bitmap) {
	tc.tick++
	tc.items[path] = &thumbEntry{bmp: bmp, used: tc.tick}
	if len(tc.items) <= thumbCacheMax {
		return
	}
	paths := make([]string, 0, len(tc.items))
	for p := range tc.items {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool { return tc.items[paths[i]].used < tc.items[paths[j]].used })
	for _, p := range paths[:thumbCacheDrop] {
		delete(tc.items, p)
	}
}

// pictureWell is a picture drawn whole (contain-fit) on bg.subtle at w x h with a hairline: a row's
// thumbnail, a card's band, a tile. A w of 0 takes the width of the parent. While the thumbnail is
// read the well is empty.
func (m *Main) pictureWell(c *ui.Context, t *theme.Tokens, key any, im data.TaskImage, w, h, radius float32) ui.Element {
	well := ui.Box(c.Key(key)).Height(h).Shrink(0).Radius(radius).Background(t.BgSubtle).Border(theme.Hairline, t.Border)
	if w > 0 {
		well.Width(w)
	} else {
		well.FillWidth()
	}
	return well.Role(ui.RoleImage).Label(im.Name).Children(func() {
		if bmp := m.thumbOf(im.Thumb); bmp != nil {
			img := ui.Image(c, bmp).Height(h).Fit(ui.Contain).Radius(radius)
			if w > 0 {
				img.Width(w)
			} else {
				img.FillWidth()
			}
		}
	})
}

// moreBadge is the +2 at the bottom right of a thumbnail with more pictures behind it: 9 pt, white
// on 55 % black.
func moreBadge(c *ui.Context, more int) {
	if more <= 0 {
		return
	}
	ui.Box(c.Key("more")).Absolute().Right(1).Bottom(1).Padding(0, 2).Radius(3).Background(ui.RGBA(0, 0, 0, 0.55)).
		PassThrough().Children(func() {
		theme.Text(c, theme.Pico, ui.Hex("#FFFFFF"), "+"+itoa(more))
	})
}

// rowThumb is a task's first picture before the due label or the age of its row, 32 x 24, with +N
// when it has more. A click opens the viewer. A task without pictures draws nothing.
func (m *Main) rowThumb(c *ui.Context, t *theme.Tokens, id int) {
	imgs := m.app.Images(id)
	if len(imgs) == 0 {
		return
	}
	btn := ui.ButtonBase(c.Key("pic-"+itoa(id))).Size(rowThumbW, rowThumbH).Shrink(0).Radius(rowThumbRadius).
		Label(m.say.ImageOpen(imgs[0].Name)).Cursor(ui.CursorPointer)
	btn.Children(func() {
		m.pictureWell(c, t, "well", imgs[0], rowThumbW, rowThumbH, rowThumbRadius)
		moreBadge(c, len(imgs)-1)
	})
	if btn.Clicked() {
		m.openViewer(id, 0)
	}
}

// cardBand is the first picture across the top of a card, 96 pt tall, with a hairline under it and
// +N at its bottom right. A click opens the viewer. It draws nothing for a task without pictures.
func (m *Main) cardBand(c *ui.Context, t *theme.Tokens, id int) {
	imgs := m.app.Images(id)
	if len(imgs) == 0 {
		return
	}
	btn := ui.ButtonBase(c.Key("band")).FillWidth().Height(cardBandH).Shrink(0).Label(m.say.ImageOpen(imgs[0].Name)).
		Cursor(ui.CursorPointer)
	btn.Children(func() {
		band := ui.Box(c.Key("band-well")).FillWidth().Height(cardBandH).Background(t.BgSubtle)
		band.Children(func() {
			if bmp := m.thumbOf(imgs[0].Thumb); bmp != nil {
				ui.Image(c, bmp).FillWidth().Height(cardBandH).Fit(ui.Contain)
			}
		})
		ui.Box(c.Key("band-line")).FillWidth().Height(theme.Hairline).Background(t.Border).Absolute().Left(0).Right(0).Bottom(0).PassThrough()
		moreBadge(c, len(imgs)-1)
	})
	if btn.Clicked() {
		m.openViewer(id, 0)
	}
}
