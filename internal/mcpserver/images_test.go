package mcpserver_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// pngBytes is a w x h picture; the noise keeps a large one from compressing to nothing.
func pngBytes(t *testing.T, w, h int, noisy bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(40)
			if noisy {
				v = uint8((x*7 + y*13 + x*y) % 251)
			}
			img.Set(x, y, color.NRGBA{R: v, G: uint8(x), B: uint8(y), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// attach stores a picture file in the rig's images folder and links it to a new task.
func (r *rig) attach(taskID int64, name, hash string, raw []byte, w, h int) {
	r.t.Helper()
	if err := os.MkdirAll(r.images, 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.images, hash+".png"), raw, 0o644); err != nil {
		r.t.Fatal(err)
	}
	_, err := r.core.AttachImage(ctx, taskID, domain.AttachImageInput{
		Image: domain.Image{Hash: hash, Ext: "png", MIME: "image/png", Bytes: int64(len(raw))}, Name: name, Width: w, Height: h})
	if err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) newTask() int64 {
	r.t.Helper()
	return int64(r.ok("create_task", map[string]any{"title": "pics"}).obj("task")["id"].(float64))
}

func (r *rig) image(taskID int64, index int) *mcp.CallToolResult {
	r.t.Helper()
	res, err := r.session.CallTool(ctx, &mcp.CallToolParams{Name: "get_task_image", Arguments: map[string]any{"task_id": taskID, "index": index}})
	if err != nil {
		r.t.Fatal(err)
	}
	return res
}

func TestGetTaskListsPicturesWithoutTheirBytes(t *testing.T) {
	r := newRig(t)
	id := r.newTask()
	if got := r.ok("get_task", map[string]any{"task_id": id}); strings.Contains(got.Text, "Images") || got.Data["images"] != nil {
		t.Fatalf("a task without pictures lists some: %q %v", got.Text, got.Data["images"])
	}
	raw := pngBytes(t, 8, 6, false)
	r.attach(id, "shot.png", "aa11", raw, 8, 6)
	got := r.ok("get_task", map[string]any{"task_id": id})
	for _, want := range []string{"Images", "get_task_image", "1. shot.png 8x6 image/png"} {
		if !strings.Contains(got.Text, want) {
			t.Errorf("text lacks %q:\n%s", want, got.Text)
		}
	}
	pics := got.list("images")
	if len(pics) != 1 {
		t.Fatalf("images = %v", pics)
	}
	p := pics[0].(map[string]any)
	path, _ := p["path"].(string)
	want := filepath.Join(r.images, "aa11.png")
	if !filepath.IsAbs(path) || path != want || p["index"] != 1.0 || p["width"] != 8.0 || p["mime"] != "image/png" || p["bytes"] != float64(len(raw)) || p["name"] != "shot.png" {
		t.Errorf("image = %v, want path %s", p, want)
	}
}

func TestGetTaskImageReturnsImageContent(t *testing.T) {
	r := newRig(t)
	id := r.newTask()
	raw := pngBytes(t, 8, 6, false)
	r.attach(id, "shot.png", "aa11", raw, 8, 6)
	res := r.image(id, 1)
	if res.IsError || len(res.Content) != 2 {
		t.Fatalf("result = %+v", res)
	}
	if txt, _ := res.Content[0].(*mcp.TextContent); txt == nil || !strings.Contains(txt.Text, "shot.png") {
		t.Errorf("text = %+v", res.Content[0])
	}
	img, _ := res.Content[1].(*mcp.ImageContent)
	if img == nil || img.MIMEType != "image/png" || !bytes.Equal(img.Data, raw) {
		t.Errorf("image content = %+v", res.Content[1])
	}
}

func TestGetTaskImageScalesALargePictureDown(t *testing.T) {
	r := newRig(t)
	id := r.newTask()
	r.attach(id, "wide.png", "bb22", pngBytes(t, 3000, 1000, true), 3000, 1000)
	img, _ := r.image(id, 1).Content[1].(*mcp.ImageContent)
	if img == nil {
		t.Fatal("no image content")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(img.Data))
	if err != nil || format != "png" || img.MIMEType != "image/png" || cfg.Width != 1568 || cfg.Height != 522 {
		t.Errorf("got %dx%d %s %s, %v", cfg.Width, cfg.Height, format, img.MIMEType, err)
	}
}

func TestGetTaskImageErrorsListWhatExists(t *testing.T) {
	r := newRig(t)
	id := r.newTask()
	if res := r.image(id, 1); !res.IsError || !strings.Contains(res.Content[0].(*mcp.TextContent).Text, "no pictures") {
		t.Errorf("no pictures: %+v", res)
	}
	r.attach(id, "a.png", "cc33", pngBytes(t, 4, 4, false), 4, 4)
	r.attach(id, "b.png", "dd44", pngBytes(t, 5, 5, false), 5, 5)
	res := r.image(id, 3)
	msg := res.Content[0].(*mcp.TextContent).Text
	if !res.IsError || !strings.Contains(msg, "index") || !strings.Contains(msg, "1 a.png, 2 b.png") {
		t.Errorf("out of range: %q", msg)
	}
	if err := os.Remove(filepath.Join(r.images, "dd44.png")); err != nil {
		t.Fatal(err)
	}
	res = r.image(id, 2)
	if msg = res.Content[0].(*mcp.TextContent).Text; !res.IsError || !strings.Contains(msg, "missing") || !strings.Contains(msg, "b.png") {
		t.Errorf("missing file: %q", msg)
	}
}
