package views

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// assistState is Settings > Assistant. At rest the pane shows the provider in use and the others
// that are set up; Change opens the grid of providers (settings_providers.go), where one is
// picked, its key and model are typed, and Save and test makes it the one in use.
type assistState struct {
	test     testResult // what the last test said, for one provider
	changing bool       // the grid is open
	pick     string     // the provider picked in the grid
	drafts   map[string]*draft
	lists    map[string]*modelList
	fs       fields // the typed fields' keyboard state
	saving   bool   // Save and test is running
	err      string // why the last Save and test failed, shown under the form
}

// testResult is what the last test said, for one provider.
type testResult struct {
	id      string
	running bool
	text    string
	ok      bool
}

// assistantPane is the provider in use and the others set up, or the grid when none is.
func (s *Settings) assistantPane(c *ui.Context, t *theme.Tokens) {
	if s.app.Models == nil {
		group(c, t, "assistant", func() { settingsRow(c, t, "none", s.say.SettingsAssistantNone(), "", true, nil) })
		return
	}
	if s.assist.drafts == nil {
		s.assist.drafts, s.assist.lists, s.assist.fs = map[string]*draft{}, map[string]*modelList{}, fields{}
	}
	ps := s.app.Models.Providers()
	var inUse *data.Provider
	var others []data.Provider
	for i, p := range ps {
		switch {
		case p.Active:
			inUse = &ps[i]
		case p.Configured:
			others = append(others, p)
		}
	}
	setUp := inUse != nil || len(others) > 0
	if s.assist.changing || !setUp {
		s.changePane(c, t, ps, inUse, setUp)
		return
	}
	if inUse != nil {
		groupLabel(c, t, s.say.SettingsInUse())
		s.inUseCard(c, t, *inUse)
	}
	if len(others) > 0 {
		groupLabel(c, t, s.say.SettingsAlsoSetUp())
		group(c, t, "also", func() {
			for i, p := range others {
				s.alsoRow(c, t, p, i == 0)
			}
		})
	}
	s.memoryGroup(c, t)
}

// memoryGroup is the row for the file where the assistant keeps notes about the user. The file is
// plain Markdown: Open file hands it to the editor macOS has for it, or shows the data folder when
// the assistant has not written anything yet.
func (s *Settings) memoryGroup(c *ui.Context, t *theme.Tokens) {
	path := s.app.Config.AssistantMemory
	if path == "" {
		return
	}
	group(c, t, "memory", func() {
		settingsRow(c, t, "memory", s.say.SettingsAssistantMemory(), s.say.SettingsAssistantMemoryBody(), true, func() {
			label := s.say.SettingsOpenFile()
			if !settingsLink(c, t, "open-memory", label, label).Clicked() {
				return
			}
			if _, err := os.Stat(path); err != nil {
				s.app.Act.ShowInFinder(filepath.Dir(path))
				return
			}
			s.app.Act.OpenFile(path)
		})
	})
}

// inUseCard is the provider the assistant uses: its logo, name, model and how the last test went,
// with Change.
func (s *Settings) inUseCard(c *ui.Context, t *theme.Tokens, p data.Provider) {
	res := s.assist.test
	if res.id != p.ID {
		res = testResult{}
	}
	group(c, t, "inuse", func() {
		ui.Row(c).Shrink(0).Padding(14, 16).Gap(14).AlignItems(ui.Center).Children(func() {
			providerLogo(c, t, p.ID, theme.LogoCard)
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				theme.Text(c, theme.BodyStrong, t.Text, providerName(s.say, p)).Ellipsis("…")
				s.statusLine(c, t, p, res)
				if !res.ok && !res.running && res.text != "" {
					theme.Paragraph(c, theme.Meta, t.Blocked(), res.text, 2)
				}
			})
			if settingsButton(c, t, "change", s.say.SettingsChange(), false).Clicked() {
				s.assist.changing, s.assist.pick = true, p.ID
				s.assist.err = ""
				s.fetchModels(p, "")
			}
		})
	})
}

