package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// Export Markdown (ui-spec, Summary): a sheet with the Markdown of the day or the week and Copy /
// Save as…. The paragraph first, then a section per person with a line per task, then what is still
// open.

// exportWidth is the width of the sheet: wider than the small sheets, for lines of Markdown.
const exportWidth float32 = 560

// exportHeight is how tall the Markdown's well is before it scrolls.
const exportHeight float32 = 300

// exportSheet is the state of the sheet.
type exportSheet struct {
	open     bool
	copiedAt time.Time
}

// exportButton is Export Markdown at the right of the header: the accent on a tint of it, dimmed
// when nothing was finished.
func (m *Main) exportButton(c *ui.Context, t *theme.Tokens, v *summaryView) {
	on := v.sum.Total() > 0
	label := m.say.SummaryExport()
	btn := ui.ButtonBase(c.Key("summary-export")).Height(26).Padding(0, 10).Gap(6).Shrink(0).Radius(7).AlignItems(ui.Center).Label(label)
	ink := t.Accent
	fill := t.Accent.Alpha(0.12)
	switch {
	case !on:
		ink, fill = t.TextTertiary, t.BgSubtle
	case btn.Hovered():
		fill = t.Accent.Alpha(0.2)
		btn.Cursor(ui.CursorPointer)
	default:
		btn.Cursor(ui.CursorPointer)
	}
	btn.Background(fill).Children(func() {
		ui.Icon(c.Key("export-icon"), theme.IconCopy).Size(theme.IconSmall, theme.IconSmall).TextColor(ink)
		theme.Text(c, theme.Meta, ink, label)
	})
	if on && btn.Clicked() {
		m.sum.export = exportSheet{open: true}
	}
}

// exportSheetView draws the sheet while it is open.
func (m *Main) exportSheetView(c *ui.Context, t *theme.Tokens, v *summaryView) {
	ex := &m.sum.export
	if !ex.open {
		return
	}
	md := m.summaryMarkdown(v)
	ui.DialogBase(c, &ex.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.32))
		panel.Width(exportWidth).Padding(24).Radius(theme.RadiusPanel).Background(t.Bg).Border(1, t.Border).Shadow(0, 20, 48, 0, t.Shadow)
		ui.Column(c).Gap(16).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, m.say.SummaryExportTitle())
			ui.Scroll(c.Key("export-well")).FillWidth().MaxHeight(exportHeight).Radius(theme.RadiusControl).Background(t.BgSubtle).Children(func() {
				theme.Paragraph(c.Key("export-text"), theme.Mono, t.Text, md, 0).FillWidth().Padding(12, 14).Selectable()
			})
			m.exportButtons(c, t, v, md)
		})
	})
}

// exportButtons are Close, Save as… and Copy, which says Copied for a moment.
func (m *Main) exportButtons(c *ui.Context, t *theme.Tokens, v *summaryView, md string) {
	ex := &m.sum.export
	now := m.app.Now()
	copied := !ex.copiedAt.IsZero() && now.Sub(ex.copiedAt) < copiedFor
	ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
		if ui.Button(c.Key("export-close"), m.say.SummaryExportClose()).Clicked() {
			ex.open = false
		}
		if ui.Button(c.Key("export-save"), m.say.SummaryExportSave()).Clicked() {
			m.app.Act.SaveText(exportFileName(v.period), md)
		}
		label := m.say.SummaryExportCopy()
		if copied {
			label = m.say.SummaryExportCopied()
			c.After(copiedFor - now.Sub(ex.copiedAt))
		}
		if ui.PrimaryButton(c.Key("export-copy"), label).AutoFocus().Clicked() {
			c.WriteClipboard(md)
			ex.copiedAt = now
		}
	})
}

// exportFileName is the file the Markdown is saved as by default.
func exportFileName(p data.Period) string {
	if p.Week {
		return "summary-week-" + p.FirstDate() + ".md"
	}
	return "summary-" + p.FirstDate() + ".md"
}

// summaryMarkdown is the Markdown of the period on screen.
func (m *Main) summaryMarkdown(v *summaryView) string {
	var b strings.Builder
	title := m.say.SummaryMdDay(v.period.FirstDate())
	if v.period.Week {
		title = m.say.SummaryMdWeek(v.period.FirstDate(), v.period.Day(6).Format("2006-01-02"))
	}
	b.WriteString("# " + title + "\n")
	if para := m.paragraphOf(v.period); para.stored != nil && strings.TrimSpace(para.stored.Paragraph) != "" {
		b.WriteString("\n" + strings.TrimSpace(para.stored.Paragraph) + "\n")
	}
	for _, p := range v.sum.People {
		name := m.say.DoneByYou()
		if !p.You() {
			name = agentName(p.Agent)
		}
		b.WriteString("\n## " + name + "\n")
		for _, task := range p.Tasks {
			b.WriteString("- " + m.markdownTask(task) + "\n")
		}
	}
	if open := m.openParts(v.sum.Open); open != "" {
		b.WriteString("\n" + m.say.SummaryMdOpen(open) + "\n")
	}
	return b.String()
}

// markdownTask is a line of the export: `title (project, PR #12): what the agent last said`.
func (m *Main) markdownTask(task data.SummaryTask) string {
	var detail []string
	if task.Project != "" {
		detail = append(detail, task.Project)
	}
	if task.PRNumber > 0 {
		detail = append(detail, "PR #"+itoa(task.PRNumber))
	}
	line := task.Title
	if len(detail) > 0 {
		line += m.say.SummaryMdParen(strings.Join(detail, ", "))
	}
	if task.Said != "" {
		line += m.say.SummaryMdSaid(task.Said)
	}
	return line
}

// openParts says what is still open as "3 in progress, 1 blocked", or "" when nothing is.
func (m *Main) openParts(o data.SummaryOpen) string {
	var parts []string
	if o.InProgress > 0 {
		parts = append(parts, m.say.SummaryOpenProgress(o.InProgress))
	}
	if o.InReview > 0 {
		parts = append(parts, m.say.SummaryOpenReview(o.InReview))
	}
	if o.Blocked > 0 {
		parts = append(parts, m.say.SummaryOpenBlocked(o.Blocked))
	}
	return strings.Join(parts, m.say.NotifyCountSeparator())
}
