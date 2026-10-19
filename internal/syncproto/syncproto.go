// Package syncproto holds the wire types of M11 sync and the synced columns of every entity. The
// server (S2a) and the clients (S2b, S3) share it, so the two sides cannot disagree about a name.
package syncproto

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
type Mutation struct {
	ID     int64          `json:"id"`
	Entity string         `json:"entity"`
	RowID  int64          `json:"row_id"`
	Op     string         `json:"op"`
	Data   map[string]any `json:"data,omitempty"`
}

// CreateDeviceRequest is POST /v1/devices.
type CreateDeviceRequest struct {
	Name string `json:"name"`
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
