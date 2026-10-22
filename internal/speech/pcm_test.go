package speech

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

func sine(rate, hz float64, seconds float64) []float32 {
	out := make([]float32, int(rate*seconds))
	for i := range out {
		out[i] = float32(0.5 * math.Sin(2*math.Pi*hz*float64(i)/rate))
	}
	return out
}

// feed writes x to p in blocks of size, as the audio tap does.
func feed(p *pcm, x []float32, size int) {
	for len(x) > 0 {
		n := min(size, len(x))
		p.write(x[:n])
		x = x[n:]
	}
}

func TestEncodeWAV(t *testing.T) {
	b := EncodeWAV([]int16{1, -2, 32767})
	if len(b) != 44+6 {
		t.Fatalf("length %d", len(b))
	}
	if !bytes.Equal(b[0:4], []byte("RIFF")) || !bytes.Equal(b[8:16], []byte("WAVEfmt ")) || !bytes.Equal(b[36:40], []byte("data")) {
		t.Fatalf("header %q", b[:44])
	}
	le := binary.LittleEndian
	if le.Uint32(b[4:]) != 36+6 || le.Uint16(b[20:]) != 1 || le.Uint16(b[22:]) != 1 ||
		le.Uint32(b[24:]) != 16000 || le.Uint32(b[28:]) != 32000 || le.Uint16(b[32:]) != 2 ||
		le.Uint16(b[34:]) != 16 || le.Uint32(b[40:]) != 6 {
		t.Fatalf("fields wrong: %v", b[:44])
	}
	if int16(le.Uint16(b[44:])) != 1 || int16(le.Uint16(b[46:])) != -2 || int16(le.Uint16(b[48:])) != 32767 {
		t.Fatal("samples wrong")
	}
}

func TestResampleLength(t *testing.T) {
	for _, rate := range []float64{16000, 44100, 48000, 24000} {
		p := newPCM(rate)
		feed(p, sine(rate, 440, 2), 1024)
		want := 2 * SampleRate
		if got := len(p.samples); got < want-2 || got > want {
			t.Errorf("%v Hz: %d samples, want about %d", rate, got, want)
		}
	}
}

func TestResampleKeepsTheTone(t *testing.T) {
	// A 440 Hz tone at 48 kHz must still be 440 Hz at 16 kHz, whatever the block size is.
	for _, block := range []int{1024, 333, 4800} {
		p := newPCM(48000)
		feed(p, sine(48000, 440, 1), block)
		crossings := 0
		for i := 1; i < len(p.samples); i++ {
			if p.samples[i-1] < 0 && p.samples[i] >= 0 {
				crossings++
			}
		}
		if crossings < 438 || crossings > 441 {
			t.Errorf("block %d: %d rising zero crossings, want 440", block, crossings)
		}
		var peak int16
		for _, s := range p.samples {
			peak = max(peak, s)
		}
		if peak < 15500 || peak > 16500 { // 0.5 of full scale
			t.Errorf("block %d: peak %d", block, peak)
		}
	}
}

func TestResampleIsIndependentOfBlockSize(t *testing.T) {
	x := sine(44100, 300, 0.5)
	a, b := newPCM(44100), newPCM(44100)
	feed(a, x, 512)
	feed(b, x, 97)
	n := min(len(a.samples), len(b.samples))
	if n < 7000 {
		t.Fatalf("only %d samples", n)
	}
	for i := 0; i < n; i++ {
		if d := int(a.samples[i]) - int(b.samples[i]); d > 1 || d < -1 {
			t.Fatalf("sample %d differs: %d vs %d", i, a.samples[i], b.samples[i])
		}
	}
}

func TestStopsAtMaxDuration(t *testing.T) {
	p := newPCM(16000)
	feed(p, make([]float32, 70*16000), 16000)
	if len(p.samples) != 60*SampleRate {
		t.Fatalf("%d samples, want 60 s", len(p.samples))
	}
}

func TestFinish(t *testing.T) {
	p := newPCM(16000)
	feed(p, sine(16000, 200, 0.05), 256)
	if _, err := p.finish(); !errors.Is(err, ErrNoAudio) {
		t.Fatalf("0.05 s = %v, want ErrNoAudio", err)
	}
	p = newPCM(16000)
	feed(p, sine(16000, 200, 1), 256)
	b, err := p.finish()
	if err != nil || len(b) != 44+2*len(p.samples) {
		t.Fatalf("1 s: %d bytes, %v", len(b), err)
	}
	n := len(p.samples)
	p.write(sine(16000, 200, 1)) // after finish nothing is added
	if len(p.samples) != n {
		t.Fatal("samples added after finish")
	}
}

func TestLevel(t *testing.T) {
	if l := levelOf(make([]float32, 100)); l != 0 {
		t.Fatalf("silence = %v", l)
	}
	loud := make([]float32, 100)
	for i := range loud {
		loud[i] = 0.9
	}
	if l := levelOf(loud); l != 1 {
		t.Fatalf("loud = %v", l)
	}
	quiet := make([]float32, 100)
	for i := range quiet {
		quiet[i] = 0.02
	}
	if l := levelOf(quiet); l < 0.1 || l > 0.2 {
		t.Fatalf("quiet = %v", l)
	}
}

func TestMixToMono(t *testing.T) {
	got := mixToMono(nil, [][]float32{{1, 0.5}, {0, 0.5}})
	if len(got) != 2 || got[0] != 0.5 || got[1] != 0.5 {
		t.Fatalf("mix = %v", got)
	}
	if got := mixToMono(nil, nil); len(got) != 0 {
		t.Fatalf("no channels = %v", got)
	}
}

// The recorder must answer the read-only questions on every platform without asking the person for
// anything, and Start must not record without permission.
func TestReadOnlyQuestions(t *testing.T) {
	r := New()
	_, _ = r.Available()
	if a := r.Authorization(); a != AuthAuthorized {
		if err := r.Start(nil); err == nil {
			r.Cancel()
			t.Fatalf("Start succeeded with authorization %s", a)
		}
	}
}
