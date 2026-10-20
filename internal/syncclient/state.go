package syncclient

import (
	"context"
	"fmt"
	"strconv"

	"github.com/QingYunA/adhd-todo/internal/store"
)

const (
	stateDeviceID    = "device_id"
	stateWorkspaceID = "workspace_id"
	stateCookie      = "cookie"
	stateEnabled     = "enabled"
	stateLastError   = "last_error"
)

// entities is the order a seeding snapshot writes the five synced tables: parents first. Within a
// table rows go by id, so a parent project, task or report comes before what hangs off it.
var entities = []string{
	"projects",
	"categories",
	"milestones",
	"tasks",
	"reports",
}

// deviceID reads the local device id, the key Link stores.
func (c *Client) deviceID(ctx context.Context) (string, error) {
	id, found, err := c.core.SyncState(ctx, stateDeviceID)
	if err != nil {
		return "", err
	}
	if !found || id == "" {
		return "", ErrNotLinked
	}
	return id, nil
}

// cookie reads the last pulled server version, 0 before the first pull.
func (c *Client) cookie(ctx context.Context) (int64, error) {
	raw, found, err := c.core.SyncState(ctx, stateCookie)
	if err != nil {
		return 0, err
	}
	if !found || raw == "" {
		return 0, nil
	}
	cookie, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("read sync cookie %q: %w", raw, err)
	}
	return cookie, nil
}

// setState writes one sync_state value in a write of its own.
func (c *Client) setState(ctx context.Context, key, value string) error {
	return c.core.ApplyChanges(ctx, func(q store.Querier) error {
		return setState(ctx, q, key, value)
	})
}

// setState writes one sync_state value inside the caller's transaction.
func setState(ctx context.Context, q store.Querier, key, value string) error {
	if _, err := q.ExecContext(ctx, `INSERT INTO sync_state (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
		return fmt.Errorf("set sync %s: %w", key, err)
	}
	return nil
}

// storeDevice writes the device id the server just gave this database.
func (c *Client) storeDevice(ctx context.Context, deviceID string) error {
	return c.setState(ctx, stateDeviceID, deviceID)
}

// DeviceID is the server's id for this database. ErrNotLinked says Link has not linked it.
func (c *Client) DeviceID(ctx context.Context) (string, error) { return c.deviceID(ctx) }

// Cookie is the last server version this device pulled, 0 before the first pull.
func (c *Client) Cookie(ctx context.Context) (int64, error) { return c.cookie(ctx) }

// DeviceWorkspaceID is the workspace the server put this device in, 0 when the database linked
// before S3 started storing it.
func (c *Client) DeviceWorkspaceID(ctx context.Context) (int64, error) {
	raw, found, err := c.core.SyncState(ctx, stateWorkspaceID)
	if err != nil || !found || raw == "" {
		return 0, err
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("read sync workspace %q: %w", raw, err)
	}
	return id, nil
}

// Pending is the number of local changes still waiting to upload.
func (c *Client) Pending(ctx context.Context) (int, error) { return c.core.SyncPending(ctx) }

// IDsLeft is how many row ids each entity's local blocks still hand out.
func (c *Client) IDsLeft(ctx context.Context) (map[string]int64, error) {
	out := make(map[string]int64, len(entities))
	for _, entity := range entities {
		left, err := c.core.SyncIDsLeft(ctx, entity)
		if err != nil {
			return nil, err
		}
		out[entity] = left
	}
	return out, nil
}

// RecordResult stores the outcome of a sync so a later `sync status` can report it. It does not
// touch the outbox: the applying write records nothing.
func (c *Client) RecordResult(ctx context.Context, syncErr error) {
	value := ""
	if syncErr != nil {
		value = syncErr.Error()
	}
	old, found, err := c.core.SyncState(ctx, stateLastError)
	if err != nil || (found && old == value) {
		return
	}
	_ = c.setState(ctx, stateLastError, value)
}
