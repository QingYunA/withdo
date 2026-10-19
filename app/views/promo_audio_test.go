//go:build promo

package views

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The sound of the promo video: the scene's own cues (theme.DayPlan.SoundCues, WeekPlan.SoundCues)
// laid at the seconds the scenes began, each file played at its cue's speed (the pitch steps of the
// lands are a faster playback, as the player does it). Mixed here rather than in an ffmpeg graph of a
// hundred inputs; the result is one mono 48 kHz 16-bit WAV that render.sh encodes.

const (
	promoRate     = 48000
	promoDuration = 28.0 // seconds of the video
	// promoGainDB lifts the scene's quiet sounds (peaks about -18 dBFS in the app) so that they are
	// heard on a video: still well under full scale.
	promoGainDB = 9.0
)

// readWAV is the samples of a mono 16-bit WAV, as floats in -1 to 1.
func readWAV(path string) ([]float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	i := bytes.Index(b, []byte("data"))
	if i < 0 || len(b) < i+8 {
		return nil, fmt.Errorf("%s: no data chunk", path)
	}
	n := int(binary.LittleEndian.Uint32(b[i+4:]))
	body := b[i+8:]
	if n > len(body) {
		n = len(body)
	}
	out := make([]float64, n/2)
	for k := range out {
		out[k] = float64(int16(binary.LittleEndian.Uint16(body[2*k:]))) / 32768
	}
	return out, nil
}

// promoMix is the WAV of the whole video's sound.
func promoMix(t *testing.T, track promoTrack) []byte {
	t.Helper()
	mix := make([]float64, int(promoDuration*promoRate))
	files := map[string][]float64{}
	for _, cue := range track.Cues {
		src, ok := files[cue.Sound]
		if !ok {
			var err error
			if src, err = readWAV(filepath.Join("..", "theme", "sounds", cue.Sound+".wav")); err != nil {
				t.Fatal(err)
			}
			files[cue.Sound] = src
		}
		start := int(math.Round(cue.At * promoRate))
		// Played at speed s the file is 1/s as long, sampled s samples apart (linear interpolation).
		for k := 0; ; k++ {
			pos := float64(k) * cue.Speed
			j := int(pos)
			if j+1 >= len(src) || start+k >= len(mix) {
				break
			}
			frac := pos - float64(j)
			mix[start+k] += src[j]*(1-frac) + src[j+1]*frac
		}
	}
	gain := math.Pow(10, promoGainDB/20)
	var data bytes.Buffer
	for _, s := range mix {
		v := math.Max(-1, math.Min(1, s*gain))
		_ = binary.Write(&data, binary.LittleEndian, int16(math.Round(v*32767)))
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(36+data.Len()))
	out.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(promoRate), uint32(promoRate * 2), uint16(2), uint16(16)} {
		_ = binary.Write(&out, binary.LittleEndian, v)
	}
	out.WriteString("data")
	_ = binary.Write(&out, binary.LittleEndian, uint32(data.Len()))
	out.Write(data.Bytes())
	return out.Bytes()
}
