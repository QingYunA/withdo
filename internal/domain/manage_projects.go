package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/gitctx"
	"github.com/QingYunA/adhd-todo/internal/store"
)

// CreateProjectInput is a project made by hand: a name, and optionally the GitHub repository
// whose agent reports belong to it.
type CreateProjectInput struct {
	Name string
	// RepoSlug is "owner/repo"; a GitHub address pasted in its place is read as one.
	RepoSlug string
}

// NormalizeRepoSlug reads s as the "owner/repo" of a GitHub repository: it trims it, and takes the
// repository out of a pasted GitHub URL or git remote. ok is false when the result is no owner/repo.
func NormalizeRepoSlug(s string) (slug string, ok bool) {
	s = strings.TrimSpace(s)
	if parsed := gitctx.ParseRemoteURL(s); parsed != "" {
		s = parsed
	}
	return s, slugPattern.MatchString(s)
}

// CreateProject adds a project the user named. With a repository, the project is the one an agent's
// report from that repository lands in; a repository another project already uses is a conflict.
func (c *Core) CreateProject(ctx context.Context, in CreateProjectInput) (*Project, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, invalid("name", "must not be empty")
	}
	var slug *string
	if strings.TrimSpace(in.RepoSlug) != "" {
		s, ok := NormalizeRepoSlug(in.RepoSlug)
		if !ok {
			return nil, invalid("repo_slug", "must look like owner/repo")
		}
		slug = &s
	}
	var out *Project
	err := c.write(ctx, func(q store.Querier) error {
		if slug != nil {
			taken, err := findProject(ctx, q, "repo_slug = ?", *slug)
			if err != nil {
				return err
			}
			if taken != nil {
				return conflict("repo_slug", "%s is already used by project %q", *slug, taken.Name)
			}
		}
		var err error
		out, err = insertProject(ctx, q, c.stamp(), name, slug, nil)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return out, nil
}

// RenameProject gives a project a new name.
func (c *Core) RenameProject(ctx context.Context, id int64, name string) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, invalid("name", "must not be empty")
	}
	return c.editProject(ctx, id, "rename", func(q store.Querier, p *Project) error {
		_, err := q.ExecContext(ctx, "UPDATE projects SET name = ?, updated_at = ? WHERE id = ?", name, c.stamp(), p.ID)
		return err
	})
}

// ArchiveProject hides a project and its tasks from the lists. Archiving twice keeps the first time.
func (c *Core) ArchiveProject(ctx context.Context, id int64) (*Project, error) {
	return c.editProject(ctx, id, "archive", func(q store.Querier, p *Project) error {
		if p.ArchivedAt != nil {
			return nil
		}
		ts := c.stamp()
		_, err := q.ExecContext(ctx, "UPDATE projects SET archived_at = ?, updated_at = ? WHERE id = ?", ts, ts, p.ID)
		return err
	})
}

// RestoreProject brings an archived project back. A project that is not archived is left as it is.
func (c *Core) RestoreProject(ctx context.Context, id int64) (*Project, error) {
	return c.editProject(ctx, id, "restore", func(q store.Querier, p *Project) error {
		return c.unarchive(ctx, q, p)
	})
}

// unarchive clears archived_at of p, if set.
func (c *Core) unarchive(ctx context.Context, q store.Querier, p *Project) error {
	if p.ArchivedAt == nil {
		return nil
	}
	_, err := q.ExecContext(ctx, "UPDATE projects SET archived_at = NULL, updated_at = ? WHERE id = ?", c.stamp(), p.ID)
	return err
}

// unarchiveByID restores the project a task belongs to, when it is archived: an agent reporting
// there is working in it again.
func (c *Core) unarchiveByID(ctx context.Context, q store.Querier, projectID *int64) error {
	if projectID == nil {
		return nil
	}
	p, err := getProject(ctx, q, *projectID)
	if err != nil || p == nil {
		return err
	}
	return c.unarchive(ctx, q, p)
}

// editProject runs change on the project in one transaction and returns the project after it.
func (c *Core) editProject(ctx context.Context, id int64, what string, change func(store.Querier, *Project) error) (*Project, error) {
	var out *Project
	err := c.write(ctx, func(q store.Querier) error {
		p, err := getProject(ctx, q, id)
		if err != nil {
			return err
		}
		if p == nil {
			return notFound("project_id", "project", id)
		}
		if err := change(q, p); err != nil {
			return err
		}
		out, err = getProject(ctx, q, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("%s project %d: %w", what, id, err)
	}
	return out, nil
}
