package pbauth

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

// realtimePingInterval is how often an open /api/realtime stream gets a PB_PING event. It is a
// variable so a test can shorten it.
var realtimePingInterval = 30 * time.Second

// keepRealtimeAlive makes every /api/realtime stream send a PB_PING event each realtimePingInterval.
// PocketBase writes nothing after PB_CONNECT until a subscribed event happens, and Cloudflare closes
// an HTTP stream that is idle for about 100 s; the Google sign-in waits on this stream for minutes.
// Delivering the ping through the client's channel makes the connect loop write it as an SSE event
// and reset PocketBase's own idle timer. Clients ignore events they did not subscribe to (the
// sign-in flow in syncclient reads only @oauth2, and PocketBase's JS SDK is the same).
func keepRealtimeAlive(e *core.RealtimeConnectRequestEvent) error {
	go pingRealtime(e.Request.Context(), e.Client, realtimePingInterval)
	return e.Next()
}

// pingRealtime sends PB_PING to client every interval until ctx is done, which is when the request
// ends (the client left, or PocketBase's idle or max timer cancelled it).
func pingRealtime(ctx context.Context, client subscriptions.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Send blocks until the connect loop takes the message; if the loop has already ended the
			// broker discards the client, which closes the channel and ends the send.
			client.Send(subscriptions.Message{Name: "PB_PING", Data: []byte("{}")})
		}
	}
}
