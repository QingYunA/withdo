package syncserver

import (
	"fmt"
	"net/http"
	"time"
)

// events is GET /v1/events: a Server-Sent Events stream that sends a poke whenever the
// workspace's version rises. A device sees pokes only for its own workspace.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r.Context())
	if _, err := s.lookupDevice(r.Context(), sess, r.URL.Query().Get("device_id")); err != nil {
		writeError(w, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, fmt.Errorf("response writer does not support flushing"))
		return
	}
	ch := s.hub.subscribe(sess.workspaceID)
	defer s.hub.unsubscribe(sess.workspaceID, ch)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case version := <-ch:
			if _, err := fmt.Fprintf(w, "event: poke\ndata: %d\n\n", version); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
