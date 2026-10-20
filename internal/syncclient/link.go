package syncclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// Link registers this database as a device of the server's workspace and does the first sync.
//
// A device with data on an empty workspace seeds the workspace: one local transaction turns every
// existing row of the five synced tables into a put in the outbox (projects, categories,
// milestones, tasks, reports; parents first), the ordinary push path uploads the snapshot, and
// then the server is told the seeding is done. A device without data on a workspace that already
// has rows joins it and pulls everything. A joining device that already has user rows gets
// ErrLocalData and nothing local changes. A workspace another device is still seeding answers 409
// and Link returns ErrSeeding; a device that has user rows sends resume_seeding and takes the
// seeding over instead.
func (c *Client) Link(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("syncclient: device name is required")
	}
	if _, found, err := c.core.SyncState(ctx, stateDeviceID); err != nil {
		return err
	} else if found {
		return errors.New("syncclient: this database is already linked")
	}

	hasData, err := c.core.SyncHasLocalData(ctx)
	if err != nil {
		return err
	}
	// Only a device that has something to seed asks to take the seeding over.
	dev, err := c.register(ctx, name, hasData)
	if err != nil {
		if isStatus(err, http.StatusConflict) {
			return ErrSeeding
		}
		return err
	}
	switch {
	case dev.Seeding:
		if err := c.startSeeding(ctx, dev.DeviceID); err != nil {
			return err
		}
		if err := c.pushAll(ctx); err != nil {
			return err
		}
		if err := c.endSeeding(ctx); err != nil {
			return err
		}
	case hasData:
		return ErrLocalData
	default:
		// Probe before writing anything: a workspace another device is seeding answers 409, and
		// the joiner must be left unlinked so that a retry can start cleanly.
		if _, err := c.pull(ctx, dev.DeviceID, 0, 1); err != nil {
			if isStatus(err, http.StatusConflict) {
				return ErrSeeding
			}
			return err
		}
		if err := c.startJoining(ctx, dev.DeviceID); err != nil {
			return err
		}
	}
	if err := c.pullAll(ctx); err != nil {
		return err
	}
	return c.refill(ctx)
}

// startSeeding makes the local database a linked seeder and queues its snapshot in one transaction.
func (c *Client) startSeeding(ctx context.Context, deviceID string) error {
	err := c.core.ApplyChanges(ctx, func(q store.Querier) error {
		if err := setState(ctx, q, stateDeviceID, deviceID); err != nil {
			return err
		}
		if err := setState(ctx, q, stateEnabled, "1"); err != nil {
			return err
		}
		if err := setState(ctx, q, stateCookie, "0"); err != nil {
			return err
		}
		for _, entity := range entities {
			if err := queueSnapshot(ctx, q, entity); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("seed the local snapshot: %w", err)
	}
	return nil
}

// queueSnapshot appends one put per row of entity to the outbox. It runs with applying set, so the
// rows it names never change; only the outbox and sync_state are written.
func queueSnapshot(ctx context.Context, q store.Querier, entity string) error {
	query := `INSERT INTO sync_outbox (entity, row_id, op, created_at)
		SELECT ?, id, 'put', strftime('%Y-%m-%dT%H:%M:%fZ', 'now') FROM ` + quoteIdent(entity) + ` ORDER BY id`
	if _, err := q.ExecContext(ctx, query, entity); err != nil {
		return fmt.Errorf("queue %s snapshot: %w", entity, err)
	}
	return nil
}

// startJoining links an empty database to the workspace, ready for the first pull.
func (c *Client) startJoining(ctx context.Context, deviceID string) error {
	err := c.core.ApplyChanges(ctx, func(q store.Querier) error {
		if err := setState(ctx, q, stateDeviceID, deviceID); err != nil {
			return err
		}
		if err := setState(ctx, q, stateEnabled, "1"); err != nil {
			return err
		}
		return setState(ctx, q, stateCookie, "0")
	})
	if err != nil {
		return fmt.Errorf("link the local database: %w", err)
	}
	return nil
}

// endSeeding tells the server the snapshot is uploaded.
func (c *Client) endSeeding(ctx context.Context) error {
	deviceID, err := c.deviceID(ctx)
	if err != nil {
		return err
	}
	return c.markSeeded(ctx, deviceID)
}
