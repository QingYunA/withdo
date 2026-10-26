package views

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// Images in the composer and the model chip: pasted, dropped (attachFiles is what a drop calls),
// chosen with +, shown as thumbnails, removed, limited to four, sent with the text; and the chip
// that lists the models set up and switches the one in use.

// pngBytes is a small picture of one color, as a screenshot is on the pasteboard.
func pngBytes(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// picFiles writes n small PNG files and returns their paths.
func picFiles(t *testing.T, names ...string) []string {
	t.Helper()
	dir := t.TempDir()
	var out []string
	for i, n := range names {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, pngBytes(t, 120, 90, color.NRGBA{R: uint8(40 * (i + 1)), G: 120, B: 200, A: 255}), 0o600); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

// withModels sets up DeepSeek (in use) and Anthropic, as Settings would.
func (r *assistRig) withModels(t *testing.T) {
	t.Helper()
	mustNil(t, r.app.Models.Save("anthropic", data.ProviderEdit{Model: "claude-haiku-5-5", Key: "sk-ant-demo"}))
	mustNil(t, r.app.Models.Save("deepseek", data.ProviderEdit{Model: "deepseek-chat", Key: "sk-ds-demo"}))
	mustNil(t, r.app.Models.SetActive("deepseek"))
	r.tt.Frame()
}

func (r *assistRig) activeProvider() string {
	for _, p := range r.app.Models.Providers() {
		if p.Active {
			return p.ID
		}
	}
	return ""
}

func (r *assistRig) pics() []picture { return r.m.cmp[surfacePage].pics }

// loaded waits for the pictures being read, whatever their number, and shows them.
func (r *assistRig) loaded(t *testing.T) {
	t.Helper()
	r.rec.drain(t)
	for r.m.cmp[surfacePage].pending > 0 {
		r.rec.drain(t)
	}
	r.tt.Frame()
}

func TestAPastedImageBecomesAThumbnailAndGoesWithTheText(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	r := newAssistRig(t, fa)
	r.withModels(t)
	r.tt.Key(ui.Cmd, ui.KeyK)
	raw := pngBytes(t, 300, 200, color.NRGBA{R: 200, G: 60, B: 60, A: 255})
	r.tt.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, raw))))
	r.tt.Key(ui.Cmd, ui.KeyV)
	r.loaded(t)
	if len(r.pics()) != 1 || r.pageBuf() != "" {
		t.Fatalf("%d pictures, field %q after the paste", len(r.pics()), r.pageBuf())
	}
	if r.m.cmp[surfacePage].pending != 0 {
		t.Error("the paste is still pending")
	}
	r.tt.Type("move these to Friday")
	r.tt.Key(0, ui.KeyEnter)
	r.rec.drain(t)
	got := fa.Pictures()
	if len(got) != 1 || len(got[0]) != 1 || got[0][0].MIME != "image/png" || len(got[0][0].Data) == 0 {
		t.Fatalf("sent %+v", got)
	}
	if asked := fa.Asked(); len(asked) != 1 || asked[0] != "move these to Friday" {
		t.Errorf("asked %q", asked)
	}
	if len(r.pics()) != 0 {
		t.Error("the picture stays in the composer after it was sent")
	}
}

func TestPastingTextStillPastesText(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.SetClipboard("buy milk")
	r.tt.Key(ui.Cmd, ui.KeyV)
	if r.pageBuf() != "buy milk" || len(r.pics()) != 0 {
		t.Errorf("field %q, %d pictures", r.pageBuf(), len(r.pics()))
	}
}

func TestAnImagePastedWithoutAModelIsLeftAlone(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	r.tt.Key(ui.Cmd, ui.KeyK)
	raw := pngBytes(t, 30, 20, color.NRGBA{A: 255})
	r.tt.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, raw))))
	r.tt.Key(ui.Cmd, ui.KeyV)
	if len(r.pics()) != 0 || r.m.cmp[surfacePage].pending != 0 {
		t.Error("a picture was taken with no model to read it")
	}
}

func TestDroppedFilesAttachInOrderAndAtMostFour(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true}
	r := newAssistRig(t, fa)
	files := picFiles(t, "a.png", "b.png", "c.png", "d.png", "e.png")
	r.m.attachFiles(surfacePage, files)
	r.loaded(t)
	if len(r.pics()) != 4 {
		t.Fatalf("%d pictures, want 4", len(r.pics()))
	}
	for i, want := range []string{"a.png", "b.png", "c.png", "d.png"} {
		if r.pics()[i].att.Name != want {
			t.Errorf("picture %d is %q, want %q", i, r.pics()[i].att.Name, want)
		}
	}
	if !r.tt.HasText("At most 4 images at a time.") {
		t.Errorf("no word about the limit in %q", r.tt.Texts())
	}
	// A fifth, later, is refused too.
	r.m.attachFiles(surfacePage, picFiles(t, "f.png"))
	r.tt.Frame()
	if len(r.pics()) != 4 {
		t.Errorf("%d pictures after a fifth", len(r.pics()))
	}
}

