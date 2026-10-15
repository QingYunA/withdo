package views

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The composer with two images attached, and the model popover open over it, in light and dark,
// English and Chinese (-zh). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensComposerPictures

// gradientFile writes a soft two-color picture, so a thumbnail has something to show.
func gradientFile(t *testing.T, name string, from, to color.NRGBA) string {
	t.Helper()
	const w, h = 640, 480
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			k := float64(x+y) / float64(w+h)
			mix := func(a, b uint8) uint8 { return uint8(float64(a)*(1-k) + float64(b)*k) }
			img.SetNRGBA(x, y, color.NRGBA{R: mix(from.R, to.R), G: mix(from.G, to.G), B: mix(from.B, to.B), A: 255})
		}
	}
	path := filepath.Join(t.TempDir(), name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

// modelsApp is the seeded fake with four providers set up and DeepSeek in use.
func modelsApp(t *testing.T) (*App, *recorder) {
	t.Helper()
	app, _, rec := readyApp(t, &data.FakeAssistant{Ready: true})
	mustNil(t, app.Models.Save("deepseek", data.ProviderEdit{Model: "deepseek-chat", Key: "sk-ds-demo"}))
	mustNil(t, app.Models.Save("qwen", data.ProviderEdit{Model: "qwen3-max", Key: "sk-qw-demo"}))
	mustNil(t, app.Models.Save("ollama", data.ProviderEdit{Model: "qwen3:8b"}))
	mustNil(t, app.Models.SetActive("deepseek"))
	return app, rec
}

func TestScreensComposerPictures(t *testing.T) {
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix + ".png"
			if zh {
				sfx = "-zh" + sfx
			}
			text := "Push these two designs to Friday afternoon, and do the urgent one first"
			if zh {
				text = "把这两张图里的待办都记下来，周五前要做完的标成紧急"
			}
			attached := func(open bool) (*App, func(*ui.Tester, *Main)) {
				app, rec := modelsApp(t)
				files := []string{
					gradientFile(t, "screenshot.png", color.NRGBA{R: 223, G: 227, B: 255, A: 255}, color.NRGBA{R: 244, G: 232, B: 255, A: 255}),
					gradientFile(t, "whiteboard.png", color.NRGBA{R: 255, G: 233, B: 214, A: 255}, color.NRGBA{R: 255, G: 217, B: 226, A: 255}),
				}
				return app, func(tt *ui.Tester, m *Main) {
					m.attachFiles(surfacePage, files)
					rec.drain(t)
					for m.cmp[surfacePage].pending > 0 {
						rec.drain(t)
					}
					tt.Frame()
					tt.Key(ui.Cmd, ui.KeyK)
					tt.Type(text)
					if open {
						tt.Frame()
						m.cmp[surfacePage].modelOpen = true
					}
				}
			}
			shotOf := func(name string, app *App, setup func(tt *ui.Tester, m *Main)) {
				if zh {
					zhShot(t, name+sfx, app, a.dark, setup)
					return
				}
				mainShot(t, name+sfx, app, a.dark, setup)
			}
			app, setup := attached(false)
			shotOf("mine-composer-pictures", app, setup)
			app, setup = attached(true)
			shotOf("mine-composer-model", app, setup)
			inspector, _ := modelsApp(t)
			shotOf("inspector-composer-model", inspector, func(tt *ui.Tester, m *Main) {
				m.OpenTask(5)
				m.focusInspectorComposer = true
				for i := 0; i < 5; i++ {
					tt.Frame()
				}
				tt.Type("move it to friday afternoon")
			})
			empty, _ := modelsApp(t)
			shotOf("mine-composer-plus", empty, func(tt *ui.Tester, m *Main) { tt.Key(ui.Cmd, ui.KeyK) })
		}
	}
}
