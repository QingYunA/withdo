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

// otherModel is the value of the model pop-up's last item, which turns it into a text field.
const otherModel = "\x00other"

// gridColumns is how many provider tiles a row of the grid has.
const gridColumns = 4

// maxModelChoices bounds the model pop-up: a gateway can list hundreds, the suggestions and a few
// more are enough, and Other… types the rest.
const maxModelChoices = 24

// draft is what is typed for one provider in the grid and not yet saved.
type draft struct {
	key, base, model string
	other            bool // the model is typed, not chosen
	protocol         int  // Custom: 0 as stored, 1 OpenAI-compatible, 2 Anthropic
}

// modelList is the models a provider offered when asked: the answer for one key.
type modelList struct {
	seq     int // counts the requests, so that an old answer is dropped
	names   []string
	forKey  string // the key typed when it was asked, "" for the stored one
	asked   bool
	running bool
}

func (s *Settings) draftOf(id string) *draft {
	if s.assist.drafts[id] == nil {
		s.assist.drafts[id] = &draft{}
	}
	return s.assist.drafts[id]
}

// providerGrid is the 12 providers in four columns: the logo and name, the green square of one
// that is set up, and the accent ring on the one in use. The grid is always on the pane.
func (s *Settings) providerGrid(c *ui.Context, t *theme.Tokens, ps []data.Provider) {
	for row := 0; row*gridColumns < len(ps); row++ {
		ui.Row(c.Key("grid-"+itoa(row))).Shrink(0).Gap(8).Margin(0, 0, 8).Children(func() {
			for _, p := range ps[row*gridColumns : min(len(ps), (row+1)*gridColumns)] {
				s.tile(c, t, p)
			}
		})
	}
}

// tile is one provider of the grid. A set-up provider becomes the one in use and is tested; any
// other opens its key and model form under the grid. The one already in use opens its form too, so
// that its key and model can still be changed.
func (s *Settings) tile(c *ui.Context, t *theme.Tokens, p data.Provider) {
	name := providerName(s.say, p)
	label := s.say.A11yProviderSetUp(name)
	switch {
	case p.Active:
		label = s.say.A11yProviderChange(name)
	case p.Configured:
		label = s.say.A11yUseProvider(name)
	}
	btn := ui.ButtonBase(c.Key("tile-"+p.ID)).Grow(1).Basis(0).MinWidth(0).Height(theme.ProviderTileH).Radius(theme.SettingsRadius).
		Row().AlignItems(ui.Center).Gap(10).Padding(0, 12).Label(label).Tooltip(label).Cursor(ui.CursorPointer).Background(t.SettingsGroup())
	switch {
	case p.Active:
		btn.Shadow(0, 0, 0, 2, t.Accent)
	case btn.Hovered():
		btn.Shadow(0, 0, 0, 1, t.Border.Alpha(0.3))
	default:
		btn.Shadow(0, 0, 0, 1, t.Border)
	}
	btn.Children(func() {
		providerLogo(c, t, p.ID, theme.LogoChip)
		theme.Text(c, theme.Meta, t.Text, name).Grow(1).MinWidth(0).Ellipsis("…")
		if p.Configured {
			ui.Box(c.Key("tile-set-up-"+p.ID)).Size(6, 6).Absolute().Top(8).Right(8).Background(t.Status(data.Done))
		}
	})
	if !btn.Clicked() {
		return
	}
	switch {
	case p.Active:
		// Already in use: the tile is how its key and model are changed.
		s.assist.pick, s.assist.err = p.ID, ""
		s.fetchModels(p, "")
	case p.Configured:
		s.assist.pick, s.assist.err = "", ""
		s.assist.test = testResult{}
		if s.app.Models.SetActive(p.ID) == nil {
			s.runTest(p)
		}
	default:
		s.assist.pick, s.assist.err = p.ID, ""
		s.fetchModels(p, "")
	}
}

// providerLogo is a provider's logo in the colors of the theme: a one-color mark takes the text
// color, so that it is white in dark mode, and Custom's plus the quiet one.
func providerLogo(c *ui.Context, t *theme.Tokens, id string, size float32) ui.Element {
	ink := t.Text
	if id == "custom" {
		ink = t.TextSecondary
	}
	return theme.ProviderLogo(c, id, size, ink)
}

// formLabelWidth is the width of the labels at the left of the fields.
const formLabelWidth float32 = 72

func formRow(c *ui.Context, t *theme.Tokens, label string, margin bool, body func()) {
	r := ui.Row(c).Shrink(0).Gap(12).AlignItems(ui.Center)
	if margin {
		r.Margin(0, 0, 10)
	}
	r.Children(func() {
		theme.Text(c, theme.Compact, t.TextSecondary, label).Width(formLabelWidth).Shrink(0)
		body()
	})
}

