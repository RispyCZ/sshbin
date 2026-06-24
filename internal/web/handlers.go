package web

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/rispycz/securedrop/internal/sharing"
)

type handler struct {
	repo    sharing.Repository
	baseURL string
	host    string
	tpl     *templates
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
	h.renderSetup(w, s, false)
}

func (h *handler) setupPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lookup(w, r.Context(), r.PathValue("id"))
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		h.render(w, http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
		return
	}

	s.ExpiresAt = parseExpiry(r.FormValue("expires"), time.Now())
	s.Public = r.FormValue("public") != ""
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

func (h *handler) shareView(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lookup(w, r.Context(), r.PathValue("id"))
	if !ok {
		return
	}
	if !s.Configured {
		h.render(w, http.StatusNotFound, "error", errData(http.StatusNotFound, "This share has not been set up yet."))
		return
	}
	if s.Expired(time.Now()) {
		h.render(w, http.StatusGone, "error", errData(http.StatusGone, "This share has expired."))
		return
	}
	h.render(w, http.StatusOK, "share", map[string]any{"Sharing": s})
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
