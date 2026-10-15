package views

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// otherModel is the value of the model pop-up's last item, which turns the row into a text field.
const otherModel = "\x00other"

// assistState is Settings > Assistant: the typed fields, what the Test link last said, and which
// providers' model row is a text field.
type assistState struct {
	fs    fields
	test  testResult
	other map[string]bool
}

// testResult is what the Test link last said, for one provider.
type testResult struct {
	id      string
	running bool
	text    string
	ok      bool
}

// assistantPane is the model the assistant uses: its provider, model and key, whether they work,
// and the base URL under Advanced.
func (s *Settings) assistantPane(c *ui.Context, t *theme.Tokens) {
	if s.app.Models == nil {
		group(c, t, "assistant", func() { settingsRow(c, t, "none", s.say.SettingsAssistantNone(), "", true, nil) })
		return
	}
	if s.assist.fs == nil {
		s.assist.fs, s.assist.other = fields{}, map[string]bool{}
	}
	ps := s.app.Models.Providers()
	p := currentProvider(ps)
	group(c, t, "assistant", func() {
		settingsRow(c, t, "provider", s.say.SettingsProvider(), s.providerSub(p), true, func() { s.providerControl(c, t, ps, p) })
		settingsRow(c, t, "model", s.say.ModelsModel(), "", false, func() { s.modelControl(c, t, p) })
		if p.HasKeyField {
			settingsRow(c, t, "key", s.say.ModelsAPIKey(), s.say.ModelsKeyHelp(), false, func() { s.keyControl(c, t, p) })
		}
		s.statusRow(c, t, p)
	})
	groupLabel(c, t, s.say.SettingsAdvanced())
	group(c, t, "advanced", func() {
		settingsRow(c, t, "url", s.say.ModelsBaseURL(), s.say.ModelsBaseURLHelp(), true, func() {
			fieldWell(c, t, "url", wellWidth, func() {
				if v, ok := textField(c, t, s.assist.fs, fieldSpec{
					name: p.ID + "/url", label: s.say.ModelsBaseURL(), value: p.BaseURL, style: theme.Body, color: t.Text,
					placeholder: p.DefaultBaseURL, escReverts: true,
				}); ok {
					s.save(p, "url", v)
				}
			})
		})
		if p.PickProtocol {
			settingsRow(c, t, "protocol", s.say.ModelsProtocol(), "", false, func() { s.protocolControl(c, t, p) })
		}
	})
}

// currentProvider is the active provider, or the first when none is: saving its key makes it the
// active one.
func currentProvider(ps []data.Provider) data.Provider {
	for _, p := range ps {
		if p.Active {
			return p
		}
	}
	return ps[0]
}

// providerSub is the line under the provider: what it says about itself, or that it needs a key.
func (s *Settings) providerSub(p data.Provider) string {
	note := providerNote(s.say, p)
	if note == "" && p.HasKeyField && !s.app.Models.KeySaved(p.ID) {
		return s.say.ModelsNeedsKey()
	}
	return note
}

func (s *Settings) providerControl(c *ui.Context, t *theme.Tokens, ps []data.Provider, cur data.Provider) {
	opts := make([]popOption, len(ps))
	for i, p := range ps {
		opts[i] = popOption{p.ID, providerName(s.say, p)}
	}
	if id, ok := popUp(c, t, "provider", cur.ID, opts); ok {
		s.assist.test = testResult{}
		s.fail(cur, s.app.Models.SetActive(id))
	}
}

// modelControl is a pop-up of the preset's suggestions and Other…, or the text field Other… turns
// into; a model the suggestions lack, and a preset with none, show the field.
func (s *Settings) modelControl(c *ui.Context, t *theme.Tokens, p data.Provider) {
	if len(p.Suggestions) > 0 && !s.assist.other[p.ID] && slices.Contains(p.Suggestions, p.Model) {
		opts := make([]popOption, 0, len(p.Suggestions)+1)
		for _, m := range p.Suggestions {
			opts = append(opts, popOption{m, m})
		}
		opts = append(opts, popOption{otherModel, s.say.ModelsOther()})
		switch v, ok := popUp(c, t, "model-"+p.ID, p.Model, opts); {
		case !ok:
		case v == otherModel:
			s.assist.other[p.ID] = true
		default:
			s.save(p, "model", v)
		}
		return
	}
	fieldWell(c, t, "model", wellWidth, func() {
		if v, ok := textField(c, t, s.assist.fs, fieldSpec{
			name: p.ID + "/model", label: s.say.ModelsModel(), value: p.Model, style: theme.Body, color: t.Text,
			escReverts: true, focus: s.assist.other[p.ID],
		}); ok {
			s.save(p, "model", v)
		}
	})
}

