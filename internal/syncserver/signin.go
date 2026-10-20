package syncserver

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

const (
	// signInMaxBody bounds POST /v1/sign-in: a pending auth request, a login name and a password.
	signInMaxBody = 4 << 10
	// signInWindow and signInLimit are the fixed-window rate limit of one source IP and of one
	// login name's failed attempts.
	signInWindow = 10 * time.Minute
	signInLimit  = 10
	// signInMaxField is the longest login name or password the handler accepts.
	signInMaxField = 200
)

var signInAuthRequest = regexp.MustCompile(`^V2_[0-9]+$`)

// passwordSignIn is the configured in-app sign-in: the Zitadel client, the client id an auth
// request must belong to, and the rate limiter shared by every request.
type passwordSignIn struct {
	zitadel  *zitadelLogin
	clientID string
	limiter  *signInLimiter
}

// EnablePasswordSignIn turns on POST /v1/sign-in and the config flag that advertises it. issuer
// and clientID are the OIDC values clients already use; pat is the login machine user's token.
// Call it before the server starts serving.
func (s *Server) EnablePasswordSignIn(issuer, clientID, pat string) error {
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(clientID) == "" {
		return errors.New("syncserver: password sign-in needs an OIDC issuer and client id")
	}
	if strings.TrimSpace(pat) == "" {
		return errors.New("syncserver: password sign-in needs a Zitadel PAT")
	}
	if s.signIn != nil {
		return errors.New("syncserver: password sign-in is already enabled")
	}
	s.signIn = &passwordSignIn{
		zitadel:  newZitadelLogin(issuer, pat),
		clientID: strings.TrimSpace(clientID),
		limiter:  newSignInLimiter(),
	}
	s.mux.HandleFunc("POST /v1/sign-in", s.signInHandler)
	return nil
}

// SetSignInClock replaces the limiter's clock. Tests use it to cross a window without sleeping.
func (s *Server) SetSignInClock(now func() time.Time) {
	if s.signIn != nil && now != nil {
		s.signIn.limiter.setClock(now)
	}
}

// signInHandler is POST /v1/sign-in. It checks the password through Zitadel and returns the
// callback URL, so the client can finish the PKCE exchange with its own verifier.
func (s *Server) signInHandler(w http.ResponseWriter, r *http.Request) {
	login := s.signIn
	if login == nil {
		http.NotFound(w, r)
		return
	}
	retry, ok := login.limiter.allowIP(signInClientIP(r))
	if !ok {
		signInRetry(w, retry)
		return
	}
	req, err := decodeSignIn(w, r)
	if err != nil {
		signInResult(w, http.StatusBadRequest, "bad_request")
		return
	}

	loginName := strings.TrimSpace(req.LoginName)
	if retry, ok := login.limiter.allowLogin(strings.ToLower(loginName)); !ok {
		signInRetry(w, retry)
		return
	}

	ctx := r.Context()
	auth, err := login.zitadel.getAuthRequest(ctx, req.AuthRequest)
	if err != nil {
		var ze *zitadelError
		if errors.As(err, &ze) && ze.status == http.StatusNotFound {
			signInResult(w, http.StatusBadRequest, "bad_request")
			return
		}
		signInResult(w, http.StatusBadGateway, "upstream")
		return
	}
	if auth.ClientID != login.clientID {
		signInResult(w, http.StatusBadRequest, "bad_request")
		return
	}

	sess, err := login.zitadel.createSession(ctx, loginName, req.Password)
	if err != nil {
		var ze *zitadelError
		if !errors.As(err, &ze) {
			signInResult(w, http.StatusBadGateway, "upstream")
			return
		}
		switch ze.code {
		case 3, 5:
			login.limiter.failLogin(strings.ToLower(loginName))
			signInResult(w, http.StatusUnauthorized, "invalid_credentials")
		case 9:
			signInResult(w, http.StatusLocked, "locked")
		default:
			signInResult(w, http.StatusBadGateway, "upstream")
		}
		return
	}

	userID, err := login.zitadel.sessionUser(ctx, sess.SessionID)
	if err != nil || userID == "" {
		signInResult(w, http.StatusBadGateway, "upstream")
		return
	}
	methods, err := login.zitadel.authMethods(ctx, userID)
	if err != nil {
		signInResult(w, http.StatusBadGateway, "upstream")
		return
	}
	for _, method := range methods {
		if method != "AUTHENTICATION_METHOD_TYPE_PASSWORD" {
			// Zitadel completes an auth request after the password alone, so a TOTP account would
			// otherwise sign in with one factor. Delete the session before saying no.
			_ = login.zitadel.deleteSession(ctx, sess.SessionID, sess.SessionToken)
			signInResult(w, http.StatusForbidden, "second_factor")
			return
		}
	}

	callback, err := login.zitadel.completeAuthRequest(ctx, req.AuthRequest, sess)
	if err != nil || callback == "" {
		signInResult(w, http.StatusBadGateway, "upstream")
		return
	}
	writeJSON(w, http.StatusOK, syncproto.SignInResponse{CallbackURL: callback})
	log.Printf("sign-in: ok")
}

