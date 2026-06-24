package web

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rispycz/securedrop/internal/auth"
	"github.com/rispycz/securedrop/internal/sharing"
)

type handler struct {
	repo          sharing.Repository
	auth          *auth.Manager
	baseURL       string
	host          string
	secureCookies bool
	tpl           *templates
}

var expiryPresets = map[string]time.Duration{
	"1h":   time.Hour,
	"24h":  24 * time.Hour,
	"168h": 7 * 24 * time.Hour,
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "index", map[string]any{"Host": h.host})
}

func (h *handler) setupGet(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lookup(w, r.Context(), r.PathValue("id"))
	if !ok {
		return
	}
	sess, _ := h.currentSession(r) // guaranteed by requireSession
	if !h.ownsOrClaimable(w, s, sess.Email) {
		return
	}
	h.renderSetup(w, s, false)
}

func (h *handler) setupPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lookup(w, r.Context(), r.PathValue("id"))
	if !ok {
		return
	}
	sess, _ := h.currentSession(r)
	if !h.ownsOrClaimable(w, s, sess.Email) {
		return
	}
	if err := r.ParseForm(); err != nil {
		h.render(w, http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
		return
	}

	s.OwnerEmail = sess.Email
	s.ExpiresAt = parseExpiry(r.FormValue("expires"), time.Now())
	s.Public = r.FormValue("visibility") != "private"
	s.AllowedEmails = nil
	if !s.Public {
		s.AllowedEmails = sharing.ParseEmails(r.FormValue("emails"))
	}
	if pw := r.FormValue("password"); pw != "" {
		hash, err := sharing.HashPassword(pw)
		if err != nil {
			log.Printf("hash password: %v", err)
			h.render(w, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not save settings."))
			return
		}
		s.PasswordHash = hash
	}
	s.Configured = true

	if err := h.repo.Update(r.Context(), s); err != nil {
		log.Printf("update sharing %s: %v", s.ID, err)
		h.render(w, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not save settings."))
		return
	}
	h.renderSetup(w, s, true)
}

// ownsOrClaimable allows access to setup when the share is unclaimed or already
// owned by the current session's email. A claimed share is private to its owner.
func (h *handler) ownsOrClaimable(w http.ResponseWriter, s sharing.Sharing, email string) bool {
	if s.OwnerEmail == "" || s.OwnerEmail == email {
		return true
	}
	h.render(w, http.StatusForbidden, "error", errData(http.StatusForbidden, "This share belongs to someone else."))
	return false
}

func (h *handler) shareView(w http.ResponseWriter, r *http.Request) {
	s, ok := h.accessibleShare(w, r)
	if !ok {
		return
	}
	if s.HasPassword() {
		h.render(w, http.StatusOK, "share_password", map[string]any{"Sharing": s})
		return
	}
	h.render(w, http.StatusOK, "share", map[string]any{"Sharing": s})
}

func (h *handler) sharePassword(w http.ResponseWriter, r *http.Request) {
	s, ok := h.accessibleShare(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		h.render(w, http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
		return
	}
	if s.HasPassword() && !s.CheckPassword(r.FormValue("password")) {
		h.render(w, http.StatusUnauthorized, "share_password", map[string]any{
			"Sharing": s, "Error": "Incorrect password.",
		})
		return
	}
	h.render(w, http.StatusOK, "share", map[string]any{"Sharing": s})
}

// accessibleShare resolves a share and enforces existence, configuration,
// expiry, and visibility (private shares require a session whose email is
// allowlisted). Password is enforced separately by the caller. It writes the
// appropriate response and returns false when access is denied.
func (h *handler) accessibleShare(w http.ResponseWriter, r *http.Request) (sharing.Sharing, bool) {
	s, ok := h.lookup(w, r.Context(), r.PathValue("id"))
	if !ok {
		return sharing.Sharing{}, false
	}
	if !s.Configured {
		h.render(w, http.StatusNotFound, "error", errData(http.StatusNotFound, "This share has not been set up yet."))
		return sharing.Sharing{}, false
	}
	if s.Expired(time.Now()) {
		h.render(w, http.StatusGone, "error", errData(http.StatusGone, "This share has expired."))
		return sharing.Sharing{}, false
	}
	if !s.Public {
		sess, ok := h.currentSession(r)
		if !ok {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusSeeOther)
			return sharing.Sharing{}, false
		}
		if s.OwnerEmail != sess.Email && !s.AllowsEmail(sess.Email) {
			h.render(w, http.StatusForbidden, "error", errData(http.StatusForbidden, "You don't have access to this share."))
			return sharing.Sharing{}, false
		}
	}
	return s, true
}

// lookup fetches a sharing by id, rendering a 404 and returning false when it is
// absent. Other repository errors render a 500.
func (h *handler) lookup(w http.ResponseWriter, ctx context.Context, id string) (sharing.Sharing, bool) {
	s, err := h.repo.Get(ctx, id)
	if errors.Is(err, sharing.ErrNotFound) {
		h.render(w, http.StatusNotFound, "error", errData(http.StatusNotFound, "We couldn't find that share."))
		return sharing.Sharing{}, false
	}
	if err != nil {
		log.Printf("get sharing %s: %v", id, err)
		h.render(w, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Something went wrong."))
		return sharing.Sharing{}, false
	}
	return s, true
}

func (h *handler) renderSetup(w http.ResponseWriter, s sharing.Sharing, saved bool) {
	expires := ""
	if s.ExpiresAt == nil {
		expires = "never"
	}
	h.render(w, http.StatusOK, "setup", map[string]any{
		"Sharing":  s,
		"Saved":    saved,
		"Expires":  expires,
		"Emails":   strings.Join(s.AllowedEmails, "\n"),
		"ShareURL": h.baseURL + "/s/" + s.ID,
	})
}

// render writes a page, buffering first so a template error doesn't emit a
// half-written response with an already-committed 200 status.
func (h *handler) render(w http.ResponseWriter, status int, page string, data any) {
	var buf bytes.Buffer
	if err := h.tpl.render(&buf, page, data); err != nil {
		log.Printf("render %s: %v", page, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func errData(status int, msg string) map[string]any {
	return map[string]any{"Status": status, "Message": msg}
}

func parseExpiry(value string, now time.Time) *time.Time {
	d, ok := expiryPresets[value]
	if !ok {
		return nil
	}
	t := now.Add(d)
	return &t
}
