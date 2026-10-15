package views

import (
	"context"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

type settingsTab int

const (
	tabGeneral settingsTab = iota
	tabModels
)

const (
	providerListWidth float32 = 188
	inputHeight       float32 = 30
)

// modelsState is Settings > Models: which provider is open, its typed fields, and the Test result.
type modelsState struct {
	tab      settingsTab
	selected string // the provider whose fields are open; the active one when empty
	fs       fields
	test     testResult
}

// testResult is what the Test button last said, for one provider.
type testResult struct {
	id      string
	running bool
	text    string
	ok      bool
}

// modelsTab is the provider list at the left and the open provider's fields at the right.
func (m *Main) modelsTab(c *ui.Context, t *theme.Tokens) {
	ps := m.app.Models.Providers()
	cur := m.openProvider(ps)
	ui.Row(c).Gap(24).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Width(providerListWidth).Shrink(0).Gap(2).Children(func() {
			for _, p := range ps {
				m.providerRow(c, t, p, p.ID == cur.ID)
			}
		})
		m.providerFields(c, t, cur)
	})
}

// openProvider is the provider whose fields show: the one picked, else the active one, else the first.
func (m *Main) openProvider(ps []data.Provider) data.Provider {
	for _, p := range ps {
		if p.ID == m.models.selected {
			return p
		}
	}
	for _, p := range ps {
		if p.Active {
			return p
		}
	}
	return ps[0]
}

func (m *Main) providerRow(c *ui.Context, t *theme.Tokens, p data.Provider, open bool) {
	row := ui.Row(c.Key("provider-"+p.ID)).Shrink(0).Height(32).Padding(0, 8).Gap(10).AlignItems(ui.Center).
		Radius(theme.RadiusCard).Cursor(ui.CursorPointer)
	switch {
	case open:
		row.Background(t.BgSelected)
	case row.Hovered():
		row.Background(t.BgHover)
	}
	row.Children(func() {
		radio(c, t, p.Active)
		theme.Text(c, theme.Body, t.Text, providerName(m.say, p)).Shrink(1).MinWidth(0).Ellipsis("…")
	})
	if row.Clicked() {
		m.selectProvider(p.ID)
	}
}

func (m *Main) selectProvider(id string) {
	if m.models.selected != id {
		m.models.selected, m.models.test = id, testResult{}
	}
}

// providerFields is the open provider: what it says about itself, Base URL, Model, key, Test.
func (m *Main) providerFields(c *ui.Context, t *theme.Tokens, p data.Provider) {
	if m.models.fs == nil {
		m.models.fs = fields{}
	}
	ui.Column(c.Key("provider-fields")).Grow(1).MinWidth(0).Gap(14).Children(func() {
		m.providerHeader(c, t, p)
		m.textRow(c, t, p, m.say.ModelsBaseURL(), "url", p.BaseURL, p.DefaultBaseURL, false)
		m.modelRow(c, t, p)
		if p.HasKeyField {
			m.keyRow(c, t, p)
		}
		m.testRow(c, t, p)
	})
}

func (m *Main) providerHeader(c *ui.Context, t *theme.Tokens, p data.Provider) {
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			theme.Text(c, theme.BodyStrong, t.Text, providerName(m.say, p))
			note := providerNote(m.say, p)
			if note == "" && p.HasKeyField && !m.app.Models.KeySaved(p.ID) {
				note = m.say.ModelsNeedsKey()
			}
			if note != "" {
				theme.Text(c, theme.Meta, t.TextSecondary, note)
			}
		})
		switch {
		case p.Active:
			ui.Row(c).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
				ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.TextSecondary)
				theme.Text(c, theme.Meta, t.TextSecondary, m.say.ModelsInUse())
			})
		case optionButton(c, t, m.say.ModelsUseThis()).Clicked():
			m.report(m.app.Models.SetActive(p.ID))
		}
	})
	if p.PickProtocol {
		m.protocolRow(c, t, p)
	}
}

// protocolRow is only for Custom: which API the endpoint speaks.
func (m *Main) protocolRow(c *ui.Context, t *theme.Tokens, p data.Provider) {
	sel := 0
	if p.Anthropic {
		sel = 1
	}
	if i := segmented(c, t, "protocol", []string{m.say.ModelsProtocolOpenAI(), m.say.ModelsProtocolAnthropic()}, sel); i >= 0 {
		e := m.edit(p)
		e.Anthropic = i == 1
		m.report(m.app.Models.Save(p.ID, e))
	}
}

// edit is what is typed in the open provider's fields right now.
func (m *Main) edit(p data.Provider) data.ProviderEdit {
	fs := m.models.fs
	return data.ProviderEdit{
		BaseURL: strings.TrimSpace(fs.get(p.ID + "/url").buf), Model: strings.TrimSpace(fs.get(p.ID + "/model").buf),
		Key: strings.TrimSpace(fs.get(p.ID + "/key").buf), Anthropic: p.Anthropic,
	}
}

// label is a small caption over a field.
func label(c *ui.Context, t *theme.Tokens, text string) {
	theme.Text(c, theme.Meta, t.TextSecondary, text).Margin(0, 0, 5)
}

// textRow is a labelled field that saves on Enter or blur.
func (m *Main) textRow(c *ui.Context, t *theme.Tokens, p data.Provider, name, key, value, placeholder string, secret bool) {
	ui.Column(c.Key("row-" + p.ID + key)).Shrink(0).Children(func() {
		label(c, t, name)
		m.inputWell(c, t, p, key, name, value, placeholder, secret)
	})
}

