package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // registers the GIF decoder with image.Decode
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder with image.Decode

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// Claude's limits for a picture: the long side in pixels, and the size in bytes.
const (
	maxImageEdge  = 1568
	maxImageBytes = 15 << 18 // 3.75 MB
)

// imageBrief is a picture as get_task lists it. Index counts from 1 and is what get_task_image takes.
type imageBrief struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	MIME   string `json:"mime"`
	Bytes  int64  `json:"bytes"`
	Path   string `json:"path"`
}

func (t *tools) imagePath(img domain.TaskImage) string {
	return filepath.Join(t.imagesDir, img.Hash+"."+img.Ext)
}

// imageBriefs lists a task's pictures with the absolute paths of their files.
func (t *tools) imageBriefs(images []domain.TaskImage) []imageBrief {
	out := make([]imageBrief, len(images))
	for i, v := range images {
		path, err := filepath.Abs(t.imagePath(v))
		if err != nil {
			path = t.imagePath(v)
		}
		out[i] = imageBrief{Index: i + 1, Name: v.Name, Width: v.Width, Height: v.Height, MIME: v.MIME, Bytes: v.Bytes, Path: path}
	}
	return out
}

// imageLines is the text get_task adds for a task with pictures.
func imageLines(taskID int64, images []imageBrief) []string {
	lines := []string{fmt.Sprintf("Images (see one with get_task_image: task_id %d and its index, or open the path):", taskID)}
	for _, v := range images {
		lines = append(lines, fmt.Sprintf("%d. %s %dx%d %s %d bytes %s", v.Index, v.Name, v.Width, v.Height, v.MIME, v.Bytes, v.Path))
	}
	return lines
}

type getTaskImageArgs struct {
	TaskID int64 `json:"task_id"`
	Index  int   `json:"index"`
}

func (t *tools) registerGetTaskImage(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "get_task_image",
		Title: "Get task image",
		Description: "Get one picture attached to a task, as an image. The index is the number get_task lists under Images, from 1. " +
			"A picture over 1568 px on the long side or 3.75 MB is scaled down first.",
		InputSchema: object(map[string]*jsonschema.Schema{
			"task_id": id("Task id, as returned by report_progress or create_task"),
			"index":   id("Number of the picture in get_task's Images list, from 1"),
		}, "task_id", "index"),
	}, t.getTaskImage)
}

func (t *tools) getTaskImage(ctx context.Context, _ *mcp.CallToolRequest, a getTaskImageArgs) (*mcp.CallToolResult, any, error) {
	images, err := t.core.ListTaskImages(ctx, a.TaskID)
	if err != nil {
		return t.fail(err)
	}
	if a.Index < 1 || a.Index > len(images) {
		return t.fail(imageIndexError(a, images))
	}
	img := images[a.Index-1]
	raw, err := os.ReadFile(t.imagePath(img))
	if err != nil {
		return t.fail(fieldErr("index", fmt.Sprintf("the file of picture %d (%s) is missing: %v", a.Index, img.Name, err)))
	}
	data, mime, err := fitImage(raw, img.MIME)
	if err != nil {
		return t.fail(fieldErr("index", fmt.Sprintf("picture %d (%s) cannot be read: %v", a.Index, img.Name, err)))
	}
	text := fmt.Sprintf("Picture %d of task #%d: %s (%dx%d)", a.Index, a.TaskID, img.Name, img.Width, img.Height)
	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: text},
		&mcp.ImageContent{Data: data, MIMEType: mime},
	}}, nil, nil
}

func imageIndexError(a getTaskImageArgs, images []domain.TaskImage) error {
	if len(images) == 0 {
		return fieldErr("index", fmt.Sprintf("task %d has no pictures", a.TaskID))
	}
	names := make([]string, len(images))
	for i, v := range images {
		names[i] = fmt.Sprintf("%d %s", i+1, v.Name)
	}
	return fieldErr("index", fmt.Sprintf("%d is out of range; task %d has: %s", a.Index, a.TaskID, strings.Join(names, ", ")))
}

// fitImage returns the picture within Claude's limits: as it is when it already fits, else
// scaled down and re-encoded as JPEG when it is a JPEG, as PNG otherwise (and as JPEG on white
// when that is still too large).
func fitImage(raw []byte, mime string) ([]byte, string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	if max(cfg.Width, cfg.Height) <= maxImageEdge && len(raw) <= maxImageBytes {
		return raw, mime, nil
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	src = shrink(src, maxImageEdge)
	if format == "jpeg" {
		if b, ok := encodeJPEG(src, 88); ok {
			return b, "image/jpeg", nil
		}
	} else if b, ok := encodePNG(src); ok {
		return b, "image/png", nil
	}
	for _, q := range []int{85, 70, 50} {
		if b, ok := encodeJPEG(onWhite(src), q); ok {
			return b, "image/jpeg", nil
		}
	}
	return nil, "", fmt.Errorf("still over %d bytes after scaling down", maxImageBytes)
}

func shrink(img image.Image, edge int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	long := max(w, h)
	if long <= edge {
		return img
	}
	dst := image.NewNRGBA(image.Rect(0, 0, max(1, w*edge/long), max(1, h*edge/long)))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)
	return dst
}

// onWhite flattens transparency onto white, as JPEG has none.
func onWhite(img image.Image) image.Image {
	dst := image.NewRGBA(img.Bounds())
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Over)
	return dst
}

func encodeJPEG(img image.Image, quality int) ([]byte, bool) {
	var buf bytes.Buffer
	if jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}) != nil || buf.Len() > maxImageBytes {
		return nil, false
	}
	return buf.Bytes(), true
}

func encodePNG(img image.Image) ([]byte, bool) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if enc.Encode(&buf, img) != nil || buf.Len() > maxImageBytes {
		return nil, false
	}
	return buf.Bytes(), true
}
