//go:build ignore

// Command gen writes the Summary scene's sounds: set A "wood box" of docs/design/mockups/v21.html,
// synthesised the way that page's Web Audio code does, at the level of its volume slider at 50%
// (the slider's 0.5 times the page's 0.6). Run it from this folder:
//
//	go run gen.go
//
// The noise has a fixed seed, so a rerun writes the same bytes.
package main

import (
	"encoding/binary"
	"log"
	"math"
	"math/rand"
	"os"
)

const (
	rate   = 48000
	master = 0.5 * 0.6
	// floor is where the mockup's exponential envelopes end: they cannot reach 0.
	floor = 0.0001
)

// hz is the pitch of a note in semitones from C5.
func hz(semi float64) float64 { return 523.25 * math.Pow(2, semi/12) }

// voice is the sum of the sounds of one file.
type voice struct{ buf []float64 }

// add mixes in samples starting at the time at (seconds).
func (v *voice) add(at float64, samples []float64) {
	start := int(math.Round(at * rate))
	if need := start + len(samples); need > len(v.buf) {
		v.buf = append(v.buf, make([]float64, need-len(v.buf))...)
	}
	for i, s := range samples {
		v.buf[start+i] += s
	}
}

// envelope is the gain of the mockup's tone: from 0 up to peak over attack, then exponentially down
// to floor at dur.
func envelope(t, attack, dur, peak float64) float64 {
	if t < attack {
		return peak * t / attack
	}
	return peak * math.Pow(floor/peak, (t-attack)/(dur-attack))
}

type toneOpt struct{ to, glide, attack float64 }

// tone is an oscillator ("sine" or "triangle") of frequency f that may glide exponentially to opt.to
// over opt.glide, under a linear attack (4 ms unless set) and an exponential decay to dur.
func (v *voice) tone(kind string, f, at, dur, peak float64, opt toneOpt) {
	if opt.attack == 0 {
		opt.attack = 0.004
	}
	if opt.glide == 0 {
		opt.glide = dur
	}
	out := make([]float64, int(math.Round(dur*rate)))
	phase := 0.0
	for i := range out {
		t := float64(i) / rate
		freq := f
		if opt.to != 0 {
			freq = f * math.Pow(opt.to/f, math.Min(t/opt.glide, 1))
		}
		var s float64
		switch kind {
		case "sine":
			s = math.Sin(2 * math.Pi * phase)
		case "triangle":
			s = 2 / math.Pi * math.Asin(math.Sin(2*math.Pi*phase))
		}
		out[i] = s * envelope(t, opt.attack, dur, peak)
		phase += freq / rate
	}
	v.add(at, out)
}

// noiseBuf is the second of white noise every noise burst reads from its start, as the mockup does.
var noiseBuf = func() []float64 {
	r := rand.New(rand.NewSource(1))
	b := make([]float64, rate)
	for i := range b {
		b[i] = r.Float64()*2 - 1
	}
	return b
}()

// noise is white noise through a band-pass biquad (Web Audio's, of centre freq and quality q),
// under a gain that starts at peak and falls exponentially to floor at dur.
func (v *voice) noise(at, dur, peak, freq, q float64) {
	w0 := 2 * math.Pi * freq / rate
	alpha := math.Sin(w0) / (2 * q)
	a0 := 1 + alpha
	b0, b2 := alpha/a0, -alpha/a0
	a1, a2 := -2*math.Cos(w0)/a0, (1-alpha)/a0
	out := make([]float64, int(math.Round(dur*rate)))
	var x1, x2, y1, y2 float64
	for i := range out {
		x := noiseBuf[i]
		y := b0*x + b2*x2 - a1*y1 - a2*y2
		x2, x1, y2, y1 = x1, x, y1, y
		out[i] = y * peak * math.Pow(floor/peak, float64(i)/rate/dur)
	}
	v.add(at, out)
}

func main() {
	files := map[string]func(*voice){
		"lid": func(v *voice) {
			v.tone("triangle", 180, 0, 0.09, 0.30, toneOpt{to: 120})
			v.noise(0, 0.03, 0.10, 900, 1)
		},
		// land is the first step; the player's speed raises it up the pentatonic.
		"land": func(v *voice) {
			f := hz(-12)
			v.tone("sine", f, 0, 0.12, 0.36, toneOpt{to: f * 0.72, glide: 0.05})
			v.tone("triangle", f*2, 0, 0.05, 0.10, toneOpt{})
			v.noise(0, 0.02, 0.10, 2400, 2)
		},
		// star starts 40 ms in, so that it comes just after the tock it rides on.
		"star": func(v *voice) { v.tone("sine", 2093, 0.04, 0.28, 0.07, toneOpt{}) },
		"done": func(v *voice) {
			for k, s := range []float64{0, 4, 7} {
				at := float64(k) * 0.10
				v.tone("sine", hz(s), at, 0.55, 0.26, toneOpt{})
				v.tone("sine", hz(s)*4, at, 0.12, 0.03, toneOpt{})
			}
		},
		"tick": func(v *voice) { v.noise(0, 0.025, 0.07, 1500, 3) },
	}
	for name, build := range files {
		var v voice
		build(&v)
		if err := os.WriteFile(name+".wav", wav(v.buf), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}

// wav is mono 16-bit PCM at 48 kHz of the samples, scaled by the master gain.
func wav(samples []float64) []byte {
	data := make([]byte, 0, len(samples)*2)
	for _, s := range samples {
		n := int16(math.Round(math.Max(-1, math.Min(1, s*master)) * 32767))
		data = binary.LittleEndian.AppendUint16(data, uint16(n))
	}
	h := []byte("RIFF")
	h = binary.LittleEndian.AppendUint32(h, uint32(36+len(data)))
	h = append(h, "WAVEfmt "...)
	h = binary.LittleEndian.AppendUint32(h, 16)
	h = binary.LittleEndian.AppendUint16(h, 1) // PCM
	h = binary.LittleEndian.AppendUint16(h, 1) // mono
	h = binary.LittleEndian.AppendUint32(h, rate)
	h = binary.LittleEndian.AppendUint32(h, rate*2)
	h = binary.LittleEndian.AppendUint16(h, 2)
	h = binary.LittleEndian.AppendUint16(h, 16)
	h = append(h, "data"...)
	h = binary.LittleEndian.AppendUint32(h, uint32(len(data)))
	return append(h, data...)
}
