package pbauth_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

const testPassword = "correct horse battery"

func openApp(t *testing.T) core.App {
	t.Helper()
	app, err := pbauth.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = pbauth.Close(app) })
	return app
}

// serve runs PocketBase with a /v1 handler that answers the caller's subject, and returns its URL.
func serve(t *testing.T, app core.App, auth syncserver.Authenticator) string {
	t.Helper()
	handler := http.NewServeMux()
	handler.HandleFunc("GET /v1/whoami", func(w http.ResponseWriter, r *http.Request) {
		subject, err := auth.Subject(r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, subject)
	})
	handler.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") })
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pbauth.Serve(ctx, app, ln, handler) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("serve: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("serve did not stop")
		}
	})
	return "http://" + ln.Addr().String()
}

func post(t *testing.T, url string, body any) (int, []byte) {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func get(t *testing.T, url, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(out)
}

func signIn(t *testing.T, base, collection, email, password string) (int, string) {
	t.Helper()
	status, raw := post(t, base+"/api/collections/"+collection+"/auth-with-password",
		map[string]string{"identity": email, "password": password})
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(raw, &out)
	return status, out.Token
}

func TestAuthenticatorAcceptsUsersAndRejectsTheRest(t *testing.T) {
	app := openApp(t)
	subject, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(subject, "pb:") || len(subject) < 5 {
		t.Fatalf("subject = %q", subject)
	}
	if err := pbauth.UpsertSuperuser(app, "root@example.com", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))

	status, token := signIn(t, base, "users", "ada@example.com", testPassword)
	if status != http.StatusOK || token == "" {
		t.Fatalf("sign in = %d", status)
	}
	if status, body := get(t, base+"/v1/whoami", token); status != http.StatusOK || body != subject {
		t.Fatalf("whoami = %d %q, want %q", status, body, subject)
	}
	if status, _ := get(t, base+"/v1/whoami", token+"x"); status != http.StatusUnauthorized {
		t.Fatalf("tampered token = %d", status)
	}
	if status, _ := get(t, base+"/v1/whoami", ""); status != http.StatusUnauthorized {
		t.Fatalf("no token = %d", status)
	}
	if status, body := get(t, base+"/healthz", ""); status != http.StatusOK || body != "ok" {
		t.Fatalf("healthz = %d %q", status, body)
	}

	// A superuser is not a sync user.
	status, super := signIn(t, base, "_superusers", "root@example.com", testPassword)
	if status != http.StatusOK || super == "" {
		t.Fatalf("superuser sign in = %d", status)
	}
	if status, _ := get(t, base+"/v1/whoami", super); status != http.StatusUnauthorized {
		t.Fatalf("superuser token on /v1 = %d, want 401", status)
	}
}

func TestSignInFailuresAndClosedSignUp(t *testing.T) {
	app := openApp(t)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))

	if status, _ := signIn(t, base, "users", "ada@example.com", "wrong password"); status != http.StatusBadRequest {
		t.Fatalf("wrong password = %d, want 400", status)
	}
	status, _ := post(t, base+"/api/collections/users/records", map[string]string{
		"email": "eve@example.com", "password": testPassword, "passwordConfirm": testPassword,
	})
	// The create rule fails a plain create with a 400 now that OAuth2 may create users.
	if status != http.StatusForbidden && status != http.StatusNotFound && status != http.StatusBadRequest {
		t.Fatalf("sign-up = %d, want it closed", status)
	}
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err == nil {
		t.Fatal("importing the same email twice should fail")
	}
}

