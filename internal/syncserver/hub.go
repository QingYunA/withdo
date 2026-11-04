package syncserver

import "sync"

// hubMsg is what a /v1/events stream sends: a poke carrying the workspace's new version, or (billing)
// a notice that the account's plan changed.
type hubMsg struct {
	version int64
	billing bool
}

// hub fans a workspace's version changes out to the open /v1/events streams of its devices. A
// subscriber that is not reading is skipped instead of blocking a push.
type hub struct {
	mu   sync.Mutex
	subs map[int64]map[chan hubMsg]struct{}
}

func newHub() *hub {
	return &hub{subs: map[int64]map[chan hubMsg]struct{}{}}
}

func (h *hub) subscribe(workspaceID int64) chan hubMsg {
	ch := make(chan hubMsg, 2)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[workspaceID] == nil {
		h.subs[workspaceID] = map[chan hubMsg]struct{}{}
	}
	h.subs[workspaceID][ch] = struct{}{}
	return ch
}

func (h *hub) unsubscribe(workspaceID int64, ch chan hubMsg) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subs[workspaceID], ch)
	if len(h.subs[workspaceID]) == 0 {
		delete(h.subs, workspaceID)
	}
}

func (h *hub) publish(workspaceID, version int64) { h.send(workspaceID, hubMsg{version: version}) }

// publishBilling tells the workspace's devices that the account's plan changed.
func (h *hub) publishBilling(workspaceID int64) { h.send(workspaceID, hubMsg{billing: true}) }

func (h *hub) send(workspaceID int64, msg hubMsg) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[workspaceID] {
		select {
		case ch <- msg:
		default:
		}
	}
}
