package views

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The Simplified Chinese screens, in light and dark at 2x, next to the English ones with a -zh
// suffix. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensChinese
//
// The region keeps 12 hours, so times read "下午2:30". The real-shape screen reads a copy of a
// real database named by ADHD_REAL_DB (never the original); it is skipped without it.

// zhShot renders the window at 2x in Simplified Chinese.
func zhShot(t *testing.T, name string, app *App, dark bool, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	inChinese(tt)
	if setup != nil {
		setup(tt, m)
		tt.Frame()
	}
	settle(tt)
	shot(t, name, tt.Image(), false, dark)
}

func TestScreensChineseSeed(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := newTestApp(t)
		zhShot(t, "mine-zh-"+a.suffix+".png", app, a.dark, nil)
		zhShot(t, "project-list-zh-"+a.suffix+".png", app, a.dark, toProject(1))
		zhShot(t, "project-board-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			toProject(1)(tt, m)
			m.board[1] = true
		})
		zhShot(t, "inbox-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenInbox}) })
		zhShot(t, "inspector-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.OpenTask(3) })
		empty, _ := newEmptyApp(t)
		zhShot(t, "mine-empty-zh-"+a.suffix+".png", empty, a.dark, nil)
	}
}

func TestScreensChineseConversation(t *testing.T) {
	for _, a := range appearances() {
		fa := &data.FakeAssistant{Ready: true}
		app, store, rec := assistApp(t, fa)
		fa.Script = func(string) (data.Reply, error) {
			task, err := store.CreateTask(data.CreateTaskInput{Title: "开会", Source: data.SourceUser})
			if err != nil {
				return data.Reply{}, err
			}
			at := fixedNow().Add(18*time.Hour + 30*time.Minute)
			if _, err = store.UpdateTask(task.ID, data.Patch{RemindAt: &at}); err != nil {
				return data.Reply{}, err
			}
			return data.Reply{Text: "好的，明天 9:00 提醒你开会。", Receipts: []data.Receipt{
				{ID: 1, TaskID: task.ID, Title: "开会", Detail: tomorrowReminder(), Status: data.Todo, Bell: true},
			}}, nil
		}
		zhShot(t, "mine-reply-receipt-zh-"+a.suffix+".png", app, a.dark, sendAnd(t, rec, "明天九点提醒我开会", true))

		q := &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Question: &data.Question{
			Text:    "有两个任务匹配“发布说明”，哪一个完成了？",
			Options: []string{"Write the release notes", "Release notes for v0.2"},
		}}, nil)}
		qapp, _, qrec := assistApp(t, q)
		zhShot(t, "mine-question-zh-"+a.suffix+".png", qapp, a.dark, sendAnd(t, qrec, "发布说明写完了", true))
	}
}

func TestScreensChineseRealShape(t *testing.T) {
	path := os.Getenv("ADHD_REAL_DB")
	if path == "" {
		t.Skip("set ADHD_REAL_DB to a copy of a real database")
	}
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	for _, a := range appearances() {
		r, err := data.OpenReal(context.Background(), path, data.RealOptions{Now: func() time.Time { return now }})
		if err != nil {
			t.Fatal(err)
		}
		app := NewApp(r, &recorder{}, testConfig())
		app.Now = func() time.Time { return now }
		zhShot(t, "mine-real-zh-"+a.suffix+".png", app, a.dark, nil)
		ps, _ := r.ListProjects()
		for _, p := range ps {
			if p.RepoSlug != "" {
				zhShot(t, "project-real-zh-"+a.suffix+".png", app, a.dark, toProject(p.ID))
			}
		}
		_ = r.Close()
	}
}
