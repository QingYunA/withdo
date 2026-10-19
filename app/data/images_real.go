package data

import (
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/paths"
)

var _ Images = (*Real)(nil)

// defaultImageGrace is how long the files of a picture outlive its last link.
const defaultImageGrace = 2 * time.Minute

// imageChunk is how many task ids one query of ImagesOf takes.
const imageChunk = 500

func imagesDir(dataFolder string) string {
	if dataFolder == "" {
		return ""
	}
	return paths.ImagesDir(dataFolder)
}

// TaskImages implements Images.
func (r *Real) TaskImages(taskID int) ([]TaskImage, error) {
	got, err := r.ImagesOf([]int{taskID})
	return got[taskID], err
}

// ImagesOf implements Images.
func (r *Real) ImagesOf(taskIDs []int) (map[int][]TaskImage, error) {
	out := map[int][]TaskImage{}
	for start := 0; start < len(taskIDs); start += imageChunk {
		ids := make([]int64, 0, imageChunk)
		for _, id := range taskIDs[start:min(start+imageChunk, len(taskIDs))] {
			ids = append(ids, int64(id))
		}
		got, err := r.core.ListImagesOf(r.ctx(), ids)
		if err != nil {
			return nil, err
		}
		for id, links := range got {
			for _, v := range links {
				out[int(id)] = append(out[int(id)], r.images.taskImage(v))
			}
		}
	}
	return out, nil
}

// AttachImages implements Images.
func (r *Real) AttachImages(taskID int, pics []Attachment) ([]TaskImage, error) {
	r.imgMu.Lock()
	defer r.imgMu.Unlock()
	var out []TaskImage
	for _, p := range pics {
		saved, err := r.images.save(p)
		if err != nil {
			return out, err
		}
		link, err := r.core.AttachImage(r.ctx(), int64(taskID), domain.AttachImageInput{
			Image: saved.image, Name: p.Name, Width: saved.width, Height: saved.height,
		})
		if err != nil {
			return out, err
		}
		out = append(out, r.images.taskImage(*link))
	}
	return out, nil
}

// DetachImage implements Images.
func (r *Real) DetachImage(taskID int, hash string) error {
	if _, err := r.core.DetachImage(r.ctx(), int64(taskID), hash); err != nil {
		return err
	}
	r.CollectSoon()
	return nil
}

// RestoreImage implements Images.
func (r *Real) RestoreImage(v TaskImage) error {
	return r.core.RestoreImage(r.ctx(), domain.TaskImage{
		Image: domain.Image{
			Hash: v.Hash, Ext: extOf(v.MIME), MIME: v.MIME, Bytes: v.Bytes, CreatedAt: domain.FormatTime(v.AddedAt),
		},
		TaskID: int64(v.TaskID), Name: v.Name, Width: v.Width, Height: v.Height, AddedAt: domain.FormatTime(v.AddedAt),
	})
}

// CollectImages implements Images.
func (r *Real) CollectImages() (int, error) {
	r.imgMu.Lock()
	defer r.imgMu.Unlock()
	gone, err := r.core.CollectImages(r.ctx())
	for _, img := range gone {
		r.images.remove(img)
	}
	return len(gone), err
}

// CollectSoon implements Images: it collects the pictures nothing links, after the grace that Undo needs. Another call
// in the meantime moves the collection back.
func (r *Real) CollectSoon() {
	grace := r.opts.ImageGrace
	if grace == 0 {
		grace = defaultImageGrace
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sweep != nil {
		r.sweep.Stop()
	}
	r.sweep = time.AfterFunc(grace, func() { _, _ = r.CollectImages() })
}
