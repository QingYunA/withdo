package syncclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/QingYunA/adhd-todo/internal/store"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

const (
	// refillBelow is the number of ids an entity may have left before Sync leases a new block.
	refillBelow = 200
	// leaseCount is how many ids one lease asks for.
	leaseCount = 1000
)

// refill leases a new block for every entity with fewer than refillBelow ids left.
func (c *Client) refill(ctx context.Context) error {
	deviceID, err := c.deviceID(ctx)
	if err != nil {
		return err
	}
	for _, entity := range entities {
		left, err := c.core.SyncIDsLeft(ctx, entity)
		if err != nil {
			return err
		}
		if left >= refillBelow {
			continue
		}
		block, err := c.leaseIDs(ctx, deviceID, entity, leaseCount)
		if err != nil {
			if isStatus(err, http.StatusConflict) {
				return ErrSeeding
			}
			return err
		}
		if err := c.storeBlock(ctx, entity, block.Start, block.End); err != nil {
			return err
		}
	}
	return nil
}

// storeBlock adds a leased half-open id range to the local blocks.
func (c *Client) storeBlock(ctx context.Context, entity string, start, end int64) error {
	if _, ok := syncproto.EntityColumns(entity); !ok {
		return fmt.Errorf("syncclient: id lease names unknown entity %q", entity)
	}
	err := c.core.ApplyChanges(ctx, func(q store.Querier) error {
		if _, err := q.ExecContext(ctx,
			"INSERT INTO sync_id_blocks (entity, next, end) VALUES (?, ?, ?)", entity, start, end); err != nil {
			return fmt.Errorf("store %s id block: %w", entity, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
