package syncclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// IsUnauthorized reports whether err is the server's 401: the token is wrong or expired, so the
// runner stops until it is started again.
func IsUnauthorized(err error) bool { return isStatus(err, http.StatusUnauthorized) }

// events opens the workspace's SSE stream for deviceID and calls onPoke for every poke until the
// stream ends or ctx is cancelled. It returns the reason the stream ended; io.EOF means the server
// closed a healthy stream.
func (c *Client) events(ctx context.Context, deviceID string, onPoke func()) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/v1/events?device_id="+deviceID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// The stream outlives any JSON request, so it must not inherit a client timeout.
	httpClient := c.http
	if httpClient.Timeout != 0 {
		clone := *httpClient
		clone.Timeout = 0
		httpClient = &clone
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET /v1/events: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return newStatusError(resp.StatusCode, raw)
	}

	reader := bufio.NewReader(resp.Body)
	event := ""
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				if event == "poke" {
					onPoke()
				}
				event = ""
			} else if !strings.HasPrefix(line, ":") {
				field, value, _ := strings.Cut(line, ":")
				value = strings.TrimPrefix(value, " ")
				if field == "event" {
					event = value
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return io.EOF
			}
			return fmt.Errorf("read /v1/events: %w", readErr)
		}
	}
}
