package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// Image is a picture kept for tasks. The files live outside the database, named by Hash (the
// SHA-256 of the stored bytes, in hex): images/<Hash>.<Ext>, with images/<Hash>.thumb.png beside it.
type Image struct {
	Hash      string `json:"hash"`
	Ext       string `json:"ext"`
	MIME      string `json:"mime"`
	Bytes     int64  `json:"bytes"`
	CreatedAt string `json:"created_at"`
}

// TaskImage is a picture as one task has it.
type TaskImage struct {
	Image
	TaskID  int64  `json:"task_id"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	AddedAt string `json:"added_at"`
}

// AttachImageInput is a picture to link to a task. The files of Image are already in place.
type AttachImageInput struct {
	Image  Image // CreatedAt is the time of the call
	Name   string
	Width  int
	Height int
}

const taskImageColumns = `i.hash, i.ext, i.mime, i.bytes, i.created_at, t.task_id, t.name, t.width, t.height, t.added_at`

func scanTaskImage(s scanner) (TaskImage, error) {
	var v TaskImage
	err := s.Scan(&v.Hash, &v.Ext, &v.MIME, &v.Bytes, &v.CreatedAt, &v.TaskID, &v.Name, &v.Width, &v.Height, &v.AddedAt)
	return v, err
}

// AttachImage links a picture to a task. Attaching the same content to the same task again changes
// nothing and returns the link as it is. A task that does not exist is a not-found error.
func (c *Core) AttachImage(ctx context.Context, taskID int64, in AttachImageInput) (*TaskImage, error) {
	var out *TaskImage
	err := c.write(ctx, func(q store.Querier) error {
		t, err := getTask(ctx, q, taskID)
		if err != nil {
			return err
		}
		if t == nil {
			return notFound("task_id", "task", taskID)
		}
		now := c.stamp()
		if _, err := q.ExecContext(ctx, `INSERT OR IGNORE INTO images (hash, ext, mime, bytes, created_at) VALUES (?, ?, ?, ?, ?)`,
			in.Image.Hash, in.Image.Ext, in.Image.MIME, in.Image.Bytes, now); err != nil {
			return fmt.Errorf("insert image: %w", err)
		}
		if _, err := q.ExecContext(ctx, `INSERT OR IGNORE INTO task_images (task_id, image_hash, name, width, height, added_at)
			VALUES (?, ?, ?, ?, ?, ?)`, taskID, in.Image.Hash, in.Name, in.Width, in.Height, now); err != nil {
			return fmt.Errorf("link image: %w", err)
		}
		v, err := one(q.QueryRowContext(ctx, `SELECT `+taskImageColumns+` FROM task_images t JOIN images i ON i.hash = t.image_hash
			WHERE t.task_id = ? AND t.image_hash = ?`, taskID, in.Image.Hash), scanTaskImage)
		out = v
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("attach image to task %d: %w", taskID, err)
	}
	return out, nil
}

// ListTaskImages is a task's pictures in the order they were added.
func (c *Core) ListTaskImages(ctx context.Context, taskID int64) ([]TaskImage, error) {
	got, err := c.ListImagesOf(ctx, []int64{taskID})
	return got[taskID], err
}

// ListImagesOf reads the pictures of many tasks in one query, each task's in the order they were
// added. A task without pictures is not in the map.
func (c *Core) ListImagesOf(ctx context.Context, taskIDs []int64) (map[int64][]TaskImage, error) {
	out := map[int64][]TaskImage{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(taskIDs)), ",")
	args := make([]any, len(taskIDs))
	for i, id := range taskIDs {
		args[i] = id
	}
	rows, err := c.db.QueryContext(ctx, `SELECT `+taskImageColumns+` FROM task_images t JOIN images i ON i.hash = t.image_hash
		WHERE t.task_id IN (`+marks+`) ORDER BY t.added_at, t.rowid`, args...)
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	all, err := collect(rows, scanTaskImage)
	if err != nil {
		return nil, err
	}
	for _, v := range all {
		out[v.TaskID] = append(out[v.TaskID], v)
	}
	return out, nil
}

// DetachImage removes a task's link to a picture and reports whether there was one. The picture's
// files stay until CollectImages finds no task links them.
func (c *Core) DetachImage(ctx context.Context, taskID int64, hash string) (bool, error) {
	var removed bool
	err := c.write(ctx, func(q store.Querier) error {
		res, err := q.ExecContext(ctx, "DELETE FROM task_images WHERE task_id = ? AND image_hash = ?", taskID, hash)
		if err != nil {
			return fmt.Errorf("detach image: %w", err)
		}
		n, err := res.RowsAffected()
		removed = n > 0
		return err
	})
	return removed, err
}

// CollectImages forgets the pictures no task links to and returns them, so the caller can delete
// their files.
func (c *Core) CollectImages(ctx context.Context) ([]Image, error) {
	var gone []Image
	err := c.write(ctx, func(q store.Querier) error {
		rows, err := q.QueryContext(ctx, `SELECT hash, ext, mime, bytes, created_at FROM images
			WHERE hash NOT IN (SELECT image_hash FROM task_images)`)
		if err != nil {
			return fmt.Errorf("find unlinked images: %w", err)
		}
		gone, err = collect(rows, func(s scanner) (Image, error) {
			var v Image
			err := s.Scan(&v.Hash, &v.Ext, &v.MIME, &v.Bytes, &v.CreatedAt)
			return v, err
		})
		if err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM images WHERE hash NOT IN (SELECT image_hash FROM task_images)`); err != nil {
			return fmt.Errorf("forget images: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect images: %w", err)
	}
	return gone, nil
}

// RestoreImage puts a link back as it was, with its name and time, for the Undo of DetachImage.
// A link that is already there is left alone.
func (c *Core) RestoreImage(ctx context.Context, im TaskImage) error {
	err := c.write(ctx, func(q store.Querier) error {
		t, err := getTask(ctx, q, im.TaskID)
		if err != nil {
			return err
		}
		if t == nil {
			return notFound("task_id", "task", im.TaskID)
		}
		return relinkImage(ctx, q, im.TaskID, im)
	})
	if err != nil {
		return fmt.Errorf("restore image of task %d: %w", im.TaskID, err)
	}
	return nil
}
