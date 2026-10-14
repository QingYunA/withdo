package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Reminder is the payload of a `reminder` event of adhd-core's /events stream.
type Reminder struct {
	TaskID      int     `json:"task_id"`
	Title       string  `json:"title"`
	ProjectName *string `json:"project_name"`
}

// text is the notification's title and body.
func (r Reminder) text() (title, body string) {
	body = "Inbox"
	if r.ProjectName != nil {
		body = *r.ProjectName
	}
	return "Reminder: " + r.Title, body
}

// notificationID is the ID of the reminder's notification; it says which task a click opens.
func (r Reminder) notificationID() string { return fmt.Sprintf("task/%d", r.TaskID) }

// taskIDFromNotification reverses notificationID.
func taskIDFromNotification(id string) (int, bool) {
	rest, ok := strings.CutPrefix(id, "task/")
	if !ok {
		return 0, false
	}
	var n int
	if _, err := fmt.Sscanf(rest, "%d", &n); err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// readSSE calls onEvent for every event of a text/event-stream until r ends.
func readSSE(r io.Reader, onEvent func(name, data string)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	var name, data string
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if name != "" {
				onEvent(name, data)
			}
			name, data = "", ""
		case strings.HasPrefix(line, ":"): // keep-alive comment
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	return sc.Err()
}

// reminderWatcher follows /events and hands every due reminder to notify. The shell does this
// itself instead of the page, so reminders fire with the main window closed or hidden.
type reminderWatcher struct {
	endpoint func(context.Context) (Endpoint, error)
	notify   func(Reminder)
	logf     func(format string, args ...any)
	retry    time.Duration
}

func (w *reminderWatcher) run(ctx context.Context) {
	for ctx.Err() == nil {
		ep, err := w.endpoint(ctx)
		if err != nil {
			// The sidecar failed for good, was stopped, or ctx ended: nothing to watch.
			return
		}
		if err := w.stream(ctx, ep); err != nil && ctx.Err() == nil {
			w.logf("reminders: stream ended: %v", err)
		}
		w.sleep(ctx)
	}
}

func (w *reminderWatcher) sleep(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(w.retry):
	}
}

func (w *reminderWatcher) stream(ctx context.Context, ep Endpoint) error {
	u := fmt.Sprintf("http://127.0.0.1:%d/events?token=%s", ep.Port, url.QueryEscape(ep.Token))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", res.Status)
	}
	return readSSE(res.Body, func(name, data string) {
		if name != "reminder" {
			return
		}
		var r Reminder
		if err := json.Unmarshal([]byte(data), &r); err != nil || r.TaskID <= 0 {
			w.logf("reminders: ignoring malformed reminder %q", data)
			return
		}
		w.notify(r)
	})
}
