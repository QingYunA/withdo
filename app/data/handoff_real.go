package data

import "errors"

// HandOff implements Store.
func (r *Real) HandOff(id int, prompt string) (HandOffResult, error) {
	return HandOffResult{}, errors.New("hand-off is not built yet")
}

// TakeBack implements Store.
func (r *Real) TakeBack(id int) (Task, error) {
	return Task{}, errors.New("take-back is not built yet")
}
