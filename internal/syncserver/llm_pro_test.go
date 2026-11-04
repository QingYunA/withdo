package syncserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// newProTestLLM is newTestLLM with a free limit of 2, a Pro limit of 4, and "bob" as the Pro account.
func newProTestLLM(t *testing.T) *llm {
	t.Helper()
	l := newTestLLM(newFakeUpstream(t, 200, chatAnswer), 2)
	l.configure(LLMOptions{APIKey: llmFakeKey, BaseURL: l.opts.BaseURL, Model: "official-model", DailyTurns: 2, DailyTurnsPro: 4})
	l.pro = func(_ context.Context, subject string) bool { return subject == "bob" }
	return l
}

func TestLLMProAccountsGetTheProLimit(t *testing.T) {
	l := newProTestLLM(t)
	for i := 0; i < 4; i++ {
		if w := llmChat(l, "tok2", "pro-"+strconv.Itoa(i), chatBody); w.Code != 200 {
			t.Fatalf("pro turn %d = %d", i, w.Code)
		}
	}
	if w := llmChat(l, "tok2", "pro-4", chatBody); w.Code != 429 || llmCode(t, w) != CodeLLMQuota {
		t.Fatalf("pro past its limit = %d %s", w.Code, w.Body)
	}
	for i := 0; i < 2; i++ {
		if w := llmChat(l, "tok", "free-"+strconv.Itoa(i), chatBody); w.Code != 200 {
			t.Fatalf("free turn %d = %d", i, w.Code)
		}
	}
	if w := llmChat(l, "tok", "free-2", chatBody); w.Code != 429 {
		t.Fatalf("free past its limit = %d", w.Code)
	}
}

func TestLLMQuotaReportsTheAccountsOwnLimitAndPlan(t *testing.T) {
	l := newProTestLLM(t)
	for token, want := range map[string]struct {
		limit int
		plan  string
	}{"tok": {2, "free"}, "tok2": {4, "pro"}} {
		req := httptest.NewRequest(http.MethodGet, "/v1/llm/quota", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		l.quota(w, req)
		var out struct {
			Limit int    `json:"limit"`
			Left  int    `json:"left"`
			Plan  string `json:"plan"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 {
			t.Fatalf("%s quota = %d %s", token, w.Code, w.Body)
		}
		if out.Limit != want.limit || out.Left != want.limit || out.Plan != want.plan {
			t.Fatalf("%s quota = %+v, want %+v", token, out, want)
		}
	}
}