func TestMigrationSetsUpUsersCollection(t *testing.T) {
	app := openApp(t)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	if users.ListRule != nil || users.ViewRule != nil || users.UpdateRule != nil || users.DeleteRule != nil {
		t.Error("the users API rules should be closed")
	}
	if users.CreateRule == nil || *users.CreateRule != `@request.context = "oauth2"` {
		t.Errorf("create rule = %v, want only OAuth2 to create users", users.CreateRule)
	}
	if !users.PasswordAuth.Enabled || !users.OTP.Enabled || !users.OAuth2.Enabled || users.MFA.Enabled {
		t.Error("password, one-time code and OAuth2 sign-in should be on, with MFA off")
	}
	if users.OAuth2.MappedFields.Name != "name" || len(users.OAuth2.Providers) != 0 {
		t.Errorf("OAuth2 = %+v, want name mapped and no provider in code", users.OAuth2)
	}
	if users.OTP.Length != 6 || users.OTP.Duration != 600 {
		t.Errorf("code = %d digits for %d s, want 6 for 600", users.OTP.Length, users.OTP.Duration)
	}
	tpl := users.OTP.EmailTemplate
	if tpl.Subject != "{OTP} is your ADHD Todo code" || !strings.Contains(tpl.Body, "{OTP}") ||
		!strings.Contains(tpl.Body, "It works for 10 minutes") {
		t.Errorf("OTP template = %q / %q", tpl.Subject, tpl.Body)
	}
	if got := app.Settings().Meta.AppName; got != "ADHD Todo" {
		t.Errorf("app name = %q", got)
	}
}

func TestChainOrder(t *testing.T) {
	app := openApp(t)
	subject, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, nil)
	_, token := signIn(t, base, "users", "ada@example.com", testPassword)

	chain := syncserver.Chain{
		pbauth.NewAuthenticator(app),
		syncserver.StaticTokens{"dev-token": "dev-subject"},
	}
	for token, want := range map[string]string{token: subject, "dev-token": "dev-subject"} {
		req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		got, err := chain.Subject(req)
		if err != nil || got != want {
			t.Errorf("chain subject = %q, %v; want %q", got, err, want)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.Header.Set("Authorization", "Bearer nope")
	if _, err := chain.Subject(req); !errors.Is(err, syncserver.ErrUnauthenticated) {
		t.Errorf("unknown token error = %v", err)
	}
}

// The admin UI is public, so guessing a superuser's password is limited harder than a user's.
func TestSuperuserSignInIsRateLimited(t *testing.T) {
	app := openApp(t)
	if err := pbauth.UpsertSuperuser(app, "root@example.com", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	tries := 0
	for ; tries < 20; tries++ {
		if status, _ := signIn(t, base, "_superusers", "root@example.com", "wrong password"); status == http.StatusTooManyRequests {
			break
		}
	}
	if tries > 5 {
		t.Fatalf("a superuser's wrong password was refused with 429 only after %d tries, want at most 5", tries)
	}
}

func TestPasswordSignInIsRateLimited(t *testing.T) {
	app := openApp(t)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	var limited bool
	for i := 0; i < 20 && !limited; i++ {
		status, _ := signIn(t, base, "users", "ada@example.com", "wrong password")
		limited = status == http.StatusTooManyRequests
	}
	if !limited {
		t.Fatal("twenty wrong passwords in a row were never refused with 429")
	}
	// Sync routes are not limited.
	for i := 0; i < 30; i++ {
		if status, _ := get(t, base+"/healthz", ""); status != http.StatusOK {
			t.Fatalf("healthz = %d after %d requests", status, i)
		}
	}
}

type sentMail struct{ to, subject, body string }

// captureMail replaces the OTP mailer, so no mail leaves the test, and returns the channel the
// mails arrive on (they are sent in the background).
func captureMail(app core.App) <-chan sentMail {
	out := make(chan sentMail, 8)
	app.OnMailerRecordOTPSend().BindFunc(func(e *core.MailerRecordEvent) error {
		// What PocketBase does after a real send; it is what lets the code verify the account.
		if otp, err := e.App.FindOTPById(e.Meta["otpId"].(string)); err == nil {
			otp.SetSentTo(e.Message.To[0].Address)
			_ = e.App.Save(otp)
		}
		out <- sentMail{to: e.Message.To[0].Address, subject: e.Message.Subject, body: e.Message.HTML}
		return nil
	})
	return out
}

func waitMail(t *testing.T, mails <-chan sentMail) sentMail {
	t.Helper()
	select {
	case m := <-mails:
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("no mail was sent")
		return sentMail{}
	}
}

func requestOTP(t *testing.T, base, email string) (int, string) {
	t.Helper()
	status, raw := post(t, base+"/api/collections/users/request-otp", map[string]string{"email": email})
	var out struct {
		OTPID string `json:"otpId"`
	}
	_ = json.Unmarshal(raw, &out)
	return status, out.OTPID
}

func authWithOTP(t *testing.T, base, otpID, code string) (int, string) {
	t.Helper()
	status, raw := post(t, base+"/api/collections/users/auth-with-otp", map[string]string{"otpId": otpID, "password": code})
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(raw, &out)
	return status, out.Token
}

var sixDigits = regexp.MustCompile(`^(\d{6}) is your ADHD Todo code$`)

func TestNewEmailGetsAnAccountAndACode(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))

	status, otpID := requestOTP(t, base, "Grace@Example.com")
	if status != http.StatusOK || otpID == "" {
		t.Fatalf("request-otp = %d %q", status, otpID)
	}
	m := waitMail(t, mails)
	match := sixDigits.FindStringSubmatch(m.subject)
	if match == nil || !strings.EqualFold(m.to, "grace@example.com") || !strings.Contains(m.body, match[1]) {
		t.Fatalf("mail = %+v", m)
	}
	code := match[1]

	users, _ := app.FindCollectionByNameOrId("users")
	record, err := app.FindAuthRecordByEmail(users, "grace@example.com")
	if err != nil {
		t.Fatalf("the user was not created: %v", err)
	}
	if record.Verified() || record.GetString("name") != "Grace" {
		t.Errorf("new user verified=%v name=%q", record.Verified(), record.GetString("name"))
	}

	// A wrong code fails, the right one signs in and marks the account verified.
	if status, _ := authWithOTP(t, base, otpID, "000000"); status != http.StatusBadRequest {
		t.Fatalf("wrong code = %d, want 400", status)
	}
	status, token := authWithOTP(t, base, otpID, code)
	if status != http.StatusOK || token == "" {
		t.Fatalf("auth-with-otp = %d", status)
	}
	subject, err := pbauth.NewAuthenticator(app).Subject(bearer(token))
	if err != nil || subject != "pb:"+record.Id {
		t.Fatalf("subject = %q, %v; want pb:%s", subject, err, record.Id)
	}
	record, _ = app.FindRecordById(users, record.Id)
	if !record.Verified() {
		t.Error("the code sign-in should verify the account")
	}
}

