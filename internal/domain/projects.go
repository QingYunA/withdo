package domain

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/gitctx"
	"github.com/QingYunA/adhd-todo/internal/store"
)

const projectCols = "id, name, repo_slug, repo_path, created_at, updated_at, archived_at"

var slugPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func scanProject(s scanner) (Project, error) {
	var p Project
	err := s.Scan(&p.ID, &p.Name, &p.RepoSlug, &p.RepoPath, &p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt)
	return p, err
}

func getProject(ctx context.Context, q store.Querier, id int64) (*Project, error) {
	return one(q.QueryRowContext(ctx, "SELECT "+projectCols+" FROM projects WHERE id = ?", id), scanProject)
}

// GetProject returns the project with the given id, or nil.
func (c *Core) GetProject(ctx context.Context, id int64) (*Project, error) {
	p, err := getProject(ctx, c.db, id)
	if err != nil {
		return nil, fmt.Errorf("get project %d: %w", id, err)
	}
	return p, nil
}

// ListProjects returns every project, oldest first.
func (c *Core) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT "+projectCols+" FROM projects ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return collect(rows, scanProject)
}

// ProjectWithCounts is an active project and its task count per status.
type ProjectWithCounts struct {
	Project
	Counts map[Status]int `json:"counts"`
}

// ListActiveProjects returns non-archived projects with a task count for
// every status (zero-filled).
func (c *Core) ListActiveProjects(ctx context.Context) ([]ProjectWithCounts, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT "+projectCols+" FROM projects WHERE archived_at IS NULL ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list active projects: %w", err)
	}
	projects, err := collect(rows, scanProject)
	if err != nil {
		return nil, err
	}
	counts, err := c.statusCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProjectWithCounts, len(projects))
	for i, p := range projects {
		m := make(map[Status]int, len(Statuses))
		for _, s := range Statuses {
			m[s] = counts[p.ID][s]
		}
		out[i] = ProjectWithCounts{Project: p, Counts: m}
	}
	return out, nil
}

func (c *Core) statusCounts(ctx context.Context) (map[int64]map[Status]int, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT project_id, status, COUNT(*) FROM tasks WHERE project_id IS NOT NULL GROUP BY project_id, status")
	if err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]map[Status]int{}
	for rows.Next() {
		var id int64
		var s Status
		var n int
		if err := rows.Scan(&id, &s, &n); err != nil {
			return nil, fmt.Errorf("count tasks: %w", err)
		}
		if out[id] == nil {
			out[id] = map[Status]int{}
		}
		out[id][s] = n
	}
	return out, rows.Err()
}

