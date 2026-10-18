package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The Summary page: the day and the week at the last frame of their scene, one frame in the middle
// of a throw, a day with nothing in it, a ledger row opened, a good with its tip and the export
// sheet. Light and dark, English and Chinese (-zh). The Settings rows of the end-of-day reminder are
// in the settings-general screens.
// Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSummary

// summaryShotApp is summaryApp with the paragraphs of today and this week already written, so the
// screens show them at once and need no assistant.
func summaryShotApp(t *testing.T, zh bool) *App {
	t.Helper()
	app, store, _ := summaryApp(t)
	day, week := "Eight finished. You answered the packaging question, replied to the design feedback and booked the dentist; Claude Code merged the native UI and the MCP server, and Codex merged the watcher. The native UI review is still waiting.",
		"Ten finished this week, Wednesday the busiest with eight. Most of it is the native UI: Claude Code merged two pull requests and Codex one. You finished four, rent and the dentist among them. Two tasks are still in progress and one is blocked."
	if zh {
		day = "今天收了 8 件。你回了设计反馈、预约了牙医，答复了包装问题；Claude Code 把原生界面和 MCP 服务送进了主干，Codex 合并了 watcher。原生界面的审查还在等你。"
		week = "这周做完 10 件，周三最忙，一天 8 件。主线是原生界面：Claude Code 合并了 2 个 PR，Codex 合并了 1 个。你这边 4 件，房租和牙医都办了。还有 2 件进行中，1 件卡住。"
	}
	kinds := map[int]string{2: "gem", 20: "gear", 1: "gem", 21: "mail", 7: "mail", 8: "crate", 10: "gear", 9: "jar", 13: "book", 11: "gem", 12: "jar"}
	_, _ = store.PutSummary(data.DayPeriod(app.Now(), 0), data.StoredSummary{Paragraph: day, Model: "fake-model", Kinds: kinds})
	_, _ = store.PutSummary(data.WeekPeriod(app.Now(), 0, time.Sunday), data.StoredSummary{Paragraph: week, Model: "fake-model", Kinds: kinds})
	app.Refresh()
	return app
}

func toSummary(week bool, mod func(tt *ui.Tester, m *Main)) func(*ui.Tester, *Main) {
	return func(tt *ui.Tester, m *Main) {
		m.sum.week = week
		m.goTo(nav{screen: screenSummary})
		tt.Frame()
		if mod != nil {
			mod(tt, m)
		}
	}
}

func TestScreensSummary(t *testing.T) {
	for _, a := range appearances() {
		app := summaryShotApp(t, false)
		mainShot(t, "summary-"+a.suffix+".png", app, a.dark, toSummary(false, nil))
		mainShot(t, "summary-week-"+a.suffix+".png", app, a.dark, toSummary(true, nil))
		mainShot(t, "summary-throw-"+a.suffix+".png", app, a.dark, toSummary(false, func(tt *ui.Tester, m *Main) {
			at := 1900 * time.Millisecond
			m.sum.freeze = &at
		}))
		mainShot(t, "summary-open-"+a.suffix+".png", app, a.dark, toSummary(false, func(tt *ui.Tester, m *Main) {
			v := m.summaryFor(app.Now())
			m.sum.open = map[string]bool{v.key + "/Claude Code": true}
		}))
		mainShot(t, "summary-tip-"+a.suffix+".png", app, a.dark, toSummary(false, func(tt *ui.Tester, m *Main) {
			v := m.summaryFor(app.Now())
			r := m.sum.scene
			top, _ := v.day.HitAt(v.day.Calm(), 133+8, 49+8)
			x := r.X + (float32(theme.SceneOffset(r.W)+top.X)+8)*theme.SceneCell
			y := r.Y + (float32(top.Y)+8)*theme.SceneCell
			tt.Move(x, y)
		}))
		mainShot(t, "summary-export-"+a.suffix+".png", app, a.dark, toSummary(false, func(tt *ui.Tester, m *Main) {
			m.sum.export.open = true
		}))
		empty, _ := newEmptyApp(t)
		empty.Assistant = &data.FakeAssistant{Ready: true}
		mainShot(t, "summary-empty-"+a.suffix+".png", empty, a.dark, toSummary(false, nil))
	}
}

func TestScreensSummaryChinese(t *testing.T) {
	for _, a := range appearances() {
		app := summaryShotApp(t, true)
		zhShot(t, "summary-zh-"+a.suffix+".png", app, a.dark, toSummary(false, nil))
		zhShot(t, "summary-week-zh-"+a.suffix+".png", app, a.dark, toSummary(true, nil))
		zhShot(t, "summary-open-zh-"+a.suffix+".png", app, a.dark, toSummary(false, func(tt *ui.Tester, m *Main) {
			v := m.summaryFor(app.Now())
			m.sum.open = map[string]bool{v.key + "/Claude Code": true}
		}))
		zhShot(t, "summary-export-zh-"+a.suffix+".png", app, a.dark, toSummary(false, func(tt *ui.Tester, m *Main) {
			m.sum.export.open = true
		}))
		empty, _ := newEmptyApp(t)
		empty.Assistant = &data.FakeAssistant{Ready: true}
		zhShot(t, "summary-empty-zh-"+a.suffix+".png", empty, a.dark, toSummary(false, nil))
	}
}
