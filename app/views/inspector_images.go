package views

import (
	"errors"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The images block of the inspector, between the properties and the notes: a task's pictures as
// tiles, three a row, a dashed + tile that opens the file panel, or with none a quiet line. Image
// files dropped on the pane are attached, with the types and limits of the composer's.

// imagesBlock draws the pictures of a task, or the quiet line that adds the first.
func (m *Main) imagesBlock(c *ui.Context, t *theme.Tokens, task data.Task) {
	imgs := m.app.Images(task.ID)
	ui.Column(c.Key("images")).Gap(8).Children(func() {
		if len(imgs) == 0 {
			m.addImagesLine(c, t, task)
		} else {
			m.tiles(c, t, task, imgs)
		}
		m.imagesProblem(c, t)
	})
}

// addImagesLine is "Add images…", quiet, like the notes' placeholder; it opens the file panel.
func (m *Main) addImagesLine(c *ui.Context, t *theme.Tokens, task data.Task) {
	btn := ui.ButtonBase(c.Key("add-images")).Height(theme.RowOneLine).Shrink(0).Radius(theme.RadiusControl).
		AlignItems(ui.Center).Justify(ui.Start).Label(m.say.ImagesAdd()).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if btn.Hovered() {
		ink = t.TextSecondary
	}
	btn.Children(func() { theme.Text(c, theme.Body, ink, m.say.ImagesAdd()) })
	if btn.Clicked() {
		m.chooseTaskPictures(task.ID)
	}
}

// tiles lays the pictures and the + tile out in rows of three.
func (m *Main) tiles(c *ui.Context, t *theme.Tokens, task data.Task, imgs []data.TaskImage) {
	cells := len(imgs) + 1 // the last is the + tile
	ui.Column(c.Key("tiles")).Gap(tileGap).Children(func() {
		for start := 0; start < cells; start += tilesPerRow {
			ui.Row(c.Key("tiles-" + itoa(start))).Gap(tileGap).Shrink(0).Children(func() {
				for i := start; i < min(start+tilesPerRow, cells); i++ {
					if i == len(imgs) {
						m.addTile(c, t, task)
					} else {
						m.imageTile(c, t, task, imgs, i)
					}
				}
			})
		}
	})
}

// imageTile is one picture, 96 x 72, whole on a light well. Hovered it shows the round x that takes
// it off the task; a click opens the viewer.
func (m *Main) imageTile(c *ui.Context, t *theme.Tokens, task data.Task, imgs []data.TaskImage, i int) {
	im := imgs[i]
	key := "tile-" + im.Hash
	tile := ui.ButtonBase(c.Key(key)).Size(tileW, tileH).Shrink(0).Radius(tileRadius).Label(m.say.ImageOpen(im.Name)).
		Cursor(ui.CursorPointer)
	hovered := tile.Hovered()
	if hovered {
		tile.Shadow(0, 0, 0, 2, t.Accent.Alpha(0.4))
	}
	tile.Children(func() {
		m.pictureWell(c, t, "well", im, tileW, tileH, tileRadius)
		if hovered {
			label := m.say.ImageDetach(im.Name)
			x := ui.ButtonBase(c.Key(key+"-x")).Absolute().Top(4).Right(4).Size(tileX, tileX).Radius(tileX / 2).
				Background(t.Text.Alpha(0.78)).Center().Label(label).Tooltip(label).Cursor(ui.CursorPointer)
			if x.Hovered() {
				x.Background(t.Text)
			}
			x.Children(func() { ui.Icon(c, theme.IconClose).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Bg) })
			if x.Clicked() {
				m.detachImage(c, task, im)
				return
			}
		}
	})
	if tile.Clicked() {
		m.openViewer(task.ID, i)
	}
}

