package web

import (
	"errors"
	"net/http"

	"net/url"
	"strings"
	"time"

	"github.com/rispycz/sshbin/internal/auth"
)

const sessionCookie = "fd_session"

func (h *handler) currentSession(r *http.Request) (auth.Session, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return auth.Session{}, false
	}
	return h.auth.Session(c.Value)
}

func (h *handler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// requireSession redirects unauthenticated requests to the login page, carrying
// the original path so the user returns there after verifying.
func (h *handler) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.currentSession(r); !ok {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func verifyError(err error) string {
	switch {
	case errors.Is(err, auth.ErrInvalidCode):
		return "That code is not correct."
	case errors.Is(err, auth.ErrTooManyAttempts):
		return "Too many attempts. Request a new code."
	case errors.Is(err, auth.ErrChallengeExpired):
		return "That code expired. Request a new one."
	default:
		return "Could not verify the code."
	}
}

func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return email
	}
	local := email[:at]
	domain := email[at:]
	mask := strings.Repeat("•", len(local))
	return mask + domain
}
