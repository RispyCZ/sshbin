package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/userprefs"
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
	HasPassword   bool     `json:"hasPassword"`
	CreatedAt     string   `json:"createdAt"`
	ShareURL      string   `json:"shareURL"`
}

func (h *handler) toShareDTO(s sharing.Sharing, now time.Time) shareDTO {
	var exp *string
	if s.ExpiresAt != nil {
		v := s.ExpiresAt.Format(time.RFC3339)
		exp = &v
	}
	emails := s.AllowedEmails
	if emails == nil {
		emails = []string{}
	}
	return shareDTO{
		ID:            s.ID,
		FileName:      s.FileName,
		Configured:    s.Configured,
		Public:        s.Public,
		Expired:       s.Expired(now),
		ExpiresAt:     exp,
		AllowedEmails: emails,
		HasPassword:   s.HasPassword(),
		CreatedAt:     s.CreatedAt.Format(time.RFC3339),
		ShareURL:      h.baseURL + "/s/" + s.ID,
	}
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
		out = append(out, h.toShareDTO(s, now))
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

// writeAccessErr maps a non-OK access result to a JSON error response and
// reports whether it wrote one (true means the caller should stop).
func writeAccessErr(w http.ResponseWriter, res accessResult) bool {
	switch res {
	case accessOK:
		return false
	case accessNotFound:
		writeErr(w, http.StatusNotFound, "We couldn't find that share.")
	case accessNotConfigured:
		writeErr(w, http.StatusNotFound, "This share has not been set up yet.")
	case accessExpired:
		writeErr(w, http.StatusGone, "This share has expired.")
	case accessNeedLogin:
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Sign in to view this share.",
			"code":  "login_required",
		})
	case accessForbidden:
		writeErr(w, http.StatusForbidden, "You don't have access to this share.")
	default:
		writeErr(w, http.StatusInternalServerError, "Something went wrong.")
	}
	return true
}

type shareViewDTO struct {
	FileName         string  `json:"fileName"`
	RequiresPassword bool    `json:"requiresPassword"`
	Unlocked         bool    `json:"unlocked"`
	ExpiresAt        *string `json:"expiresAt"`
	DownloadURL      string  `json:"downloadURL"`
}

func (h *handler) shareViewDTO(r *http.Request, s sharing.Sharing) shareViewDTO {
	var exp *string
	if s.ExpiresAt != nil {
		v := s.ExpiresAt.Format(time.RFC3339)
		exp = &v
	}
	return shareViewDTO{
		FileName:         s.FileName,
		RequiresPassword: s.HasPassword(),
		Unlocked:         h.hasPasswordGrant(r, s),
		ExpiresAt:        exp,
		DownloadURL:      "/s/" + s.ID + "/download",
	}
}

func (h *handler) apiShareView(w http.ResponseWriter, r *http.Request) {
	s, res := h.resolveShare(r, r.PathValue("id"))
	if writeAccessErr(w, res) {
		return
	}
	writeJSON(w, http.StatusOK, h.shareViewDTO(r, s))
}

func (h *handler) apiSharePassword(w http.ResponseWriter, r *http.Request) {
	s, res := h.resolveShare(r, r.PathValue("id"))
	if writeAccessErr(w, res) {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request.")
		return
	}
	if s.HasPassword() && !s.CheckPassword(in.Password) {
		writeErr(w, http.StatusUnauthorized, "Incorrect password.")
		return
	}
	h.setPasswordGrant(w, s.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"unlocked":    true,
		"downloadURL": "/s/" + s.ID + "/download",
	})
}

var expiryPresets = map[string]time.Duration{
	"1h":   time.Hour,
	"24h":  24 * time.Hour,
	"168h": 7 * 24 * time.Hour,
}

// parseExpiry maps a preset key to an absolute time. Unknown keys (including
// "never") yield nil, meaning the share never expires.
func parseExpiry(value string, now time.Time) *time.Time {
	d, ok := expiryPresets[value]
	if !ok {
		return nil
	}
	t := now.Add(d)
	return &t
}

type setupInput struct {
	Expires    string   `json:"expires"`
	Visibility string   `json:"visibility"`
	Emails     []string `json:"emails"`
	Password   string   `json:"password"`
}

func (h *handler) apiSetup(w http.ResponseWriter, r *http.Request) {
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
	// A share is claimable while unowned; once claimed it is private to its owner.
	if s.OwnerEmail != "" && s.OwnerEmail != sess.Email {
		writeErr(w, http.StatusForbidden, "This share belongs to someone else.")
		return
	}

	var in setupInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request.")
		return
	}

	s.OwnerEmail = sess.Email
	s.ExpiresAt = parseExpiry(in.Expires, time.Now())
	s.Public = in.Visibility != "private"
	s.AllowedEmails = nil
	if !s.Public {
		s.AllowedEmails = sharing.ParseEmails(strings.Join(in.Emails, "\n"))
	}
	if in.Password != "" {
		hash, err := sharing.HashPassword(in.Password)
		if err != nil {
			log.Error("hash password", "err", err)
			writeErr(w, http.StatusInternalServerError, "Could not save settings.")
			return
		}
		s.PasswordHash = hash
	}
	s.Configured = true

	if err := h.repo.Update(r.Context(), s); err != nil {
		log.Error("update sharing", "id", s.ID, "err", err)
		writeErr(w, http.StatusInternalServerError, "Could not save settings.")
		return
	}
	writeJSON(w, http.StatusOK, h.toShareDTO(s, time.Now()))
}

func (h *handler) apiProfileGet(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	prefs, err := h.prefs.Get(r.Context(), sess.Email)
	if err != nil {
		log.Error("get user prefs", "err", err)
		writeErr(w, http.StatusInternalServerError, "Could not load preferences.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email":         sess.Email,
		"defaultPublic": prefs.DefaultPublic,
	})
}

func (h *handler) apiProfileSave(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	var in struct {
		DefaultPublic bool `json:"defaultPublic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request.")
		return
	}
	prefs := userprefs.UserPrefs{Email: sess.Email, DefaultPublic: in.DefaultPublic}
	if err := h.prefs.Upsert(r.Context(), prefs); err != nil {
		log.Error("upsert user prefs", "err", err)
		writeErr(w, http.StatusInternalServerError, "Could not save preferences.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) apiProfileDeleteAll(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	email := sess.Email
	if err := h.repo.DeleteByOwner(r.Context(), email); err != nil {
		log.Error("delete shares by owner", "email", email, "err", err)
		writeErr(w, http.StatusInternalServerError, "Could not delete data.")
		return
	}
	if err := h.prefs.Delete(r.Context(), email); err != nil {
		log.Error("delete user prefs", "email", email, "err", err)
	}
	if err := h.auth.DeleteSessionsByEmail(email); err != nil {
		log.Error("delete sessions", "email", email, "err", err)
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
