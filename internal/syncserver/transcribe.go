package syncserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Limits and defaults of POST /v1/transcribe.
const (
	// DefaultMiMoBaseURL is the recognizer's OpenAI-compatible API root.
	DefaultMiMoBaseURL = "https://api.xiaomimimo.com/v1"
	// DefaultVoiceDailyLimit is how many recordings one account may transcribe per UTC day.
	DefaultVoiceDailyLimit = 200
	// voiceMaxBytes bounds the WAV: MiMo takes at most 10 MB of base64, which is 7.5 MB of audio.
	voiceMaxBytes = 7 << 20
	// voiceMaxSeconds bounds one recording; the app stops at 60 s, the rest is slack for the clock.
	voiceMaxSeconds = 61
	voiceModel      = "mimo-v2.5-asr"
	voiceUpstreamTO = 30 * time.Second
	voiceReplyMax   = 1 << 20
)

// VoiceOptions configure POST /v1/transcribe. An empty APIKey leaves voice off: the route answers
// 503 voice_disabled.
type VoiceOptions struct {
	// APIKey is the recognizer's bearer key. It is never logged or returned.
	APIKey string
	// BaseURL is the recognizer's API root; empty uses DefaultMiMoBaseURL.
	BaseURL string
	// DailyLimit is the requests per account per UTC day; zero uses DefaultVoiceDailyLimit.
	DailyLimit int
	// HTTP is the transport to the recognizer; nil uses a client with a 30 second timeout.
	HTTP *http.Client
}

// voice is the transcription endpoint's state. Its zero options mean "voice off".
type voice struct {
	auth Authenticator

	mu    sync.Mutex
	opts  VoiceOptions
	now   func() time.Time
	used  map[string]int
	swept string
}

func newVoice(auth Authenticator) *voice {
	return &voice{auth: auth, now: time.Now, used: map[string]int{}}
}

// EnableVoice turns on POST /v1/transcribe. Call it before serving.
func (s *Server) EnableVoice(opts VoiceOptions) { s.voice.configure(opts) }

func (v *voice) configure(opts VoiceOptions) {
	opts.APIKey = strings.TrimSpace(opts.APIKey)
	opts.BaseURL = strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultMiMoBaseURL
	}
	if opts.DailyLimit <= 0 {
		opts.DailyLimit = DefaultVoiceDailyLimit
	}
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Timeout: voiceUpstreamTO}
	}
	v.mu.Lock()
	v.opts = opts
	v.mu.Unlock()
}

// take counts one request of subject today. ok is false past the limit, with the seconds until the
// UTC day ends.
func (v *voice) take(subject string) (ok bool, retry int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	now := v.now().UTC()
	day := now.Format("2006-01-02")
	if v.swept != day { // a new day: every old count goes
		v.used, v.swept = map[string]int{}, day
	}
	if v.used[subject] >= v.opts.DailyLimit {
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		return false, int(next.Sub(now).Seconds()) + 1
	}
	v.used[subject]++
	return true, 0
}

// refund gives back a request whose recognition failed on the recognizer's side.
func (v *voice) refund(subject string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.used[subject] > 0 {
		v.used[subject]--
	}
}

// handler is POST /v1/transcribe: a WAV in, {"text": "..."} out.
func (v *voice) handler(w http.ResponseWriter, r *http.Request) {
	subject, err := v.auth.Subject(r)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
			return
		}
		writeError(w, err)
		return
	}
	v.mu.Lock()
	opts := v.opts
	v.mu.Unlock()
	if opts.APIKey == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "voice_disabled"})
		return
	}
	switch mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt {
	case "audio/wav", "audio/x-wav", "audio/wave":
	default:
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "unsupported_media"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, voiceMaxBytes)
	wav, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "audio_too_large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_audio"})
		return
	}
	seconds, err := wavSeconds(wav)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad_audio"})
		return
	}
	if seconds > voiceMaxSeconds {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "audio_too_long"})
		return
	}
	ok, retry := v.take(subject)
	if !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "voice_quota"})
		return
	}
	text, err := recognize(r.Context(), opts, wav)
	if err != nil {
		v.refund(subject)
		// The classification only: the recognizer's body and our key stay out of the log.
		log.Printf("transcribe: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "transcribe_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}

// recognize sends the WAV to MiMo and returns the text of its answer. Its errors never carry the
// key, the audio or the recognizer's body.
func recognize(ctx context.Context, opts VoiceOptions, wav []byte) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model": voiceModel,
		"messages": []any{map[string]any{
			"role": "user",
			"content": []any{map[string]any{
				"type":        "input_audio",
				"input_audio": map[string]string{"data": "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(wav)},
			}},
		}},
		"asr_options": map[string]string{"language": "auto"},
	})
	if err != nil {
		return "", errors.New("encode the request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", errors.New("build the request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	resp, err := opts.HTTP.Do(req)
	if err != nil {
		return "", errors.New("recognizer unreachable")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, voiceReplyMax))
	if err != nil {
		return "", errors.New("read the recognizer's answer")
	}
	if resp.StatusCode/100 != 2 {
		return "", errors.New("recognizer answered status " + strconv.Itoa(resp.StatusCode))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return "", errors.New("recognizer answer has no choices")
	}
	text, _ := out.Choices[0].Message.Content.(string)
	return strings.TrimSpace(text), nil
}

// wavSeconds checks that b is a PCM16 WAV and returns its length in seconds. It walks the RIFF
// chunks, so a LIST chunk before data is fine.
func wavSeconds(b []byte) (float64, error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return 0, errors.New("not a WAV")
	}
	var rate, channels, bits uint32
	for pos := 12; pos+8 <= len(b); {
		id, size := string(b[pos:pos+4]), int(binary.LittleEndian.Uint32(b[pos+4:pos+8]))
		pos += 8
		switch id {
		case "fmt ":
			if size < 16 || pos+16 > len(b) || binary.LittleEndian.Uint16(b[pos:]) != 1 {
				return 0, errors.New("not PCM")
			}
			channels = uint32(binary.LittleEndian.Uint16(b[pos+2:]))
			rate = binary.LittleEndian.Uint32(b[pos+4:])
			bits = uint32(binary.LittleEndian.Uint16(b[pos+14:]))
		case "data":
			if rate == 0 || channels == 0 || bits != 16 {
				return 0, errors.New("no usable fmt chunk before data")
			}
			if size > len(b)-pos { // a streamed header may over-state the size
				size = len(b) - pos
			}
			return float64(size) / float64(rate*channels*2), nil
		}
		if size < 0 || size > len(b) {
			return 0, errors.New("bad chunk size")
		}
		pos += size + size%2
	}
	return 0, errors.New("no data chunk")
}
