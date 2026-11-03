package providers

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strings"
)

// Kind is what sort of failure a provider error is.
type Kind int

const (
	KindOther Kind = iota // anything not recognized
	KindNotConfigured
	KindNoKey
	KindCancelled
	KindTimeout
	KindUnreachable // connection refused or an unknown host
	KindKeyRejected
	KindUnknownModel
	KindRateLimited
	KindNoImages // the model does not take pictures
	// The official model's own refusals (see Official).
	KindOfficialQuota     // today's turns are used up
	KindOfficialSignedOut // no session, or the server refused it
	KindOfficialOff       // the server has no official model
)

// address matches a URL in an error's text, so the digits of its port or path are not read as a
// reply's status.
var address = regexp.MustCompile(`https?://[^\s"]+`)

// Classify says what sort of failure err is. It never looks at a key.
func Classify(err error) Kind {
	if err == nil {
		return KindOther
	}
	low := strings.ToLower(address.ReplaceAllString(err.Error(), ""))
	var netErr net.Error
	switch {
	case errors.Is(err, ErrNotConfigured):
		return KindNotConfigured
	case errors.Is(err, ErrNoKey):
		return KindNoKey
	case errors.Is(err, ErrOfficialQuota) || strings.Contains(low, "llm_quota"):
		return KindOfficialQuota
	case errors.Is(err, ErrOfficialSignedOut) || strings.Contains(low, "llm_unauthenticated"):
		return KindOfficialSignedOut
	case errors.Is(err, ErrOfficialOff) || strings.Contains(low, "llm_disabled"):
		return KindOfficialOff
	case errors.Is(err, context.Canceled):
		return KindCancelled
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) || strings.Contains(low, "timeout"):
		return KindTimeout
	case strings.Contains(low, "connection refused") || strings.Contains(low, "no such host") ||
		strings.Contains(low, "dial tcp"):
		return KindUnreachable
	case hasStatus(low, "image_url", "image input", "image content", "support image", "support vision", "vision input",
		"multimodal", "multi-modal", "images are not"):
		return KindNoImages
	case hasStatus(low, "401", "403", "invalid x-api-key", "invalid api key", "authentication", "unauthorized"):
		return KindKeyRejected
	case hasStatus(low, "404", "model_not_found", "not_found"):
		return KindUnknownModel
	case hasStatus(low, "429", "rate limit", "overloaded", "insufficient_quota"):
		return KindRateLimited
	}
	return KindOther
}

// Friendly turns a provider failure into the neutral English sentence for logs and the CLI. It
// never blames and never includes a key. Anything it does not recognize keeps its own first line.
func Friendly(err error) string {
	if err == nil {
		return ""
	}
	switch Classify(err) {
	case KindNotConfigured:
		return "No model is connected yet."
	case KindNoKey:
		return "This provider needs an API key."
	case KindCancelled:
		return "Cancelled."
	case KindTimeout:
		return "The model took too long to answer."
	case KindUnreachable:
		return "Couldn't reach the model. Is the address right and the server running?"
	case KindKeyRejected:
		return "The provider didn't accept the API key."
	case KindUnknownModel:
		return "The provider doesn't know this model or address."
	case KindNoImages:
		return "This model cannot read images."
	case KindOfficialQuota:
		return "Today's limit of the official model is reached. It comes back at midnight UTC."
	case KindOfficialSignedOut:
		return "Sign in to use the official model."
	case KindOfficialOff:
		return "This server does not offer the official model."
	case KindRateLimited:
		return "The provider is busy or out of quota. Try again in a moment."
	}
	return firstLine(err.Error())
}

func hasStatus(low string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return s
}
