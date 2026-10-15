package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The composer: empty, typed, with a reply, on Now, in a project and in the inspector, in light and
// dark, English and Chinese (-zh). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensComposer

// readyApp is the seeded fake with a scripted assistant and Anthropic's Haiku connected.
func readyApp(t *testing.T, fa *data.FakeAssistant) (*App, *data.Fake, *recorder) {
	t.Helper()
	app, store, rec := assistApp(t, fa)
	mustNil(t, app.Models.Save("anthropic", data.ProviderEdit{Model: "claude-haiku-5-5", Key: "sk-ant-demo"}))
	mustNil(t, app.Models.SetActive("anthropic"))
	return app, store, rec
}

// fridayP1 is the reply of "fix the login redirect loop, p1, friday": a task in the project with
// what the words made of it.
func fridayP1(store *data.Fake, project int, title, say string) func(string) (data.Reply, error) {
	return func(string) (data.Reply, error) {
		due := fixedNow().Add(2*24*time.Hour + 3*time.Hour + 30*time.Minute) // Friday evening
		task, err := store.CreateTask(data.CreateTaskInput{Title: title, Source: data.SourceUser,
			ProjectID: data.IntPtr(project), Priority: data.P1})
		if err != nil {
			return data.Reply{}, err
		}
		if _, err = store.UpdateTask(task.ID, data.Patch{DueAt: &due}); err != nil {
			return data.Reply{}, err
		}
		return data.Reply{Text: say, Receipts: []data.Receipt{{
			ID: 1, TaskID: task.ID, Title: title, Status: data.Todo,
			Detail: data.Detail{Kind: data.DetailDue, At: due, Priority: "P1", Project: "adhd-todo"},
		}}}, nil
	}
}

// remindsTomorrow is the reply of "remind me tomorrow morning": the task's reminder really moves to
// tomorrow at 9:00, and the receipt says so.
func remindsTomorrow(store *data.Fake, id int, say string) func(string) (data.Reply, error) {
	return func(string) (data.Reply, error) {
		at := fixedNow().Add(18*time.Hour + 30*time.Minute)
		task, err := store.UpdateTask(id, data.Patch{RemindAt: &at})
		if err != nil {
			return data.Reply{}, err
		}
		return data.Reply{Text: say, Receipts: []data.Receipt{{
			ID: 1, TaskID: id, Title: task.Title, Status: task.Status, Bell: true,
			Detail: data.Detail{Kind: data.DetailUpdated, Changes: []data.Change{{Kind: data.ChangeReminder, At: at}}},
		}}}, nil
	}
}

func TestScreensComposer(t *testing.T) {
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix + ".png"
			if zh {
				sfx = "-zh" + sfx
			}
			shotOf := func(name string, app *App, setup func(tt *ui.Tester, m *Main)) {
				if zh {
					zhShot(t, name+sfx, app, a.dark, setup)
					return
				}
				mainShot(t, name+sfx, app, a.dark, setup)
			}
			text, project, saidAdded := "review PR 12 before standup, it's urgent", "fix the login redirect loop, p1, friday", "Added it to adhd-todo, due Friday, P1."
			made := "Fix the login redirect loop"
			if zh {
				text, project, saidAdded = "站会前审查 PR 12，很急", "修复登录重定向死循环，p1，周五", "已添加到 adhd-todo，周五截止，P1。"
				made = "修复登录重定向死循环"
			}

			empty, _, _ := readyApp(t, &data.FakeAssistant{Ready: true})
			shotOf("mine-composer-empty", empty, nil)
			typed, _, _ := readyApp(t, &data.FakeAssistant{Ready: true})
			shotOf("mine-composer-typed", typed, func(tt *ui.Tester, m *Main) {
				tt.Key(ui.Cmd, ui.KeyK)
				tt.Type(text)
			})
			long, _, _ := readyApp(t, &data.FakeAssistant{Ready: true})
			shotOf("mine-composer-long", long, func(tt *ui.Tester, m *Main) {
				tt.Key(ui.Cmd, ui.KeyK)
				for i, line := range []string{text, "ask Anna about the invoice", "book the dentist for next week", "check the build on main"} {
					if i > 0 {
						tt.Key(ui.Shift, ui.KeyEnter)
					}
					tt.Type(line)
				}
			})
			off, _, _ := assistApp(t, &data.FakeAssistant{Ready: false})
			shotOf("mine-composer-no-model", off, func(tt *ui.Tester, m *Main) { tt.Key(ui.Cmd, ui.KeyK) }) // focused: the one extra line shows

			fa := &data.FakeAssistant{Ready: true}
			app, store, rec := readyApp(t, fa)
			fa.Script = fridayP1(store, 1, made, saidAdded)
			shotOf("project-composer", app, func(tt *ui.Tester, m *Main) { // one line says what the words made
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				tt.Key(ui.Cmd, ui.KeyK)
				tt.Type(project)
				tt.Key(0, ui.KeyEnter)
				rec.drain(t)
			})
			typedInProject, _, _ := readyApp(t, &data.FakeAssistant{Ready: true})
			shotOf("project-composer-typed", typedInProject, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				tt.Key(ui.Cmd, ui.KeyK)
				tt.Type(project)
			})
			inbox, _, _ := readyApp(t, &data.FakeAssistant{Ready: true})
			shotOf("inbox-composer", inbox, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenInbox}) })
			board, _, _ := readyApp(t, &data.FakeAssistant{Ready: true})
			shotOf("project-board-composer", board, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				m.board[1] = true
			})

			ask, movedIt := "remind me tomorrow morning", "Moved it to tomorrow at 9:00 AM."
			if zh {
				ask, movedIt = "明天早上提醒我", "已改为明天上午 9:00 提醒。"
			}
			ifa := &data.FakeAssistant{Ready: true}
			iapp, istore, irec := readyApp(t, ifa)
			ifa.Script = remindsTomorrow(istore, 7, movedIt)
			shotOf("inspector-composer", iapp, func(tt *ui.Tester, m *Main) { m.OpenTask(7) })
			shotOf("inspector-composer-reply", iapp, func(tt *ui.Tester, m *Main) {
				m.OpenTask(7)
				m.focusInspectorComposer = true
				tt.Frame()
				tt.Frame()
				tt.Frame()
				tt.Type(ask)
				tt.Key(0, ui.KeyEnter)
				irec.drain(t)
			})
		}
	}
}
