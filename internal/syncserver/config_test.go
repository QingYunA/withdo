package syncserver_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncserver"
)

// TestS3UIb1ConfigIsPublic is the server half of S3-UI-b: GET /v1/config needs no token and
// returns the issuer, client id and scopes the server was started with.
func TestS3UIb1ConfigIsPublic(t *testing.T) {
	h := newHarness(t)
	h.server.SetPublicConfig(syncserver.PublicConfig{
		Issuer: "https://issuer.example", ClientID: "adhd-todo",
		Scopes: []string{"openid", "offline_access", "urn:example:audience"},
	})
	resp, body := h.do(t, http.MethodGet, "/v1/config", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/config = %d %s", resp.StatusCode, body)
	}
	var got struct {
		Issuer   string   `json:"issuer"`
		ClientID string   `json:"client_id"`
		Scopes   []string `json:"scopes"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.Issuer != "https://issuer.example" || got.ClientID != "adhd-todo" ||
		len(got.Scopes) != 3 || got.Scopes[2] != "urn:example:audience" {
		t.Fatalf("config = %+v", got)
	}
}
