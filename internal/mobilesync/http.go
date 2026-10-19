package mobilesync

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const MaxRequestBytes = 1 << 20

// NewHandler requires an explicit shared secret. It is a single-owner private
// transport, not a multiuser authentication or tenant-isolation system.
func NewHandler(s *Store, token string) (http.Handler, error) {
	if s == nil || strings.TrimSpace(token) == "" || strings.TrimSpace(token) != token || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("a nonempty bearer token without surrounding whitespace is required")
	}
	digest := sha256.Sum256([]byte("Bearer " + token))
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			got := sha256.Sum256([]byte(r.Header.Get("Authorization")))
			if subtle.ConstantTimeCompare(got[:], digest[:]) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next(w, r)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.PingContext(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/sync/push", auth(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBytes)
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeError(w, http.StatusRequestEntityTooLarge, "request_too_large")
			} else {
				writeError(w, http.StatusBadRequest, "invalid_json")
			}
			return
		}
		var request PushRequest
		if err := decodeStrict(raw, &request); err != nil || !validID(request.DeviceID) || request.Mutations == nil || len(request.Mutations) > MaxMutations {
			writeError(w, http.StatusBadRequest, "invalid_push")
			return
		}
		out, err := s.Push(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}))
	mux.HandleFunc("GET /v1/sync/pull", auth(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		cursor, cursorErr := strconv.ParseInt(query.Get("cursor"), 10, 64)
		limit := 100
		var limitErr error
		if text := query.Get("limit"); text != "" {
			limit, limitErr = strconv.Atoi(text)
		}
		if cursorErr != nil || limitErr != nil || cursor < 0 || cursor > maxSafeInteger || limit < 1 || limit > MaxPullLimit || len(query.Get("epoch")) > 128 {
			writeError(w, http.StatusBadRequest, "invalid_cursor")
			return
		}
		out, err := s.Pull(r.Context(), cursor, limit, query.Get("epoch"))
		if errors.Is(err, ErrResnapshot) {
			writeError(w, http.StatusConflict, "resnapshot_required")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}))
	mux.HandleFunc("GET /v1/sync/snapshot", auth(func(w http.ResponseWriter, r *http.Request) {
		out, err := s.Snapshot(r.Context())
		if errors.Is(err, ErrSnapshotTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "snapshot_too_large")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}))
	// Keep even method/route errors JSON; never reflect request bodies or tokens.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h, pattern := mux.Handler(r)
		if pattern == "" {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		h.ServeHTTP(w, r)
	}), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
