package views

import (
	"errors"
	"image"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// Images in the composer: pasted with Cmd+V, dropped on the box, or chosen with the + button. They
// show as 64 x 48 thumbnails above the text and go to the assistant with it. They live in memory
// only: never in the database, never on disk.

const (
	thumbW, thumbH float32 = 64, 48
	thumbRadius    float32 = 8
	thumbX         float32 = 18 // the round x that removes one
)

// pictureNoticeFor is how long the line about a picture that was not taken stays.
const pictureNoticeFor = 4 * time.Second

// picture is an image waiting in a composer: what is sent, and what is drawn.
type picture struct {
	att   data.Attachment
	thumb *ui.Bitmap
}

// attachments are the pictures as the assistant takes them.
func attachments(ps []picture) []data.Attachment {
	out := make([]data.Attachment, len(ps))
	for i, p := range ps {
		out[i] = p.att
	}
	return out
}

// pickFiles opens the file panel for images and returns what was chosen, nil when it was
// cancelled. It blocks until the panel closes, so it runs off the UI thread. Tests replace it.
var pickFiles = func(title string) []string {
	return openFiles(title, true, "png", "jpg", "jpeg", "gif", "webp")
}

// openFiles is the file panel with a filter on the extensions.
func openFiles(title string, multiple bool, exts ...string) []string {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title: title, Multiple: multiple, Filters: []mygo.FileFilter{{Name: title, Extensions: exts}},
	})
	if err != nil {
		return nil
	}
	return paths
}

// attachFiles reads and prepares the files, off the UI thread, and adds the pictures among them in
// order as each is ready.
func (m *Main) attachFiles(s surface, paths []string) {
	m.attach(s, len(paths), func(i int) (data.Attachment, error) { return data.ReadImageFile(paths[i]) })
}

// attachBytes prepares one picture that came as bytes, as a paste does.
func (m *Main) attachBytes(s surface, name string, raw []byte) {
	m.attach(s, 1, func(int) (data.Attachment, error) { return data.PrepareImage(name, raw) })
}

func (m *Main) attach(s surface, n int, load func(i int) (data.Attachment, error)) {
	cs := &m.cmp[s]
	room := max(0, data.MaxPictures-len(cs.pics)-cs.pending)
	if n > room {
		m.pictureProblem(s, m.say.ComposerPictureLimit())
		n = room
	}
	if n == 0 {
		return
	}
	cs.pending += n
	post := m.app.Act.Post
	go func() {
		for i := 0; i < n; i++ {
			att, err := load(i)
			post(func() { m.pictureReady(s, att, err) })
		}
	}()
}

// pictureReady runs on the UI thread when a picture was prepared, or could not be.
func (m *Main) pictureReady(s surface, att data.Attachment, err error) {
	cs := &m.cmp[s]
	cs.pending--
	var thumb image.Image
	if err == nil {
		thumb, err = att.Thumbnail()
		if err != nil {
			err = data.ErrPictureFormat
		}
	}
	switch {
	case errors.Is(err, data.ErrPictureFormat):
		m.pictureProblem(s, m.say.ComposerPictureFormat())
	case errors.Is(err, data.ErrPictureSize):
		m.pictureProblem(s, m.say.ComposerPictureSize())
	case err != nil:
		m.pictureProblem(s, m.say.ComposerPictureUnreadable())
	case len(cs.pics) >= data.MaxPictures:
		m.pictureProblem(s, m.say.ComposerPictureLimit())
	default:
		cs.pics = append(append([]picture(nil), cs.pics...), picture{att: att, thumb: ui.NewBitmap(thumb)})
	}
}

// removePicture takes the i-th picture out of the composer.
func (m *Main) removePicture(s surface, i int) {
	cs := &m.cmp[s]
	if i < 0 || i >= len(cs.pics) {
		return
	}
	cs.pics = append(append([]picture(nil), cs.pics[:i]...), cs.pics[i+1:]...)
}

// pictureProblem is the quiet line under the composer about a picture that was not taken.
func (m *Main) pictureProblem(s surface, text string) {
	cs := &m.cmp[s]
	cs.problem, cs.problemAt = text, m.app.Now()
}

