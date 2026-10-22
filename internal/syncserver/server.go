// Package syncserver is the self-hosted M11 sync server: Postgres keeps every workspace's rows,
// and the HTTP API pushes a device's mutations and pulls the changes after a cursor.
package syncserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// Server owns the database pool and the HTTP API. It is safe for concurrent use.
type Server struct {
	pool *pgxpool.Pool
	auth Authenticator
	hub  *hub
	mux  *http.ServeMux
	// public is what GET /v1/config tells clients; set before serving.
	public PublicConfig
	// voice is POST /v1/transcribe; off until EnableVoice gives it a key.
	voice *voice
}

// Open builds the connection pool from databaseURL, applies the schema migrations and returns the
// server. Close releases the pool.
func Open(ctx context.Context, databaseURL string, auth Authenticator) (*Server, error) {
	if auth == nil {
		return nil, errors.New("syncserver: nil authenticator")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	s, err := newServer(ctx, pool, auth)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// NewServer applies the migrations and wires the routes around an existing pool. Open is the usual
// constructor; a caller that already owns a pool, such as a test with its own schema, uses this.
func NewServer(ctx context.Context, pool *pgxpool.Pool, auth Authenticator) (*Server, error) {
	return newServer(ctx, pool, auth)
}

// newServer applies the migrations and wires the routes.
func newServer(ctx context.Context, pool *pgxpool.Pool, auth Authenticator) (*Server, error) {
	if err := migrate(ctx, pool); err != nil {
		return nil, err
	}
	s := &Server{pool: pool, auth: auth, hub: newHub(), mux: http.NewServeMux(), voice: newVoice(auth)}
	s.mux.HandleFunc("GET /healthz", s.healthz)
	s.mux.HandleFunc("GET /v1/config", s.config)
	s.mux.HandleFunc("POST /v1/devices", s.withSession(s.createDevice))
	s.mux.HandleFunc("GET /v1/devices", s.withSession(s.listDevices))
	s.mux.HandleFunc("POST /v1/devices/seeded", s.withSession(s.deviceSeeded))
	s.mux.HandleFunc("POST /v1/ids", s.withSession(s.leaseIDs))
	s.mux.HandleFunc("POST /v1/push", s.withSession(s.push))
	s.mux.HandleFunc("GET /v1/pull", s.withSession(s.pull))
	s.mux.HandleFunc("POST /v1/transcribe", s.voice.handler)
	s.mux.HandleFunc("GET /v1/events", s.withSession(s.events))
	return s, nil
}

// Handler returns the HTTP handler, including /healthz.
func (s *Server) Handler() http.Handler { return s.mux }

// Close releases the database pool.
func (s *Server) Close() { s.pool.Close() }

// session is the caller's user and workspace, resolved once per request.
type session struct {
	userID      int64
	workspaceID int64
}

type sessionContextKey struct{}

// withSession authenticates the request and makes its workspace available to the handler. Every
// /v1 request is scoped to this workspace; a device id from another user never matches it.
func (s *Server) withSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject, err := s.auth.Subject(r)
		if err != nil {
			if errors.Is(err, ErrUnauthenticated) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
				return
			}
			writeError(w, err)
			return
		}
		sess, err := s.ensureWorkspace(r.Context(), subject)
		if err != nil {
			writeError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey{}, &sess)
		next(w, r.WithContext(ctx))
	}
}

func sessionFrom(ctx context.Context) *session {
	sess, _ := ctx.Value(sessionContextKey{}).(*session)
	return sess
}

// ensureWorkspace resolves the subject (a user's own or an alias of one), creating the user and
// their one workspace on the first request of an unknown subject. The unique owner_id makes concurrent first requests end at the same workspace.
func (s *Server) ensureWorkspace(ctx context.Context, subject string) (sess session, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return sess, fmt.Errorf("begin workspace lookup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// A primary subject wins over an alias; only a subject known to neither creates a user.
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE subject = $1
		UNION ALL SELECT user_id FROM subject_aliases WHERE subject = $1
		LIMIT 1`, subject).Scan(&sess.userID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO users (subject) VALUES ($1)
			ON CONFLICT (subject) DO UPDATE SET subject = EXCLUDED.subject
			RETURNING id`, subject).Scan(&sess.userID)
	}
	if err != nil {
		return sess, fmt.Errorf("resolve user: %w", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO workspaces (owner_id) VALUES ($1)
		ON CONFLICT (owner_id) DO NOTHING
		RETURNING id`, sess.userID).Scan(&sess.workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, "SELECT id FROM workspaces WHERE owner_id = $1", sess.userID).Scan(&sess.workspaceID)
	}
	if err != nil {
		return sess, fmt.Errorf("find workspace: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO workspace_versions (workspace_id, version) VALUES ($1, 0)
		ON CONFLICT (workspace_id) DO NOTHING`, sess.workspaceID); err != nil {
		return sess, fmt.Errorf("start workspace clock: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return sess, fmt.Errorf("commit workspace lookup: %w", err)
	}
	return sess, nil
}

// deviceRecord is the part of a device row the handlers need.
type deviceRecord struct {
	userID         int64
	workspaceID    int64
	lastMutationID int64
}

// lookupDevice reads a device and checks that it belongs to the caller's workspace. A device of
// another user, or one that does not exist, is a 403 rather than a 404.
func (s *Server) lookupDevice(ctx context.Context, sess *session, id string) (deviceRecord, error) {
	var d deviceRecord
	if id == "" {
		return d, badRequest("device_id is required")
	}
	err := s.pool.QueryRow(ctx,
		"SELECT user_id, workspace_id, last_mutation_id FROM devices WHERE id = $1", id).
		Scan(&d.userID, &d.workspaceID, &d.lastMutationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, forbidden("device does not belong to this user")
	}
	if err != nil {
		return d, fmt.Errorf("read device: %w", err)
	}
	if d.userID != sess.userID || d.workspaceID != sess.workspaceID {
		return d, forbidden("device does not belong to this user")
	}
	return d, nil
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "ok")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("sync server: write response: %v", err)
	}
}

// decodeJSON reads one bounded JSON value into dst and rejects trailing data.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, syncproto.MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return statusErrorf(http.StatusRequestEntityTooLarge, "request body too large")
		}
		return badRequest("invalid JSON: %v", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return badRequest("invalid JSON: trailing data")
	}
	return nil
}

// decodeData decodes a row's JSONB text. Numbers stay json.Number so a value above 2^53 survives
// a merge and the next pull instead of being rounded through float64.
func decodeData(raw string) (map[string]any, error) {
	data := map[string]any{}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]any{}
	}
	return data, nil
}
