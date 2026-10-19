package syncserver

import "sync"

// hub fans a workspace's version changes out to the open /v1/events streams of its devices. A
// subscriber that is not reading is skipped instead of blocking a push.
type hub struct {
	mu   sync.Mutex
	subs map[int64]map[chan int64]struct{}
}

func newHub() *hub {
	return &hub{subs: map[int64]map[chan int64]struct{}{}}
}

func (h *hub) subscribe(workspaceID int64) chan int64 {
	ch := make(chan int64, 1)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[workspaceID] == nil {
		h.subs[workspaceID] = map[chan int64]struct{}{}
	}
	h.subs[workspaceID][ch] = struct{}{}
	return ch
}

func (h *hub) unsubscribe(workspaceID int64, ch chan int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subs[workspaceID], ch)
	if len(h.subs[workspaceID]) == 0 {
		delete(h.subs, workspaceID)
	}
}

func (h *hub) publish(workspaceID, version int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[workspaceID] {
		select {
		case ch <- version:
		default:
		}
	}
}