func TestAFileThatIsNotAnImageIsRefusedWithAQuietLine(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	dir := t.TempDir()
	bad := filepath.Join(dir, "notes.pdf")
	if err := os.WriteFile(bad, []byte("%PDF-1.4"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.m.attachFiles(surfacePage, []string{bad})
	r.loaded(t)
	if len(r.pics()) != 0 || !r.tt.HasText("That file isn't a PNG, JPEG, GIF or WebP image.") {
		t.Errorf("%d pictures, texts %q", len(r.pics()), r.tt.Texts())
	}
	r.m.attachFiles(surfacePage, []string{filepath.Join(dir, "missing.png")})
	r.loaded(t)
	if !r.tt.HasText("Couldn't read that file.") {
		t.Errorf("texts %q", r.tt.Texts())
	}
}

func TestTheXRemovesAThumbnail(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.m.attachFiles(surfacePage, picFiles(t, "a.png", "b.png"))
	r.loaded(t)
	if len(r.pics()) != 2 {
		t.Fatalf("%d pictures", len(r.pics()))
	}
	if err := r.tt.Click("Remove a.png"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if len(r.pics()) != 1 || r.pics()[0].att.Name != "b.png" {
		t.Errorf("after the x: %+v", r.pics())
	}
}

func TestPicturesAloneCanBeSentAndAFailedSendKeepsThemForRetry(t *testing.T) {
	calls := 0
	fa := &data.FakeAssistant{Ready: true, Script: func(string) (data.Reply, error) {
		calls++
		if calls == 1 {
			return data.Reply{}, &data.ProblemError{Problem: data.ProblemNoImages, Text: "This model cannot read images."}
		}
		return data.Reply{Text: "Seen."}, nil
	}}
	r := newAssistRig(t, fa)
	r.withModels(t)
	r.m.attachFiles(surfacePage, picFiles(t, "a.png"))
	r.loaded(t)
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Key(0, ui.KeyEnter) // no text, one picture
	r.answer(t)
	if got := fa.Pictures(); len(got) != 1 || len(got[0]) != 1 {
		t.Fatalf("sent %+v", got)
	}
	if !r.tt.HasText("deepseek-chat cannot read images.") {
		t.Errorf("no friendly error in %q", r.tt.Texts())
	}
	if !r.m.cmp[surfacePage].modelOpen || !r.tt.HasText("Switch model") {
		t.Error("the model chip is not open to pick another model")
	}
	if err := r.tt.Click("Anthropic"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.activeProvider() != "anthropic" {
		t.Errorf("active %q", r.activeProvider())
	}
	if err := r.tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	r.rec.drain(t)
	if got := fa.Pictures(); len(got) != 2 || len(got[1]) != 1 || got[1][0].Name != "a.png" {
		t.Errorf("retry sent %+v", got)
	}
}

func TestThePlusOpensTheFilePanelAndTakesWhatIsChosen(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	files := picFiles(t, "one.png", "two.png")
	var title string
	old := pickFiles
	pickFiles = func(ti string) []string { title = ti; return files }
	t.Cleanup(func() { pickFiles = old })
	r.tt.Key(ui.Cmd, ui.KeyK)
	rect, ok := r.tt.Find("Add images")
	if !ok {
		t.Fatalf("no + in the focused empty box: %q", r.tt.Texts())
	}
	x, y := center(rect)
	r.tt.Move(x, y) // the pointer arrives before it presses: the + must stay under it
	r.tt.ClickAt(x, y)
	r.loaded(t)
	if title != "Choose images" || len(r.pics()) != 2 {
		t.Errorf("title %q, %d pictures", title, len(r.pics()))
	}
	r.tt.Type("hi")
	if r.pageBuf() != "hi" {
		t.Errorf("the cursor did not come back to the field: %q", r.pageBuf())
	}
}

func TestTheChipNamesTheModelInUseAndSwitchesIt(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.withModels(t)
	r.onProject() // Home has no chip: the model is chosen in Settings
	if !r.tt.HasText("DeepSeek · deepseek-chat") {
		t.Error("no chip on the empty box at rest")
	}
	r.tt.Key(ui.Cmd, ui.KeyK)
	if !r.tt.HasText("DeepSeek · deepseek-chat") {
		t.Error("no chip on the empty focused box")
	}
	r.tt.Type("hello")
	if !r.tt.HasText("DeepSeek · deepseek-chat") {
		t.Fatalf("no chip in %q", r.tt.Texts())
	}
	if err := r.tt.Click("DeepSeek · deepseek-chat"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	for _, want := range []string{"Switch model", "Anthropic", "claude-haiku-5-5", "deepseek-chat"} {
		if !r.tt.HasText(want) {
			t.Errorf("the popover lacks %q: %q", want, r.tt.Texts())
		}
	}
	if err := r.tt.Click("Anthropic"); err != nil {
		t.Fatal(err)
	}
	r.tt.Frame()
	if r.activeProvider() != "anthropic" {
		t.Errorf("active %q, want anthropic", r.activeProvider())
	}
	if !r.tt.HasText("Anthropic · claude-haiku-5-5") || r.tt.HasText("Switch model") {
		t.Errorf("the chip did not follow, or the popover stayed: %q", r.tt.Texts())
	}
	r.tt.Type(" there")
	if r.pageBuf() != "hello there" {
		t.Errorf("the cursor did not come back to the field: %q", r.pageBuf())
	}
}

func TestTheChipIsHiddenWhenNoModelIsSetUp(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	r.tt.Key(ui.Cmd, ui.KeyK)
	r.tt.Type("hello")
	for _, s := range r.tt.Texts() {
		if len(s) > 4 && s[len(s)-3:] == "⌄" {
			t.Errorf("a chip says %q with no model set up", s)
		}
	}
	if _, err := func() (ui.Rect, error) { rect, _ := r.tt.Find("Add images"); return rect, nil }(); err != nil {
		t.Fatal(err)
	}
}
