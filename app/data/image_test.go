package data

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func noise(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewSource(1))
	for i := range img.Pix {
		img.Pix[i] = byte(rng.Intn(256))
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	return img
}

func pngOf(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func sizeOf(t *testing.T, b []byte) image.Point {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return image.Pt(cfg.Width, cfg.Height)
}

func TestASmallPictureGoesOutAsItCame(t *testing.T) {
	raw := pngOf(t, image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	a, err := PrepareImage("a.png", raw)
	if err != nil || a.MIME != "image/png" || !bytes.Equal(a.Data, raw) || a.Name != "a.png" {
		t.Fatalf("got %+v, %v", a, err)
	}
}

func TestALargePictureIsDownscaledToTheLongSide(t *testing.T) {
	skipUnderRace(t)
	big := image.NewNRGBA(image.Rect(0, 0, 3200, 1800))
	a, err := PrepareImage("big.png", pngOf(t, big))
	if err != nil {
		t.Fatal(err)
	}
	if got := sizeOf(t, a.Data); got != image.Pt(1600, 900) {
		t.Errorf("size %v, want 1600x900", got)
	}
	if a.MIME != "image/png" {
		t.Errorf("mime %q", a.MIME)
	}
}

func TestAJPEGStaysAJPEGWhenItIsDownscaled(t *testing.T) {
	skipUnderRace(t)
	var b bytes.Buffer
	if err := jpeg.Encode(&b, noise(2000, 1000), nil); err != nil {
		t.Fatal(err)
	}
	a, err := PrepareImage("p.jpg", b.Bytes())
	if err != nil || a.MIME != "image/jpeg" || sizeOf(t, a.Data) != image.Pt(1600, 800) {
		t.Fatalf("got %s %v, %v", a.MIME, sizeOf(t, a.Data), err)
	}
}

func TestAGIFIsReEncodedAsPNG(t *testing.T) {
	var b bytes.Buffer
	pal := color.Palette{color.White, color.Black}
	if err := gif.Encode(&b, image.NewPaletted(image.Rect(0, 0, 20, 20), pal), nil); err != nil {
		t.Fatal(err)
	}
	a, err := PrepareImage("g.gif", b.Bytes())
	if err != nil || a.MIME != "image/png" {
		t.Fatalf("got %s, %v", a.MIME, err)
	}
}

func TestAHugeNoisyPNGFallsBackToAJPEGUnder5MB(t *testing.T) {
	skipUnderRace(t)
	a, err := PrepareImage("n.png", pngOf(t, noise(1600, 1600)))
	if err != nil {
		t.Fatal(err)
	}
	if a.MIME != "image/jpeg" || len(a.Data) > MaxPictureSize {
		t.Errorf("got %s, %d bytes", a.MIME, len(a.Data))
	}
}

func TestSomethingThatIsNotAPictureIsRefused(t *testing.T) {
	for name, raw := range map[string][]byte{"text": []byte("hello"), "empty": nil, "heic": []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic")} {
		if _, err := PrepareImage(name, raw); !errors.Is(err, ErrPictureFormat) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestReadImageFileReadsAndPrepares(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(path, pngOf(t, image.NewNRGBA(image.Rect(0, 0, 8, 8))), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := ReadImageFile(path)
	if err != nil || a.Name != "shot.png" {
		t.Fatalf("got %+v, %v", a, err)
	}
	if _, err := ReadImageFile(filepath.Join(dir, "missing.png")); err == nil {
		t.Error("a missing file read without an error")
	}
	img, err := a.Thumbnail()
	if err != nil || img.Bounds().Dx() != 8 {
		t.Errorf("thumbnail %v, %v", img, err)
	}
}

func TestAPictureRefusedByTheModelIsAProblemOfItsOwn(t *testing.T) {
	err := problemFor(errors.New(`400 {"error":{"message":"unknown variant image_url"}}`))
	if ProblemOf(err) != ProblemNoImages {
		t.Errorf("got %v for %v", ProblemOf(err), err)
	}
}

func TestTheFakeAssistantRecordsPictures(t *testing.T) {
	f := &FakeAssistant{}
	pic := Attachment{Name: "a.png", MIME: "image/png", Data: []byte{1}}
	_, _ = f.Ask(bg, "plain", Place{})
	_, _ = f.Ask(bg, "with", Place{}, pic, pic)
	got := f.Pictures()
	if len(got) != 2 || len(got[0]) != 0 || len(got[1]) != 2 || got[1][0].Name != "a.png" {
		t.Errorf("pictures %+v", got)
	}
}
