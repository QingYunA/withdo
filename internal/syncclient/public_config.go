package syncclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return out, fmt.Errorf("syncclient: server URL is required")
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return out, fmt.Errorf("syncclient: invalid server URL %q", base)
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
	if out.Scopes == nil {
		out.Scopes = []string{}
	}
	return out, nil
}
