package pbauth_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/QingYunA/adhd-todo/internal/syncserver/pbauth"
)

const resetURL = "/api/withdo/reset-password"

// errBody is PocketBase's error answer.
type errBody struct {
	Message string                       `json:"message"`
	Data    map[string]map[string]string `json:"data"`
}

func decodeErr(raw []byte) errBody {
	var out errBody
	_ = json.Unmarshal(raw, &out)
	return out
}

func signUp(t *testing.T, base, email, password, lang string) (int, []byte) {
	t.Helper()
	body := map[string]any{"email": email, "password": password, "passwordConfirm": password}
	if lang != "" {
		body["lang"] = lang
	}
	return post(t, base+"/api/collections/users/records", body)
}

func findUser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	r, err := app.FindAuthRecordByEmail("users", email)
	if err != nil {
		t.Fatalf("no user %s: %v", email, err)
	}
	return r
}

var anyCode = regexp.MustCompile(`\d{6}`)

// mailedCode requests a code for the email and returns it with the OTP id and the mail.
func mailedCode(t *testing.T, base string, mails <-chan sentMail, email string) (otpID, code string, mail sentMail) {
	t.Helper()
	status, otpID := requestOTP(t, base, email)
	if status != http.StatusOK || otpID == "" {
		t.Fatalf("request-otp = %d %q", status, otpID)
	}
	mail = waitMail(t, mails)
	code = anyCode.FindString(mail.subject)
	if code == "" || !strings.Contains(mail.body, code) {
		t.Fatalf("mail = %+v", mail)
	}
	return otpID, code, mail
}

func TestSignUpCreatesAnUnverifiedUser(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))

	status, raw := signUp(t, base, "Grace@Example.com", testPassword, "zh-Hans")
	if status != http.StatusOK {
		t.Fatalf("sign-up = %d %s", status, raw)
	}
	user := findUser(t, app, "grace@example.com")
	if user.Verified() || user.Email() != "grace@example.com" || user.GetString("name") != "grace" ||
		user.GetString("lang") != "zh-Hans" {
		t.Errorf("user = verified %v email %q name %q lang %q", user.Verified(), user.Email(),
			user.GetString("name"), user.GetString("lang"))
	}
	if !user.ValidatePassword(testPassword) {
		t.Error("the password chosen at sign-up should be stored")
	}
}

func TestSignUpRefusals(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))

	status, raw := signUp(t, base, "ada@example.com", "short", "en")
	if e := decodeErr(raw); status != http.StatusBadRequest || e.Data["password"]["code"] != "validation_min_text_constraint" {
		t.Fatalf("short password = %d %s", status, raw)
	}
	if status, _ := signUp(t, base, "ada@example.com", testPassword, "fr"); status != http.StatusBadRequest {
		t.Fatalf("unknown lang = %d, want 400", status)
	}
	status, raw = signUp(t, base, "not an email", testPassword, "en")
	if e := decodeErr(raw); status != http.StatusBadRequest || e.Data["email"]["code"] == "" {
		t.Fatalf("bad email = %d %s", status, raw)
	}
}

func TestSignUpWithATakenEmail(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, raw := signUp(t, base, "ada@example.com", testPassword, "en"); status != http.StatusOK {
		t.Fatalf("sign-up = %d %s", status, raw)
	}
	for _, again := range []string{"ada@example.com", "Ada@Example.com"} {
		status, raw := signUp(t, base, again, testPassword, "en")
		if e := decodeErr(raw); status != http.StatusBadRequest || e.Data["email"]["code"] != "validation_not_unique" {
			t.Fatalf("taken email %s = %d %s", again, status, raw)
		}
	}
	if total, _ := app.CountRecords("users"); total != 1 {
		t.Fatalf("users = %d, want 1", total)
	}
}