// inputWell is the field itself: a quiet well, borderless text inside.
func (m *Main) inputWell(c *ui.Context, t *theme.Tokens, p data.Provider, key, name, value, placeholder string, secret bool) {
	well := ui.Row(c.Key("well-"+p.ID+key)).Height(inputHeight).Shrink(0).Padding(0, 10).AlignItems(ui.Center).
		Radius(theme.RadiusControl).Background(t.BgSubtle).Border(1, ui.Transparent)
	well.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			saved, ok := textField(c, t, m.models.fs, fieldSpec{
				name: p.ID + "/" + key, label: name, value: value, style: theme.Body, color: t.Text, placeholder: placeholder, secret: secret,
			})
			if ok {
				m.save(p, key, saved)
			}
		})
	})
}

// save stores one field the user finished editing.
func (m *Main) save(p data.Provider, key, v string) {
	e := m.edit(p)
	e.Key = ""
	switch key {
	case "url":
		e.BaseURL = strings.TrimSpace(v)
	case "model":
		e.Model = strings.TrimSpace(v)
	case "key":
		e.Key = strings.TrimSpace(v)
		m.models.fs.get(p.ID + "/key").buf = "" // the key lives in the keychain now, not on screen
	}
	m.report(m.app.Models.Save(p.ID, e))
}

// report keeps an error from the settings store where it is seen: the banner under the title bar.
func (m *Main) report(err error) {
	if err != nil {
		m.app.snap.err = err
	}
}

// modelRow is the Model field with a few suggestions under it.
func (m *Main) modelRow(c *ui.Context, t *theme.Tokens, p data.Provider) {
	m.textRow(c, t, p, m.say.ModelsModel(), "model", p.Model, "", false)
	if len(p.Suggestions) == 0 {
		return
	}
	ui.Row(c.Key("suggest-"+p.ID)).Shrink(0).Margin(-8, 0, 0).Gap(6).Wrap().Children(func() {
		for _, s := range p.Suggestions {
			if s == p.Model {
				continue
			}
			if m.chip(c, t, p.ID, s).Clicked() {
				m.models.fs.get(p.ID + "/model").buf = s
				e := m.edit(p)
				e.Model = s
				m.report(m.app.Models.Save(p.ID, e))
			}
		}
	})
}

// chip is a small suggestion in the mono face.
func (m *Main) chip(c *ui.Context, t *theme.Tokens, id, text string) ui.Element {
	btn := ui.ButtonBase(c.Key("chip-"+id+text)).Padding(2, 7).Radius(theme.RadiusControl).Shrink(0).
		Background(t.BgSubtle).Label(m.say.A11yUseModel(text)).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgSelected)
	}
	btn.Children(func() { theme.Text(c, theme.Mono, t.TextSecondary, text) })
	return btn
}

func (m *Main) keyRow(c *ui.Context, t *theme.Tokens, p data.Provider) {
	placeholder := m.say.ModelsKeyPlaceholder()
	if m.app.Models.KeySaved(p.ID) {
		placeholder = m.say.ModelsKeySaved()
	}
	ui.Column(c.Key("row-" + p.ID + "key")).Shrink(0).Children(func() {
		label(c, t, m.say.ModelsAPIKey())
		m.inputWell(c, t, p, "key", m.say.ModelsAPIKey(), "", placeholder, true)
	})
}

// testRow is the Test button and what it said.
func (m *Main) testRow(c *ui.Context, t *theme.Tokens, p data.Provider) {
	res := m.models.test
	ui.Row(c.Key("test-" + p.ID)).Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
		btn := ui.ButtonBase(c.Key("test")).Padding(4, 12).Radius(theme.RadiusControl).Shrink(0).Label(m.say.ModelsTestConnection()).
			Background(t.BgSubtle).Border(1, t.Border).Cursor(ui.CursorPointer)
		if btn.Hovered() {
			btn.Background(t.BgHover)
		}
		btn.Children(func() { theme.Text(c, theme.MetaStrong, t.Text, m.testLabel(res, p.ID)) })
		if btn.Clicked() && !(res.running && res.id == p.ID) {
			m.runTest(p)
		}
		if res.id == p.ID && res.text != "" {
			m.testOutcome(c, t, res)
		}
	})
}

func (m *Main) testLabel(res testResult, id string) string {
	if res.running && res.id == id {
		return m.say.ModelsTesting()
	}
	return m.say.ModelsTest()
}

func (m *Main) testOutcome(c *ui.Context, t *theme.Tokens, res testResult) {
	ui.Row(c).Gap(5).AlignItems(ui.Center).Shrink(1).MinWidth(0).Children(func() {
		if res.ok {
			ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.TextSecondary)
		}
		theme.Paragraph(c, theme.Meta, t.TextSecondary, res.text, 0).Shrink(1).MinWidth(0)
	})
}

// runTest sends a one-token request off the UI thread and shows how it went.
func (m *Main) runTest(p data.Provider) {
	e := m.edit(p)
	m.models.test = testResult{id: p.ID, running: true}
	models, post := m.app.Models, m.app.Act.Post
	go func() {
		d, err := models.Test(context.Background(), p.ID, e)
		post(func() { m.models.test = m.outcomeOf(p.ID, d, err) })
	}()
}

func (m *Main) outcomeOf(id string, d time.Duration, err error) testResult {
	if err != nil {
		return testResult{id: id, text: problemText(m.say, err)}
	}
	return testResult{id: id, ok: true, text: m.say.ModelsWorks(itoa(int(d.Milliseconds())))}
}
