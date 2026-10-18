package data

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// ThumbEdge is the long side, in pixels, of the thumbnail kept beside every picture.
const ThumbEdge = 256

// TaskImage is a picture as one task has it. The files are read by path: lists, cards and tiles
// read only Thumb, never the original.
type TaskImage struct {
	TaskID        int
	Hash          string // of the stored bytes; what two tasks sharing a picture have in common
	Name          string // the file's name when it was added
	MIME          string
	Width, Height int // of the stored picture, in pixels
	Bytes         int64
	AddedAt       time.Time
	File          string // the stored picture
	Thumb         string // its 256 px thumbnail, a PNG
}

// Images is the pictures of tasks: kept from the composer, shown in lists, cards and the
// inspector. The files live under the data folder in images/, named by the hash of their content.
type Images interface {
	// TaskImages lists a task's pictures, oldest first.
	TaskImages(taskID int) ([]TaskImage, error)
	// ImagesOf reads the pictures of many tasks at once; a task without any is not in the map.
	ImagesOf(taskIDs []int) (map[int][]TaskImage, error)
	// AttachImages keeps the pictures with a task: the original when the attachment has one,
	// else its prepared copy. A picture the task already has is left alone. It reads and writes
	// files, so call it off the UI thread.
	AttachImages(taskID int, pics []Attachment) ([]TaskImage, error)
	// DetachImage removes the task's link to a picture. The files stay a while, for RestoreImage,
	// and go once no task links them (see CollectImages).
	DetachImage(taskID int, hash string) error
	// RestoreImage links a detached picture again, as it was: the Undo of DetachImage.
	RestoreImage(TaskImage) error
	// CollectImages deletes the files of the pictures no task links to and says how many pictures
	// that was.
	CollectImages() (int, error)
	// CollectSoon collects them after the grace Undo needs; deleting a task and detaching a
	// picture do so themselves.
	CollectSoon()
}

// ErrNoImageFolder means the store has nowhere to keep pictures.
var ErrNoImageFolder = errors.New("no folder to keep pictures in")

// imageFiles are the pictures on disk: images/<hash>.<ext> and images/<hash>.thumb.png.
type imageFiles struct{ dir string }

// savedImage is a picture whose files are in place, ready to be linked.
type savedImage struct {
	image         domain.Image
	width, height int
}

func (f imageFiles) file(hash, ext string) string { return filepath.Join(f.dir, hash+"."+ext) }
func (f imageFiles) thumb(hash string) string     { return filepath.Join(f.dir, hash+".thumb.png") }

// extOf is the file extension of a stored picture's MIME type.
func extOf(mime string) string {
	switch mime {
	case "image/jpeg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	}
	return "png"
}

// save writes a picture and its thumbnail, once: a picture whose files are there is not written
// again.
func (f imageFiles) save(att Attachment) (savedImage, error) {
	if f.dir == "" {
		return savedImage{}, ErrNoImageFolder
	}
	raw, mime := att.Stored()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return savedImage{}, ErrPictureFormat
	}
	sum := sha256.Sum256(raw)
	img := domain.Image{Hash: hex.EncodeToString(sum[:]), Ext: extOf(mime), MIME: mime, Bytes: int64(len(raw))}
	if err := os.MkdirAll(f.dir, 0o755); err != nil {
		return savedImage{}, fmt.Errorf("create the pictures folder: %w", err)
	}
	if err := writeOnce(f.file(img.Hash, img.Ext), func() ([]byte, error) { return raw, nil }); err != nil {
		return savedImage{}, err
	}
	if err := writeOnce(f.thumb(img.Hash), func() ([]byte, error) { return thumbnailOf(raw) }); err != nil {
		return savedImage{}, err
	}
	return savedImage{image: img, width: cfg.Width, height: cfg.Height}, nil
}

// thumbnailOf is the PNG thumbnail of a picture: the whole picture, 256 px on its long side.
func thumbnailOf(raw []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrPictureFormat
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, downscale(src, ThumbEdge)); err != nil {
		return nil, fmt.Errorf("encode thumbnail: %w", err)
	}
	return buf.Bytes(), nil
}

// writeOnce writes the bytes make returns to path through a temporary file, unless path exists.
func writeOnce(path string, make func() ([]byte, error)) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	b, err := make()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	_, werr := tmp.Write(b)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), path)
	}
	if werr != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("write %s: %w", filepath.Base(path), werr)
	}
	return nil
}

// remove deletes a picture's files; files that are not there are fine.
func (f imageFiles) remove(img domain.Image) {
	for _, p := range []string{f.file(img.Hash, img.Ext), f.thumb(img.Hash)} {
		_ = os.Remove(p)
	}
}

// taskImage is a link as the UI reads it.
func (f imageFiles) taskImage(v domain.TaskImage) TaskImage {
	return TaskImage{
		TaskID: int(v.TaskID), Hash: v.Hash, Name: v.Name, MIME: v.MIME, Width: v.Width, Height: v.Height,
		Bytes: v.Bytes, AddedAt: parseTime(v.AddedAt), File: f.file(v.Hash, v.Ext), Thumb: f.thumb(v.Hash),
	}
}

// LoadThumbnail decodes a thumbnail file, for drawing it small. It reads a small file, but still
// off the UI thread.
func LoadThumbnail(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// LoadPicture decodes a stored picture, shrunk to at most edge pixels on the long side, for the
// viewer. It can take a while for a large original, so call it off the UI thread.
func LoadPicture(path string, edge int) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return downscale(img, edge), nil
}
