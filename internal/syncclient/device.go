package syncclient

import (
	"context"
	"net/http"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// Device is one device of this account's workspace as the server lists it. LastSeenAt is when the
// device last pushed or pulled, and nil before its first sync.
type Device = syncproto.Device

// Devices lists the account's devices, the one that synced most recently first. It reads the
// server, so a Mac that is offline cannot show the list.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	var out syncproto.DevicesResponse
	if err := c.do(ctx, http.MethodGet, "/v1/devices", nil, &out); err != nil {
		return nil, err
	}
	if out.Devices == nil {
		out.Devices = []Device{}
	}
	return out.Devices, nil
}
