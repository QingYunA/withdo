package views

import (
	"errors"
	"strconv"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// Import image… of the category sheet (v24 frames 7c and 7d): the picture and its 12 x 12 result,
// a threshold stepper and an Outline only switch.

const (
	importSide    float32 = 144 // the picture's box and the result's, 12 pt a pixel
	thresholdStep         = 4
)

// importing is the state of an import: the file as it was read and the two settings that turn it
// into the sheet's icon.
type importing struct {
	src     data.IconImport
	thumb   *ui.Bitmap // the picture, to show; nil for an SVG
	percent int        // the threshold
	outline bool
}

// pickIconFiles opens the file panel for the Import image… button: the pictures of the composer and
// SVG. Tests replace it.
var pickIconFiles = func(title string) []string {
	return openFiles(title, false, "png", "jpg", "jpeg", "gif", "webp", "svg")
}

// chooseIcon opens the file panel and reads what is chosen, off the UI thread.
func (m *Main) chooseIcon() {
	title := m.say.CategoryImportTitle()
	go func() {
		if paths := pickIconFiles(title); len(paths) > 0 {
			m.readIcon(paths[0])
		}
	}()
}

// readIcon reads a file for the icon and hands the result to the UI thread. It blocks, so it runs
// off that thread: from the file panel's goroutine, or for a file dropped on the sheet.
func (m *Main) readIcon(path string) {
	imp, err := data.ReadIconFile(path)
	m.app.Act.Post(func() { m.iconRead(imp, err) })
}

// iconRead shows what a file came to: the import, or a line about what was wrong with it.
func (m *Main) iconRead(imp data.IconImport, err error) {
	s := &m.manage.category
	if !s.open {
		return
	}
	s.iconErr = err
	if err != nil {
		return
	}
	s.imp = importing{src: imp, percent: imp.Threshold()}
	if !imp.IsVector() {
		s.imp.thumb = ui.NewBitmap(imp.Thumb)
	}
	s.mode = modeImport
	s.icon = s.imp.icon()
}

// icon is the 12 x 12 result for the threshold and the switch.
func (i importing) icon() pixmap.Picture {
	p := i.src.Picture(i.percent, theme.PaletteRGBA())
	if i.outline {
		p = p.Outline()
	}
	return p
}

// categoryImport is the picture and its result, the stepper with the live previews, and the switch.
// An SVG is already 12 x 12: it has no picture beside it and no threshold.
func (m *Main) categoryImport(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	imp := &s.imp
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		if !imp.src.IsVector() {
			m.importThumb(c, t, imp)
			ui.Icon(c, theme.IconMove).Size(theme.IconRow, theme.IconRow).TextColor(t.TextSecondary).Shrink(0)
		}
		ui.Column(c).Gap(6).Children(func() {
			ui.Box(c.Key("import-result")).Size(importSide, importSide).Shrink(0).Radius(6).Background(t.BgSubtle).Center().Children(func() {
				categoryGlyph(c, data.Category{Icon: s.icon, Color: s.color}, importSide)
			})
			theme.Text(c, theme.Meta, t.TextSecondary, "12 × 12")
		})
	})
	ui.Row(c).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
		if !imp.src.IsVector() {
			m.thresholdStepper(c, t, s)
		}
		m.categoryPreview(c, t, s, theme.IconBar, theme.IconRow, theme.IconSmall)
	})
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		label := m.say.CategoryOutline()
		if ui.Switch(c.Key("import-outline"), &imp.outline).Label(label).Changed() {
			s.icon = imp.icon()
		}
		theme.Text(c, theme.Compact, t.TextSecondary, label)
	})
}

// importThumb is the picture as it was, 144 pt square on a light well, with the file's name under it.
func (m *Main) importThumb(c *ui.Context, t *theme.Tokens, imp *importing) {
	ui.Column(c).Gap(6).Width(importSide).Shrink(0).Children(func() {
		ui.Box(c.Key("import-source")).Size(importSide, importSide).Radius(6).Background(t.BgSubtle).Border(theme.Hairline, t.Border).Role(ui.RoleImage).
			Label(imp.src.Name).Children(func() {
			ui.Image(c, imp.thumb).Size(importSide, importSide).Fit(ui.Contain).Radius(6)
		})
		theme.Text(c, theme.Meta, t.TextSecondary, imp.src.Name).Ellipsis("…")
	})
}

// thresholdStepper is "Threshold − 42% +": how much ink a cell needs to become a pixel, in steps of 4.
func (m *Main) thresholdStepper(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		theme.Text(c, theme.Compact, t.TextSecondary, m.say.CategoryThreshold())
		if m.toolButton(c, t, "threshold-less", m.say.CategoryThresholdLess(), theme.IconMinus) {
			s.stepThreshold(-thresholdStep)
		}
		theme.Text(c, theme.Compact, t.Text, strconv.Itoa(s.imp.percent)+"%").Width(36).TextAlign(ui.Center)
		if m.toolButton(c, t, "threshold-more", m.say.CategoryThresholdMore(), theme.IconPlus) {
			s.stepThreshold(thresholdStep)
		}
	})
}

// stepThreshold moves the threshold, between 0 and 100 percent, and redraws the icon from it.
func (s *categorySheet) stepThreshold(by int) {
	s.imp.percent = max(0, min(100, s.imp.percent+by))
	s.icon = s.imp.icon()
}

// iconProblemLine words what the last import found wrong with the file.
func (m *Main) iconProblemLine() string {
	switch err := m.manage.category.iconErr; {
	case err == nil:
		return ""
	case errors.Is(err, data.ErrIconCurves):
		return m.say.CategoryImportErrCurves()
	}
	return m.say.CategoryImportErrRead()
}