// statusLine is the model and how the connection to it is: connected with the time a request
// took, being tested, not tested, or failed (the words of the failure are under it).
func (s *Settings) statusLine(c *ui.Context, t *theme.Tokens, p data.Provider, res testResult) {
	ui.Row(c).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
		if p.Model != "" {
			theme.Text(c, theme.Meta, t.TextSecondary, p.Model).Ellipsis("…")
		}
		switch {
		case res.running:
			theme.Text(c, theme.Meta, t.TextTertiary, "· "+s.say.ModelsTesting())
		case res.ok:
			theme.Text(c, theme.Meta, t.TextTertiary, "·")
			statusDot(c, t.Status(data.Done))
			theme.Text(c, theme.Meta, t.Status(data.Done), res.text)
		default:
			if res.text == "" {
				theme.Text(c, theme.Meta, t.TextTertiary, "· "+s.say.ModelsStatusNone())
			}
			link := s.say.ModelsTest()
			if res.text != "" {
				link = s.say.ModelsTestAgain()
			}
			if settingsLink(c, t, "test", link, link).Clicked() {
				s.runTest(p)
			}
		}
	})
}

// alsoRow is a provider that is set up but not in use, with Use.
func (s *Settings) alsoRow(c *ui.Context, t *theme.Tokens, p data.Provider, first bool) {
	sub := p.Model
	if !p.HasKeyField {
		sub += " · " + providerNote(s.say, p)
	}
	settingsRowLead(c, t, "also-"+p.ID, providerName(s.say, p), sub, first, func() {
		use := s.say.SettingsUse()
		if settingsLink(c, t, "use-"+p.ID, use, s.say.A11yUseProvider(providerName(s.say, p))).Clicked() {
			s.assist.test = testResult{}
			if s.app.Models.SetActive(p.ID) == nil {
				s.runTest(p)
			}
		}
	}, func() { providerLogo(c, t, p.ID, theme.LogoRow) })
}

// edit is the provider as it is stored, to which one change is made.
func (s *Settings) edit(p data.Provider) data.ProviderEdit {
	return data.ProviderEdit{BaseURL: p.BaseURL, Model: p.Model, Anthropic: p.Anthropic}
}

// runTest sends a one-token request off the UI thread and shows how it went.
func (s *Settings) runTest(p data.Provider) {
	if s.assist.test.running && s.assist.test.id == p.ID {
		return
	}
	e := s.edit(p)
	s.assist.test = testResult{id: p.ID, running: true}
	models, post := s.app.Models, s.app.Act.Post
	go func() {
		d, err := models.Test(context.Background(), p.ID, e)
		post(func() { s.assist.test = s.outcomeOf(p.ID, d, err) })
	}()
}

func (s *Settings) outcomeOf(id string, d time.Duration, err error) testResult {
	if err != nil {
		return testResult{id: id, text: problemText(s.say, err)}
	}
	return testResult{id: id, ok: true, text: s.say.SettingsConnected(itoa(int(d.Milliseconds())))}
}

// settingsButton is a button of a settings pane: a quiet raised one, or the accent one for the
// action the pane is for.
func settingsButton(c *ui.Context, t *theme.Tokens, key, text string, primary bool) ui.Element {
	btn := ui.ButtonBase(c.Key(key)).Shrink(0).Padding(5, 13).Radius(7).Label(text).Cursor(ui.CursorPointer).Center()
	ink := t.Text
	switch {
	case primary:
		btn.Background(t.Accent)
		if btn.Hovered() {
			btn.Background(t.Accent.Mix(t.Text, 0.12))
		}
		ink = t.AccentText
	default:
		btn.Background(t.SettingsGroup()).Border(1, t.ComposerBorder()).Shadow(0, 1, 1, 0, t.Shadow.Alpha(0.06))
		if btn.Hovered() {
			btn.Background(t.SettingsGroup().Mix(t.Text, 0.06))
		}
	}
	btn.Children(func() { theme.Text(c, theme.Body, ink, text) })
	return btn
}