// decodeSignIn reads and validates one bounded request. The password is passed through unchanged;
// only its length and that it is not blank are checked.
func decodeSignIn(w http.ResponseWriter, r *http.Request) (syncproto.SignInRequest, error) {
	var req syncproto.SignInRequest
	r.Body = http.MaxBytesReader(w, r.Body, signInMaxBody)
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	if err := dec.Decode(&req); err != nil {
		return req, err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return req, errors.New("trailing data")
	}
	if !signInAuthRequest.MatchString(req.AuthRequest) {
		return req, errors.New("bad auth request")
	}
	loginName := strings.TrimSpace(req.LoginName)
	if loginName == "" || utf8.RuneCountInString(loginName) > signInMaxField {
		return req, errors.New("bad login name")
	}
	if strings.TrimSpace(req.Password) == "" || utf8.RuneCountInString(req.Password) > signInMaxField {
		return req, errors.New("bad password")
	}
	return req, nil
}

// signInResult writes one error and logs its classification, never the request or upstream text.
func signInResult(w http.ResponseWriter, status int, outcome string) {
	writeJSON(w, status, map[string]string{"error": outcome})
	log.Printf("sign-in: %s", outcome)
}

// signInRetry answers a rate-limited request with the seconds until the window ends.
func signInRetry(w http.ResponseWriter, retry int) {
	w.Header().Set("Retry-After", strconv.Itoa(retry))
	signInResult(w, http.StatusTooManyRequests, "too_many_attempts")
}

// signInClientIP is the source a rate limit counts. A loopback peer is the host's nginx, which
// appends the real address to X-Forwarded-For, so the last value is the source. Any other peer is
// the source itself; its header is not trusted.
func signInClientIP(r *http.Request) string {
	host := r.RemoteAddr
	if parsed, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = parsed
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
				return last
			}
		}
	}
	return host
}

// signInCounter is one fixed window.
type signInCounter struct {
	start time.Time
	count int
}

// signInLimiter counts requests per source IP and failed passwords per login name in memory. The
// mutex guards both maps; expired entries are dropped as they are seen.
type signInLimiter struct {
	mu     sync.Mutex
	now    func() time.Time
	ips    map[string]*signInCounter
	logins map[string]*signInCounter
}

func newSignInLimiter() *signInLimiter {
	return &signInLimiter{
		now:    time.Now,
		ips:    map[string]*signInCounter{},
		logins: map[string]*signInCounter{},
	}
}

func (l *signInLimiter) setClock(now func() time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.now = now
}

// allowIP counts one request from ip, including malformed ones, and refuses the request after the
// window's limit.
func (l *signInLimiter) allowIP(ip string) (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.pruneLocked(now)
	entry := l.entryLocked(l.ips, ip, now)
	entry.count++
	if entry.count > signInLimit {
		return signInRetryAfter(entry, now), false
	}
	return 0, true
}

// allowLogin refuses a login name whose failed attempts filled its window. It does not count a
// request: only an invalid_credentials answer calls failLogin.
func (l *signInLimiter) allowLogin(login string) (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.pruneLocked(now)
	entry := l.logins[login]
	if entry == nil {
		return 0, true
	}
	if entry.count >= signInLimit {
		return signInRetryAfter(entry, now), false
	}
	return 0, true
}

// failLogin counts one invalid_credentials answer for login.
func (l *signInLimiter) failLogin(login string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.pruneLocked(now)
	l.entryLocked(l.logins, login, now).count++
}

// entryLocked returns key's current window, starting one when it is absent or expired.
func (l *signInLimiter) entryLocked(m map[string]*signInCounter, key string, now time.Time) *signInCounter {
	entry := m[key]
	if entry == nil || now.Sub(entry.start) >= signInWindow {
		entry = &signInCounter{start: now}
		m[key] = entry
	}
	return entry
}

// pruneLocked drops every window that has expired, so idle keys do not accumulate.
func (l *signInLimiter) pruneLocked(now time.Time) {
	for key, entry := range l.ips {
		if now.Sub(entry.start) >= signInWindow {
			delete(l.ips, key)
		}
	}
	for key, entry := range l.logins {
		if now.Sub(entry.start) >= signInWindow {
			delete(l.logins, key)
		}
	}
}

// signInRetryAfter is the whole seconds left in entry's window, at least one.
func signInRetryAfter(entry *signInCounter, now time.Time) int {
	left := int(math.Ceil(entry.start.Add(signInWindow).Sub(now).Seconds()))
	if left < 1 {
		return 1
	}
	return left
}
