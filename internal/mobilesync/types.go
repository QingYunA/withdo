// Package mobilesync provides the standalone, single-owner mobile sync store.
// It deliberately does not open or migrate the desktop domain database.
package mobilesync

import "encoding/json"

type Record struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Version int64           `json:"version"`
	Deleted bool            `json:"deleted"`
	Data    json.RawMessage `json:"data"`
}

type Mutation struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	EntityID    string          `json:"entityId"`
	BaseVersion int64           `json:"baseVersion"`
	Data        json.RawMessage `json:"data"`
}

type PushRequest struct {
	DeviceID  string     `json:"deviceId"`
	Mutations []Mutation `json:"mutations"`
}

type PushResult struct {
	MutationID string  `json:"mutationId"`
	Status     string  `json:"status"`
	Record     *Record `json:"record"`
	Error      string  `json:"error,omitempty"`
}

type PushResponse struct {
	Results []PushResult `json:"results"`
}

type Change struct {
	Seq    int64  `json:"seq"`
	Record Record `json:"record"`
}

type PullResponse struct {
	Epoch   string   `json:"epoch"`
	Cursor  int64    `json:"cursor"`
	Changes []Change `json:"changes"`
	HasMore bool     `json:"hasMore"`
}

type SnapshotResponse struct {
	Epoch   string   `json:"epoch"`
	Cursor  int64    `json:"cursor"`
	Records []Record `json:"records"`
}

type project struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	RepoSlug   string  `json:"repoSlug"`
	ArchivedAt *string `json:"archivedAt"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

type task struct {
	ID          string  `json:"id"`
	ProjectID   *string `json:"projectId"`
	ParentID    *string `json:"parentId"`
	Title       string  `json:"title"`
	Notes       string  `json:"notes"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	Source      string  `json:"source"`
	DueAt       *string `json:"dueAt"`
	RemindAt    *string `json:"remindAt"`
	Branch      string  `json:"branch"`
	PRURL       string  `json:"prUrl"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	CompletedAt *string `json:"completedAt"`
}

type report struct {
	ID        string  `json:"id"`
	TaskID    string  `json:"taskId"`
	Agent     string  `json:"agent"`
	Status    *string `json:"status"`
	Summary   string  `json:"summary"`
	NextStep  string  `json:"nextStep"`
	CreatedAt string  `json:"createdAt"`
}
