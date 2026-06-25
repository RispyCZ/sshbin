package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/rispycz/sshbin/internal/sharing"
)

// Password grants are stateless: a correct password sets a cookie holding an
// HMAC of the share ID under the server secret. The cookie cannot be forged
// without the secret, and is scoped (by path) to the one share.

func grantCookieName(id string) string {
	return "fd_pw_" + id
}

func (h *handler) grantValue(id string) string {
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(id))
	return hex.EncodeToString(mac.Sum(nil))
}

func (h *handler) setPasswordGrant(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     grantCookieName(id),
		Value:    h.grantValue(id),
		Path:     "/s/" + id,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// hasPasswordGrant reports whether the request may bypass the password prompt:
// either the share has no password, or it carries a valid grant cookie.
func (h *handler) hasPasswordGrant(r *http.Request, s sharing.Sharing) bool {
	if !s.HasPassword() {
		return true
	}
	c, err := r.Cookie(grantCookieName(s.ID))
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(c.Value), []byte(h.grantValue(s.ID)))
}
