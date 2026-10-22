// Package syncproto holds the wire types of M11 sync and the synced columns of every entity. The
// server (S2a) and the clients (S2b, S3) share it, so the two sides cannot disagree about a name.
package syncproto

import "time"

// The five entities synced since S1. images, task_images and summaries arrive in S3.
const (
	EntityProjects   = "projects"
	EntityMilestones = "milestones"
	EntityTasks      = "tasks"
	EntityReports    = "reports"
	EntityCategories = "categories"
)

// Columns maps each entity to its synced columns, in the order migration 014's triggers name them.
// id is the row key and device-local columns (projects.repo_path, reports.cwd, tasks.reminded_at)
// are missing on purpose. TestColumnsMatchMigration014 keeps this list and the triggers together.
var Columns = map[string][]string{
	EntityProjects:   {"name", "repo_slug", "created_at", "updated_at", "archived_at", "parent_id"},
	EntityMilestones: {"project_id", "title", "due_at", "status", "created_at"},
	EntityTasks: {
		"project_id", "milestone_id", "title", "notes", "status", "priority", "due_at",
		"remind_at", "branch", "pr_url", "pr_number", "source", "created_at", "updated_at",
		"completed_at", "pr_state", "pr_checks", "pr_review", "pr_checked_at",
		"pr_closed_seen_at", "parent_id", "handed_to", "handed_at", "category_id",
	},
	EntityReports:    {"task_id", "agent", "status", "summary", "next_step", "created_at", "kind", "links"},
	EntityCategories: {"builtin", "name", "icon", "color", "position", "created_at"},
}

// EntityColumns returns the synced columns of entity and whether the server knows it.
func EntityColumns(entity string) ([]string, bool) {
	columns, ok := Columns[entity]
	return columns, ok
}

// ValidData reports whether data names only synced columns of entity. A delete carries no data.
func ValidData(entity string, data map[string]any) bool {
	columns, ok := Columns[entity]
	if !ok {
		return false
	}
	known := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		known[c] = struct{}{}
	}
	for name := range data {
		if _, ok := known[name]; !ok {
			return false
		}
	}
	return true
}

// Mutation ops.
const (
	OpPut    = "put"
	OpDelete = "delete"
)

// Mutation is one client change in a push. ID is the device's mutation id, which rises as the
// device writes; RowID is the row's integer id. Data is column → value and is empty for a delete.
// Full marks an insert: the outbox entry had no column list, so Data holds every synced column.
// A full put brings a deleted row back; a partial put on a deleted row is ignored (delete wins).
type Mutation struct {
	ID     int64          `json:"id"`
	Entity string         `json:"entity"`
	RowID  int64          `json:"row_id"`
	Op     string         `json:"op"`
	Data   map[string]any `json:"data,omitempty"`
	Full   bool           `json:"full,omitempty"`
}

// CreateDeviceRequest is POST /v1/devices. ResumeSeeding asks to take the seeding over from the
// device that is doing it; without it a registration while another device seeds is 409.
type CreateDeviceRequest struct {
	Name          string `json:"name"`
	ResumeSeeding bool   `json:"resume_seeding,omitempty"`
}

// CreateDeviceResponse returns the new device and its workspace. Empty says the workspace has no
// rows yet. Seeding says this device may push the initial snapshot: it may use any row ids and
// raises the workspace's id counters above them.
type CreateDeviceResponse struct {
	DeviceID    string `json:"device_id"`
	WorkspaceID int64  `json:"workspace_id"`
	Empty       bool   `json:"empty"`
	Seeding     bool   `json:"seeding"`
}

// SeededRequest ends seeding for a device that started a snapshot.
type SeededRequest struct {
	DeviceID string `json:"device_id"`
}

// SeededResponse reports that the workspace is no longer seeding.
type SeededResponse struct {
	Seeded bool `json:"seeded"`
}