func TestClientCannotSetProtectedFields(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	for name, extra := range map[string]map[string]any{
		"verified true":  {"verified": true},
		"verified false": {"verified": false},
		"custom id":      {"id": "abcdefghijklmno"},
	} {
		body := map[string]any{"email": "eve@example.com", "password": testPassword, "passwordConfirm": testPassword}
		for k, v := range extra {
			body[k] = v
		}
		if status, raw := post(t, base+"/api/collections/users/records", body); status < 400 {
			t.Errorf("%s: sign-up = %d %s, want it refused", name, status, raw)
		}
	}
	if total, _ := app.CountRecords("users"); total != 0 {
		t.Fatalf("users = %d, want none made by a refused sign-up", total)
	}
	// A token key from the client is ignored by PocketBase (it is a hidden system field), never stored.
	if status, raw := post(t, base+"/api/collections/users/records", map[string]any{
		"email": "eve@example.com", "password": testPassword, "passwordConfirm": testPassword,
		"tokenKey": "0123456789abcdef0123456789abcdef",
	}); status < 400 {
		if findUser(t, app, "eve@example.com").TokenKey() == "0123456789abcdef0123456789abcdef" {
			t.Errorf("the client set the token key: %s", raw)
		}
	}
	// The rest of the collection stays closed: no list, no edit, no delete.
	if status, _ := get(t, base+"/api/collections/users/records", ""); status < 400 {
		t.Errorf("list = %d", status)
	}
}

func TestUnverifiedUserSignsInOnlyWithTheCode(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, raw := signUp(t, base, "ada@example.com", testPassword, "en"); status != http.StatusOK {
		t.Fatalf("sign-up = %d %s", status, raw)
	}

	// The right password of an unverified account: the documented 403, no token.
	status, raw := post(t, base+"/api/collections/users/auth-with-password",
		map[string]string{"identity": "ada@example.com", "password": testPassword})
	if e := decodeErr(raw); status != http.StatusForbidden || e.Data["email"]["code"] != pbauth.ErrorCodeEmailNotVerified ||
		strings.Contains(string(raw), `"token"`) {
		t.Fatalf("unverified sign-in = %d %s", status, raw)
	}
	// A wrong password is the usual 400, whether or not the account is verified.
	if status, _ := signIn(t, base, "users", "ada@example.com", "wrong password"); status != http.StatusBadRequest {
		t.Fatalf("wrong password = %d, want 400", status)
	}

	// OTP auth works for it, verifies it, and the password chosen at sign-up still works.
	otpID, code, _ := mailedCode(t, base, mails, "ada@example.com")
	if status, _ := authWithOTP(t, base, otpID, "000000"); status != http.StatusBadRequest {
		t.Fatalf("wrong code = %d, want 400", status)
	}
	status, token := authWithOTP(t, base, otpID, code)
	if status != http.StatusOK || token == "" {
		t.Fatalf("auth-with-otp = %d", status)
	}
	user := findUser(t, app, "ada@example.com")
	if !user.Verified() {
		t.Fatal("the code should verify the account")
	}
	if subject, err := pbauth.NewAuthenticator(app).Subject(bearer(token)); err != nil || subject != "pb:"+user.Id {
		t.Fatalf("subject = %q, %v", subject, err)
	}
	if status, token := signIn(t, base, "users", "ada@example.com", testPassword); status != http.StatusOK || token == "" {
		t.Fatalf("password sign-in after verifying = %d, want 200", status)
	}
	// One use.
	if status, _ := authWithOTP(t, base, otpID, code); status != http.StatusBadRequest {
		t.Fatalf("a used code = %d, want 400", status)
	}
}

func resetPasswordCall(t *testing.T, base, otpID, code, password string) (int, []byte) {
	t.Helper()
	return post(t, base+resetURL, map[string]string{
		"otpId": otpID, "code": code, "password": password, "passwordConfirm": password,
	})
}

func TestResetPasswordWithACode(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))

	otpID, code, _ := mailedCode(t, base, mails, "ada@example.com")
	const fresh = "a brand new password"
	status, raw := resetPasswordCall(t, base, otpID, code, fresh)
	var out struct {
		Token  string         `json:"token"`
		Record map[string]any `json:"record"`
	}
	_ = json.Unmarshal(raw, &out)
	if status != http.StatusOK || out.Token == "" || out.Record["email"] != "ada@example.com" {
		t.Fatalf("reset = %d %s", status, raw)
	}
	if subject, err := pbauth.NewAuthenticator(app).Subject(bearer(out.Token)); err != nil ||
		subject != "pb:"+findUser(t, app, "ada@example.com").Id {
		t.Fatalf("token subject = %q, %v", subject, err)
	}
	if status, _ := signIn(t, base, "users", "ada@example.com", testPassword); status != http.StatusBadRequest {
		t.Errorf("old password = %d, want it refused", status)
	}
	if status, token := signIn(t, base, "users", "ada@example.com", fresh); status != http.StatusOK || token == "" {
		t.Errorf("new password = %d", status)
	}
	// One use.
	if status, _ := resetPasswordCall(t, base, otpID, code, "yet another password"); status != http.StatusBadRequest {
		t.Errorf("a used code = %d, want 400", status)
	}
}

