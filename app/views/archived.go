package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// archivedRows is how many archived projects Settings shows before the list scrolls.
const archivedRows = 5

// archivedBlock lists the archived projects with a Restore button each.
func (m *Main) archivedBlock(c *ui.Context, t *theme.Tokens) {
	ui.Column(c).Gap(4).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, m.say.ManageArchivedTitle())
		archived := m.app.snap.archived
		if len(archived) == 0 {
			theme.Paragraph(c, theme.Meta, t.TextSecondary, m.say.ManageArchivedNone(), 0)
			return
		}
		scroll := ui.Scroll(c.Key("archived")).MaxHeight(archivedRows * theme.RowOneLine)
		scroll.Children(func() {
			for _, p := range archived {
				ui.Row(c.Key(p.ID)).Height(theme.RowOneLine).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
					theme.Text(c, theme.Body, t.Text, p.Name).Shrink(1).MinWidth(0).Ellipsis("…")
					if p.RepoSlug != "" {
						theme.Text(c, theme.Meta, t.TextTertiary, p.RepoSlug).Shrink(1).MinWidth(0).Ellipsis("…")
					}
					ui.Spacer(c)
					if actionButton(c, t, p.ID, m.say.ManageRestore(), m.say.ManageA11yRestore(p.Name)).Clicked() {
						_ = m.app.RestoreProject(p.ID)
					}
				})
			}
		})
	})
}
