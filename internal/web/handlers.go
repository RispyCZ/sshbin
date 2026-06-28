package web

import (
	"bytes"
	"errors"
	"mime"

	"github.com/charmbracelet/log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/rispycz/sshbin/internal/auth"
	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
	"github.com/rispycz/sshbin/internal/userprefs"
)

type handler struct {
	repo          sharing.Repository
	storage       storage.Storage
	auth          *auth.Manager
	prefs         userprefs.Repository
	baseURL       string
	host          string
	secureCookies bool
	secret        []byte
	tpl           *templates
}

var expiryPresets = map[string]time.Duration{
	"1h":   time.Hour,
	"24h":  24 * time.Hour,
	"168h": 7 * 24 * time.Hour,
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, "index", map[string]any{"Host": h.host})
}

type shareListItem struct {
	S        sharing.Sharing
	Expired  bool
	ShareURL string
	Expires  string
	Emails   string
}

func (h *handler) sharesList(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	shares, err := h.repo.ListByOwner(r.Context(), sess.Email)
	if err != nil {
		log.Error("list shares", "email", sess.Email, "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not load shares."))
		return
	}
	now := time.Now()
	items := make([]shareListItem, len(shares))
	for i, s := range shares {
		items[i] = shareListItem{
			S:        s,
			Expired:  s.Expired(now),
			ShareURL: h.baseURL + "/s/" + s.ID,
			Expires:  expiresBucket(s),
			Emails:   strings.Join(s.AllowedEmails, "\n"),
		}
	}
	h.render(w, r, http.StatusOK, "shares", map[string]any{"Items": items})
}

func (h *handler) shareQR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.repo.Get(r.Context(), id); err != nil {
		http.NotFound(w, r)
		return
	}
	png, err := qrcode.Encode(h.baseURL+"/s/"+id, qrcode.Medium, 256)
	if err != nil {
		log.Error("qr encode", "id", id, "err", err)
		http.Error(w, "qr error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(png)
}

func (h *handler) deleteShare(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s, ok := h.lookup(w, r, id)
	if !ok {
		return
	}
	sess, _ := h.currentSession(r)
	if s.OwnerEmail != sess.Email {
		h.render(w, r, http.StatusForbidden, "error", errData(http.StatusForbidden, "This share belongs to someone else."))
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		log.Error("delete share", "id", id, "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not delete share."))
		return
	}
	http.Redirect(w, r, "/shares", http.StatusSeeOther)
}

func (h *handler) setupGet(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lookup(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	sess, _ := h.currentSession(r) // guaranteed by requireSession
	if !h.ownsOrClaimable(w, r, s, sess.Email) {
		return
	}
	if !s.Configured {
		if prefs, err := h.prefs.Get(r.Context(), sess.Email); err == nil {
			s.Public = prefs.DefaultPublic
		}
	}
	h.renderSetup(w, r, s, false)
}

func (h *handler) setupPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.lookup(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	sess, _ := h.currentSession(r)
	if !h.ownsOrClaimable(w, r, s, sess.Email) {
		return
	}
	if err := r.ParseForm(); err != nil {
		h.render(w, r, http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
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
			log.Error("hash password", "err", err)
			h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not save settings."))
			return
		}
		s.PasswordHash = hash
	}
	s.Configured = true

	if err := h.repo.Update(r.Context(), s); err != nil {
		log.Error("update sharing", "id", s.ID, "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not save settings."))
		return
	}
	if r.FormValue("from") == "shares" {
		http.Redirect(w, r, "/shares", http.StatusSeeOther)
		return
	}
	h.renderSetup(w, r, s, true)
}

// ownsOrClaimable allows access to setup when the share is unclaimed or already
// owned by the current session's email. A claimed share is private to its owner.
func (h *handler) ownsOrClaimable(w http.ResponseWriter, r *http.Request, s sharing.Sharing, email string) bool {
	if s.OwnerEmail == "" || s.OwnerEmail == email {
		return true
	}
	h.render(w, r, http.StatusForbidden, "error", errData(http.StatusForbidden, "This share belongs to someone else."))
	return false
}

func (h *handler) shareView(w http.ResponseWriter, r *http.Request) {
	s, ok := h.accessibleShare(w, r)
	if !ok {
		return
	}
	if !h.hasPasswordGrant(r, s) {
		h.render(w, r, http.StatusOK, "share_password", map[string]any{"Sharing": s})
		return
	}
	h.render(w, r, http.StatusOK, "share", map[string]any{
		"Sharing":     s,
		"DownloadURL": "/s/" + s.ID + "/download",
	})
}

