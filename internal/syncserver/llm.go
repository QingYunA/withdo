package syncserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Limits and defaults of POST /v1/llm/chat/completions (M15, the official model).
const (
	// DefaultLLMBaseURL is the upstream's API root; its chat route is <root>/chat/completions.
	DefaultLLMBaseURL = "https://api.deepseek.com"
	// DefaultLLMModel is the model the server asks the upstream for, whatever the client names.
	DefaultLLMModel = "deepseek-chat"
	// DefaultLLMDailyTurns is how many turns a free account may take per UTC day.
	DefaultLLMDailyTurns = 20
	// DefaultLLMDailyTurnsPro is how many a Pro account (M16) may take.
	DefaultLLMDailyTurnsPro = 100
	// LLMRequestsPerTurn is the hard cap of requests that carry one turn id: an agent turn makes a
	// few model calls (a tool loop), never more than this.
	LLMRequestsPerTurn = 6
	// HeaderLLMTurn names the turn a request belongs to: one id per thing the user sent.
	HeaderLLMTurn = "X-Withdo-Turn"
	// HeaderLLMTurnsLeft is on every answer: how many turns the account has left today.
	HeaderLLMTurnsLeft = "X-Withdo-Turns-Left"
	// CodeLLMQuota and CodeLLMDisabled are the "code" of the 429 and 503 answers.
	CodeLLMQuota    = "llm_quota"
	CodeLLMDisabled = "llm_disabled"

	llmMaxBody     = 1 << 20
	llmUpstreamTO  = 90 * time.Second
	llmTurnIDMax   = 64
	llmErrBodyMax  = 4 << 10
	llmCopyBufSize = 4 << 10
)

// LLMOptions configure the official model. An empty APIKey leaves it off: the route answers 503
// llm_disabled and GET /v1/config says "llm": false.
type LLMOptions struct {
	// APIKey is the upstream's bearer key. It is never logged or returned.
	APIKey string
	// BaseURL is the upstream's API root; empty uses DefaultLLMBaseURL.
	BaseURL string
	// Model is the upstream model; empty uses DefaultLLMModel.
	Model string
	// DailyTurns is the turns per free account per UTC day; zero uses DefaultLLMDailyTurns.
	DailyTurns int
	// DailyTurnsPro is the turns per Pro account per UTC day; zero uses DefaultLLMDailyTurnsPro.
	DailyTurnsPro int
	// HTTP is the transport to the upstream; nil uses http.DefaultClient (the 90 s bound is on the
	// request's context, so a stream is bounded too).
	HTTP *http.Client
}

// llm is the official model's state. Counts live in memory like the voice limiter's: a restart
// forgets today's turns, which is a free refill and not worth a table.
type llm struct {
	auth Authenticator
	// pro says whether the account a subject maps to is on Pro; nil (billing off) means no one is.
	pro func(ctx context.Context, subject string) bool

	mu       sync.Mutex
	opts     LLMOptions
	now      func() time.Time
	day      string
	turns    map[string]map[string]int // subject -> turn id -> requests so far
	requests map[string]int            // subject -> requests today
	anon     int                       // makes the ids of requests that send none
}

func newLLM(auth Authenticator) *llm {
	return &llm{auth: auth, now: time.Now, turns: map[string]map[string]int{}, requests: map[string]int{}}
}

// EnableLLM turns on POST /v1/llm/chat/completions. Call it before serving.
func (s *Server) EnableLLM(opts LLMOptions) { s.llm.configure(opts) }

func (l *llm) configure(opts LLMOptions) {
	opts.APIKey = strings.TrimSpace(opts.APIKey)
	opts.BaseURL = strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultLLMBaseURL
	}
	opts.Model = strings.TrimSpace(opts.Model)
	if opts.Model == "" {
		opts.Model = DefaultLLMModel
	}
	if opts.DailyTurns <= 0 {
		opts.DailyTurns = DefaultLLMDailyTurns
	}
	if opts.DailyTurnsPro <= 0 {
		opts.DailyTurnsPro = DefaultLLMDailyTurnsPro
	}
	if opts.HTTP == nil {
		opts.HTTP = http.DefaultClient
	}
	l.mu.Lock()
	l.opts = opts
	l.mu.Unlock()
}