// addTile is the dashed tile after the pictures; it opens the file panel.
func (m *Main) addTile(c *ui.Context, t *theme.Tokens, task data.Task) {
	label := m.say.ImagesAddTile()
	tile := ui.ButtonBase(c.Key("add-tile")).Size(tileW, tileH).Shrink(0).Radius(tileRadius).Center().Label(label).Tooltip(label).
		Border(1, t.TextTertiary.Alpha(0.6)).BorderStyle(ui.BorderDashed).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if tile.Hovered() {
		tile.Background(t.BgHover)
		ink = t.TextSecondary
	}
	tile.Children(func() { ui.Icon(c, theme.IconPlus).Size(theme.IconRow, theme.IconRow).TextColor(ink) })
	if tile.Clicked() {
		m.chooseTaskPictures(task.ID)
	}
}

// detachImage takes a picture off a task at once and offers Undo, as deleting a task does.
func (m *Main) detachImage(c *ui.Context, task data.Task, im data.TaskImage) {
	if m.app.DetachImage(task.ID, im.Hash) != nil {
		return
	}
	c.AddToast(ui.Toast{
		ID: "image-detached", Title: m.say.ImageDetached(im.Name), Action: m.say.MenuUndo(), Timeout: undoWithin,
		OnAction: func() { _ = m.app.RestoreImage(im) },
	})
}

// chooseTaskPictures opens the file panel for the + tile and the quiet line.
func (m *Main) chooseTaskPictures(id int) {
	post, title := m.app.Act.Post, m.say.ComposerAttachTitle()
	go func() {
		paths := pickFiles(title)
		post(func() { m.attachToTask(id, paths) })
	}()
}

// attachToTask reads the image files, off the UI thread, and keeps them with the task. What cannot
// be taken is said in a quiet line under the block, as the composer says it.
func (m *Main) attachToTask(id int, paths []string) {
	if len(paths) == 0 {
		return
	}
	if len(paths) > data.MaxPictures {
		m.imagesNotice(m.say.ComposerPictureLimit())
		paths = paths[:data.MaxPictures]
	}
	post := m.app.Act.Post
	store := m.app.Store
	go func() {
		var pics []data.Attachment
		var failed error
		for _, p := range paths {
			att, err := data.ReadImageFile(p)
			if err != nil {
				failed = err
				continue
			}
			pics = append(pics, att)
		}
		if len(pics) > 0 {
			if _, err := store.AttachImages(id, pics); err != nil {
				failed = err
			}
		}
		post(func() {
			if failed != nil {
				m.imagesNotice(m.pictureErrorText(failed))
			}
			m.app.Refresh()
		})
	}()
}

// pictureErrorText words why a picture was not taken, as the composer's line does.
func (m *Main) pictureErrorText(err error) string {
	switch {
	case errors.Is(err, data.ErrPictureFormat):
		return m.say.ComposerPictureFormat()
	case errors.Is(err, data.ErrPictureSize):
		return m.say.ComposerPictureSize()
	}
	return m.say.ComposerPictureUnreadable()
}

// imagesNotice shows a line under the block for a few seconds.
func (m *Main) imagesNotice(text string) {
	m.inspector.imgProblem, m.inspector.imgProblemAt = text, m.app.Now()
}

// imagesProblem is the quiet line under the images block about a file that was not taken.
func (m *Main) imagesProblem(c *ui.Context, t *theme.Tokens) {
	in := &m.inspector
	age := m.app.Now().Sub(in.imgProblemAt)
	if in.imgProblem == "" || age >= pictureNoticeFor {
		return
	}
	c.After(pictureNoticeFor - age)
	theme.Text(c, theme.Compact, t.TextSecondary, in.imgProblem).Ellipsis("…")
}

// dropOverlay is the accent tint over the inspector while image files are dragged over it.
func dropOverlay(c *ui.Context, t *theme.Tokens) {
	ui.Box(c.Key("drop-over")).Absolute().Left(0).Top(0).Right(0).Bottom(0).Background(t.Accent.Alpha(0.06)).
		Border(1, t.Accent).PassThrough()
}