// pasteHook is what a paste into the composer's field does: a picture on the pasteboard becomes an
// attachment; anything else, text above all, is pasted as usual.
func (m *Main) pasteHook(c *ui.Context, s surface) func(string) bool {
	svc := c.Services()
	return func(text string) bool {
		if text != "" || !m.assistReady() {
			return false
		}
		formats := svc.ClipboardFormats()
		switch {
		case hasFormat(formats, transfer.PNG):
			if raw, err := svc.ReadClipboardData(transfer.PNG).Read(transfer.PNG); err == nil && len(raw) > 0 {
				m.attachBytes(s, "pasted.png", raw)
				return true
			}
		case hasFormat(formats, transfer.FileList), hasFormat(formats, transfer.URIList):
			if files, err := svc.ReadClipboardData(transfer.FileList, transfer.URIList).Files(); err == nil && len(files) > 0 {
				m.attachFiles(s, files)
				return true
			}
		}
		return false
	}
}

func hasFormat(formats []transfer.Format, f transfer.Format) bool {
	for _, g := range formats {
		if g == f {
			return true
		}
	}
	return false
}

// refocus puts the cursor back in the field of s, after a button of the composer took the focus.
func (m *Main) refocus(s surface) {
	if s == surfaceInspector {
		m.focusInspectorComposer = true
		return
	}
	m.focusComposer = true
}

// choosePictures opens the file panel for the + button.
func (m *Main) choosePictures(s surface) {
	post, title := m.app.Act.Post, m.say.ComposerAttachTitle()
	go func() {
		paths := pickFiles(title)
		post(func() {
			m.refocus(s)
			m.attachFiles(s, paths)
		})
	}()
}

// attachButton is the + at the left of the composer, in the line of a one-line box, else in
// its bottom row.
func (m *Main) attachButton(c *ui.Context, t *theme.Tokens, s surface) ui.Element {
	full := len(m.cmp[s].pics)+m.cmp[s].pending >= data.MaxPictures
	label := m.say.ComposerAttach()
	btn := ui.ButtonBase(c.Key("attach-"+itoa(int(s)))).Size(theme.SendSize, theme.SendSize).Shrink(0).Center().
		Radius(theme.SendSize / 2).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgHover)
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	ink := t.TextSecondary
	if full {
		ink = t.TextTertiary
	}
	btn.Children(func() { ui.Icon(c, theme.IconPlus).Size(theme.IconRow, theme.IconRow).TextColor(ink) })
	if btn.Clicked() && !full {
		m.choosePictures(s)
	}
	return btn
}

// thumbnails is the row of attached pictures above the text.
func (m *Main) thumbnails(c *ui.Context, t *theme.Tokens, s surface) {
	cs := &m.cmp[s]
	if len(cs.pics) == 0 {
		return
	}
	ui.Row(c.Key("thumbs-"+itoa(int(s)))).Shrink(0).Gap(8).Padding(4, 6, 0, 0).Children(func() {
		for i, p := range cs.pics {
			m.thumbnail(c, t, s, i, p)
		}
	})
}

// thumbnail is one picture, 64 x 48, with a round x at its corner that removes it. The whole
// picture shows, letterboxed on a light well, so two similar screenshots can be told apart.
func (m *Main) thumbnail(c *ui.Context, t *theme.Tokens, s surface, i int, p picture) {
	key := "thumb-" + itoa(int(s)) + "-" + itoa(i)
	ui.Box(c.Key(key)).Size(thumbW, thumbH).Shrink(0).Radius(thumbRadius).Background(t.BgSubtle).Border(1, t.Border).Role(ui.RoleImage).
		Label(m.say.ComposerPicture(p.att.Name)).Children(func() {
		ui.Image(c, p.thumb).Size(thumbW, thumbH).Fit(ui.Contain).Radius(thumbRadius)
		label := m.say.ComposerPictureRemove(p.att.Name)
		x := ui.ButtonBase(c.Key(key+"-x")).Absolute().Top(-6).Right(-6).Size(thumbX, thumbX).Radius(thumbX / 2).
			Background(t.Text.Alpha(0.78)).Center().Label(label).Tooltip(label).Cursor(ui.CursorPointer)
		if x.Hovered() {
			x.Background(t.Text)
		}
		x.Children(func() { ui.Icon(c, theme.IconClose).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Bg) })
		if x.Clicked() {
			m.removePicture(s, i)
		}
	})
}

// pictureNotice is the quiet line under the composer about a picture that was not taken.
func (m *Main) pictureNotice(c *ui.Context, t *theme.Tokens, s surface) {
	cs := &m.cmp[s]
	age := m.app.Now().Sub(cs.problemAt)
	if strings.TrimSpace(cs.problem) == "" || age >= pictureNoticeFor {
		return
	}
	c.After(pictureNoticeFor - age)
	ui.Row(c.Key("picture-notice-"+itoa(int(s)))).Height(answerLineHeight).Shrink(0).Margin(8, 0, 0).Padding(0, 12, 0, 16).
		AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Compact, t.TextSecondary, cs.problem).Ellipsis("…")
	})
}
