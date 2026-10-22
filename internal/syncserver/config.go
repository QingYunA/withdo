package syncserver

import (
	"net/http"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// PublicConfig is what GET /v1/config tells a client so that it can sign in. All of it is public.
type PublicConfig struct {
	// Auth names the account system clients sign in with: "pocketbase", or empty when the server
	// only knows development tokens.
	Auth string
}

// SetPublicConfig sets the values GET /v1/config returns. Call it before serving.
func (s *Server) SetPublicConfig(cfg PublicConfig) { s.public = cfg }

// config is GET /v1/config. It needs no bearer token: a client asks for it before it has one.
func (s *Server) config(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, syncproto.ConfigResponse{Auth: s.public.Auth})
}
