package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/rispycz/sshbin/internal/sharing"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Error("encode json", "err", err)
		}
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// requireSessionAPI rejects unauthenticated API calls with 401 JSON instead of
// the HTML redirect used by the server-rendered pages.
func (h *handler) requireSessionAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.currentSession(r); !ok {
			writeErr(w, http.StatusUnauthorized, "Not signed in.")
			return
		}
		next(w, r)
	}
}

func (h *handler) apiSession(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.currentSession(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not signed in.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"email": sess.Email})
}

func (h *handler) apiLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request.")
		return
	}
	email := strings.TrimSpace(body.Email)
	if email == "" {
		writeErr(w, http.StatusBadRequest, "Enter an email address.")
		return
	}
	if err := h.auth.StartLogin(r.Context(), email); err != nil {
		log.Error("start login", "err", err)
		writeErr(w, http.StatusInternalServerError, "Could not send a code.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"maskedEmail": maskEmail(email)})
}

func (h *handler) apiVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request.")
		return
	}
	token, sess, err := h.auth.Verify(strings.TrimSpace(body.Email), strings.TrimSpace(body.Code))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, verifyError(err))
		return
	}
	h.setSessionCookie(w, token, sess.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]string{"email": sess.Email})
}

func (h *handler) apiLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		h.auth.Logout(c.Value)
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

type shareDTO struct {
	ID            string   `json:"id"`
	FileName      string   `json:"fileName"`
	Configured    bool     `json:"configured"`
	Public        bool     `json:"public"`
	Expired       bool     `json:"expired"`
	ExpiresAt     *string  `json:"expiresAt"`
	AllowedEmails []string `json:"allowedEmails"`
	ShareURL      string   `json:"shareURL"`
}

func (h *handler) apiShares(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	shares, err := h.repo.ListByOwner(r.Context(), sess.Email)
	if err != nil {
		log.Error("list shares", "email", sess.Email, "err", err)
		writeErr(w, http.StatusInternalServerError, "Could not load shares.")
		return
	}
	now := time.Now()
	out := make([]shareDTO, 0, len(shares))
	for _, s := range shares {
		var exp *string
		if s.ExpiresAt != nil {
			v := s.ExpiresAt.Format(time.RFC3339)
			exp = &v
		}
		emails := s.AllowedEmails
		if emails == nil {
			emails = []string{}
		}
		out = append(out, shareDTO{
			ID:            s.ID,
			FileName:      s.FileName,
			Configured:    s.Configured,
			Public:        s.Public,
			Expired:       s.Expired(now),
			ExpiresAt:     exp,
			AllowedEmails: emails,
			ShareURL:      h.baseURL + "/s/" + s.ID,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) apiDeleteShare(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, sharing.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "Share not found.")
		return
	}
	if err != nil {
		log.Error("get sharing", "id", id, "err", err)
		writeErr(w, http.StatusInternalServerError, "Something went wrong.")
		return
	}
	sess, _ := h.currentSession(r)
	if s.OwnerEmail != sess.Email {
		writeErr(w, http.StatusForbidden, "This share belongs to someone else.")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		log.Error("delete share", "id", id, "err", err)
		writeErr(w, http.StatusInternalServerError, "Could not delete share.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