// keyControl is the secure field; once a key is stored it shows dots and a check, never the key.
func (s *Settings) keyControl(c *ui.Context, t *theme.Tokens, p data.Provider) {
	saved := s.app.Models.KeySaved(p.ID)
	placeholder := s.say.ModelsKeyPlaceholder()
	if saved {
		placeholder = "••••••••••••••••••••"
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		fieldWell(c, t, "key", wellWidth, func() {
			if v, ok := textField(c, t, s.assist.fs, fieldSpec{
				name: p.ID + "/key", label: s.say.ModelsAPIKey(), style: theme.Body, color: t.Text, placeholder: placeholder, secret: true,
			}); ok {
				s.save(p, "key", v)
			}
		})
		if saved {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.Status(data.Done)).Shrink(0)
		}
	})
}

func (s *Settings) protocolControl(c *ui.Context, t *theme.Tokens, p data.Provider) {
	sel := 0
	if p.Anthropic {
		sel = 1
	}
	if i := segmented(c, t, "protocol", []string{s.say.ModelsProtocolOpenAI(), s.say.ModelsProtocolAnthropic()}, sel); i >= 0 {
		e := s.edit(p)
		e.Anthropic = i == 1
		s.fail(p, s.app.Models.Save(p.ID, e))
	}
}

// edit is the provider as it is stored, to which one change is made.
func (s *Settings) edit(p data.Provider) data.ProviderEdit {
	return data.ProviderEdit{BaseURL: p.BaseURL, Model: p.Model, Anthropic: p.Anthropic}
}

// save stores one field the user finished editing. A key goes to the keychain, makes the provider
// the one in use and is tested at once.
func (s *Settings) save(p data.Provider, field, v string) {
	e, v := s.edit(p), strings.TrimSpace(v)
	switch field {
	case "url":
		e.BaseURL = v
	case "model":
		e.Model = v
		s.assist.other[p.ID] = false
	case "key":
		if v == "" {
			return
		}
		e.Key = v
	}
	if s.fail(p, s.app.Models.Save(p.ID, e)) != nil {
		return
	}
	if field == "key" {
		if !p.Active {
			s.fail(p, s.app.Models.SetActive(p.ID))
		}
		s.runTest(p)
	}
}

// fail shows an error from the settings store in the status row, where it is seen.
func (s *Settings) fail(p data.Provider, err error) error {
	if err != nil {
		s.assist.test = testResult{id: p.ID, text: problemText(s.say, err)}
	}
	return err
}

// statusRow says whether the connection works, and offers to test it.
func (s *Settings) statusRow(c *ui.Context, t *theme.Tokens, p data.Provider) {
	res := s.assist.test
	if res.id != p.ID {
		res = testResult{}
	}
	text, dot := s.say.ModelsStatusNone(), t.TextTertiary
	switch {
	case res.running:
		text = s.say.ModelsTesting()
	case res.ok:
		text, dot = res.text, t.Status(data.Done)
	case res.text != "":
		text = res.text
	}
	ui.Box(c.Key("rule-status")).Height(1).Shrink(0).Margin(0, 14).Background(t.Border)
	ui.Row(c.Key("row-status")).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(8, 14).Gap(8).AlignItems(ui.Center).Children(func() {
		statusDot(c, dot)
		ink := t.TextSecondary
		if res.ok {
			ink = t.Text
		}
		theme.Paragraph(c, theme.Body, ink, text, 0).Grow(1).MinWidth(0)
		if !res.running {
			link := s.say.ModelsTest()
			if res.ok || res.text != "" {
				link = s.say.ModelsTestAgain()
			}
			if settingsLink(c, t, "test", link, link).Clicked() {
				s.runTest(p)
			}
		}
	})
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
	return testResult{id: id, ok: true, text: s.say.ModelsWorks(itoa(int(d.Milliseconds())))}
}
