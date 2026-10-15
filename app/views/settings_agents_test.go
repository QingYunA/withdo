package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

func TestAgentsPaneListsTheAgentsThatReportedWithTodayAndTheLastReport(t *testing.T) {
	r := newSettingsRig(t)
	store := r.app.Store.(*data.Fake)
	now := fixedNow()
	store.ReportAs(1, "codex", "one", now.Add(-6*time.Minute))
	store.ReportAs(1, "codex", "two", now.Add(-50*time.Minute))
	store.ReportAs(1, "opencode", "yesterday", now.Add(-30*time.Hour))
	r.app.Refresh()
	r.pane(t, "Agents")
	hasFold(t, r.tt, "Agents that reported")
	has(t, r.tt, "Claude Code", "Codex", "2 reports today", "6m", "opencode", "No reports today", "1d",
		"Connect Claude Code", r.app.Config.ConnectCommand)
	hasNot(t, r.tt, "github") // the poller is not an agent
	claude, _ := r.tt.Find("Claude Code")
	codex, _ := r.tt.Find("Codex")
	opencode, _ := r.tt.Find("opencode")
	if !(claude.Y < codex.Y && codex.Y < opencode.Y) {
		t.Errorf("not the latest first: claude %v, codex %v, opencode %v", claude.Y, codex.Y, opencode.Y)
	}
}

func TestAgentsPaneCountsOneReportInTheSingular(t *testing.T) {
	r := newSettingsRig(t)
	store := r.app.Store.(*data.Fake)
	store.ReportAs(1, "cursor-agent", "one", fixedNow().Add(-time.Minute))
	r.app.Refresh()
	r.pane(t, "Agents")
	has(t, r.tt, "Cursor", "1 report today", "1m")
}

func TestAgentsPaneInChinese(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.app.Store.(*data.Fake).ReportAs(1, "codex", "one", fixedNow().Add(-6*time.Minute))
	r.app.Refresh()
	r.s.Show(PaneAgents)
	r.tt.Frame()
	has(t, r.tt, "Codex", "今天汇报 1 次", "6分钟前")
	hasFold(t, r.tt, "接入的 Agent")
}
