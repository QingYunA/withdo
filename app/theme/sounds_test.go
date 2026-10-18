package theme

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/audio"
)

func TestEverySoundHasAMono48kHzFileThatDecodes(t *testing.T) {
	for _, name := range soundFiles {
		data, err := soundFS.ReadFile("sounds/" + string(name) + ".wav")
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		d, _, err := audio.NewDecoder(bytes.NewReader(data))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if d.SampleRate() != 48000 || d.Channels() != 1 || d.Len() <= 0 {
			t.Errorf("%s: %d Hz, %d channels, %d frames", name, d.SampleRate(), d.Channels(), d.Len())
		}
	}
}

func TestEverySoundOfACueIsLoaded(t *testing.T) {
	v := LoadVoices()
	for _, s := range []Sound{SoundLid, SoundLand, SoundStar, SoundDone, SoundTick} {
		if v.sounds[s] == nil {
			t.Errorf("%s is not loaded", s)
		}
	}
}

func TestAMissingSoundIsSilenceNotAnError(t *testing.T) {
	v := &Voices{sounds: map[Sound]*audio.Sound{}}
	v.Play(Cue{Sound: SoundLand})
	v.Play(Cue{Sound: "nothing"})
	v.Stop()
}

func TestPlayedCuesCanBeStopped(t *testing.T) {
	v := LoadVoices()
	v.Play(Cue{Sound: SoundDone})
	v.Play(Cue{Sound: SoundLand, Step: 3})
	v.Stop()
	if len(v.playing) != 0 {
		t.Errorf("%d players kept", len(v.playing))
	}
}
