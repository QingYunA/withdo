package syncclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Errors Transcribe returns for the answers the voice UI words for the person. Test them with
// errors.Is.
var (
	// ErrVoiceNotSignedIn says there is no signed-in account to speak for: the token source has
	// none, or the server refused the token.
	ErrVoiceNotSignedIn = errors.New("syncclient: sign in to use voice")
	// ErrVoiceOff says the server has no speech recognizer configured.
	ErrVoiceOff = errors.New("syncclient: voice is off on this server")
	// ErrVoiceQuota says the account used up today's recordings.
	ErrVoiceQuota = errors.New("syncclient: the voice limit for today is reached")
	// ErrVoiceNetwork says the server could not be reached.
	ErrVoiceNetwork = errors.New("syncclient: the server could not be reached")
	// ErrVoiceNoSpeech says the recognizer heard no words.
	ErrVoiceNoSpeech = errors.New("syncclient: no speech was recognized")
	// ErrVoiceFailed is every other refusal: a bad recording, or the recognizer failing.
	ErrVoiceFailed = errors.New("syncclient: the recording could not be transcribed")
)

// Transcribe uploads a WAV recording (16 kHz mono PCM16, 60 s at most) to the sync server, which
// sends it to its speech recognizer, and returns the text. It uses the signed-in session's token
// and does not retry: a recording is cheap to take again and a retry would count twice.
func (c *Client) Transcribe(ctx context.Context, wav []byte) (string, error) {
	token, err := c.token(ctx)
	if err != nil {
		if IsSignedOut(err) {
			return "", ErrVoiceNotSignedIn
		}
		return "", fmt.Errorf("%w: %v", ErrVoiceNetwork, err)
	}
	if token == "" {
		return "", ErrVoiceNotSignedIn
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/v1/transcribe", bytes.NewReader(wav))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "audio/wav")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("%w: %v", ErrVoiceNetwork, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return "", ErrVoiceNotSignedIn
	case resp.StatusCode == http.StatusServiceUnavailable:
		return "", ErrVoiceOff
	case resp.StatusCode == http.StatusTooManyRequests:
		return "", ErrVoiceQuota
	case resp.StatusCode/100 != 2:
		return "", fmt.Errorf("%w: %s (status %d)", ErrVoiceFailed, errorMessage(raw), resp.StatusCode)
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("%w: unreadable answer", ErrVoiceFailed)
	}
	text := strings.TrimSpace(out.Text)
	if text == "" {
		return "", ErrVoiceNoSpeech
	}
	return text, nil
}
