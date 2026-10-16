package data

import "fmt"

// HandOff implements Store. The Fake hands every task to Claude Code and pretends to start it.
func (f *Fake) HandOff(id int, prompt string) (HandOffResult, error) {
	f.mu.Lock()
	t, ok := f.find(id)
	if !ok {
		f.mu.Unlock()
		return HandOffResult{}, fmt.Errorf("task %d not found", id)
	}
	now := f.Now()
	t.Source, t.Status, t.HandedTo, t.HandedAt, t.CompletedAt, t.UpdatedAt = SourceAgent, Todo, "claude", &now, nil, now
	out := *t
	f.mu.Unlock()
	f.notify()
	return HandOffResult{Task: out, Agent: "claude", AgentName: "Claude Code", Launched: true}, nil
}

// TakeBack implements Store.
func (f *Fake) TakeBack(id int) (Task, error) {
	f.mu.Lock()
	t, ok := f.find(id)
	if !ok {
		f.mu.Unlock()
		return Task{}, fmt.Errorf("task %d not found", id)
	}
	t.Source, t.HandedTo, t.HandedAt, t.UpdatedAt = SourceUser, "", nil, f.Now()
	out := *t
	f.mu.Unlock()
	f.notify()
	return out, nil
}
