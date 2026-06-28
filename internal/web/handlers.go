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
