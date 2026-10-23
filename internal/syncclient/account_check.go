package syncclient

import (
	"context"
	"fmt"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// CheckDeviceAccount asks whether the signed-in account owns this database's device, by looking
// for the device id in the account's device list. It returns nil when it does, and
// ErrDeviceNotInAccount when the database is linked to another account. ErrNotLinked says the
// database was never linked, so there is nothing to check. Any other error is a failed request;
// the caller must not read it as a verdict.
func (c *Client) CheckDeviceAccount(ctx context.Context) error {
	id, err := c.deviceID(ctx)
	if err != nil {
		return err
	}
	var devices []Device
	err = retryTransient(ctx, func() error {
		var listErr error
		devices, listErr = c.Devices(ctx)
		return listErr
	})
	if err != nil {
		return err
	}
	for _, d := range devices {
		if d.ID == id {
			return nil
		}
	}
	return ErrDeviceNotInAccount
}

// Unlink forgets this database's device link, so the next sign-in registers it as a new device of
// whatever account is signed in. The rows stay: Link seeds them into an empty account, and
// LinkMerge or LinkReplace take the choice when the account has data. What goes is only the
// bookkeeping of the old link: the device id, workspace and cookie, the id blocks the old account
// leased, and the outbox. Every outbox entry names a row the new link queues again as a full put,
// or a delete the new account never saw, so no change of this database is lost. Unlinking a
// database that is not linked does nothing.
func (c *Client) Unlink(ctx context.Context) error {
	err := c.core.ApplyChanges(ctx, func(q store.Querier) error {
		for _, key := range []string{
			stateDeviceID, stateWorkspaceID, stateCookie, stateEnabled, stateLastError, stateSeedTotal,
		} {
			if _, err := q.ExecContext(ctx, "DELETE FROM sync_state WHERE key = ?", key); err != nil {
				return fmt.Errorf("forget sync %s: %w", key, err)
			}
		}
		for _, table := range []string{"sync_outbox", "sync_id_blocks"} {
			if _, err := q.ExecContext(ctx, "DELETE FROM "+table); err != nil {
				return fmt.Errorf("empty %s: %w", table, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("unlink the local database: %w", err)
	}
	return nil
}