func insertProject(ctx context.Context, q store.Querier, ts, name string, slug, repoPath *string) (*Project, error) {
	res, err := q.ExecContext(ctx,
		"INSERT INTO projects (name, repo_slug, repo_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		name, slug, repoPath, ts, ts)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}
	return getProject(ctx, q, id)
}

func findProject(ctx context.Context, q store.Querier, where string, arg string) (*Project, error) {
	return one(q.QueryRowContext(ctx, "SELECT "+projectCols+" FROM projects WHERE "+where, arg), scanProject)
}

// ResolvedProject is a project together with the git context it was found from.
type ResolvedProject struct {
	Project *Project
	Git     *gitctx.Info
}

// ResolveProjectFromCwd finds or creates the project that owns cwd. A GitHub
// remote keys the project by "owner/repo"; otherwise it is keyed by the main
// working tree's path, so every worktree of one repo shares a project. It
// returns (nil, nil) when cwd is not inside a git repository.
func (c *Core) ResolveProjectFromCwd(ctx context.Context, cwd string) (*ResolvedProject, error) {
	info, err := c.git(ctx, cwd)
	if errors.Is(err, gitctx.ErrNotRepo) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", cwd, err)
	}
	var project *Project
	err = c.write(ctx, func(q store.Querier) error {
		var e error
		project, e = c.findOrCreateByGit(ctx, q, info)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("resolve project for %s: %w", cwd, err)
	}
	return &ResolvedProject{Project: project, Git: info}, nil
}

func (c *Core) findOrCreateByGit(ctx context.Context, q store.Querier, info *gitctx.Info) (*Project, error) {
	var existing *Project
	var err error
	if info.RepoSlug != "" {
		existing, err = findProject(ctx, q, "repo_slug = ?", info.RepoSlug)
	} else {
		existing, err = findProject(ctx, q, "repo_slug IS NULL AND repo_path = ?", info.RepoRoot)
	}
	if err != nil {
		return nil, err
	}
	if existing == nil {
		var slug *string
		if info.RepoSlug != "" {
			slug = &info.RepoSlug
		}
		return insertProject(ctx, q, c.stamp(), info.RepoName, slug, &info.RepoRoot)
	}
	if existing.RepoPath != nil && *existing.RepoPath == info.RepoRoot {
		return existing, nil
	}
	if _, err := q.ExecContext(ctx, "UPDATE projects SET repo_path = ?, updated_at = ? WHERE id = ?",
		info.RepoRoot, c.stamp(), existing.ID); err != nil {
		return nil, fmt.Errorf("update project path: %w", err)
	}
	return getProject(ctx, q, existing.ID)
}

// UpsertProjectInput selects a project by exactly one of Cwd or RepoSlug.
type UpsertProjectInput struct {
	Cwd      string
	RepoSlug string
	// Name renames the project when non-empty.
	Name string
}

// UpsertProject creates the project for a repo (by cwd or owner/repo slug), or
// renames it when Name is given.
func (c *Core) UpsertProject(ctx context.Context, in UpsertProjectInput) (*Project, error) {
	switch {
	case in.Cwd != "" && in.RepoSlug != "":
		return nil, invalid("cwd", "provide either cwd or repo_slug, not both")
	case in.Cwd != "":
		resolved, err := c.ResolveProjectFromCwd(ctx, in.Cwd)
		if err != nil {
			return nil, err
		}
		if resolved == nil {
			return nil, invalid("cwd", "%s is not inside a git repository", in.Cwd)
		}
		return c.rename(ctx, resolved.Project, in.Name)
	case in.RepoSlug != "":
		if !slugPattern.MatchString(in.RepoSlug) {
			return nil, invalid("repo_slug", "must look like owner/repo")
		}
		project, err := c.findOrCreateBySlug(ctx, in.RepoSlug)
		if err != nil {
			return nil, err
		}
		return c.rename(ctx, project, in.Name)
	}
	return nil, invalid("cwd", "provide cwd or repo_slug")
}

func (c *Core) findOrCreateBySlug(ctx context.Context, slug string) (*Project, error) {
	var project *Project
	err := c.write(ctx, func(q store.Querier) error {
		var e error
		if project, e = findProject(ctx, q, "repo_slug = ?", slug); e != nil || project != nil {
			return e
		}
		name := slug[strings.Index(slug, "/")+1:]
		project, e = insertProject(ctx, q, c.stamp(), name, &slug, nil)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project %s: %w", slug, err)
	}
	return project, nil
}

func (c *Core) rename(ctx context.Context, p *Project, name string) (*Project, error) {
	if name == "" || name == p.Name {
		return p, nil
	}
	var out *Project
	err := c.write(ctx, func(q store.Querier) error {
		if _, err := q.ExecContext(ctx, "UPDATE projects SET name = ?, updated_at = ? WHERE id = ?",
			name, c.stamp(), p.ID); err != nil {
			return err
		}
		var err error
		out, err = getProject(ctx, q, p.ID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("rename project %d: %w", p.ID, err)
	}
	return out, nil
}

// ResolveProjectID picks a project from an explicit id or a cwd. The result
// is nil for "no project" (the inbox): no id given and cwd outside any repo.
func (c *Core) ResolveProjectID(ctx context.Context, projectID *int64, cwd string) (*int64, error) {
	if projectID != nil {
		p, err := c.GetProject(ctx, *projectID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, notFound("project_id", "project", *projectID)
		}
		return projectID, nil
	}
	if cwd == "" {
		return nil, nil
	}
	resolved, err := c.ResolveProjectFromCwd(ctx, cwd)
	if err != nil || resolved == nil {
		return nil, err
	}
	return &resolved.Project.ID, nil
}
