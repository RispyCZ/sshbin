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

func (h *handler) download(w http.ResponseWriter, r *http.Request) {
	s, ok := h.accessibleShareHTML(w, r)
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

type accessResult int

const (
	accessOK accessResult = iota
	accessNotFound
	accessNotConfigured
	accessExpired
	accessNeedLogin
	accessForbidden
	accessError
)

// resolveShare fetches a share and classifies access: existence, configuration,
// expiry, and visibility (private shares require a session whose email is
// allowlisted). It is pure — callers render HTML or JSON from the result.
// Password enforcement is separate (see hasPasswordGrant).
func (h *handler) resolveShare(r *http.Request, id string) (sharing.Sharing, accessResult) {
	s, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, sharing.ErrNotFound) {
		return sharing.Sharing{}, accessNotFound
	}
	if err != nil {
		log.Error("get sharing", "id", id, "err", err)
		return sharing.Sharing{}, accessError
	}
	if !s.Configured {
		return s, accessNotConfigured
	}
	if s.Expired(time.Now()) {
		return s, accessExpired
	}
	if !s.Public {
		sess, ok := h.currentSession(r)
		if !ok {
			return s, accessNeedLogin
		}
		if s.OwnerEmail != sess.Email && !s.AllowsEmail(sess.Email) {
			return s, accessForbidden
		}
	}
	return s, accessOK
}

// accessibleShareHTML resolves a share for the server-rendered download route,
// writing an HTML error page (or login redirect) and returning false on denial.
func (h *handler) accessibleShareHTML(w http.ResponseWriter, r *http.Request) (sharing.Sharing, bool) {
	s, res := h.resolveShare(r, r.PathValue("id"))
	switch res {
	case accessOK:
		return s, true
	case accessNotFound:
		h.render(w, r, http.StatusNotFound, "error", errData(http.StatusNotFound, "We couldn't find that share."))
	case accessNotConfigured:
		h.render(w, r, http.StatusNotFound, "error", errData(http.StatusNotFound, "This share has not been set up yet."))
	case accessExpired:
		h.render(w, r, http.StatusGone, "error", errData(http.StatusGone, "This share has expired."))
	case accessNeedLogin:
		http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusSeeOther)
	case accessForbidden:
		h.render(w, r, http.StatusForbidden, "error", errData(http.StatusForbidden, "You don't have access to this share."))
	default:
		h.render(w, r, http.StatusInternalServerError, "error", errData(http.StatusInternalServerError, "Something went wrong."))
	}
	return sharing.Sharing{}, false
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
