package theme

import (
	"embed"
	"log"
	"sync"

	"github.com/egoist/mygo/audio"
)

// The sounds are WAV files under sounds/, mono 48 kHz 16-bit, written by sounds/gen.go. To change
// one, replace its file with one of the same name. Unlike a sprite, a file that is missing or cannot
// be decoded is silence, not a mistake in the program: the scene plays without it.
//
//go:embed sounds/*.wav
var soundFS embed.FS

// soundFiles are the sounds the scene makes, each behind the file of its name.
var soundFiles = []Sound{SoundLid, SoundLand, SoundStar, SoundDone, SoundTick}

// Voices plays the cues. A sound that has no usable file is left out.
type Voices struct {
	sounds map[Sound]*audio.Sound

	mu      sync.Mutex
	playing []*audio.Player
}

// LoadVoices decodes the sound files once.
func LoadVoices() *Voices {
	v := &Voices{sounds: map[Sound]*audio.Sound{}}
	for _, name := range soundFiles {
		data, err := soundFS.ReadFile("sounds/" + string(name) + ".wav")
		if err == nil {
			var s *audio.Sound
			if s, err = audio.DecodeSound(data); err == nil {
				v.sounds[name] = s
			}
		}
		if err != nil {
			log.Printf("sound %s: %v", name, err)
		}
	}
	return v
}

// Play makes the sound of a cue, at its speed, over whatever is still sounding. Safe to call from
// any goroutine.
func (v *Voices) Play(c Cue) {
	s := v.sounds[c.Sound]
	if s == nil {
		return
	}
	p := s.NewPlayer().SetSpeed(c.Speed())
	p.Play()
	v.mu.Lock()
	defer v.mu.Unlock()
	v.playing = append(keepPlaying(v.playing), p)
}

// Stop silences what is still sounding.
func (v *Voices) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, p := range v.playing {
		p.Stop()
	}
	v.playing = nil
}

// keepPlaying drops the players that have ended.
func keepPlaying(ps []*audio.Player) []*audio.Player {
	out := ps[:0]
	for _, p := range ps {
		if p.Playing() {
			out = append(out, p)
		}
	}
	return out
}
