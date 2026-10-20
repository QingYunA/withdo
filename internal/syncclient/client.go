// Package syncclient is the M11 sync client: it moves a local database's changes to the S2a sync
// server and the server's changes back. It talks HTTP with net/http and keeps its state in the
// local database's sync_state/outbox/id-block tables. There is no UI and no background loop yet.
package syncclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

var (
	// ErrLocalData says a joining device's database already holds user rows. S3 decides how to
	// merge them; this package leaves the database alone.
	ErrLocalData = errors.New("sync: the local database already has data")
	// ErrSeeding says another device is uploading the workspace's first snapshot. Try again later.
	ErrSeeding = errors.New("sync: another device is seeding the workspace")
	// ErrNotLinked says Sync was called on a database Link has not linked yet.
	ErrNotLinked = errors.New("sync: the database is not linked")
)

// Options configure a Client.
type Options struct {
	// BaseURL is the sync server's root, e.g. https://sync.example.com.
	BaseURL string
	// Token is the bearer token the S2a server's authenticator knows.
	Token string
	// HTTP is the transport; nil uses a client with a 30 second timeout.
	HTTP *http.Client
}

// Client is one linked local database talking to one sync server. It is safe for concurrent use
// only as far as domain.Core is; callers serialize Sync and Link.
type Client struct {
	core  *domain.Core
	base  string
	token string
	http  *http.Client
}

// New returns a client for core. It does not touch the network.
func New(core *domain.Core, opts Options) (*Client, error) {
	if core == nil {
		return nil, errors.New("syncclient: nil core")
	}
	base := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if base == "" {
		return nil, errors.New("syncclient: BaseURL is required")
	}
	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("syncclient: invalid BaseURL: %w", err)
	}
	client := opts.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{core: core, base: base, token: opts.Token, http: client}, nil
}

// statusError is a non-2xx answer from the server.
type statusError struct {
	status  int
	message string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("sync server: %s (status %d)", e.message, e.status)
}

// isStatus reports whether err is an HTTP answer with the given status.
func isStatus(err error, status int) bool {
	var se *statusError
	return errors.As(err, &se) && se.status == status
}

// do sends one JSON request and decodes the JSON answer with UseNumber, so a number above 2^53
// survives into SQLite instead of being rounded through float64.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &statusError{status: resp.StatusCode, message: errorMessage(raw)}
	}
	if out == nil {
		return nil
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return nil
}

// errorMessage reads the server's {"error": ...} body, or falls back to the raw text.
func errorMessage(raw []byte) string {
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err == nil && body.Error != "" {
		return body.Error
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "request failed"
	}
	return text
}

// register asks the server for a device in the caller's workspace. resumeSeeding asks to take the
// seeding over from another device; the server answers 409 without it while one is seeding.
func (c *Client) register(ctx context.Context, name string, resumeSeeding bool) (syncproto.CreateDeviceResponse, error) {
	var out syncproto.CreateDeviceResponse
	err := c.do(ctx, http.MethodPost, "/v1/devices",
		syncproto.CreateDeviceRequest{Name: name, ResumeSeeding: resumeSeeding}, &out)
	if err != nil {
		return out, err
	}
	if out.DeviceID == "" {
		return out, errors.New("sync server returned an empty device_id")
	}
	return out, nil
}

func (c *Client) markSeeded(ctx context.Context, deviceID string) error {
	var out syncproto.SeededResponse
	return c.do(ctx, http.MethodPost, "/v1/devices/seeded",
		syncproto.SeededRequest{DeviceID: deviceID}, &out)
}

func (c *Client) leaseIDs(ctx context.Context, deviceID, entity string, count int) (syncproto.IDsResponse, error) {
	var out syncproto.IDsResponse
	err := c.do(ctx, http.MethodPost, "/v1/ids",
		syncproto.IDsRequest{DeviceID: deviceID, Entity: entity, Count: count}, &out)
	if err != nil {
		return out, err
	}
	if out.End <= out.Start {
		return out, fmt.Errorf("sync server returned an empty id block [%d,%d)", out.Start, out.End)
	}
	return out, nil
}

func (c *Client) push(ctx context.Context, req syncproto.PushRequest) error {
	var out syncproto.PushResponse
	return c.do(ctx, http.MethodPost, "/v1/push", req, &out)
}

func (c *Client) pull(ctx context.Context, deviceID string, cookie int64, limit int) (syncproto.PullResponse, error) {
	var out syncproto.PullResponse
	q := url.Values{}
	q.Set("device_id", deviceID)
	q.Set("cookie", strconv.FormatInt(cookie, 10))
	q.Set("limit", strconv.Itoa(limit))
	err := c.do(ctx, http.MethodGet, "/v1/pull?"+q.Encode(), nil, &out)
	return out, err
}
