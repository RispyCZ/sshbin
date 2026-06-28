package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rispycz/sshbin/internal/sharing"
)

func TestAPISession_Unauthenticated(t *testing.T) {
	h, _ := newTestHandler(t, sharing.NewMemoryRepository())
	rec := httptest.NewRecorder()
	h.apiSession(rec, httptest.NewRequest("GET", "/api/session", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPISession_Authenticated(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("GET", "/api/session", nil)
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.apiSession(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct{ Email string }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Email != "owner@example.com" {
		t.Errorf("email = %q, want owner@example.com", body.Email)
	}
}

func TestAPIShares_ReturnsOwnerSharesAsJSON(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{
		ID: "abc", FileName: "f.txt", OwnerEmail: "owner@example.com",
		Configured: true, Public: true, CreatedAt: time.Now(),
	})
	repo.Create(context.Background(), sharing.Sharing{
		ID: "other", FileName: "g.txt", OwnerEmail: "someone@example.com",
		Configured: true, Public: true, CreatedAt: time.Now(),
	})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/api/shares", nil)
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.requireSessionAPI(h.apiShares)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []shareDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (only owner's share)", len(got))
	}
	if got[0].ID != "abc" || got[0].ShareURL != "http://example.com/s/abc" {
		t.Errorf("unexpected share %+v", got[0])
	}
	if got[0].AllowedEmails == nil {
		t.Error("allowedEmails should serialize as [] not null")
	}
}

func TestAPIShares_Unauthenticated(t *testing.T) {
	h, _ := newTestHandler(t, sharing.NewMemoryRepository())
	rec := httptest.NewRecorder()
	h.requireSessionAPI(h.apiShares)(rec, httptest.NewRequest("GET", "/api/shares", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIDeleteShare_Owner(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", OwnerEmail: "owner@example.com"})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("DELETE", "/api/shares/abc", nil)
	req.SetPathValue("id", "abc")
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.apiDeleteShare(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if _, err := repo.Get(context.Background(), "abc"); err == nil {
		t.Error("share should be deleted")
	}
}

func TestAPIDeleteShare_NotOwner(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", OwnerEmail: "owner@example.com"})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("DELETE", "/api/shares/abc", nil)
	req.SetPathValue("id", "abc")
	req.AddCookie(login(t, h, sender, "intruder@example.com"))
	rec := httptest.NewRecorder()

	h.apiDeleteShare(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAPIDeleteShare_NotFound(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("DELETE", "/api/shares/missing", nil)
	req.SetPathValue("id", "missing")
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.apiDeleteShare(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAPILogin_SendsCode(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"email":"user@example.com"}`))
	rec := httptest.NewRecorder()

	h.apiLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if sender.code == "" {
		t.Error("login should send a code")
	}
	var body struct{ MaskedEmail string }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasSuffix(body.MaskedEmail, "@example.com") {
		t.Errorf("maskedEmail = %q, want masked local part", body.MaskedEmail)
	}
}

func TestAPILogin_EmptyEmail(t *testing.T) {
	h, _ := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"email":"  "}`))
	rec := httptest.NewRecorder()

	h.apiLogin(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAPIVerify_SetsCookie(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	if err := h.auth.StartLogin(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	body := `{"email":"user@example.com","code":"` + sender.code + `"}`
	req := httptest.NewRequest("POST", "/api/verify", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.apiVerify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), sessionCookie) {
		t.Error("verify should set session cookie")
	}
}

func TestAPIVerify_BadCode(t *testing.T) {
	h, _ := newTestHandler(t, sharing.NewMemoryRepository())
	if err := h.auth.StartLogin(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/verify", strings.NewReader(`{"email":"user@example.com","code":"000000"}`))
	rec := httptest.NewRecorder()

	h.apiVerify(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPISetup_PersistsAndClaims(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	h, sender := newTestHandler(t, repo)

	body := `{"expires":"24h","visibility":"private","emails":["Alice@Example.com","bob@example.com"],"password":"secret"}`
	req := httptest.NewRequest("POST", "/api/setup/abc", strings.NewReader(body))
	req.SetPathValue("id", "abc")
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.apiSetup(rec, req)
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
		t.Error("password not stored/hashed")
	}
	if got.ExpiresAt == nil {
		t.Error("24h preset should set an expiry")
	}

	var dto shareDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !dto.HasPassword || dto.Public {
		t.Errorf("response DTO not reflecting saved state: %+v", dto)
	}
}

func TestAPISetup_PublicNever(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("POST", "/api/setup/abc", strings.NewReader(`{"expires":"never","visibility":"public"}`))
	req.SetPathValue("id", "abc")
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.apiSetup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got, _ := repo.Get(context.Background(), "abc")
	if !got.Configured || !got.Public || got.ExpiresAt != nil {
		t.Errorf("expected configured public never-expiring share, got %+v", got)
	}
}

func TestAPISetup_ForbiddenNonOwner(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", OwnerEmail: "owner@example.com", Configured: true})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("POST", "/api/setup/abc", strings.NewReader(`{"expires":"never","visibility":"public"}`))
	req.SetPathValue("id", "abc")
	req.AddCookie(login(t, h, sender, "intruder@example.com"))
	rec := httptest.NewRecorder()

	h.apiSetup(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAPISetup_NotFound(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("POST", "/api/setup/missing", strings.NewReader(`{"expires":"never","visibility":"public"}`))
	req.SetPathValue("id", "missing")
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.apiSetup(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAPIProfile_GetAndSave(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	cookie := login(t, h, sender, "owner@example.com")

	get := httptest.NewRequest("GET", "/api/profile", nil)
	get.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	h.apiProfileGet(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getRec.Code)
	}
	var got struct {
		Email         string
		DefaultPublic bool
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Email != "owner@example.com" || got.DefaultPublic {
		t.Errorf("unexpected profile %+v", got)
	}

	save := httptest.NewRequest("PUT", "/api/profile", strings.NewReader(`{"defaultPublic":true}`))
	save.AddCookie(cookie)
	saveRec := httptest.NewRecorder()
	h.apiProfileSave(saveRec, save)
	if saveRec.Code != http.StatusNoContent {
		t.Fatalf("save status = %d, want 204", saveRec.Code)
	}
	prefs, _ := h.prefs.Get(context.Background(), "owner@example.com")
	if !prefs.DefaultPublic {
		t.Error("DefaultPublic not persisted")
	}
}

func TestAPIProfile_DeleteAll(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", OwnerEmail: "owner@example.com"})
	h, sender := newTestHandler(t, repo)

	req := httptest.NewRequest("DELETE", "/api/profile", nil)
	req.AddCookie(login(t, h, sender, "owner@example.com"))
	rec := httptest.NewRecorder()

	h.apiProfileDeleteAll(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if shares, _ := repo.ListByOwner(context.Background(), "owner@example.com"); len(shares) != 0 {
		t.Error("owner shares should be deleted")
	}
	if sc := rec.Header().Get("Set-Cookie"); !strings.Contains(sc, sessionCookie) {
		t.Error("session cookie should be cleared")
	}
}
