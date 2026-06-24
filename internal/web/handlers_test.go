package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rispycz/securedrop/internal/auth"
	"github.com/rispycz/securedrop/internal/sharing"
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
		auth:    auth.NewManager(sender, auth.Options{}),
		baseURL: "http://example.com",
		host:    "example.com",
		tpl:     tpl,
	}
	return h, sender
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

	// Correct password -> share page.
	ok := url.Values{"password": {"open-sesame"}}
	okReq := httptest.NewRequest("POST", "/s/abc", strings.NewReader(ok.Encode()))
	okReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	okReq.SetPathValue("id", "abc")
	okRec := httptest.NewRecorder()
	h.sharePassword(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("correct password status = %d, want 200", okRec.Code)
	}
	body := okRec.Body.String()
	if !strings.Contains(body, "Shared file") {
		t.Error("expected share page after correct password")
	}
	if strings.Contains(body, "Unlock") {
		t.Error("should not re-prompt (Unlock button present) after correct password")
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
	if loc := rec.Header().Get("Location"); loc != "/setup/abc" {
		t.Errorf("Location = %q, want /setup/abc", loc)
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
