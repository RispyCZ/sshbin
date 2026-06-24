package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rispycz/securedrop/internal/sharing"
)

func newTestHandler(t *testing.T, repo sharing.Repository) *handler {
	t.Helper()
	tpl, err := parseTemplates()
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	return &handler{repo: repo, baseURL: "http://example.com", host: "example.com", tpl: tpl}
}

func TestIndex(t *testing.T) {
	h := newTestHandler(t, sharing.NewMemoryRepository())
	rec := httptest.NewRecorder()
	h.index(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "example.com:") {
		t.Error("index missing scp host hint")
	}
}

func TestSetupGet_NotFound(t *testing.T) {
	h := newTestHandler(t, sharing.NewMemoryRepository())
	req := httptest.NewRequest("GET", "/setup/missing", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.setupGet(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestSetupPost_PersistsSettings(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	h := newTestHandler(t, repo)

	form := url.Values{"expires": {"24h"}, "public": {"on"}, "password": {"secret"}}
	req := httptest.NewRequest("POST", "/setup/abc", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.setupPost(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	got, _ := repo.Get(context.Background(), "abc")
	if !got.Configured {
		t.Error("Configured not set")
	}
	if !got.Public {
		t.Error("Public not set")
	}
	if got.ExpiresAt == nil {
		t.Error("ExpiresAt not set")
	}
	if !got.CheckPassword("secret") {
		t.Error("password not stored/hashed correctly")
	}
	if !strings.Contains(rec.Body.String(), "http://example.com/s/abc") {
		t.Error("response missing share URL")
	}
}

func TestShareView_Unconfigured(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	h := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/s/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.shareView(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for unconfigured share", rec.Code)
	}
}

func TestShareView_Expired(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	past := time.Now().Add(-time.Hour)
	repo.Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", Configured: true, ExpiresAt: &past})
	h := newTestHandler(t, repo)

	req := httptest.NewRequest("GET", "/s/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.shareView(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410 for expired share", rec.Code)
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
