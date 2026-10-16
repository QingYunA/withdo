package data

import (
	"fmt"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// AnyAgent is what Task.HandedTo holds for a task handed over: any agent.
const AnyAgent = domain.AnyAgent

// HandOff implements Store. The Fake hands a task to any agent.
func (f *Fake) HandOff(id int) (Task, error) {
	f.mu.Lock()
	t, ok := f.find(id)
	if !ok {
		f.mu.Unlock()
		return Task{}, fmt.Errorf("task %d not found", id)
	}
	now := f.Now()
	t.Source, t.Status, t.HandedTo, t.HandedAt, t.CompletedAt, t.UpdatedAt = SourceAgent, Todo, AnyAgent, &now, nil, now
	out := *t
	f.mu.Unlock()
	f.notify()
	return out, nil
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

// RestoreHandOff implements Store.
func (f *Fake) RestoreHandOff(id int, agent string, at *time.Time, source Source) (Task, error) {
	f.mu.Lock()
	t, ok := f.find(id)
	if !ok {
		f.mu.Unlock()
		return Task{}, fmt.Errorf("task %d not found", id)
	}
	t.Source, t.HandedTo, t.HandedAt, t.UpdatedAt = source, agent, at, f.Now()
	out := *t
	f.mu.Unlock()
	f.notify()
	return out, nil
}