// providerForm is what the provider picked needs: the key (not for Magpie and Ollama) with a link
// to where keys are handed out, the model, Save and test, and, for Custom, the address and the
// protocol. A failure shows under it.
func (s *Settings) providerForm(c *ui.Context, t *theme.Tokens, p data.Provider) {
	d := s.draftOf(p.ID)
	ui.Column(c.Key("form-"+p.ID)).Shrink(0).Margin(8, 0, 18).Padding(16).Radius(theme.SettingsRadius).
		Background(t.SettingsGroup()).Border(theme.Hairline, t.Border.Alpha(0.08)).Children(func() {
		if p.ID == "custom" {
			formRow(c, t, s.say.ModelsBaseURL(), true, func() { s.baseField(c, t, p, d) })
			formRow(c, t, s.say.ModelsProtocol(), true, func() { s.protocolPick(c, t, p, d) })
		}
		if p.HasKeyField {
			formRow(c, t, s.say.ModelsAPIKey(), true, func() { s.keyField(c, t, p, d) })
		}
		formRow(c, t, s.say.ModelsModel(), false, func() {
			s.modelField(c, t, p, d)
			s.saveButton(c, t, p, d)
		})
		if s.assist.err != "" {
			theme.Paragraph(c, theme.Meta, t.Blocked(), s.assist.err, 3).Margin(10, 0, 0, formLabelWidth+12)
		}
	})
}

func (s *Settings) keyField(c *ui.Context, t *theme.Tokens, p data.Provider, d *draft) {
	placeholder := s.say.ModelsKeyFor(providerName(s.say, p))
	if p.ID == "custom" {
		placeholder = s.say.ModelsKeyPlaceholder()
	}
	if s.app.Models.KeySaved(p.ID) {
		placeholder = "••••••••••••••••••••"
	}
	var entered, left bool
	fieldWell(c, t, "key", 0, func() {
		entered, left = draftField(c, t, s.assist.fs, draftSpec{
			name: p.ID + "/key", label: s.say.ModelsAPIKey(), placeholder: placeholder, secret: true,
		}, &d.key)
	}).Grow(1)
	if p.KeyURL != "" && settingsOutLink(c, t, "get-key", s.say.SettingsGetKey(), s.say.SettingsGetKey()).Clicked() {
		s.app.Act.OpenURL(p.KeyURL)
	}
	if key := strings.TrimSpace(d.key); (entered || left) && key != "" {
		s.fetchModels(p, key)
	}
	if entered {
		s.saveAndTest(p)
	}
}

func (s *Settings) baseField(c *ui.Context, t *theme.Tokens, p data.Provider, d *draft) {
	var entered bool
	fieldWell(c, t, "base", 0, func() {
		entered, _ = draftField(c, t, s.assist.fs, draftSpec{
			name: p.ID + "/base", label: s.say.ModelsBaseURL(), placeholder: "https://…", value: p.BaseURL,
		}, &d.base)
	}).Grow(1)
	if entered {
		s.saveAndTest(p)
	}
}

func (s *Settings) protocolPick(c *ui.Context, t *theme.Tokens, p data.Provider, d *draft) {
	sel := 0
	if s.anthropicOf(p, d) {
		sel = 1
	}
	if i := segmented(c, t, "protocol", s.say.ModelsProtocol(), []string{s.say.ModelsProtocolOpenAI(), s.say.ModelsProtocolAnthropic()}, sel); i >= 0 {
		d.protocol = i + 1
	}
}

// modelField is a pop-up of the models the provider offers and Other…, or a text field when it
// offers none or Other… was chosen.
func (s *Settings) modelField(c *ui.Context, t *theme.Tokens, p data.Provider, d *draft) {
	current := s.modelOf(p, d)
	if choices := s.modelChoices(p, current); len(choices) > 0 && !d.other {
		opts := make([]popOption, 0, len(choices)+1)
		for _, m := range choices {
			opts = append(opts, popOption{m, m})
		}
		opts = append(opts, popOption{otherModel, s.say.ModelsOther()})
		var v string
		var ok bool
		ui.Row(c).Grow(1).MinWidth(0).Children(func() { v, ok = popUpFill(c, t, "model-"+p.ID, s.say.A11yChooseModel(), current, opts) })
		switch {
		case !ok:
		case v == otherModel:
			d.other = true
		default:
			d.model = v
		}
		return
	}
	var entered bool
	fieldWell(c, t, "model", 0, func() {
		entered, _ = draftField(c, t, s.assist.fs, draftSpec{
			name: p.ID + "/model", label: s.say.ModelsModel(), value: current, focus: d.other,
		}, &d.model)
	}).Grow(1)
	if entered {
		s.saveAndTest(p)
	}
}

