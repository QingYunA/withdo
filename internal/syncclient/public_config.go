package syncclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// FetchConfig reads GET /v1/config, the public values a client needs before it has a token. It
// needs no Client and no linked database: Settings asks for it, then signs in through the issuer
// it names.
func FetchConfig(ctx context.Context, baseURL string, httpClient *http.Client) (syncproto.ConfigResponse, error) {
	var out syncproto.ConfigResponse
	base, err := ServerURL(baseURL)
	if err != nil {
		return out, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/config", nil)
	if err != nil {
		return out, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("GET /v1/config: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return out, &statusError{status: resp.StatusCode, message: errorMessage(raw)}
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode /v1/config: %w", err)
	}
	if _, err := ServerURL(out.Issuer); out.Issuer != "" && err != nil {
		return out, fmt.Errorf("the server names an issuer that is not https: %w", err)
	}
	if out.Scopes == nil {
		out.Scopes = []string{}
	}
	return out, nil
}

// ServerURL checks an address the client is about to trust and returns it without a trailing
// slash. It must be https: over plain http anyone on the network could answer /v1/config with an
// issuer of their own and collect the sign-in. Only a loopback host (a server on this machine, as
// in tests) may use http.
func ServerURL(raw string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return "", errors.New("the server address is empty")
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return "", errors.New("the server address must be a URL such as https://todo.example.com")
	}
	switch {
	case u.Scheme == "https":
	case u.Scheme == "http" && loopbackHost(u.Hostname()):
	default:
		return "", errors.New("the server address must start with https://")
	}
	return base, nil
}

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
