package billing

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

func genKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// memStore is a Store in memory with the semantics of PGStore.
type memStore struct {
	mu       sync.Mutex
	accounts map[int64]bool
	subs     map[int64]Subscription
	events   map[string]bool
	subjects map[string]int64
}

func newMemStore(accounts ...int64) *memStore {
	s := &memStore{accounts: map[int64]bool{}, subs: map[int64]Subscription{}, events: map[string]bool{}, subjects: map[string]int64{}}
	for _, a := range accounts {
		s.accounts[a] = true
		s.subjects["subject-"+strconv.FormatInt(a, 10)] = a
	}
	return s
}

func (m *memStore) Get(_ context.Context, account int64) (Subscription, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.subs[account]
	return sub, ok, nil
}

func (m *memStore) GetBySubject(ctx context.Context, subject string) (Subscription, bool, error) {
	return m.Get(ctx, m.subjects[subject])
}

func (m *memStore) Apply(_ context.Context, eventID, _ string, account int64,
	update func(Subscription, bool) (Subscription, bool)) (before, after Subscription, dup, saved bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.accounts[account] {
		return before, after, false, false, ErrNoAccount
	}
	if m.events[eventID] {
		return before, after, true, false, nil
	}
	m.events[eventID] = true
	cur, exists := m.subs[account]
	next, changed := update(cur, exists)
	if changed {
		m.subs[account] = next
		return cur, next, false, true, nil
	}
	return cur, cur, false, false, nil
}

// fakeWaffo is Waffo's API: it verifies the request signature with the public half of the key the
// service signs with, and records what it was asked.
type fakeWaffo struct {
	*httptest.Server
	mu       sync.Mutex
	pub      *rsa.PublicKey
	requests []fakeCall
	reply    func(call fakeCall) (status int, body string)
}

type fakeCall struct {
	Path, Merchant, Timestamp string
	Body                      map[string]any
	SignatureOK               bool
}

func newFakeWaffo(t *testing.T, key *rsa.PrivateKey) *fakeWaffo {
	t.Helper()
	f := &fakeWaffo{pub: &key.PublicKey}
	f.reply = func(call fakeCall) (int, string) {
		if call.Path == pathSessionToken {
			return 200, `{"data":{"token":"tok-1","portalUrl":"https://portal.example/store","expiresAt":"2026-11-04T12:15:00Z"}}`
		}
		return 200, `{"data":{"sessionId":"cs_1","checkoutUrl":"https://checkout.example/c/1","expiresAt":"2026-11-05T12:00:00Z"}}`
	}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		call := fakeCall{Path: r.URL.Path, Merchant: r.Header.Get("X-Merchant-Id"), Timestamp: r.Header.Get("X-Timestamp")}
		_ = json.Unmarshal(raw, &call.Body)
		sum := sha256.Sum256(raw)
		canonical := r.Method + "\n" + r.URL.Path + "\n" + call.Timestamp + "\n" + base64.StdEncoding.EncodeToString(sum[:])
		sig, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Signature"))
		digest := sha256.Sum256([]byte(canonical))
		call.SignatureOK = err == nil && rsa.VerifyPKCS1v15(f.pub, crypto.SHA256, digest[:], sig) == nil
		f.mu.Lock()
		f.requests = append(f.requests, call)
		reply := f.reply
		f.mu.Unlock()
		status, body := reply(call)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeWaffo) calls() []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeCall(nil), f.requests...)
}

// testService is a Service with a generated API key, a generated webhook key for the test mode
// and the fake Waffo behind it.
type testService struct {
	*Service
	store    *memStore
	waffo    *fakeWaffo
	webhook  *rsa.PrivateKey
	notified []int64
	notifyMu sync.Mutex
	now      time.Time
	nowMu    sync.Mutex
}

func newTestService(t *testing.T, accounts ...int64) *testService {
	t.Helper()
	apiKey, webhookKey := genKey(t), genKey(t)
	ts := &testService{store: newMemStore(accounts...), waffo: newFakeWaffo(t, apiKey), webhook: webhookKey, now: testNow}
	svc, err := New(Config{
		MerchantID: "MER_test", Key: apiKey, StoreID: "STO_withdo",
		ProductMonthly: "PROD_monthly", ProductYearly: "PROD_yearly", Mode: ModeTest,
		BaseURL:     ts.waffo.URL,
		WebhookKeys: map[string]*rsa.PublicKey{ModeTest: &webhookKey.PublicKey, ModeProd: &genKey(t).PublicKey},
		Now:         func() time.Time { ts.nowMu.Lock(); defer ts.nowMu.Unlock(); return ts.now },
	}, ts.store, func(account int64) {
		ts.notifyMu.Lock()
		defer ts.notifyMu.Unlock()
		ts.notified = append(ts.notified, account)
	})
	if err != nil {
		t.Fatal(err)
	}
	ts.Service = svc
	return ts
}

func (ts *testService) advance(d time.Duration) {
	ts.nowMu.Lock()
	defer ts.nowMu.Unlock()
	ts.now = ts.now.Add(d)
}

func (ts *testService) notifications() []int64 {
	ts.notifyMu.Lock()
	defer ts.notifyMu.Unlock()
	return append([]int64(nil), ts.notified...)
}

// signedHeader signs body as Waffo does: "t=<ms>,v1=<base64 RSA-SHA256 over t.body>".
func signedHeader(t *testing.T, key *rsa.PrivateKey, at time.Time, body []byte) string {
	t.Helper()
	ms := strconv.FormatInt(at.UnixMilli(), 10)
	sum := sha256.Sum256([]byte(ms + "." + string(body)))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return "t=" + ms + ",v1=" + base64.StdEncoding.EncodeToString(sig)
}

// webhookBody builds a payload. extra overrides or adds fields of "data".
func webhookBody(id, eventType string, account int64, at time.Time, extra map[string]any) []byte {
	data := map[string]any{
		"orderId":       "ORD_1",
		"orderMetadata": map[string]any{"account": strconv.FormatInt(account, 10)},
		"productName":   "Withdo Pro",
		"buyerEmail":    "buyer@example.com",
	}
	for k, v := range extra {
		data[k] = v
	}
	body, err := json.Marshal(map[string]any{
		"id": id, "timestamp": at.UTC().Format(time.RFC3339), "eventType": eventType,
		"eventId": "evt-" + id, "storeId": "STO_withdo", "mode": ModeTest, "data": data,
	})
	if err != nil {
		panic(fmt.Sprint(err))
	}
	return body
}
