package syncclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Login runs the authorization code flow with PKCE (S256) and a random state. It opens a listener
// on 127.0.0.1 with a random port (RFC 8252), passes the authorization URL to open, and waits for
// the browser's redirect. The listener answers with a short page and stops. Nothing is stored
// unless the exchange succeeds.
func (s *OIDCSource) Login(ctx context.Context, open func(url string) error) error {
	if s == nil || s.Secrets == nil {
		return errors.New("syncclient: OIDC sign-in needs a secrets store")
	}
	if open == nil {
		return errors.New("syncclient: OIDC sign-in needs a way to open the browser")
	}
	if strings.TrimSpace(s.Issuer) == "" || strings.TrimSpace(s.ClientID) == "" {
		return errors.New("syncclient: OIDC issuer and client id are required")
	}
	state, err := randomURLToken()
	if err != nil {
		return err
	}
	verifier, err := randomURLToken()
	if err != nil {
		return err
	}
	challenge := pkceChallenge(verifier)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("syncclient: open the sign-in listener: %w", err)
	}
	redirect := "http://" + listener.Addr().String() + callbackPath

	s.mu.Lock()
	base, err := s.oauthConfigLocked(ctx)
	s.mu.Unlock()
	if err != nil {
		_ = listener.Close()
		return err
	}
	oauthConfig := *base
	oauthConfig.RedirectURL = redirect
	authURL := oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	type loginResult struct {
		token *oauth2.Token
		err   error
	}
	results := make(chan loginResult, 1)
	var finishOnce sync.Once
	finish := func(result loginResult) {
		finishOnce.Do(func() { results <- result })
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("state") != state {
			// Not our redirect: any page in the browser can reach a loopback port, so a request
			// with the wrong state is refused without ending the sign-in it would otherwise abort.
			http.Error(w, "sign-in failed: state mismatch", http.StatusBadRequest)
			return
		}
		if msg := query.Get("error"); msg != "" {
			http.Error(w, "sign-in failed", http.StatusBadRequest)
			finish(loginResult{err: fmt.Errorf("syncclient: OIDC sign-in was refused (%s)", msg)})
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(w, "sign-in failed: no code", http.StatusBadRequest)
			finish(loginResult{err: errors.New("syncclient: OIDC sign-in returned no code")})
			return
		}
		// The phone goes back to the app first; the exchange's retries cover the moment until
		// Android gives the app its network back. Errors then reach the app, not the browser.
		answered := false
		if s.DoneURL != "" {
			http.Redirect(w, r, s.DoneURL, http.StatusFound)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			answered = true
		}
		var token *oauth2.Token
		err := retryTransient(ctx, func() error {
			var exchangeErr error
			token, exchangeErr = oauthConfig.Exchange(httpContext(ctx, s.HTTP), code,
				oauth2.SetAuthURLParam("code_verifier", verifier))
			return exchangeErr
		})
		if err != nil {
			if !answered {
				http.Error(w, "sign-in failed", http.StatusBadGateway)
			}
			finish(loginResult{err: fmt.Errorf("syncclient: exchange the OIDC code: %w", err)})
			return
		}
		if err := s.storeToken(token); err != nil {
			if !answered {
				http.Error(w, "sign-in failed", http.StatusInternalServerError)
			}
			finish(loginResult{err: err})
			return
		}
		if !answered {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<!doctype html><title>Signed in</title><p>You can close this tab.</p>")
		}
		finish(loginResult{token: token})
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()

	if err := open(authURL); err != nil {
		_ = server.Close()
		return fmt.Errorf("syncclient: open the sign-in page: %w", err)
	}

	timeout := s.LoginTimeout
	if timeout <= 0 {
		timeout = LoginTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-results:
		_ = server.Close()
		return result.err
	case <-timer.C:
		_ = server.Close()
		return errors.New("syncclient: OIDC sign-in timed out")
	case <-ctx.Done():
		_ = server.Close()
		return ctx.Err()
	}
}
