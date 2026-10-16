package data

import (
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// HandOff implements Store.
func (r *Real) HandOff(id int) (Task, error) {
	t, err := r.core.HandOff(r.ctx(), int64(id))
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}

// TakeBack implements Store.
func (r *Real) TakeBack(id int) (Task, error) {
	t, err := r.core.TakeBack(r.ctx(), int64(id))
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}

// RestoreHandOff implements Store.
func (r *Real) RestoreHandOff(id int, agent string, at *time.Time, source Source) (Task, error) {
	t, err := r.core.RestoreHandOff(r.ctx(), int64(id), agent, formatTimePtr(at), domain.Source(source))
	if err != nil {
		return Task{}, err
	}
	return fromTask(*t), nil
}