func TestResetPasswordVerifiesAnUnverifiedAccount(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, raw := signUp(t, base, "ada@example.com", testPassword, "en"); status != http.StatusOK {
		t.Fatalf("sign-up = %d %s", status, raw)
	}
	otpID, code, _ := mailedCode(t, base, mails, "ada@example.com")
	if status, raw := resetPasswordCall(t, base, otpID, code, "the password I chose later"); status != http.StatusOK {
		t.Fatalf("reset = %d %s", status, raw)
	}
	if !findUser(t, app, "ada@example.com").Verified() {
		t.Fatal("a reset with the mailed code proves the email")
	}
	if status, _ := signIn(t, base, "users", "ada@example.com", testPassword); status != http.StatusBadRequest {
		t.Errorf("the pre-hijacker's password = %d, want it replaced", status)
	}
}

func TestResetPasswordRefusals(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	otpID, code, _ := mailedCode(t, base, mails, "ada@example.com")
	const fresh = "a brand new password"

	// A wrong code, and the same answer for an otpId that never existed (an email with no account).
	status, wrong := resetPasswordCall(t, base, otpID, "000000", fresh)
	if status != http.StatusBadRequest {
		t.Fatalf("wrong code = %d", status)
	}
	// request-otp answers an email with no account with a random otpId and sends nothing.
	status, ghost := requestOTP(t, base, "nobody@example.com")
	if status != http.StatusOK || ghost == "" {
		t.Fatalf("request-otp for no account = %d %q", status, ghost)
	}
	status, unknown := resetPasswordCall(t, base, ghost, "123456", fresh)
	if status != http.StatusBadRequest {
		t.Fatalf("unknown email reset = %d", status)
	}
	if string(wrong) != string(unknown) {
		t.Errorf("a wrong code answers %s but an unknown email answers %s", wrong, unknown)
	}
	select {
	case m := <-mails:
		t.Fatalf("a mail was sent for an unknown email: %+v", m)
	case <-time.After(300 * time.Millisecond):
	}

	// A short password or a mismatch is refused before the code is looked at, and burns nothing.
	if status, raw := resetPasswordCall(t, base, otpID, code, "short"); status != http.StatusBadRequest ||
		decodeErr(raw).Data["password"]["code"] != "validation_min_text_constraint" {
		t.Errorf("short password = %d %s", status, raw)
	}
	if status, _ := post(t, base+resetURL, map[string]string{
		"otpId": otpID, "code": code, "password": fresh, "passwordConfirm": fresh + "x",
	}); status != http.StatusBadRequest {
		t.Errorf("mismatch = %d", status)
	}
	// The password never changed.
	if status, _ := signIn(t, base, "users", "ada@example.com", testPassword); status != http.StatusOK {
		t.Fatalf("the old password stopped working: %d", status)
	}
	// The right code still works after all that.
	if status, raw := resetPasswordCall(t, base, otpID, code, fresh); status != http.StatusOK {
		t.Fatalf("right code = %d %s", status, raw)
	}
}

func TestResetPasswordRefusesAnExpiredCode(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	otpID, code, _ := mailedCode(t, base, mails, "ada@example.com")
	if _, err := app.DB().NewQuery("UPDATE _otps SET created = {:old}, updated = {:old}").
		Bind(map[string]any{"old": time.Now().Add(-11 * time.Minute).UTC().Format("2006-01-02 15:04:05.000Z")}).Execute(); err != nil {
		t.Fatal(err)
	}
	if status, _ := resetPasswordCall(t, base, otpID, code, "a brand new password"); status != http.StatusBadRequest {
		t.Fatalf("expired code = %d, want 400", status)
	}
	if status, _ := authWithOTP(t, base, otpID, code); status != http.StatusBadRequest {
		t.Fatalf("the same expired code on auth-with-otp = %d, want 400", status)
	}
	if status, _ := signIn(t, base, "users", "ada@example.com", testPassword); status != http.StatusOK {
		t.Fatalf("the password changed anyway: %d", status)
	}
}

