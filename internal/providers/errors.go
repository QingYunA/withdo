package providers

import (
	"context"
	"errors"
	"net"
	"strings"
)

// Friendly turns a provider failure into the neutral sentence the UI shows. It never blames and
// never includes a key. Anything it does not recognize keeps its own first line.
func Friendly(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	var netErr net.Error
	switch {
	case errors.Is(err, ErrNotConfigured):
		return "No model is connected yet."
	case errors.Is(err, ErrNoKey):
		return "This provider needs an API key."
	case errors.Is(err, context.Canceled):
		return "Cancelled."
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) || strings.Contains(low, "timeout"):
		return "The model took too long to answer."
	case strings.Contains(low, "connection refused") || strings.Contains(low, "no such host") ||
		strings.Contains(low, "dial tcp"):
		return "Couldn't reach the model. Is the address right and the server running?"
	case hasStatus(low, "401", "403", "invalid x-api-key", "invalid api key", "authentication", "unauthorized"):
		return "The provider didn't accept the API key."
	case hasStatus(low, "404", "model_not_found", "not_found"):
		return "The provider doesn't know this model or address."
	case hasStatus(low, "429", "rate limit", "overloaded", "insufficient_quota"):
		return "The provider is busy or out of quota. Try again in a moment."
	}
	return firstLine(msg)
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
