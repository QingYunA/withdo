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

// assistState is Settings > Model. The pane shows the provider in use and, under it, the grid of
// every provider (settings_providers.go): a set-up one becomes the one in use and is tested, another
// opens its key and model form under the grid.
type assistState struct {
	test   testResult // what the last test said, for one provider
	pick   string     // the provider whose form is open under the grid
	drafts map[string]*draft
	lists  map[string]*modelList
	fs     fields // the typed fields' keyboard state
	saving bool   // Save and test is running
	err    string // why the last Save and test failed, shown under the form
}

// testResult is what the last test said, for one provider.
type testResult struct {
	id      string
	running bool
	text    string
	ok      bool
}

// modelPane is the user's own model: the provider in use, then the grid to switch, and the form of
// a provider that is not set up yet.
func (s *Settings) modelPane(c *ui.Context, t *theme.Tokens) {
	paneHeader(c, t, s.say.SettingsPaneModel(), s.say.SettingsPaneModelLine())
	s.assistantPane(c, t)
}

// assistantPane is the provider in use (or one line saying there is none), the grid of every
// provider, and the form of the one picked from it.
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
	for i := range ps {
		if ps[i].Active {
			inUse = &ps[i]
			break
		}
	}
	if inUse != nil {
		s.inUseCard(c, t, *inUse)
	} else {
		theme.Paragraph(c, theme.Body, t.TextSecondary, s.say.SettingsModelNone(), 0).Margin(0, 4, 16)
	}
	groupLabel(c, t, s.say.SettingsSwitchTo())
	s.providerGrid(c, t, ps)
	for _, p := range ps {
		if p.ID == s.assist.pick {
			s.providerForm(c, t, p)
			break
		}
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
// with Test.
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
			if settingsButton(c, t, "test", s.say.ModelsTest(), false).Clicked() {
				s.runTest(p)
			}
		})
	})
}

// statusLine is the model and how the connection to it is: connected with the time a request
// took, being tested, or not tested (the Test button on the card, and the words of a failure under
// the line).
func (s *Settings) statusLine(c *ui.Context, t *theme.Tokens, p data.Provider, res testResult) {
	ui.Row(c).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
		if p.Model != "" {
			theme.Text(c, theme.Meta, t.TextSecondary, p.Model).Ellipsis("…")
		}
		switch {
		case res.running:
			theme.Text(c, theme.Meta, t.TextSecondary, "· "+s.say.ModelsTesting())
		case res.ok:
			theme.Text(c, theme.Meta, t.TextSecondary, "·")
			statusDot(c, t.Status(data.Done))
			theme.Text(c, theme.Meta, t.Status(data.Done), res.text)
		default:
			theme.Text(c, theme.Meta, t.TextSecondary, "· "+s.say.ModelsStatusNone())
		}
	})
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
		btn.Background(t.SettingsGroup()).Border(theme.Hairline, t.ComposerBorder()).Shadow(0, 1, 1, 0, t.Shadow.Alpha(0.06))
		if btn.Hovered() {
			btn.Background(t.SettingsGroup().Mix(t.Text, 0.06))
		}
	}
	btn.Children(func() { theme.Text(c, theme.Body, ink, text) })
	return btn
}