func TestResetPasswordGuessesAreLimited(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	otpID, code, _ := mailedCode(t, base, mails, "ada@example.com")
	wrong := "000000"
	if code == wrong {
		wrong = "000001"
	}
	var got []int
	for i := 0; i < 6; i++ {
		status, _ := resetPasswordCall(t, base, otpID, wrong, "a brand new password")
		got = append(got, status)
	}
	if got[4] != http.StatusBadRequest || got[5] != http.StatusTooManyRequests {
		t.Fatalf("statuses = %v, want five 400s then a 429", got)
	}
	// Even the right code is refused while the limit holds.
	if status, _ := resetPasswordCall(t, base, otpID, code, "a brand new password"); status != http.StatusTooManyRequests {
		t.Fatalf("right code during the limit = %d, want 429", status)
	}
}

func TestSignUpIsRateLimited(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	var statuses []int
	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com", "d@example.com", "e@example.com", "f@example.com"} {
		status, _ := signUp(t, base, email, testPassword, "en")
		statuses = append(statuses, status)
	}
	if statuses[4] != http.StatusOK || statuses[5] != http.StatusTooManyRequests {
		t.Fatalf("statuses = %v, want five 200s then a 429", statuses)
	}
}

func TestResetRouteIsRateLimited(t *testing.T) {
	app := openApp(t)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	limited := -1
	for i := 0; i < 15; i++ {
		// Each call is for another (nonexistent) code, so only the route's own limit can stop it.
		if status, _ := resetPasswordCall(t, base, "nope", "123456", "a brand new password"); status == http.StatusTooManyRequests {
			limited = i
			break
		}
	}
	if limited != 10 {
		t.Fatalf("the 429 came on call %d, want the 11th (index 10)", limited)
	}
}

func TestCodeMailFollowsTheAccountsLanguage(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, raw := signUp(t, base, "zh@example.com", testPassword, "zh-Hans"); status != http.StatusOK {
		t.Fatal(status, string(raw))
	}
	if status, raw := signUp(t, base, "en@example.com", testPassword, "en"); status != http.StatusOK {
		t.Fatal(status, string(raw))
	}
	if status, raw := signUp(t, base, "none@example.com", testPassword, ""); status != http.StatusOK {
		t.Fatal(status, string(raw))
	}

	_, code, m := mailedCode(t, base, mails, "zh@example.com")
	if m.subject != code+" 是你的 Withdo 验证码" || !strings.Contains(m.body, "10 分钟内有效") {
		t.Errorf("zh-Hans mail = %q / %q", m.subject, m.body)
	}
	for _, email := range []string{"en@example.com", "none@example.com"} {
		_, code, m = mailedCode(t, base, mails, email)
		if m.subject != code+" is your Withdo code" || !strings.Contains(m.body, "It works for 10 minutes") {
			t.Errorf("%s mail = %q / %q", email, m.subject, m.body)
		}
	}
}

// A code that PocketBase mailed in Chinese still verifies the account: the localized message goes
// through PocketBase's own send, which marks where the code went.
func TestChineseCodeStillVerifies(t *testing.T) {
	app := openApp(t)
	mails := captureMail(app)
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, raw := signUp(t, base, "zh@example.com", testPassword, "zh-Hans"); status != http.StatusOK {
		t.Fatal(status, string(raw))
	}
	otpID, code, _ := mailedCode(t, base, mails, "zh@example.com")
	if status, token := authWithOTP(t, base, otpID, code); status != http.StatusOK || token == "" {
		t.Fatalf("auth-with-otp = %d", status)
	}
	if !findUser(t, app, "zh@example.com").Verified() {
		t.Fatal("not verified")
	}
}

func TestPasswordSignInTypedInOtherCase(t *testing.T) {
	app := openApp(t)
	if _, err := pbauth.ImportUser(app, "ada@example.com", "Ada", testPassword); err != nil {
		t.Fatal(err)
	}
	base := serve(t, app, pbauth.NewAuthenticator(app))
	if status, token := signIn(t, base, "users", "Ada@Example.com", testPassword); status != http.StatusOK || token == "" {
		t.Fatalf("sign-in with another case = %d", status)
	}
}
