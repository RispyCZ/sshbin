package web

import (
	"errors"
	"log"
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

func (h *handler) loginGet(w http.ResponseWriter, r *http.Request) {
	h.render(w, r,http.StatusOK, "login", map[string]any{"Next": safeNext(r.URL.Query().Get("next"))})
}

func (h *handler) loginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r,http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	next := safeNext(r.FormValue("next"))
	if email == "" {
		h.render(w, r,http.StatusBadRequest, "login", map[string]any{"Next": next, "Error": "Enter an email address."})
		return
	}
	if err := h.auth.StartLogin(r.Context(), email); err != nil {
		log.Printf("start login: %v", err)
		h.render(w, r,http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not send a code."))
		return
	}
	h.render(w, r,http.StatusOK, "verify", map[string]any{"Email": email, "Next": next})
}

func (h *handler) verifyPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r,http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	code := strings.TrimSpace(r.FormValue("code"))
	next := safeNext(r.FormValue("next"))

	token, sess, err := h.auth.Verify(email, code)
	if err != nil {
		h.render(w, r,http.StatusUnauthorized, "verify", map[string]any{
			"Email": email, "Next": next, "Error": verifyError(err),
		})
		return
	}
	h.setSessionCookie(w, token, sess.ExpiresAt)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		h.auth.Logout(c.Value)
	}
	h.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
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

// safeNext returns a safe local redirect target, rejecting absolute or
// protocol-relative URLs to prevent open redirects.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/"
	}
	return next
}