func (s *Settings) saveButton(c *ui.Context, t *theme.Tokens, p data.Provider, d *draft) {
	if s.assist.saving {
		theme.Text(c, theme.Body, t.TextSecondary, s.say.ModelsTesting()).Shrink(0)
		return
	}
	if settingsButton(c, t, "save-test", s.say.SettingsSaveAndTest(), true).Clicked() {
		s.saveAndTest(p)
	}
}

// modelOf is the model the form stands on: what was chosen or typed, else what is stored.
func (s *Settings) modelOf(p data.Provider, d *draft) string {
	if m := strings.TrimSpace(d.model); m != "" {
		return m
	}
	return p.Model
}

func (s *Settings) anthropicOf(p data.Provider, d *draft) bool {
	if d.protocol != 0 {
		return d.protocol == 2
	}
	return p.Anthropic
}

func (s *Settings) baseOf(p data.Provider, d *draft) string {
	if b := strings.TrimSpace(d.base); b != "" {
		return b
	}
	return p.BaseURL
}

// modelChoices are the models to offer: the preset's suggestions, then what the provider listed,
// and the current model first when neither has it.
func (s *Settings) modelChoices(p data.Provider, current string) []string {
	out := slices.Clone(p.Suggestions)
	if l := s.assist.lists[p.ID]; l != nil {
		for _, m := range l.names {
			if len(out) >= maxModelChoices {
				break
			}
			if !slices.Contains(out, m) {
				out = append(out, m)
			}
		}
	}
	if len(out) > 0 && current != "" && !slices.Contains(out, current) {
		out = append([]string{current}, out...)
	}
	return out
}

// fetchModels asks the provider for its models in the background, with the stored key or, when
// key is not empty, that one. It asks once for a key.
func (s *Settings) fetchModels(p data.Provider, key string) {
	if p.HasKeyField && key == "" && !s.app.Models.KeySaved(p.ID) {
		return // nothing to ask with yet
	}
	l := s.assist.lists[p.ID]
	if l == nil {
		l = &modelList{}
		s.assist.lists[p.ID] = l
	}
	if l.asked && l.forKey == key {
		return
	}
	l.asked, l.forKey, l.running = true, key, true
	l.seq++
	seq, e := l.seq, s.edit(p)
	e.Key = key
	e.BaseURL = s.baseOf(p, s.draftOf(p.ID))
	models, post := s.app.Models, s.app.Act.Post
	go func() {
		names, err := models.ListModels(context.Background(), p.ID, e)
		post(func() {
			if l.seq != seq {
				return
			}
			l.running = false
			if err == nil {
				l.names = names
			}
		})
	}()
}

// saveAndTest stores what is typed, tests it, and, when it works, makes the provider the one in
// use and goes back to the pane at rest. A failure stays under the form in plain words; the key
// is kept, so the next try needs only the fix.
func (s *Settings) saveAndTest(p data.Provider) {
	if s.assist.saving {
		return
	}
	d := s.draftOf(p.ID)
	e := data.ProviderEdit{
		BaseURL: s.baseOf(p, d), Model: s.modelOf(p, d), Key: strings.TrimSpace(d.key), Anthropic: s.anthropicOf(p, d),
	}
	switch {
	case p.ID == "custom" && e.BaseURL == "":
		s.assist.err = s.say.SettingsNeedsURL()
		return
	case e.Model == "":
		s.assist.err = s.say.SettingsNeedsModel()
		return
	case p.KeyRequired && e.Key == "" && !s.app.Models.KeySaved(p.ID):
		s.assist.err = s.say.ModelsNeedsKey() + "."
		return
	}
	s.assist.saving, s.assist.err = true, ""
	s.assist.test = testResult{id: p.ID, running: true}
	models, post := s.app.Models, s.app.Act.Post
	go func() {
		var took time.Duration
		err := models.Save(p.ID, e)
		if err == nil {
			e.Key = "" // it is stored now
			took, err = models.Test(context.Background(), p.ID, e)
		}
		post(func() { s.finishSave(p, took, err) })
	}()
}

func (s *Settings) finishSave(p data.Provider, took time.Duration, err error) {
	s.assist.saving = false
	if err != nil {
		s.assist.test = testResult{id: p.ID, text: problemText(s.say, err)}
		s.assist.err = s.assist.test.text
		return
	}
	if err := s.app.Models.SetActive(p.ID); err != nil {
		s.assist.test = testResult{id: p.ID, text: problemText(s.say, err)}
		s.assist.err = s.assist.test.text
		return
	}
	s.assist.test = testResult{id: p.ID, ok: true, text: s.say.SettingsConnected(itoa(int(took.Milliseconds())))}
	delete(s.assist.drafts, p.ID)
	s.assist.pick = ""
}
