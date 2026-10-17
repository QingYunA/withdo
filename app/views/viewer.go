package views

import (
	"slices"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The viewer opens a task's picture over the window: the window dims, the picture fits it whole, a
// bar says which it is and offers Preview. The arrows step through the task's pictures, Esc or a
// click outside closes. Its colors are fixed: it is a dark room in light and dark mode alike.

const (
	viewerMargin float32 = 56 // between the window's edge and the picture
	viewerBarH   float32 = 48
	viewerButton float32 = 32 // the round chevrons and close
	viewerEdge   int     = 2400
	viewerKept           = 3 // full pictures kept, so stepping back is instant
)

var (
	viewerInk    = ui.Hex("#C7C7CC")
	viewerLink   = ui.Hex("#A5A5FF")
	viewerWell   = ui.RGBA(255, 255, 255, 0.12)
	viewerWellOn = ui.RGBA(255, 255, 255, 0.22)
	viewerBack   = ui.RGBA(20, 20, 22, viewerDim)
)

// viewer is the state of the picture viewer: which task's pictures, which one, and the full-size
// bitmaps read so far.
type viewer struct {
	open    bool
	task    int
	at      int
	full    map[string]*ui.Bitmap // by original file, the latest few
	order   []string
	loading map[string]bool
}

// openViewer shows the at-th picture of a task, when it has one.
func (m *Main) openViewer(task, at int) {
	imgs := m.app.Images(task)
	if len(imgs) == 0 {
		return
	}
	m.viewer.open, m.viewer.task, m.viewer.at = true, task, max(0, min(at, len(imgs)-1))
}

// viewerImages are the pictures being viewed; the viewer closes when there are none left (the
// task or its pictures went away).
func (m *Main) viewerImages() []data.TaskImage {
	if !m.viewer.open {
		return nil
	}
	imgs := m.app.Images(m.viewer.task)
	if len(imgs) == 0 {
		m.viewer.open = false
		return nil
	}
	m.viewer.at = max(0, min(m.viewer.at, len(imgs)-1))
	return imgs
}

// step moves to the next or previous picture, without wrapping.
func (m *Main) viewerStep(dir, n int) {
	m.viewer.at = max(0, min(m.viewer.at+dir, n-1))
}

// viewerBitmap is the picture to draw: its full size once it is read, meanwhile its thumbnail.
func (m *Main) viewerBitmap(im data.TaskImage) *ui.Bitmap {
	v := &m.viewer
	if b, ok := v.full[im.File]; ok {
		return b
	}
	if !v.loading[im.File] {
		if v.loading == nil {
			v.loading, v.full = map[string]bool{}, map[string]*ui.Bitmap{}
		}
		v.loading[im.File] = true
		post := m.app.Act.Post
		go func() {
			img, err := data.LoadPicture(im.File, viewerEdge)
			post(func() {
				delete(v.loading, im.File)
				if err == nil {
					v.keep(im.File, ui.NewBitmap(img))
				}
			})
		}()
	}
	return m.thumbOf(im.Thumb)
}

// keep remembers a full picture, and forgets the oldest beyond the last few.
func (v *viewer) keep(path string, b *ui.Bitmap) {
	v.full[path] = b
	v.order = append(slices.DeleteFunc(v.order, func(p string) bool { return p == path }), path)
	for len(v.order) > viewerKept {
		delete(v.full, v.order[0])
		v.order = v.order[1:]
	}
}

// viewerView draws the viewer while it is open.
func (m *Main) viewerView(c *ui.Context, t *theme.Tokens) {
	imgs := m.viewerImages()
	if len(imgs) == 0 {
		return
	}
	v := &m.viewer
	w, h := c.Size()
	ui.DialogBase(c, &v.open, func(backdrop, panel ui.Element) {
		backdrop.Background(viewerBack)
		panel.Size(w, h)
		if len(imgs) > 1 { // the arrows go to the overlay: the window's own keys are shut out
			switch {
			case panel.OverlayShortcut(0, ui.KeyLeft):
				m.viewerStep(-1, len(imgs))
			case panel.OverlayShortcut(0, ui.KeyRight):
				m.viewerStep(1, len(imgs))
			}
		}
		im := imgs[v.at]
		scrim := ui.Box(c.Key("viewer-scrim")).Absolute().Left(0).Top(0).Size(w, h)
		if scrim.Clicked() { // outside the picture
			v.open = false
		}
		m.viewerPicture(c, im, w, h)
		if len(imgs) > 1 {
			m.viewerChevron(c, "prev", theme.IconBack, m.say.ViewerPrevious(), 16, v.at > 0, func() { m.viewerStep(-1, len(imgs)) }, h)
			m.viewerChevron(c, "next", theme.IconForward, m.say.ViewerNext(), w-16-viewerButton, v.at < len(imgs)-1, func() { m.viewerStep(1, len(imgs)) }, h)
		}
		if m.viewerRound(c, "close", theme.IconClose, m.say.ViewerClose(), w-14-viewerButton, 14) {
			v.open = false
		}
		m.viewerBar(c, im, v.at, len(imgs), w, h)
	})
}

// viewerPicture is the picture, whole, in the room the window leaves; a click on it does nothing.
func (m *Main) viewerPicture(c *ui.Context, im data.TaskImage, w, h float32) {
	room := ui.Box(c.Key("viewer-room")).Absolute().Left(viewerMargin).Top(viewerMargin).Size(w-2*viewerMargin, h-viewerMargin-viewerBarH-8)
	room.Role(ui.RoleImage).Label(m.say.ViewerImage(im.Name)).Children(func() {
		if bmp := m.viewerBitmap(im); bmp != nil {
			ui.Image(c, bmp).Size(w-2*viewerMargin, h-viewerMargin-viewerBarH-8).Fit(ui.Contain).Radius(4)
		}
	})
}

// viewerChevron is a round chevron at a window edge, level with the middle of the window.
func (m *Main) viewerChevron(c *ui.Context, key string, icon *ui.SVG, label string, x float32, enabled bool, step func(), h float32) {
	if !enabled {
		return
	}
	if m.viewerRound(c, key, icon, label, x, h/2-viewerButton/2) {
		step()
	}
}

// viewerRound is a round 32 pt button of the viewer at x, y with a pixel icon; it reports a click.
func (m *Main) viewerRound(c *ui.Context, key string, icon *ui.SVG, label string, x, y float32) bool {
	btn := ui.ButtonBase(c.Key("viewer-"+key)).Absolute().Left(x).Top(y).Size(viewerButton, viewerButton).Radius(viewerButton / 2).
		Background(viewerWell).Center().Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(viewerWellOn)
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { ui.Icon(c, icon).Size(theme.IconRow, theme.IconRow).TextColor(ui.Hex("#F5F5F7")) })
	return btn.Clicked()
}

// viewerBar is the bottom bar: the name, the place among the pictures and the size, and Preview.
func (m *Main) viewerBar(c *ui.Context, im data.TaskImage, at, n int, w, h float32) {
	ui.Row(c.Key("viewer-bar")).Absolute().Left(0).Top(h-viewerBarH).Size(w, viewerBarH).Padding(0, 18).Gap(14).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Meta, viewerInk, m.say.ViewerPosition(im.Name, at+1, n, itoa(im.Width), itoa(im.Height))).Shrink(1).MinWidth(0).Ellipsis("…")
		ui.Spacer(c)
		open := ui.ButtonBase(c.Key("viewer-preview")).Padding(4, 8).Radius(theme.RadiusControl).Shrink(0).
			Label(m.say.ViewerOpenInPreview()).Cursor(ui.CursorPointer)
		if open.Hovered() {
			open.Background(viewerWell)
		}
		open.Children(func() { theme.Text(c, theme.Meta, viewerLink, m.say.ViewerOpenInPreview()) })
		if open.Clicked() {
			m.app.Act.OpenFile(im.File)
		}
	})
}
