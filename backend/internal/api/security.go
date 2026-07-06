package api

// Durcissement (audit) : anti brute-force sur le PIN et CORS configurable.

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── Anti brute-force (EF-002 / ENF-004) ───────────────────────────────────────
// Un PIN court se brute-force en secondes sans garde-fou : 5 échecs sur un
// profil verrouillent les tentatives 15 minutes (en mémoire — suffisant pour
// une instance foyer ; redémarrer le serveur réinitialise, assumé).

const (
	maxLoginFailures = 5
	loginLockWindow  = 15 * time.Minute
)

type loginLimiter struct {
	mu    sync.Mutex
	state map[string]*loginAttempts
}

type loginAttempts struct {
	failures    int
	lockedUntil time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{state: map[string]*loginAttempts{}}
}

// locked renvoie le temps d'attente restant si le profil est verrouillé.
func (l *loginLimiter) locked(profileID string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a, ok := l.state[profileID]
	if !ok {
		return 0, false
	}
	if remaining := time.Until(a.lockedUntil); remaining > 0 {
		return remaining, true
	}
	return 0, false
}

// fail enregistre un échec ; au 5e, le profil est verrouillé 15 minutes.
func (l *loginLimiter) fail(profileID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a, ok := l.state[profileID]
	if !ok {
		a = &loginAttempts{}
		l.state[profileID] = a
	}
	a.failures++
	if a.failures >= maxLoginFailures {
		a.lockedUntil = time.Now().Add(loginLockWindow)
		a.failures = 0
	}
}

// reset efface le compteur après une connexion réussie.
func (l *loginLimiter) reset(profileID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.state, profileID)
}

// writeLocked répond 429 avec le délai à respecter.
func writeLocked(w http.ResponseWriter, remaining time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(remaining.Seconds())+1))
	writeError(w, http.StatusTooManyRequests, "too_many_attempts",
		"Trop de tentatives : profil verrouillé pendant "+
			strconv.Itoa(int(remaining.Minutes())+1)+" minute(s).")
}

// ── CORS (préparation de l'app web) ───────────────────────────────────────────
// Désactivé par défaut ; OPALE_CORS_ORIGINS = liste d'origines autorisées
// séparées par des virgules (ou « * » — déconseillé hors dev).

func corsMiddleware(originsCSV string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	allowAll := false
	for _, o := range strings.Split(originsCSV, ",") {
		o = strings.TrimSpace(strings.TrimRight(o, "/"))
		if o == "*" {
			allowAll = true
		} else if o != "" {
			allowed[o] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || allowed[strings.TrimRight(origin, "/")]) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "3600")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
