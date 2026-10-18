package views

import (
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/app/theme"
)

// CuePlayer makes the Summary scene's sounds: theme.Voices is the real one. Stop silences what is
// still sounding.
type CuePlayer interface {
	Play(theme.Cue)
	Stop()
}

// soundtrack is the cues of the scene still to come: a timer for each, from the time the scene
// began, so they follow the plan's times and not the frames. start and stop run on the UI thread,
// the timers on goroutines of their own. The two funcs are the tests' to replace.
type soundtrack struct {
	mu     sync.Mutex
	gen    int // changes on every start and stop, so that a start that is late finds it is stale
	timers []stopper

	after func(time.Duration, func()) stopper // time.AfterFunc
	async func(func())                        // go
}

type stopper interface{ Stop() bool }

// startSound plays the cues of the scene that began at, ago (the time the page has played it), unless
// the sounds are off. The system's setting is read off the UI thread, once.
func (m *Main) startSound(v *summaryView, ago time.Duration, calm bool) {
	a := m.app
	if a.Sound == nil || m.sum.freeze != nil || !a.SoundEffects() {
		return
	}
	var cues []theme.Cue
	if v.week != nil {
		cues = v.week.SoundCues(calm)
	} else {
		cues = v.day.SoundCues(calm)
	}
	if len(cues) == 0 {
		return
	}
	if calm {
		ago = 0 // the last frame at once: the done is now, not long past
	}
	tr := &m.sum.track
	gen := tr.next()
	began := time.Now()
	tr.run(func() {
		if a.SystemSounds != nil && !a.SystemSounds() {
			return
		}
		lag := ago + time.Since(began)
		tr.schedule(gen, cues, lag, func(c theme.Cue) {
			if a.WindowKey == nil || a.WindowKey() {
				a.Sound.Play(c)
			}
		})
	})
}

// stopSound drops the cues still to come and silences what is sounding: the page was left or the
// scene starts over.
func (m *Main) stopSound() {
	if !m.sum.sounding {
		return
	}
	m.sum.sounding = false
	m.sum.track.next()
	if m.app.Sound != nil {
		m.app.Sound.Stop()
	}
}

// next forgets the cues waiting and returns the number of the new start.
func (s *soundtrack) next() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gen++
	for _, t := range s.timers {
		t.Stop()
	}
	s.timers = nil
	return s.gen
}

func (s *soundtrack) run(f func()) {
	if s.async != nil {
		s.async(f)
		return
	}
	go f()
}

// lateCue is how long past its time a cue still sounds: the first frame of a scene comes a little
// after it began. A cue later than that is dropped rather than played on top of the others.
const lateCue = 100 * time.Millisecond

// schedule sets a timer for each cue, lag after the scene began; a start that is no longer the
// current one sets none.
func (s *soundtrack) schedule(gen int, cues []theme.Cue, lag time.Duration, play func(theme.Cue)) {
	after := s.after
	if after == nil {
		after = func(d time.Duration, f func()) stopper { return time.AfterFunc(d, f) }
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if gen != s.gen {
		return
	}
	for _, c := range cues {
		if c.At+lateCue < lag {
			continue
		}
		s.timers = append(s.timers, after(max(c.At-lag, 0), func() { play(c) }))
	}
}
