package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rispycz/sshbin/internal/auth"
	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
	"github.com/rispycz/sshbin/internal/userprefs"
)

type testSender struct{ code string }

func (s *testSender) Send(_ context.Context, _, code string) error {
	s.code = code
	return nil
}

func newTestHandler(t *testing.T, repo sharing.Repository) (*handler, *testSender) {
	t.Helper()
	tpl, err := parseTemplates()
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	sender := &testSender{}
	h := &handler{
		repo:    repo,
		storage: &storage.LocalStorage{BaseDir: t.TempDir()},
		auth:    auth.NewManager(sender, auth.NewMemorySessionStore(), auth.Options{}),
		prefs:   userprefs.NewMemoryRepository(),
		baseURL: "http://example.com",
		host:    "example.com",
		secret:  []byte("test-secret-32-bytes-padding-xxx"),
		tpl:     tpl,
	}
	return h, sender
}

// seedFile writes a stored file for a share so downloads have content.
func seedFile(t *testing.T, h *handler, id, name, content string) {
	t.Helper()
	w, err := h.storage.Create(context.Background(), id, name)
	if err != nil {
		t.Fatalf("storage.Create: %v", err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("write: %v", err)
	}
	w.Close()
}

func login(t *testing.T, h *handler, sender *testSender, email string) *http.Cookie {
	t.Helper()
	if err := h.auth.StartLogin(context.Background(), email); err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	token, _, err := h.auth.Verify(email, sender.code)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return &http.Cookie{Name: sessionCookie, Value: token}
}

func TestRequireSession_RedirectsAnonymous(t *testing.T) {
	h, _ := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("GET", "/shares/abc/qr", nil)
	rec := httptest.NewRecorder()

	called := false
	h.requireSession(func(http.ResponseWriter, *http.Request) { called = true })(rec, req)
	if called {
		t.Error("wrapped handler should not run for anonymous request")
	}
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 redirect", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/login?next=") {
		t.Errorf("Location = %q, want /login redirect", loc)
	}
}

func TestShareView_PublicNoSession(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", Configured: true, Public: true})
	h, _ := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/s/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.shareView(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for public share", rec.Code)
	}
}

func TestShareView_PrivateRedirectsAnonymous(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", Configured: true})
	h, _ := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/s/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.shareView(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 redirect to login", rec.Code)
	}
}

func TestShareView_PrivateNotAllowed(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{
		ID: "abc", FileName: "f.txt", Configured: true,
		OwnerEmail: "owner@example.com", AllowedEmails: []string{"friend@example.com"},
	})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/s/abc", nil)
	req.AddCookie(login(t, h, sender, "stranger@example.com"))
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.shareView(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for non-allowlisted email", rec.Code)
	}
}

func TestShareView_PrivateAllowed(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{
		ID: "abc", FileName: "f.txt", Configured: true,
		OwnerEmail: "owner@example.com", AllowedEmails: []string{"friend@example.com"},
	})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/s/abc", nil)
	req.AddCookie(login(t, h, sender, "friend@example.com"))
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.shareView(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for allowlisted email", rec.Code)
	}
}

