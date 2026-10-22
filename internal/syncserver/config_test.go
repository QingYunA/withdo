package syncserver_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

// TestConfigIsPublic checks that GET /v1/config needs no token.
func TestConfigIsPublic(t *testing.T) {
	h := newHarness(t)
	resp, body := h.do(t, http.MethodGet, "/v1/config", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/config = %d %s", resp.StatusCode, body)
	}
}

// TestConfigNamesTheAccountSystem checks that "auth" is in /v1/config when set and absent when not.
func TestConfigNamesTheAccountSystem(t *testing.T) {
	h := newHarness(t)
	_, body := h.do(t, http.MethodGet, "/v1/config", "", nil)
	if strings.Contains(string(body), `"auth"`) {
		t.Fatalf("config without an account system names one: %s", body)
	}
	h.server.SetPublicConfig(syncserver.PublicConfig{Auth: "pocketbase"})
	_, body = h.do(t, http.MethodGet, "/v1/config", "", nil)
	var got struct {
		Auth string `json:"auth"`
	}
	if err := json.Unmarshal(body, &got); err != nil || got.Auth != "pocketbase" {
		t.Fatalf("config = %s (%v)", body, err)
	}
}
