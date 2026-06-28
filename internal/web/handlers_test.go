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

func TestIndex(t *testing.T) {
	h, _ := newTestHandler(t, sharing.NewMemoryRepository())
	rec := httptest.NewRecorder()
	h.index(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "example.com:") {
		t.Error("index missing scp host hint")
	}
}

func TestRequireSession_RedirectsAnonymous(t *testing.T) {
	h, _ := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("GET", "/setup/abc", nil)
	rec := httptest.NewRecorder()

	h.requireSession(h.setupGet)(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 redirect", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/login?next=") {
		t.Errorf("Location = %q, want /login redirect", loc)
	}
}

func TestSetupGet_NotFound(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("GET", "/setup/missing", nil)
	req.AddCookie(login(t, h, sender, "u@e.com"))
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.setupGet(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestSetupPost_PersistsAndClaims(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	h, sender := newTestHandler(t, repo)

	form := url.Values{
		"expires":    {"24h"},
		"visibility": {"private"},
		"emails":     {"Alice@Example.com, bob@example.com"},
		"password":   {"secret"},
	}
	req := httptest.NewRequest("POST", "/setup/abc", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.setupPost(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	got, _ := repo.Get(context.Background(), "abc")
	if !got.Configured || got.Public {
		t.Errorf("expected configured private share, got %+v", got)
	}
	if got.OwnerEmail != "owner@example.com" {
		t.Errorf("OwnerEmail = %q", got.OwnerEmail)
	}
	if !got.AllowsEmail("alice@example.com") || !got.AllowsEmail("bob@example.com") {
		t.Errorf("allowlist not parsed: %v", got.AllowedEmails)
	}
	if !got.CheckPassword("secret") {
		t.Error("password not stored/hashed correctly")
	}
	if !strings.Contains(rec.Body.String(), "http://example.com/s/abc") {
		t.Error("response missing share URL")
	}
}

func TestSetupPost_FromShares_Redirects(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	h, sender := newTestHandler(t, repo)

	form := url.Values{
		"expires":    {"never"},
		"visibility": {"public"},
		"from":       {"shares"},
	}
	req := httptest.NewRequest("POST", "/setup/abc", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.setupPost(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/shares" {
		t.Errorf("Location = %q, want /shares", loc)
	}

	got, _ := repo.Get(context.Background(), "abc")
	if !got.Configured || !got.Public || got.ExpiresAt != nil {
		t.Errorf("expected configured public never-expiring share, got %+v", got)
	}
}

func TestSharesList_IncludesEditData(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{
		ID: "abc", FileName: "f.txt", OwnerEmail: "owner@example.com",
		Configured: true, Public: true, CreatedAt: time.Now(),
	})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/shares", nil)
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.sharesList(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<sb-setup-modal") {
		t.Error("response missing sb-setup-modal")
	}
	if !strings.Contains(body, `expires="never"`) {
		t.Error("response missing expires bucket")
	}
	if !strings.Contains(body, "configured") {
		t.Error("response missing configured flag")
	}
}

func TestSetup_ForbiddenForNonOwner(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", OwnerEmail: "owner@example.com", Configured: true})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/setup/abc", nil)
	req.AddCookie(login(t, h, sender, "intruder@example.com"))
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.setupGet(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
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

func TestVerifyPost_SetsSessionAndRedirects(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	h.auth.StartLogin(context.Background(), "u@e.com")

	form := url.Values{"email": {"u@e.com"}, "code": {sender.code}, "next": {"/setup/abc"}}
	req := httptest.NewRequest("POST", "/verify", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.verifyPost(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/setup/abc?flash=signed_in" {
		t.Errorf("Location = %q, want /setup/abc?flash=signed_in", loc)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), sessionCookie) {
		t.Error("session cookie not set")
	}
}

func TestParseExpiry(t *testing.T) {
	now := time.Now()
	if parseExpiry("never", now) != nil {
		t.Error(`"never" should yield nil`)
	}
	if parseExpiry("bogus", now) != nil {
		t.Error("unknown preset should yield nil")
	}
	got := parseExpiry("1h", now)
	if got == nil || !got.Equal(now.Add(time.Hour)) {
		t.Errorf(`"1h" = %v, want %v`, got, now.Add(time.Hour))
	}
}

func TestSafeNext(t *testing.T) {
	cases := map[string]string{
		"/setup/x":       "/setup/x",
		"":               "/",
		"//evil.com":     "/",
		"https://evil":   "/",
		"javascript:foo": "/",
	}
	for in, want := range cases {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", in, got, want)
		}
	}
}