// enabled says whether the route has a key.
func (l *llm) enabled() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.opts.APIKey != ""
}

// sweep starts a new day when the UTC date moved on and returns the time. Call with mu held.
func (l *llm) sweep() time.Time {
	now := l.now().UTC()
	if day := now.Format("2006-01-02"); l.day != day {
		l.day, l.turns, l.requests = day, map[string]map[string]int{}, map[string]int{}
	}
	return now
}

// resetsAt is the start of the UTC day after now.
func resetsAt(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

// limit is the turns a day subject may take and the plan that gives it: the Pro limit for a Pro
// account, else the free one. It reads the account's plan, so call it without mu held.
func (l *llm) limit(ctx context.Context, subject string) (turns int, plan string) {
	pro := l.pro != nil && l.pro(ctx, subject)
	l.mu.Lock()
	defer l.mu.Unlock()
	if pro {
		return l.opts.DailyTurnsPro, "pro"
	}
	return l.opts.DailyTurns, "free"
}

// left is the turns subject has left today under limit. Call with mu held.
func (l *llm) left(subject string, limit int) int {
	return max(0, limit-len(l.turns[subject]))
}

// take counts one request of turn for subject against a limit of turns a day. A turn id the
// account has not used today is a new turn; a request without a usable id is a turn of its own. ok
// is false past the quota or a cap; left is the turns remaining either way, and id is the turn the
// request was counted under.
func (l *llm) take(subject, turn string, limit int) (id string, ok bool, left int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep()
	if turn == "" || len(turn) > llmTurnIDMax {
		l.anon++
		turn = "anon-" + strconv.Itoa(l.anon)
	}
	mine := l.turns[subject]
	n, known := mine[turn]
	switch {
	case l.requests[subject] >= limit*LLMRequestsPerTurn, // a client that lies about turns
		known && n >= LLMRequestsPerTurn,
		!known && len(mine) >= limit:
		return turn, false, l.left(subject, limit)
	}
	if mine == nil {
		mine = map[string]int{}
		l.turns[subject] = mine
	}
	mine[turn]++
	l.requests[subject]++
	return turn, true, l.left(subject, limit)
}

// refund gives back a request the upstream never took (it was unreachable or failed on its side)
// and returns the turns left under limit.
func (l *llm) refund(subject, turn string, limit int) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mine := l.turns[subject]; mine[turn] > 0 {
		if mine[turn]--; mine[turn] == 0 {
			delete(mine, turn)
		}
		if l.requests[subject] > 0 {
			l.requests[subject]--
		}
	}
	return l.left(subject, limit)
}

// llmError answers a refusal of the chat route in the shape OpenAI clients parse, with the
// machine-readable code at the top as well: {"error":{"message":code,"type":"withdo","code":code},
// "code":code}. A client library that finds a string in "error" reports an empty message.
func llmError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"message": code, "type": "withdo", "code": code},
		"code":  code,
	})
}

// plainError is the quota route's refusal, in the shape of the other /v1 errors.
func plainError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code, "code": code})
}

// authenticate resolves the caller; it writes the answer and returns false when there is none.
func (l *llm) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	subject, err := l.auth.Subject(r)
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		} else {
			writeError(w, err)
		}
		return "", false
	}
	return subject, true
}

// quota is GET /v1/llm/quota:
// {"limit":100,"left":86,"resets_at":"2026-11-04T00:00:00Z","plan":"pro"}. The limit is the
// account's own: Pro accounts get the Pro one.
func (l *llm) quota(w http.ResponseWriter, r *http.Request) {
	subject, ok := l.authenticate(w, r)
	if !ok {
		return
	}
	limit, plan := l.limit(r.Context(), subject)
	l.mu.Lock()
	on := l.opts.APIKey != ""
	now := l.sweep()
	left := l.left(subject, limit)
	l.mu.Unlock()
	if !on {
		plainError(w, http.StatusServiceUnavailable, CodeLLMDisabled)
		return
	}
	w.Header().Set(HeaderLLMTurnsLeft, strconv.Itoa(left))
	writeJSON(w, http.StatusOK, map[string]any{
		"limit": limit, "left": left, "resets_at": resetsAt(now).Format(time.RFC3339), "plan": plan})
}