func (h *handler) sharePassword(w http.ResponseWriter, r *http.Request) {
	s, ok := h.accessibleShare(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		h.render(w, r, http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
		return
	}
	if s.HasPassword() && !s.CheckPassword(r.FormValue("password")) {
		h.render(w, r, http.StatusUnauthorized, "share_password", map[string]any{
			"Sharing": s, "Error": "Incorrect password.",
		})
		return
	}
	// Grant access, then redirect (POST/redirect/GET) so refresh and the
	// download link work without re-prompting.
	h.setPasswordGrant(w, s.ID)
	http.Redirect(w, r, "/s/"+s.ID, http.StatusSeeOther)
}

func (h *handler) download(w http.ResponseWriter, r *http.Request) {
	s, ok := h.accessibleShare(w, r)
	if !ok {
		return
	}
	if !h.hasPasswordGrant(r, s) {
		http.Redirect(w, r, "/s/"+s.ID, http.StatusSeeOther)
		return
	}

	rc, err := h.storage.Open(r.Context(), s.FileID, s.FileName)
	if errors.Is(err, storage.ErrNotFound) {
		h.render(w, r, http.StatusNotFound, "error", errData(http.StatusNotFound, "The file is no longer available."))
		return
	}
	if err != nil {
		log.Error("open file", "id", s.FileID, "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not read the file."))
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", contentDisposition(s.FileName))
	modTime := s.CreatedAt
	http.ServeContent(w, r, s.FileName, modTime, rc)
}

// accessibleShare resolves a share and enforces existence, configuration,
// expiry, and visibility (private shares require a session whose email is
// allowlisted). Password is enforced separately by the caller. It writes the
// appropriate response and returns false when access is denied.
func (h *handler) accessibleShare(w http.ResponseWriter, r *http.Request) (sharing.Sharing, bool) {
	s, ok := h.lookup(w, r, r.PathValue("id"))
	if !ok {
		return sharing.Sharing{}, false
	}
	if !s.Configured {
		h.render(w, r, http.StatusNotFound, "error", errData(http.StatusNotFound, "This share has not been set up yet."))
		return sharing.Sharing{}, false
	}
	if s.Expired(time.Now()) {
		h.render(w, r, http.StatusGone, "error", errData(http.StatusGone, "This share has expired."))
		return sharing.Sharing{}, false
	}
	if !s.Public {
		sess, ok := h.currentSession(r)
		if !ok {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusSeeOther)
			return sharing.Sharing{}, false
		}
		if s.OwnerEmail != sess.Email && !s.AllowsEmail(sess.Email) {
			h.render(w, r, http.StatusForbidden, "error", errData(http.StatusForbidden, "You don't have access to this share."))
			return sharing.Sharing{}, false
		}
	}
	return s, true
}

// lookup fetches a sharing by id, rendering a 404 and returning false when it is
// absent. Other repository errors render a 500.
func (h *handler) lookup(w http.ResponseWriter, r *http.Request, id string) (sharing.Sharing, bool) {
	s, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, sharing.ErrNotFound) {
		h.render(w, r, http.StatusNotFound, "error", errData(http.StatusNotFound, "We couldn't find that share."))
		return sharing.Sharing{}, false
	}
	if err != nil {
		log.Error("get sharing", "id", id, "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Something went wrong."))
		return sharing.Sharing{}, false
	}
	return s, true
}

func (h *handler) renderSetup(w http.ResponseWriter, r *http.Request, s sharing.Sharing, saved bool) {
	h.render(w, r, http.StatusOK, "setup", map[string]any{
		"Sharing":  s,
		"Saved":    saved,
		"Expires":  expiresBucket(s),
		"Emails":   strings.Join(s.AllowedEmails, "\n"),
		"ShareURL": h.baseURL + "/s/" + s.ID,
	})
}

// expiresBucket maps a share's expiry to the radio value used by the setup form.
// A concrete future timestamp has no preset bucket and returns "".
func expiresBucket(s sharing.Sharing) string {
	if s.ExpiresAt == nil {
		return "never"
	}
	return ""
}

// render writes a page, buffering first so a template error doesn't emit a
// half-written response with an already-committed 200 status. It injects the
// current session into data so the base layout can render the user menu.
func (h *handler) render(w http.ResponseWriter, r *http.Request, status int, page string, data map[string]any) {
	if sess, ok := h.currentSession(r); ok {
		data["Session"] = sess
	}
	var buf bytes.Buffer
	if err := h.tpl.render(&buf, page, data); err != nil {
		log.Error("render", "page", page, "err", err)
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

func (h *handler) profileGet(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	prefs, err := h.prefs.Get(r.Context(), sess.Email)
	if err != nil {
		log.Error("get user prefs", "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not load preferences."))
		return
	}
	h.render(w, r, http.StatusOK, "profile", map[string]any{"Prefs": prefs})
}

func (h *handler) profilePost(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	if err := r.ParseForm(); err != nil {
		h.render(w, r, http.StatusBadRequest, "error", errData(http.StatusBadRequest, "Invalid form submission."))
		return
	}
	prefs := userprefs.UserPrefs{
		Email:         sess.Email,
		DefaultPublic: r.FormValue("default_visibility") == "public",
	}
	if err := h.prefs.Upsert(r.Context(), prefs); err != nil {
		log.Error("upsert user prefs", "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not save preferences."))
		return
	}
	http.Redirect(w, r, "/profile?flash=saved", http.StatusSeeOther)
}

func (h *handler) profileDelete(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.currentSession(r)
	email := sess.Email
	if err := h.repo.DeleteByOwner(r.Context(), email); err != nil {
		log.Error("delete shares by owner", "email", email, "err", err)
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Could not delete data."))
		return
	}
	if err := h.prefs.Delete(r.Context(), email); err != nil {
		log.Error("delete user prefs", "email", email, "err", err)
	}
	if err := h.auth.DeleteSessionsByEmail(email); err != nil {
		log.Error("delete sessions", "email", email, "err", err)
	}
	h.clearSessionCookie(w)
	http.Redirect(w, r, "/?flash=deleted", http.StatusSeeOther)
}

// contentDisposition builds a safe attachment header, dropping path components
// and control characters and letting mime encode non-ASCII names (RFC 2231).
func contentDisposition(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "download"
	}
	if v := mime.FormatMediaType("attachment", map[string]string{"filename": name}); v != "" {
		return v
	}
	return "attachment"
}

func parseExpiry(value string, now time.Time) *time.Time {
	d, ok := expiryPresets[value]
	if !ok {
		return nil
	}
	t := now.Add(d)
	return &t
}
