package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/opale-app/opale/internal/auth"
	"github.com/opale-app/opale/internal/store"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyProfile
)

// requestID attribue un identifiant unique à chaque requête (traçabilité — ENF-011).
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		id := hex.EncodeToString(b)
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// statusRecorder capture le code de statut pour la journalisation.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// logRequests journalise chaque requête (logs structurés — ENF-011).
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		s.log.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sr.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", r.Context().Value(ctxKeyRequestID),
		)
	})
}

// recoverer transforme un panic en réponse 500 propre.
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "recover", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// requireAuth exige un jeton de session valide et place le profil dans le contexte.
// Accepte « Authorization: Bearer <token> » ou l'en-tête « X-Session-Token ».
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Session-Token")
		if token == "" {
			if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				token = strings.TrimPrefix(h, "Bearer ")
			}
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "jeton de session requis")
			return
		}

		profile, err := s.store.ProfileForSession(r.Context(), auth.HashToken(token))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "session invalide ou expirée")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyProfile, profile)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// profileFromContext récupère le profil authentifié (présent après requireAuth).
func profileFromContext(ctx context.Context) store.Profile {
	p, _ := ctx.Value(ctxKeyProfile).(store.Profile)
	return p
}
