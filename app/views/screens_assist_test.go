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

// TestScreensAssistThinking renders the working state (v9 B): the second row with the stop button and
// the working line under it, 3 s after the send and past 15 s, in English and Chinese.
func TestScreensAssistThinking(t *testing.T) {
	for _, a := range appearances() {
		for _, w := range []struct {
			name string
			wait time.Duration
			zh   bool
		}{
			{"mine-thinking", 3 * time.Second, false},
			{"mine-thinking-slow", 16 * time.Second, false},
			{"mine-thinking-zh", 3 * time.Second, true},
		} {
			fa := &data.FakeAssistant{Ready: true, Block: make(chan struct{})}
			app, _, rec := readyApp(t, fa)
			now := fixedNow()
			app.Now = func() time.Time { return now }
			send := sendAnd(t, rec, "remind me about the team meeting tomorrow at 9", false)
			setup := func(tt *ui.Tester, m *Main) {
				send(tt, m)
				now = now.Add(w.wait)
				tt.Frame()
			}
			if w.zh {
				zhShot(t, w.name+"-"+a.suffix+".png", app, a.dark, setup)
				continue
			}
			mainShot(t, w.name+"-"+a.suffix+".png", app, a.dark, setup)
		}
	}
}

func TestScreensAssistReplyAndReceipt(t *testing.T) {
	for _, a := range appearances() {
		fa := &data.FakeAssistant{Ready: true}
		app, store, rec := assistApp(t, fa)
		fa.Script = func(string) (data.Reply, error) {
			at := fixedNow().Add(18*time.Hour + 30*time.Minute) // tomorrow 9:00
			task, err := store.CreateTask(data.CreateTaskInput{Title: "Team meeting", Source: data.SourceUser})
			if err != nil {
				return data.Reply{}, err
			}
			if _, err = store.UpdateTask(task.ID, data.Patch{RemindAt: &at}); err != nil {
				return data.Reply{}, err
			}
			return data.Reply{Text: "Okay, I'll remind you tomorrow at 9.", Receipts: []data.Receipt{
				{ID: 1, TaskID: task.ID, Title: "Team meeting", Detail: tomorrowReminder(), Status: data.Todo, Bell: true},
			}}, nil
		}
		mainShot(t, "mine-reply-receipt-"+a.suffix+".png", app, a.dark, sendAnd(t, rec, "remind me about the team meeting tomorrow at 9", true))
	}
}

// TestScreensAssistSentenceAndSeveral renders a turn that changed nothing, and one that changed
// four tasks (three lines and "and 1 more").
func TestScreensAssistSentenceAndSeveral(t *testing.T) {
	for _, a := range appearances() {
		sentence := &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Text: "That one is already in adhd-todo, so I didn't add it again."}, nil)}
		app, _, rec := assistApp(t, sentence)
		mainShot(t, "mine-reply-sentence-"+a.suffix+".png", app, a.dark, sendAnd(t, rec, "write the release notes", true))

		fa := &data.FakeAssistant{Ready: true}
		several, store, rec2 := assistApp(t, fa)
		fa.Script = func(string) (data.Reply, error) {
			var reply data.Reply
			for i, title := range []string{"Buy milk", "Call the dentist", "Water the plants", "Send the invoice to Anna"} {
				task, err := store.CreateTask(data.CreateTaskInput{Title: title, Source: data.SourceUser})
				if err != nil {
					return data.Reply{}, err
				}
				detail := data.Detail{Kind: data.DetailAdded}
				if i == 1 {
					due := fixedNow().Add(18*time.Hour + 30*time.Minute)
					_, _ = store.UpdateTask(task.ID, data.Patch{DueAt: &due})
					detail = data.Detail{Kind: data.DetailDue, At: due}
				}
				reply.Receipts = append(reply.Receipts, data.Receipt{ID: i + 1, TaskID: task.ID, Title: title, Status: data.Todo, Detail: detail})
			}
			return reply, nil
		}
		mainShot(t, "mine-reply-receipts-"+a.suffix+".png", several, a.dark, sendAnd(t, rec2, "buy milk, call the dentist tomorrow, water the plants, send the invoice to Anna", true))
	}
}

func TestScreensAssistQuestion(t *testing.T) {
	for _, a := range appearances() {
		fa := &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Question: &data.Question{
			Text:    "Two tasks match “release notes”. Which one is done?",
			Options: []string{"Write the release notes", "Release notes for v0.2"},
		}}, nil)}
		app, _, rec := assistApp(t, fa)
		mainShot(t, "mine-question-"+a.suffix+".png", app, a.dark, sendAnd(t, rec, "release notes are done", true))
	}
}

func TestScreensAssistNoModel(t *testing.T) {
	for _, a := range appearances() {
		app, _, _ := assistApp(t, &data.FakeAssistant{Ready: false})
		mainShot(t, "mine-no-model-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { tt.Key(ui.Cmd, ui.KeyK) })
	}
}

func mustNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
