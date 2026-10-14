package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestReadSSE(t *testing.T) {
	stream := ": keepalive\n\n" +
		"event: changed\ndata: {}\n\n" +
		"event: reminder\ndata: {\"task_id\":3}\n\n" +
		"data: orphan without event\n\n" +
		"event: reminder\ndata: {\"task_id\":4}" // no blank line: never dispatched
	type ev struct{ name, data string }
	var got []ev
	if err := readSSE(strings.NewReader(stream), func(n, d string) { got = append(got, ev{n, d}) }); err != nil {
		t.Fatal(err)
	}
	want := []ev{{"changed", "{}"}, {"reminder", `{"task_id":3}`}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestReminderText(t *testing.T) {
	project := "Home"
	title, body := Reminder{TaskID: 7, Title: "Call mum", ProjectName: &project}.text()
	if title != "Reminder: Call mum" || body != "Home" {
		t.Fatalf("got %q, %q", title, body)
	}
	if _, body := (Reminder{TaskID: 1, Title: "x"}).text(); body != "Inbox" {
		t.Fatalf("body = %q, want Inbox", body)
	}
}

func TestNotificationIDRoundTrip(t *testing.T) {
	id := Reminder{TaskID: 42}.notificationID()
	if n, ok := taskIDFromNotification(id); !ok || n != 42 {
		t.Fatalf("round trip = %d, %v", n, ok)
	}
	for _, bad := range []string{"", "task/", "task/abc", "task/0", "other/5"} {
		if _, ok := taskIDFromNotification(bad); ok {
			t.Errorf("%q should not parse", bad)
		}
	}
}

// fakeCore serves /events: the first connection gets one reminder and is dropped, the second gets
// another and stays open.
type fakeCore struct {
	srv   *httptest.Server
	mu    sync.Mutex
	conns int
	token string
}

func newFakeCore(t *testing.T) *fakeCore {
	t.Helper()
	f := &fakeCore{token: "secret"}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" || r.URL.Query().Get("token") != f.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		f.conns++
		n := f.conns
		f.mu.Unlock()
		w.Header().Set("content-type", "text/event-stream")
		fmt.Fprint(w, "event: changed\ndata: {}\n\n")
		fmt.Fprintf(w, "event: reminder\ndata: {\"task_id\":%d,\"title\":\"t%d\",\"project_name\":null}\n\n", n, n)
		w.(http.Flusher).Flush()
		if n > 1 {
			<-r.Context().Done()
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeCore) endpoint(context.Context) (Endpoint, error) {
	u, _ := url.Parse(f.srv.URL)
	port, _ := strconv.Atoi(u.Port())
	return Endpoint{Port: port, Token: f.token}, nil
}

func TestWatcherDeliversRemindersAndReconnects(t *testing.T) {
	core := newFakeCore(t)
	got := make(chan Reminder, 4)
	w := &reminderWatcher{
		endpoint: core.endpoint,
		notify:   func(r Reminder) { got <- r },
		logf:     t.Logf,
		retry:    20 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w.run(ctx); close(done) }()

	for want := 1; want <= 2; want++ {
		select {
		case r := <-got:
			if r.TaskID != want || r.Title != fmt.Sprintf("t%d", want) {
				t.Fatalf("reminder %d = %+v", want, r)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("reminder %d never arrived", want)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not stop when its context ended")
	}
}

func TestWatcherStopsWhenThereIsNoSidecar(t *testing.T) {
	w := &reminderWatcher{
		endpoint: func(context.Context) (Endpoint, error) { return Endpoint{}, errStopped },
		notify:   func(Reminder) {},
		logf:     t.Logf,
		retry:    10 * time.Millisecond,
	}
	done := make(chan struct{})
	go func() { w.run(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("watcher kept running after the sidecar was stopped")
	}
}
