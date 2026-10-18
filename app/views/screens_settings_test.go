package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The Settings window, a pane at a time, in light and dark, English and Chinese (-zh). Every pane
// is the same size: 680 x 520. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSettings

// settingsShot renders the window at 2x, at its fixed size, on a pane.
func settingsShot(t *testing.T, name string, app *App, dark, zh bool, pane Pane, setup func(r *settingsRig)) {
	t.Helper()
	s := NewSettings(app)
	w, h := s.ContentSize()
	tt := ui.NewTester(s.View, int(w), int(h))
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: SettingsTitleBarHeight, Left: 78})
	still(tt)
	if zh {
		inChinese(tt)
	}
	s.Show(pane)
	r := &settingsRig{tt: tt, s: s, app: app}
	r.fm, _ = app.Models.(*data.FakeModels)
	r.rec, _ = app.Act.(*recorder)
	tt.Frame()
	tt.Frame()
	if setup != nil {
		setup(r)
		tt.Frame()
	}
	settle(tt)
	shot(t, name, tt.Image(), true, dark)
}

// settingsApp is the app with DeepSeek in use, Anthropic and Ollama also set up, and a few agents
// that reported.
func settingsApp(t *testing.T, st data.GitHubStatus) *App {
	t.Helper()
	app, store, _ := newTestApp(t)
	mac := newAgentRig(t)
	app.Config.AgentSetup = mac.env
	mac.has("claude", "codex", "opencode", "code")
	mac.write(".claude.json", `{"mcpServers":{"adhd-todo":{"command":"x"}}}`)
	mac.write(".codex/config.toml", "[mcp_servers.adhd-todo]\ncommand = \"x\"\n")
	mac.write(".cursor/mcp.json", "{}")
	// DeepSeek Harness is here by its config folder alone, as in the v14 frame.
	mac.write(".dsh/cordis.patch.yml", "[]\n")
	app.Models = newFakeModels(t)
	app.GitHub = data.NewFakeGitHub(st)
	mustNil(t, app.Models.Save("anthropic", data.ProviderEdit{Model: "claude-haiku-5-5", Key: "sk-ant-demo"}))
	mustNil(t, app.Models.Save("ollama", data.ProviderEdit{Model: "qwen3:8b"}))
	mustNil(t, app.Models.Save("deepseek", data.ProviderEdit{Model: "deepseek-chat", Key: "sk-ds-demo"}))
	mustNil(t, app.Models.SetActive("deepseek"))
	now := fixedNow()
	store.ReportAs(1, "codex", "Added the store migration test", now.Add(-6*time.Minute))
	store.ReportAs(1, "codex", "Read the migration", now.Add(-50*time.Minute))
	store.ReportAs(1, "opencode", "Checked the parser", now.Add(-30*time.Hour))
	return app
}

func TestScreensSettings(t *testing.T) {
	screensTest(t)
	ready := data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-2 * time.Minute)}
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh" + sfx
			}
			general := settingsApp(t, ready)
			mustNil(t, general.ArchiveProject(3))
			settingsShot(t, "settings-general"+sfx+".png", general, a.dark, zh, PaneGeneral, nil)

			// macOS refused the last notification: the line warns, a link goes to System Settings.
			refused := settingsApp(t, ready)
			mustNil(t, refused.ArchiveProject(3))
			refused.Act.(*recorder).denied = true
			settingsShot(t, "settings-general-denied"+sfx+".png", refused, a.dark, zh, PaneGeneral, nil)

			// At rest: the provider in use with the last test, the others under it.
			settingsShot(t, "settings-assistant"+sfx+".png", settingsApp(t, ready), a.dark, zh, PaneAssistant, func(r *settingsRig) {
				r.s.runTest(currentProviderOf(r.app))
				r.rec.drain(t)
			})
			// Change: a tile picked, its key empty.
			settingsShot(t, "settings-change"+sfx+".png", settingsApp(t, ready), a.dark, zh, PaneAssistant, func(r *settingsRig) {
				r.s.assist.changing, r.s.assist.pick = true, "kimi"
			})
			// A failed test stays under the form, in plain words.
			settingsShot(t, "settings-change-error"+sfx+".png", settingsApp(t, ready), a.dark, zh, PaneAssistant, func(r *settingsRig) {
				r.s.assist.changing, r.s.assist.pick = true, "openai"
				r.s.assist.err = "The provider didn't accept the API key."
				if zh {
					r.s.assist.err = "服务商不接受这个 API 密钥。"
				}
			})
			// Custom adds the address and the protocol, so it scrolls.
			settingsShot(t, "settings-custom"+sfx+".png", settingsApp(t, ready), a.dark, zh, PaneAssistant, func(r *settingsRig) {
				r.s.assist.changing, r.s.assist.pick = true, "custom"
			})
			// Nothing set up: the grid shows at once.
			fresh := settingsApp(t, ready)
			fresh.Models = newFakeModels(t)
			settingsShot(t, "settings-assistant-new"+sfx+".png", fresh, a.dark, zh, PaneAssistant, nil)

			settingsShot(t, "settings-agents"+sfx+".png", settingsApp(t, ready), a.dark, zh, PaneAgents, nil)
			// Under the agents: where Review and Answer on a handed-over row go.
			settingsShot(t, "settings-agents-open-with"+sfx+".png", settingsApp(t, ready), a.dark, zh, PaneAgents, func(r *settingsRig) {
				r.tt.Scroll(340, 300, 0, 3000)
			})
			for _, s := range githubStates() {
				gh := settingsApp(t, s.st)
				settingsShot(t, "settings-github"+s.name+sfx+".png", gh, a.dark, zh, PaneGitHub, nil)
			}
		}
	}
}

// currentProviderOf is the provider in use.
func currentProviderOf(app *App) data.Provider {
	for _, p := range app.Models.Providers() {
		if p.Active {
			return p
		}
	}
	return data.Provider{}
}