// Device is one device of the caller's workspace as GET /v1/devices lists it. LastSeenAt is when
// the device last pushed or pulled, and nil before its first sync.
type Device struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	LastSeenAt *time.Time `json:"last_seen_at"`
}

// DevicesResponse is GET /v1/devices: the caller's devices, newest first.
type DevicesResponse struct {
	Devices []Device `json:"devices"`
}

// ConfigResponse is GET /v1/config: the public values a client needs to sign in. Auth is the
// account system to sign in with, "pocketbase"; it is empty on a server that only knows
// development tokens.
type ConfigResponse struct {
	Auth string `json:"auth,omitempty"`
}

// IDsRequest asks for a half-open block of fresh row ids for entity. Count is 1–10000.
type IDsRequest struct {
	DeviceID string `json:"device_id"`
	Entity   string `json:"entity"`
	Count    int    `json:"count"`
}

// IDsResponse is the leased half-open range [Start, End).
type IDsResponse struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// PushRequest carries a device's mutations, applied in order.
type PushRequest struct {
	DeviceID  string     `json:"device_id"`
	Mutations []Mutation `json:"mutations"`
}

// PushResponse reports the device's highest accepted mutation id.
type PushResponse struct {
	LastMutationID int64 `json:"last_mutation_id"`
}

// Change is one row in a pull. Deleted rows keep their tombstone and have no data.
type Change struct {
	Entity  string         `json:"entity"`
	RowID   int64          `json:"row_id"`
	Deleted bool           `json:"deleted"`
	Data    map[string]any `json:"data"`
	Version int64          `json:"version"`
}

// PullResponse returns the changes after the request's cookie. Cookie is the client's next
// cursor, and More says that another page is waiting.
type PullResponse struct {
	Cookie         int64    `json:"cookie"`
	LastMutationID int64    `json:"last_mutation_id"`
	Changes        []Change `json:"changes"`
	More           bool     `json:"more"`
}

// Limits shared by both sides.
const (
	// MaxPushMutations is the most mutations one push request may carry.
	MaxPushMutations = 1000
	// MaxIDsPerRequest is the largest id block POST /v1/ids leases.
	MaxIDsPerRequest = 10000
	// DefaultPullLimit and MaxPullLimit bound a pull page.
	DefaultPullLimit = 500
	MaxPullLimit     = 1000
	// MaxBodyBytes bounds a request body.
	MaxBodyBytes = 8 << 20
)

// ReferenceAction is what a delete of a reference's target does to the rows that point at it.
type ReferenceAction int

const (
	// DeleteRef deletes the referencing row, recursively.
	DeleteRef ReferenceAction = iota
	// SetNullRef clears the referencing column.
	SetNullRef
)

// Reference is one foreign key of the synced data. The server keeps these whole inside a push:
// a delete cascades, and a put that points at a deleted row is nulled or becomes a delete. The
// client relies on the server for this, so a pulled page never holds an orphan.
type Reference struct {
	Entity   string
	Column   string
	Target   string
	OnDelete ReferenceAction
}

// References lists every reference of the five synced entities, in the order the server resolves
// them. It matches the local schema and internal/domain: deleting a project or task deletes what
// hangs off it, the other references are cleared.
var References = []Reference{
	{EntityMilestones, "project_id", EntityProjects, DeleteRef},
	{EntityTasks, "project_id", EntityProjects, DeleteRef},
	{EntityReports, "task_id", EntityTasks, DeleteRef},
	{EntityTasks, "milestone_id", EntityMilestones, SetNullRef},
	{EntityTasks, "parent_id", EntityTasks, SetNullRef},
	{EntityTasks, "category_id", EntityCategories, SetNullRef},
	{EntityProjects, "parent_id", EntityProjects, SetNullRef},
}

// ReferencesTo returns the references whose target is entity.
func ReferencesTo(entity string) []Reference {
	var out []Reference
	for _, ref := range References {
		if ref.Target == entity {
			out = append(out, ref)
		}
	}
	return out
}