// chat is POST /v1/llm/chat/completions: an OpenAI chat completions request in, the upstream's
// answer (streamed or not) out. Neither the request nor the reply is logged; the line says the
// account, the turns it has left and the status.
func (l *llm) chat(w http.ResponseWriter, r *http.Request) {
	subject, ok := l.authenticate(w, r)
	if !ok {
		return
	}
	l.mu.Lock()
	opts := l.opts
	l.mu.Unlock()
	if opts.APIKey == "" {
		llmError(w, http.StatusServiceUnavailable, CodeLLMDisabled)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, llmMaxBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		status := http.StatusBadRequest
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		llmError(w, status, "bad_request")
		return
	}
	payload, err := withModel(body, opts.Model)
	if err != nil {
		llmError(w, http.StatusBadRequest, "bad_request")
		return
	}
	limit, _ := l.limit(r.Context(), subject)
	turn, ok, left := l.take(subject, strings.TrimSpace(r.Header.Get(HeaderLLMTurn)), limit)
	w.Header().Set(HeaderLLMTurnsLeft, strconv.Itoa(left))
	if !ok {
		l.mu.Lock()
		retry := int(resetsAt(l.sweep()).Sub(l.now().UTC()).Seconds()) + 1
		l.mu.Unlock()
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		log.Printf("llm: account=%s turns_left=%d status=%d", subject, left, http.StatusTooManyRequests)
		llmError(w, http.StatusTooManyRequests, CodeLLMQuota)
		return
	}
	status, taken := forward(r.Context(), w, opts, payload)
	if !taken {
		left = l.refund(subject, turn, limit)
	}
	log.Printf("llm: account=%s turns_left=%d status=%d", subject, left, status)
}

// withModel sets the request's model to the server's, leaving every other field as it came.
func withModel(body []byte, model string) ([]byte, error) {
	var req map[string]json.RawMessage
	if err := json.Unmarshal(body, &req); err != nil || req == nil {
		return nil, errors.New("not a JSON object")
	}
	req["model"], _ = json.Marshal(model)
	return json.Marshal(req)
}

// forward sends the request to the upstream and copies its answer back, flushing as it goes so a
// stream reaches the client as it is made. It returns the status the client got, and whether the
// upstream took the request (false: it was unreachable or failed on its side, the turn is given
// back).
func forward(ctx context.Context, w http.ResponseWriter, opts LLMOptions, payload []byte) (status int, taken bool) {
	ctx, cancel := context.WithTimeout(ctx, llmUpstreamTO)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		llmError(w, http.StatusBadGateway, "llm_upstream")
		return http.StatusBadGateway, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	resp, err := opts.HTTP.Do(req)
	if err != nil {
		llmError(w, http.StatusBadGateway, "llm_upstream")
		return http.StatusBadGateway, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return upstreamFailure(w, resp)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(resp.StatusCode)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, llmCopyBufSize)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return resp.StatusCode, true
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			return resp.StatusCode, true
		}
	}
}

// upstreamFailure answers a refusal of the upstream. A rejection of the request itself (400, 404,
// 413, 422) goes through with its body so the client can read why and counts as a turn; a refusal
// of the server's own key or plan (401, 402, 403) must not look like the account's session
// failing, so it becomes 502 and the turn is given back; 429 stays 429 with its own code, apart
// from the account's quota, and is given back too.
func upstreamFailure(w http.ResponseWriter, resp *http.Response) (status int, taken bool) {
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusRequestEntityTooLarge, http.StatusUnprocessableEntity:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, llmErrBodyMax))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(raw)
		return resp.StatusCode, true
	case http.StatusTooManyRequests:
		llmError(w, http.StatusTooManyRequests, "llm_busy")
		return http.StatusTooManyRequests, false
	}
	llmError(w, http.StatusBadGateway, "llm_upstream")
	return http.StatusBadGateway, false
}