func bearer(token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestExistingAccountGetsACodeWithoutANewUser(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	subject, err := pbauth.ImportUser(app, "ada@example.com", "Ada Lovelace", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, _ := requestOTP(t, base, "ada@example.com"); status != http.StatusOK {
		t.Fatalf("request-otp = %d", status)
	}
	waitMail(t, mails)
	total, _ := app.CountRecords("users")
	if total != 1 {
		t.Fatalf("users = %d, want 1", total)
	}
	users, _ := app.FindCollectionByNameOrId("users")
	if r, _ := app.FindAuthRecordByEmail(users, "ada@example.com"); "pb:"+r.Id != subject || r.GetString("name") != "Ada Lovelace" {
		t.Error("the existing account was changed")
	}
}

func TestExistingAccountTypedInOtherCaseGetsACode(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada Lovelace", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, _ := requestOTP(t, base, "Ada@Example.com"); status != http.StatusOK {
		t.Fatalf("request-otp = %d", status)
	}
	waitMail(t, mails)
	if total, _ := app.CountRecords("users"); total != 1 {
		t.Fatalf("users = %d, want 1", total)
	}
}

func TestRequestOTPRejectsABadEmail(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, _ := requestOTP(t, base, "not an email"); status != http.StatusBadRequest {
		t.Fatalf("bad email = %d, want 400", status)
	}
	if total, _ := app.CountRecords("users"); total != 0 {
		t.Fatalf("users = %d, want 0", total)
	}
}

func TestMigrationAddsRateLimits(t *testing.T) {
	app := openApp(t)
	want := map[string]struct {
		duration int64
		max      int
	}{
		"users:requestOTP":           {60, 3},
		"users:authWithOTP":          {60, 10},
		"users:requestPasswordReset": {60, 3},
	}
	for _, rule := range app.Settings().RateLimits.Rules {
		if w, ok := want[rule.Label]; ok {
			if rule.Duration != w.duration || rule.MaxRequests != w.max {
				t.Errorf("%s = %d per %d s", rule.Label, rule.MaxRequests, rule.Duration)
			}
			delete(want, rule.Label)
		}
	}
	for label := range want {
		t.Errorf("no rate limit rule %s", label)
	}
}

func TestRequestOTPIsRateLimited(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	var limited int
	for i := 0; i < 6; i++ {
		if status, _ := requestOTP(t, base, "ada@example.com"); status == http.StatusTooManyRequests {
			limited = i
			break
		}
	}
	if limited != 3 {
		t.Fatalf("the 429 came on request %d, want the 4th (index 3)", limited)
	}
	// The codes are mailed in the background: let those sends finish before the app closes.
	waitMail(t, mails)
	for quiet := false; !quiet; {
		select {
		case <-mails:
		case <-time.After(500 * time.Millisecond):
			quiet = true
		}
	}
}

// sseEvent reads one event (name and data) from a server-sent event stream.
func sseEvent(t *testing.T, r *bufio.Reader) (string, string) {
	t.Helper()
	var name, data string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read the event stream: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if name != "" {
				return name, data
			}
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}

func TestOAuth2RedirectReachesTheSubscribedClient(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/realtime", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	stream := bufio.NewReader(resp.Body)
	name, data := sseEvent(t, stream)
	var connect struct {
		ClientID string `json:"clientId"`
	}
	if name != "PB_CONNECT" || json.Unmarshal([]byte(data), &connect) != nil || connect.ClientID == "" {
		t.Fatalf("first event = %q %q, want PB_CONNECT with a client id", name, data)
	}
	if status, _ := post(t, base+"/api/realtime", map[string]any{
		"clientId": connect.ClientID, "subscriptions": []string{"@oauth2"},
	}); status != http.StatusNoContent {
		t.Fatalf("subscribe = %d, want 204", status)
	}
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	redirect, err := noFollow.Get(base + "/api/oauth2-redirect?state=" + connect.ClientID + "&code=abc")
	if err != nil {
		t.Fatal(err)
	}
	_ = redirect.Body.Close()
	name, data = sseEvent(t, stream)
	var msg struct{ State, Code, Error string }
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatal(err)
	}
	if name != "@oauth2" || msg.State != connect.ClientID || msg.Code != "abc" || msg.Error != "" {
		t.Fatalf("event = %q %+v, want @oauth2 with the state and code", name, msg)
	}
}

func TestPlainSignUpStaysClosedAndCodeSignUpStillWorks(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	status, _ := post(t, base+"/api/collections/users/records", map[string]string{
		"email": "eve@example.com", "password": testPassword, "passwordConfirm": testPassword,
	})
	if status < 400 {
		t.Fatalf("plain sign-up = %d, want it refused", status)
	}
	if total, _ := app.CountRecords("users"); total != 0 {
		t.Fatalf("users = %d, want 0", total)
	}
	if status, body := requestOTP(t, base, "new@example.com"); status != http.StatusOK {
		t.Fatalf("code request = %d %s, want 200", status, body)
	}
	waitMail(t, mails)
	if total, _ := app.CountRecords("users"); total != 1 {
		t.Fatalf("users = %d, want the code sign-up's 1", total)
	}
}
