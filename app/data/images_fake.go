package data

import (
	"os"
	"slices"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

var _ Images = (*Fake)(nil)

// fakeImages is what the Fake keeps of pictures: the links in memory, the files in a folder.
type fakeImages struct {
	files imageFiles
	links map[int][]TaskImage
	known map[string]domain.Image // every picture whose files are in the folder
}

// SetImageDir makes dir the folder the pictures of tasks are written to. Without it the first
// picture creates a temporary folder.
func (f *Fake) SetImageDir(dir string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pics.files = imageFiles{dir: dir}
}

// TaskImages implements Images.
func (f *Fake) TaskImages(taskID int) ([]TaskImage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.pics.links[taskID]), nil
}

// ImagesOf implements Images.
func (f *Fake) ImagesOf(taskIDs []int) (map[int][]TaskImage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[int][]TaskImage{}
	for _, id := range taskIDs {
		if links := f.pics.links[id]; len(links) > 0 {
			out[id] = slices.Clone(links)
		}
	}
	return out, nil
}

// AttachImages implements Images.
func (f *Fake) AttachImages(taskID int, pics []Attachment) ([]TaskImage, error) {
	f.mu.Lock()
	if f.pics.files.dir == "" {
		dir, err := os.MkdirTemp("", "adhd-todo-demo-images-")
		if err != nil {
			f.mu.Unlock()
			return nil, err
		}
		f.pics.files = imageFiles{dir: dir}
	}
	var out []TaskImage
	for _, p := range pics {
		saved, err := f.pics.files.save(p)
		if err != nil {
			f.mu.Unlock()
			return out, err
		}
		if f.pics.known == nil {
			f.pics.known = map[string]domain.Image{}
		}
		f.pics.known[saved.image.Hash] = saved.image
		at := slices.IndexFunc(f.pics.links[taskID], func(l TaskImage) bool { return l.Hash == saved.image.Hash })
		if at < 0 {
			link := f.pics.files.taskImage(domain.TaskImage{
				Image: saved.image, TaskID: int64(taskID), Name: p.Name, Width: saved.width, Height: saved.height,
				AddedAt: domain.FormatTime(f.Now()),
			})
			f.pics.add(taskID, link)
			at = len(f.pics.links[taskID]) - 1
		}
		out = append(out, f.pics.links[taskID][at])
	}
	f.mu.Unlock()
	f.notify()
	return out, nil
}

func (p *fakeImages) add(taskID int, link TaskImage) {
	if p.links == nil {
		p.links = map[int][]TaskImage{}
	}
	p.links[taskID] = append(slices.Clone(p.links[taskID]), link)
}

// dropTask forgets a task's links; the files stay for CollectImages.
func (p *fakeImages) dropTask(id int) { delete(p.links, id) }

// DetachImage implements Images.
func (f *Fake) DetachImage(taskID int, hash string) error {
	f.mu.Lock()
	f.pics.links[taskID] = slices.DeleteFunc(slices.Clone(f.pics.links[taskID]), func(l TaskImage) bool { return l.Hash == hash })
	f.mu.Unlock()
	f.notify()
	return nil
}

// RestoreImage implements Images.
func (f *Fake) RestoreImage(v TaskImage) error {
	f.mu.Lock()
	if !slices.ContainsFunc(f.pics.links[v.TaskID], func(l TaskImage) bool { return l.Hash == v.Hash }) {
		f.pics.add(v.TaskID, v)
	}
	f.mu.Unlock()
	f.notify()
	return nil
}

// CollectImages implements Images.
func (f *Fake) CollectImages() (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	linked := map[string]bool{}
	for _, links := range f.pics.links {
		for _, l := range links {
			linked[l.Hash] = true
		}
	}
	n := 0
	for hash, img := range f.pics.known {
		if !linked[hash] {
			f.pics.files.remove(img)
			delete(f.pics.known, hash)
			n++
		}
	}
	return n, nil
}

// CollectSoon implements Images: the fake has no one to wait for Undo, so a test collects when it
// wants to.
func (f *Fake) CollectSoon() {}
