package syncserver

import (
	"net/http"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// PublicConfig is what GET /v1/config tells a client so that it can sign in: the OIDC issuer, the
// client id, and any scopes beyond openid and offline_access. All of it is public.
type PublicConfig struct {
	Issuer   string
	ClientID string
	Scopes   []string
}

// SetPublicConfig sets the values GET /v1/config returns. Call it before serving.
func (s *Server) SetPublicConfig(cfg PublicConfig) { s.public = cfg }

// config is GET /v1/config. It needs no bearer token: a client asks for it before it has one.
func (s *Server) config(w http.ResponseWriter, _ *http.Request) {
	scopes := s.public.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	writeJSON(w, http.StatusOK, syncproto.ConfigResponse{
		Issuer: s.public.Issuer, ClientID: s.public.ClientID, Scopes: scopes,
	})
}
