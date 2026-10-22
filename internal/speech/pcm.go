package speech

import (
	"encoding/binary"
	"math"
	"sync"
)

// SampleRate is the rate of every recording: what the recognizer wants, and plenty for speech.
const SampleRate = 16000

const maxSamples = int(MaxDuration/1e9) * SampleRate

// minSamples is a fifth of a second.
const minSamples = SampleRate / 5

// pcm turns the microphone's buffers (any rate, any number of channels) into one 16 kHz mono PCM16
// stream. It resamples by linear interpolation, which is plenty for speech at 16 kHz: the content
// above 8 kHz is not wanted and the recognizer does not care about the slight aliasing.
type pcm struct {
	mu      sync.Mutex
	ratio   float64 // input samples per output sample
	next    float64 // position of the next output sample, relative to the next block's first input sample
	prev    float32 // the last input sample of the previous block, index -1 of the next
	samples []int16
	closed  bool
}

func newPCM(inputRate float64) *pcm {
	return &pcm{ratio: inputRate / SampleRate}
}

// write adds one block of mono input samples. It returns the block's loudness, 0 to 1, and drops
// the samples after MaxDuration or after close.
func (p *pcm) write(x []float32) float32 {
	if len(x) == 0 {
		return 0
	}
	level := levelOf(x)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return level
	}
	n := float64(len(x))
	at := func(i int) float32 {
		if i < 0 {
			return p.prev
		}
		return x[i]
	}
	for p.next < n-1 && len(p.samples) < maxSamples {
		i := int(math.Floor(p.next))
		frac := float32(p.next - float64(i))
		v := at(i)*(1-frac) + at(i+1)*frac
		p.samples = append(p.samples, toPCM16(v))
		p.next += p.ratio
	}
	p.next -= n
	p.prev = x[len(x)-1]
	return level
}

// finish stops the stream and returns the recording as a WAV, or ErrNoAudio when it is too short.
func (p *pcm) finish() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if len(p.samples) < minSamples {
		return nil, ErrNoAudio
	}
	return EncodeWAV(p.samples), nil
}

func (p *pcm) cancel() {
	p.mu.Lock()
	p.closed, p.samples = true, nil
	p.mu.Unlock()
}

func toPCM16(v float32) int16 {
	switch {
	case v >= 1:
		return math.MaxInt16
	case v <= -1:
		return math.MinInt16
	}
	return int16(math.Round(float64(v) * 32767))
}

// levelOf is the loudness of a block for the waveform: its RMS, scaled so ordinary speech reaches
// about half and shouting the top.
func levelOf(x []float32) float32 {
	var sum float64
	for _, v := range x {
		sum += float64(v) * float64(v)
	}
	rms := math.Sqrt(sum / float64(len(x)))
	return float32(math.Min(1, rms*8))
}

// mixToMono averages the channels of one non-interleaved buffer into dst (resized as needed).
func mixToMono(dst []float32, channels [][]float32) []float32 {
	if len(channels) == 0 {
		return dst[:0]
	}
	n := len(channels[0])
	if cap(dst) < n {
		dst = make([]float32, n)
	}
	dst = dst[:n]
	if len(channels) == 1 {
		copy(dst, channels[0])
		return dst
	}
	inv := 1 / float32(len(channels))
	for i := range dst {
		var s float32
		for _, ch := range channels {
			s += ch[i]
		}
		dst[i] = s * inv
	}
	return dst
}

// EncodeWAV wraps 16 kHz mono PCM16 samples in a 44 byte WAV header.
func EncodeWAV(samples []int16) []byte {
	dataLen := len(samples) * 2
	b := make([]byte, 44+dataLen)
	copy(b[0:], "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+dataLen))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1) // PCM
	binary.LittleEndian.PutUint16(b[22:], 1) // mono
	binary.LittleEndian.PutUint32(b[24:], SampleRate)
	binary.LittleEndian.PutUint32(b[28:], SampleRate*2) // bytes a second
	binary.LittleEndian.PutUint16(b[32:], 2)            // bytes a frame
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(dataLen))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(b[44+2*i:], uint16(s))
	}
	return b
}
