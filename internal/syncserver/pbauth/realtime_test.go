package pbauth_test

import (
	"bufio"
	"context"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

// pingGoroutines counts the keepalive goroutines of open realtime streams.
func pingGoroutines() int {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	return strings.Count(string(buf), "pbauth.pingRealtime(")
}

func waitPingGoroutines(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for pingGoroutines() != want {
		if time.Now().After(deadline) {
			t.Fatalf("keepalive goroutines = %d, want %d", pingGoroutines(), want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A realtime stream gets PB_PING on its own, so a proxy that closes idle streams keeps it open, and
// the keepalive ends with the connection.
func TestRealtimeStreamGetsPingsAndTheKeepaliveStops(t *testing.T) {
	defer pbauth.SetRealtimePingInterval(100 * time.Millisecond)()
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	waitPingGoroutines(t, 0) // streams of earlier tests have ended

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/realtime", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	stream := bufio.NewReader(resp.Body)
	if name, _ := sseEvent(t, stream); name != "PB_CONNECT" {
		t.Fatalf("first event = %q, want PB_CONNECT", name)
	}
	for range 2 {
		if name, data := sseEvent(t, stream); name != "PB_PING" || data != "{}" {
			t.Fatalf("event = %q %q, want PB_PING {}", name, data)
		}
	}
	waitPingGoroutines(t, 1)
	if n := len(app.SubscriptionsBroker().Clients()); n != 1 {
		t.Fatalf("registered clients = %d, want 1", n)
	}

	_ = resp.Body.Close()
	waitPingGoroutines(t, 0)
	deadline := time.Now().Add(5 * time.Second)
	for len(app.SubscriptionsBroker().Clients()) != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the client stayed registered after the stream closed")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
