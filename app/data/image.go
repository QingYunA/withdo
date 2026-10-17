package data

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // registers the GIF decoder with image.Decode
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder with image.Decode
)

// The limits of the pictures of one request.
const (
	MaxPictures     = 4        // at most this many ride along with a text
	MaxPictureEdge  = 1600     // the long side, in pixels, after downscaling
	MaxPictureSize  = 5 << 20  // bytes, after downscaling
	MaxOriginalSize = 20 << 20 // an original up to this size is kept with a task; a larger one keeps the prepared copy
	maxSourceSize   = 64 << 20 // a file larger than this is not even read
	maxSourcePixels = 100_000_000
)

// originalLimit is MaxOriginalSize; tests lower it.
var originalLimit = MaxOriginalSize

var (
	// ErrPictureFormat means the bytes are not a PNG, JPEG, GIF or WebP picture (a HEIC photo
	// lands here too: Go cannot decode it).
	ErrPictureFormat = errors.New("not a PNG, JPEG, GIF or WebP picture")
	// ErrPictureSize means the picture is still over 5 MB after downscaling.
	ErrPictureSize = errors.New("the picture is too large")
)

// PrepareImage turns the bytes of a picture into an Attachment: downscaled to 1600 pixels on the
// long side, as JPEG when it came as one and has no transparency, else as PNG (a PNG that is
// still over 5 MB goes out as a JPEG on white). The original bytes ride along when they are up to
// 20 MB and differ from the prepared copy, for the task that keeps the picture. Nothing is
// written anywhere.
func PrepareImage(name string, raw []byte) (Attachment, error) {
	att, err := prepareImage(name, raw)
	if err != nil {
		return att, err
	}
	if len(raw) <= originalLimit && !bytes.Equal(raw, att.Data) {
		_, format, _ := image.DecodeConfig(bytes.NewReader(raw))
		att.Original, att.OriginalMIME = raw, "image/"+format
	}
	return att, nil
}

func prepareImage(name string, raw []byte) (Attachment, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxSourcePixels {
		return Attachment{}, ErrPictureFormat
	}
	small := max(cfg.Width, cfg.Height) <= MaxPictureEdge
	if small && (format == "jpeg" || format == "png") && len(raw) <= MaxPictureSize {
		return Attachment{Name: name, MIME: "image/" + format, Data: raw}, nil // as it came
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Attachment{}, ErrPictureFormat
	}
	src = downscale(src, MaxPictureEdge)
	opaque := isOpaque(src)
	if format == "jpeg" && opaque {
		if b, ok := encodeJPEG(src, 88); ok {
			return Attachment{Name: name, MIME: "image/jpeg", Data: b}, nil
		}
	} else if b, ok := encodePNG(src); ok {
		return Attachment{Name: name, MIME: "image/png", Data: b}, nil
	}
	flat := onWhite(src)
	for _, q := range []int{85, 70, 50} {
		if b, ok := encodeJPEG(flat, q); ok {
			return Attachment{Name: name, MIME: "image/jpeg", Data: b}, nil
		}
	}
	return Attachment{}, ErrPictureSize
}

// ReadImageFile reads a picture from disk and prepares it.
func ReadImageFile(path string) (Attachment, error) {
	f, err := os.Open(path)
	if err != nil {
		return Attachment{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxSourceSize+1))
	if err != nil {
		return Attachment{}, err
	}
	if len(raw) > maxSourceSize {
		return Attachment{}, ErrPictureSize
	}
	return PrepareImage(filepath.Base(path), raw)
}

// Stored is the picture as a task keeps it: the original when the attachment has one, else the
// prepared copy, with its MIME type.
func (a Attachment) Stored() (data []byte, mime string) {
	if len(a.Original) > 0 {
		return a.Original, a.OriginalMIME
	}
	return a.Data, a.MIME
}

// Thumbnail decodes a prepared picture, for drawing it small.
func (a Attachment) Thumbnail() (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(a.Data))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", a.Name, err)
	}
	return downscale(img, 256), nil
}

// downscale shrinks img so its long side is at most edge pixels; a smaller one is left alone.
func downscale(img image.Image, edge int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	long := max(w, h)
	if long <= edge {
		return img
	}
	nw, nh := max(1, w*edge/long), max(1, h*edge/long)
	dst := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)
	return dst
}

func isOpaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
				return false
			}
		}
	}
	return true
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
	if jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}) != nil || buf.Len() > MaxPictureSize {
		return nil, false
	}
	return buf.Bytes(), true
}

func encodePNG(img image.Image) ([]byte, bool) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if enc.Encode(&buf, img) != nil || buf.Len() > MaxPictureSize {
		return nil, false
	}
	return buf.Bytes(), true
}
