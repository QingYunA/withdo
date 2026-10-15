package providers

import (
	"context"
	"errors"
	"net"
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
)

// Classify says what sort of failure err is. It never looks at a key.
func Classify(err error) Kind {
	if err == nil {
		return KindOther
	}
	low := strings.ToLower(err.Error())
	var netErr net.Error
	switch {
	case errors.Is(err, ErrNotConfigured):
		return KindNotConfigured
	case errors.Is(err, ErrNoKey):
		return KindNoKey
	case errors.Is(err, context.Canceled):
		return KindCancelled
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) || strings.Contains(low, "timeout"):
		return KindTimeout
	case strings.Contains(low, "connection refused") || strings.Contains(low, "no such host") ||
		strings.Contains(low, "dial tcp"):
		return KindUnreachable
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