func TestSharePassword_Gate(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	hash, _ := sharing.HashPassword("open-sesame")
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", Configured: true, Public: true, PasswordHash: hash})
	h, _ := newTestHandler(t, repo)

	// GET shows the password prompt, not the file.
	getReq := httptest.NewRequest("GET", "/s/abc", nil)
	getReq.SetPathValue("id", "abc")
	getRec := httptest.NewRecorder()
	h.shareView(getRec, getReq)
	if !strings.Contains(getRec.Body.String(), "Password required") {
		t.Error("expected password prompt on GET")
	}

	// Wrong password -> 401.
	bad := url.Values{"password": {"nope"}}
	badReq := httptest.NewRequest("POST", "/s/abc", strings.NewReader(bad.Encode()))
	badReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badReq.SetPathValue("id", "abc")
	badRec := httptest.NewRecorder()
	h.sharePassword(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want 401", badRec.Code)
	}

	// Correct password -> grant cookie + redirect (PRG).
	ok := url.Values{"password": {"open-sesame"}}
	okReq := httptest.NewRequest("POST", "/s/abc", strings.NewReader(ok.Encode()))
	okReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	okReq.SetPathValue("id", "abc")
	okRec := httptest.NewRecorder()
	h.sharePassword(okRec, okReq)
	if okRec.Code != http.StatusSeeOther {
		t.Fatalf("correct password status = %d, want 303 redirect", okRec.Code)
	}
	grant := grantCookieFrom(t, okRec)

	// With the grant cookie, GET shows the share page, not the prompt.
	viewReq := httptest.NewRequest("GET", "/s/abc", nil)
	viewReq.AddCookie(grant)
	viewReq.SetPathValue("id", "abc")
	viewRec := httptest.NewRecorder()
	h.shareView(viewRec, viewReq)
	if body := viewRec.Body.String(); !strings.Contains(body, "Shared file") || strings.Contains(body, "Unlock") {
		t.Error("grant cookie should reveal the share page without re-prompt")
	}
}

func grantCookieFrom(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if strings.HasPrefix(c.Name, "fd_pw_") {
			return c
		}
	}
	t.Fatal("no grant cookie set")
	return nil
}

func TestDownload_PublicNoPassword(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileID: "abc", FileName: "f.txt", Configured: true, Public: true})
	h, _ := newTestHandler(t, repo)
	seedFile(t, h, "abc", "f.txt", "hello bytes")

	req := httptest.NewRequest("GET", "/s/abc/download", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.download(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "hello bytes" {
		t.Errorf("body = %q, want file content", rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "f.txt") {
		t.Errorf("Content-Disposition = %q", cd)
	}
}

func TestDownload_PasswordRedirectsWithoutGrant(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	hash, _ := sharing.HashPassword("pw")
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileID: "abc", FileName: "f.txt", Configured: true, Public: true, PasswordHash: hash})
	h, _ := newTestHandler(t, repo)
	seedFile(t, h, "abc", "f.txt", "secret")

	req := httptest.NewRequest("GET", "/s/abc/download", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.download(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 redirect to prompt", rec.Code)
	}
}

func TestDownload_PasswordWithGrant(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	hash, _ := sharing.HashPassword("pw")
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileID: "abc", FileName: "f.txt", Configured: true, Public: true, PasswordHash: hash})
	h, _ := newTestHandler(t, repo)
	seedFile(t, h, "abc", "f.txt", "secret")

	req := httptest.NewRequest("GET", "/s/abc/download", nil)
	req.AddCookie(&http.Cookie{Name: grantCookieName("abc"), Value: h.grantValue("abc")})
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.download(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with valid grant", rec.Code)
	}
	if rec.Body.String() != "secret" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestDownload_ForgedGrantRejected(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	hash, _ := sharing.HashPassword("pw")
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileID: "abc", FileName: "f.txt", Configured: true, Public: true, PasswordHash: hash})
	h, _ := newTestHandler(t, repo)
	seedFile(t, h, "abc", "f.txt", "secret")

	req := httptest.NewRequest("GET", "/s/abc/download", nil)
	req.AddCookie(&http.Cookie{Name: grantCookieName("abc"), Value: "deadbeef"})
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.download(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("forged grant status = %d, want 303 redirect", rec.Code)
	}
}

func TestContentDisposition(t *testing.T) {
	if got := contentDisposition("../../etc/passwd"); !strings.Contains(got, "passwd") || strings.Contains(got, "/") {
		t.Errorf("path not stripped: %q", got)
	}
	if got := contentDisposition("a\r\nb.txt"); strings.ContainsAny(got, "\r\n") {
		t.Errorf("control chars not stripped: %q", got)
	}
}

func TestShareView_Expired(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	past := time.Now().Add(-time.Hour)
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", Configured: true, Public: true, ExpiresAt: &past})
	h, _ := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/s/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.shareView(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410 for expired share", rec.Code)
	}
}

