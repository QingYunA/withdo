package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The M3 screens: the ask field with the built-in assistant, and Settings > Models. Rendered with
// the fake store and a scripted assistant, at 2x in light and dark:
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensAssist

// assistApp is the seeded fake with a scripted assistant and fake model settings.
func assistApp(t *testing.T, fa *data.FakeAssistant) (*App, *data.Fake, *recorder) {
	t.Helper()
	app, store, rec := newTestApp(t)
	app.Assistant, app.Models = fa, newFakeModels(t)
	return app, store, rec
}

// sendAnd types a request, sends it and, when answer is set, lets the assistant's reply arrive.
func sendAnd(t *testing.T, rec *recorder, text string, answer bool) func(*ui.Tester, *Main) {
	return func(tt *ui.Tester, m *Main) {
		tt.Key(ui.Cmd, ui.KeyK)
		tt.Type(text)
		tt.Key(0, ui.KeyEnter)
		if answer {
			rec.drain(t)
		}
	}
}

func TestScreensAssistThinking(t *testing.T) {
	for _, a := range appearances() {
		fa := &data.FakeAssistant{Ready: true, Block: make(chan struct{})}
		app, _, rec := assistApp(t, fa)
		mainShot(t, "now-thinking-"+a.suffix+".png", app, a.dark, sendAnd(t, rec, "明天九点提醒我开会", false))
	}
}

func TestScreensAssistReplyAndReceipt(t *testing.T) {
	for _, a := range appearances() {
		fa := &data.FakeAssistant{Ready: true}
		app, store, rec := assistApp(t, fa)
		fa.Script = func(string) (data.Reply, error) {
			at := fixedNow().Add(18*time.Hour + 30*time.Minute) // tomorrow 9:00
			task, err := store.CreateTask(data.CreateTaskInput{Title: "开会", Source: data.SourceUser})
			if err != nil {
				return data.Reply{}, err
			}
			if _, err = store.UpdateTask(task.ID, data.Patch{RemindAt: &at}); err != nil {
				return data.Reply{}, err
			}
			return data.Reply{Text: "好的，明天 9:00 提醒你开会。", Receipts: []data.Receipt{
				{ID: 1, TaskID: task.ID, Title: "开会", Detail: "Tomorrow 9:00 AM · reminder", Status: data.Todo, Bell: true},
			}}, nil
		}
		mainShot(t, "now-reply-receipt-"+a.suffix+".png", app, a.dark, sendAnd(t, rec, "明天九点提醒我开会", true))
	}
}

func TestScreensAssistQuestion(t *testing.T) {
	for _, a := range appearances() {
		fa := &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Question: &data.Question{
			Text:    "Two tasks match “release notes”. Which one is done?",
			Options: []string{"Write the release notes", "Release notes for v0.2"},
		}}, nil)}
		app, _, rec := assistApp(t, fa)
		mainShot(t, "now-question-"+a.suffix+".png", app, a.dark, sendAnd(t, rec, "release notes are done", true))
	}
}

func TestScreensAssistNoModel(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := assistApp(t, &data.FakeAssistant{Ready: false})
		mainShot(t, "now-no-model-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { tt.Key(ui.Cmd, ui.KeyK) })
	}
}

func TestScreensSettingsModels(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := assistApp(t, &data.FakeAssistant{Ready: true})
		mustNil(t, app.Models.Save("anthropic", data.ProviderEdit{Model: "claude-haiku-5-5", Key: "sk-ant-demo"}))
		mustNil(t, app.Models.SetActive("anthropic"))
		mainShot(t, "settings-models-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.openModels() })
	}
}

func TestScreensSettingsModelsEdit(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := assistApp(t, &data.FakeAssistant{Ready: true})
		mustNil(t, app.Models.Save("anthropic", data.ProviderEdit{Model: "claude-haiku-5-5", Key: "sk-ant-demo"}))
		mustNil(t, app.Models.SetActive("anthropic"))
		mainShot(t, "settings-models-edit-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			m.openModels()
			tt.Frame()
			mustNil(t, tt.Click("OpenAI"))
			tt.Frame()
			mustNil(t, tt.Click("Paste your API key"))
			tt.Type("sk-proj-demo")
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			mustNil(t, tt.Click("Test"))
			app.Act.(*recorder).drain(t)
		})
	}
}

func mustNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
