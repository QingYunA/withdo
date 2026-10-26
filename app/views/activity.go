package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// activity is the report timeline, newest first: the agent's avatar on a thin line, who and when, what they
// did, and what comes next.
func (m *Main) activity(c *ui.Context, t *theme.Tokens, task data.Task) {
	reports := m.app.snap.details[task.ID].Reports
	ui.Column(c).Gap(8).Children(func() {
		theme.Text(c, theme.Section, t.TextSecondary, m.say.ActivityTitle())
		if len(reports) == 0 {
			theme.Paragraph(c, theme.Body, t.TextSecondary, m.say.ActivityEmpty(), 0)
			return
		}
		for i, r := range reports {
			m.reportRow(c, t, r, i == len(reports)-1, i == 0)
		}
	})
}

func (m *Main) reportRow(c *ui.Context, t *theme.Tokens, r data.Report, last, first bool) {
	you := r.Kind == data.ReportReply && (r.Agent == data.You || r.Agent == "")
	ui.Row(c.Key(r.ID)).AlignItems(ui.Stretch).Gap(10).Children(func() {
		ui.Column(c).Width(theme.AvatarRow).Shrink(0).AlignItems(ui.Center).Children(func() {
			m.reportAvatar(c, t, r)
			if !last {
				ui.Box(c).Width(theme.Hairline).Grow(1).Margin(4, 0, 0).Background(t.Border)
			}
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Padding(0, 0, 16).Children(func() {
			ui.Row(c).Gap(6).Children(func() {
				who := agentName(r.Agent)
				if you {
					who = m.say.DoneByYou()
				}
				name := theme.Text(c, theme.Meta, t.TextSecondary, who)
				if first && r.Status == data.Blocked {
					name.TextColor(t.Blocked()) // the one report that needs attention
				}
				theme.Text(c, theme.Meta, t.TextTertiary, m.clock.Age(r.CreatedAt, m.app.Now()))
			})
			theme.Paragraph(c, theme.Body, t.Text, r.Summary, 0)
			if r.NextStep != "" {
				theme.Paragraph(c, theme.Body, t.TextSecondary, m.say.ActivityNext(r.NextStep), 0)
			}
			if len(r.Links) > 0 {
				m.linkChips(c, t, r)
			}
		})
	})
}

// reportAvatar starts a report: the avatar of the agent that wrote it, or GitHub's mark for the
// events the poller files.
func (m *Main) reportAvatar(c *ui.Context, t *theme.Tokens, r data.Report) {
	if r.Kind == data.ReportReply && (r.Agent == data.You || r.Agent == "") {
		theme.AvatarIcon(c, theme.You, theme.AvatarRow).Shrink(0)
		return
	}
	if r.Agent == data.GitHubAgent {
		ui.Box(c).Size(theme.AvatarRow, theme.AvatarRow).Shrink(0).Center().Children(func() {
			ui.Icon(c, theme.IconOctocat).Size(theme.AvatarRow, theme.AvatarRow).TextColor(t.TextSecondary)
		})
		return
	}
	theme.AvatarIcon(c, theme.AgentAvatar(r.Agent), theme.AvatarRow).Shrink(0)
}
