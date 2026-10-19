package syncclient

import (
	"context"
	"fmt"
	"strconv"

	"github.com/QingYunA/adhd-todo/internal/store"
)

const (
	stateDeviceID = "device_id"
	stateCookie   = "cookie"
	stateEnabled  = "enabled"
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
