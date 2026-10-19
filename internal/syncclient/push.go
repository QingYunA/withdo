package syncclient

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// Sync uploads the local changes, downloads the server's changes after the cookie, and leases new
// id blocks for entities that are running out. It is the whole client loop; S3 calls it when the
// app is active and after a sync event.
func (c *Client) Sync(ctx context.Context) error {
	if _, err := c.deviceID(ctx); err != nil {
		return err
	}
	if err := c.pushAll(ctx); err != nil {
		return err
	}
	if err := c.pullAll(ctx); err != nil {
		return err
	}
	return c.refill(ctx)
}

// pushAll sends the outbox oldest first, at most one push request's worth at a time, and
// acknowledges a batch only after the server accepted it. A queued put whose row is already gone
// sends nothing; its delete follows in the same or a later batch.
func (c *Client) pushAll(ctx context.Context) error {
	deviceID, err := c.deviceID(ctx)
	if err != nil {
		return err
	}
	for {
		changes, err := c.core.Outbox(ctx, math.MaxInt64)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			return nil
		}
		if len(changes) > syncproto.MaxPushMutations {
			changes = changes[:syncproto.MaxPushMutations]
		}
		mutations, err := c.mutations(ctx, changes)
		if err != nil {
			return err
		}
		lastSeq := changes[len(changes)-1].Seq
		if len(mutations) > 0 {
			if err := c.push(ctx, syncproto.PushRequest{DeviceID: deviceID, Mutations: mutations}); err != nil {
				if isStatus(err, http.StatusConflict) {
					return ErrSeeding
				}
				return err
			}
		}
		if err := c.core.DeleteOutbox(ctx, lastSeq); err != nil {
			return err
		}
	}
}

// mutations turns outbox entries into wire mutations. A put carries the row's current values of
// the columns the entry names, or every synced column for an insert.
func (c *Client) mutations(ctx context.Context, changes []domain.SyncChange) ([]syncproto.Mutation, error) {
	out := make([]syncproto.Mutation, 0, len(changes))
	for _, ch := range changes {
		switch ch.Op {
		case syncproto.OpDelete:
			out = append(out, syncproto.Mutation{
				ID: ch.Seq, Entity: ch.Entity, RowID: ch.RowID, Op: syncproto.OpDelete,
			})
		case syncproto.OpPut:
			columns := ch.Columns
			if columns == nil {
				var ok bool
				columns, ok = syncproto.EntityColumns(ch.Entity)
				if !ok {
					return nil, fmt.Errorf("syncclient: outbox names unknown entity %q", ch.Entity)
				}
			}
			row, ok, err := c.core.SyncRow(ctx, ch.Entity, ch.RowID, columns)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			out = append(out, syncproto.Mutation{
				ID: ch.Seq, Entity: ch.Entity, RowID: ch.RowID, Op: syncproto.OpPut,
				Data: row.Data, Full: ch.Columns == nil,
			})
		default:
			return nil, fmt.Errorf("syncclient: outbox has unknown op %q", ch.Op)
		}
	}
	return out, nil
}
